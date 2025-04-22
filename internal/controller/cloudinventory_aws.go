package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awsinventory "github.com/shrapk2/openproject-operator/internal/controller/aws_sdk"

	"github.com/go-logr/logr"
	v1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcileAWS handles AWS inventory logic
func (r *CloudInventoryReconciler) reconcileAWS(ctx context.Context, ci *v1alpha1.CloudInventory, log logr.Logger) (ctrl.Result, error) {
	if ci.Spec.AWS == nil {
		err := fmt.Errorf("AWS config is missing in spec")
		log.Error(err, "cannot reconcile AWS")
		return ctrl.Result{}, err
	}

	log.Info("Starting AWS Cloud Inventory",
		"resources", ci.Spec.AWS.Resources,
	)

	awsConfig, accountID, err := r.buildAWSConfig(ctx, ci, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Placeholder: call out to real inventory functions per service
	summary := make(map[string]int)
	var ec2Instances []awsinventory.EC2InstanceInfo

	for _, svc := range ci.Spec.AWS.Resources {
		switch svc {
		case "ec2":
			var err error
			ec2Instances, err = awsinventory.InventoryEC2Instances(ctx, *awsConfig, ci.Spec.AWS.TagFilter)
			if err != nil {
				log.Error(err, "EC2 inventory failed")
				continue
			}
			summary["ec2"] = len(ec2Instances)
		default:
			summary[svc] = simulateAWSInventory(svc)
		}
	}

	itemCount := 0
	for _, count := range summary {
		itemCount += count
	}

	// Patch status
	original := ci.DeepCopy()
	// Convert ec2Instances to the appropriate type
	var convertedEC2Instances []v1alpha1.EC2InstanceInfo
	for _, instance := range ec2Instances {
		convertedEC2Instances = append(convertedEC2Instances, v1alpha1.EC2InstanceInfo{
			Name:       instance.Name,
			InstanceID: instance.InstanceID,
			State:      instance.State,
			Type:       instance.Type,
			AZ:         instance.AZ,
			Platform:   instance.Platform,
			PublicIP:   instance.PublicIP,
			PrivateDNS: instance.PrivateDNS,
			PrivateIP:  instance.PrivateIP,
			ImageID:    instance.ImageID,
			VPCID:      instance.VPCID,
			Tags:       instance.Tags,
		})

	}
	ci.Status.EC2 = convertedEC2Instances
	ci.Status.ItemCount = itemCount
	ci.Status.Summary = summary
	ci.Status.LastRunTime = metav1.Now()
	ci.Status.Message = fmt.Sprintf("AWS Inventory complete for account %s", accountID)

	if err := r.Status().Patch(ctx, ci, client.MergeFrom(original)); err != nil {
		log.Error(err, "failed to patch EC2 inventory status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

// buildAWSConfig constructs an AWS config from a secret or IRSA
func (r *CloudInventoryReconciler) buildAWSConfig(ctx context.Context, ci *v1alpha1.CloudInventory, log logr.Logger) (*aws.Config, string, error) {
	if ci.Spec.AWS.CredentialsSecretRef != nil {
		// Load static credentials from secret
		secret := &corev1.Secret{}
		secretKey := ci.Spec.AWS.CredentialsSecretRef

		err := r.Get(ctx, client.ObjectKey{
			Name:      secretKey.Name,
			Namespace: ci.Namespace,
		}, secret)
		if err != nil {
			return nil, "", fmt.Errorf("failed to get AWS credentials secret: %w", err)
		}

		accessKey := string(secret.Data["aws_access_key_id"])
		secretKeyVal := string(secret.Data["aws_secret_access_key"])
		region := string(secret.Data["aws_region"])
		account := string(secret.Data["aws_account_id"])

		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(accessKey, secretKeyVal, ""),
			),
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load AWS config: %w", err)
		}
		log.Info("Loaded static AWS credentials", "region", region, "account", account)
		return &cfg, account, nil
	}

	// Fall back to IRSA or environment
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load default AWS config: %w", err)
	}

	// Optionally assume a role
	if ci.Spec.AWS.AssumeRoleARN != "" {
		stsSvc := sts.NewFromConfig(cfg)
		resp, err := stsSvc.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         aws.String(ci.Spec.AWS.AssumeRoleARN),
			RoleSessionName: aws.String("openproject-inventory"),
		})
		if err != nil {
			return nil, "", fmt.Errorf("failed to assume role: %w", err)
		}

		cfg.Credentials = credentials.NewStaticCredentialsProvider(
			*resp.Credentials.AccessKeyId,
			*resp.Credentials.SecretAccessKey,
			*resp.Credentials.SessionToken,
		)

		log.Info("Assumed AWS role", "role", ci.Spec.AWS.AssumeRoleARN)
	}

	// Get account info for status
	stsSvc := sts.NewFromConfig(cfg)
	caller, err := stsSvc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get caller identity: %w", err)
	}

	return &cfg, *caller.Account, nil
}

// simulateAWSInventory is a placeholder for service-specific logic
func simulateAWSInventory(service string) int {
	switch service {
	case "ec2":
		return 3
	case "rds":
		return 1
	case "elbv2":
		return 2
	default:
		return 0
	}
}
