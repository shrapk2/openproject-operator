package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"
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

	original := ci.DeepCopy()

	// Avoid retry storm on repeated failure
	if ci.Status.LastFailedTime != nil && time.Since(ci.Status.LastFailedTime.Time) < 30*time.Second && !ci.Status.LastRunSuccess {
		log.Info("🛑 Skipping AWS inventory due to recent failure, requeued")
		return ctrl.Result{RequeueAfter: 30 * time.Minute}, nil
	}

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
		log.Error(err, "Failed to build AWS config")
		ci.Status.LastFailedTime = &metav1.Time{Time: time.Now()}
		_ = r.Status().Patch(ctx, ci, client.MergeFrom(original))
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// summary counts and raw inventories
	summary := make(map[string]int)
	rawResults := make(map[string]interface{})

	// Loop through monitored services defined in common.go
	for _, svc := range ci.Spec.AWS.Resources {
		lower := strings.ToLower(svc)
		handled := false
		for _, info := range monitoredServices {
			if strings.ToLower(info.Name) == lower {
				items, err := info.Inventory(ctx, *awsConfig, ci.Spec.AWS.TagFilter)
				if err != nil {
					log.Error(err, fmt.Sprintf("%s inventory failed", info.Name))
					ci.Status.LastFailedTime = &metav1.Time{Time: time.Now()}
					_ = r.Status().Patch(ctx, ci, client.MergeFrom(original))
					return ctrl.Result{RequeueAfter: 30 * time.Minute}, nil
				}
				summary[lower] = info.Count(items)
				rawResults[lower] = items
				handled = true
				break
			}
		}
		if !handled {
			summary[lower] = simulateAWSInventory(lower)
		}
	}

	// Convert raw results into v1alpha1 types
	converted := make(map[string]interface{})
	for _, info := range monitoredServices {
		key := strings.ToLower(info.Name)
		if raw, ok := rawResults[key]; ok {
			converted[key] = info.Convert(raw)
		}
	}

	// Fetch the most recent report
	var reports v1alpha1.CloudInventoryReportList
	if err := r.List(ctx, &reports, client.InNamespace(ci.Namespace)); err == nil {
		var latestReport *v1alpha1.CloudInventoryReport
		for _, rep := range reports.Items {
			if rep.Spec.SourceRef.Name == ci.Name {
				if latestReport == nil || rep.CreationTimestamp.After(latestReport.CreationTimestamp.Time) {
					latestReport = &rep
				}
			}
		}
		if latestReport != nil {
			if time.Since(latestReport.CreationTimestamp.Time) < 2*time.Minute {
				log.Info("🛑 Skipping report creation — last report is recent", "age", time.Since(latestReport.CreationTimestamp.Time))
				ci.Status.LastRunTime = metav1.Now()
				ci.Status.Message = "Recent inventory already reported"
				_ = r.Status().Patch(ctx, ci, client.MergeFrom(original))
				return ctrl.Result{}, nil
			}
		}
	}

	// Create the CloudInventoryReport without Status first
	report := &v1alpha1.CloudInventoryReport{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: ci.Name + "-" + "aws" + "-",
			Namespace:    ci.Namespace,
		},
		Spec: v1alpha1.CloudInventoryReportSpec{
			SourceRef: corev1.ObjectReference{
				Name:       ci.Name,
				Namespace:  ci.Namespace,
				Kind:       "CloudInventory",
				APIVersion: "openproject.org/v1alpha1",
			},
			Timestamp: metav1.Now(),
		},
	}

	// Create the report resource first
	if err := r.Create(ctx, report); err != nil {
		log.Error(err, "Failed to create CloudInventoryReport")
		ci.Status.LastFailedTime = &metav1.Time{Time: time.Now()}
		_ = r.Status().Patch(ctx, ci, client.MergeFrom(original))
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	log.Info("Created empty CloudInventoryReport", "name", report.Name)

	reportCopy := report.DeepCopy()
	report.Status = v1alpha1.CloudInventoryReportStatus{
		Summary: summary,
	}
	reportStatusValue := reflect.ValueOf(&report.Status).Elem()
	for _, svc := range monitoredServices {
		key := strings.ToLower(svc.Name)
		if data, exists := converted[key]; exists {
			// Get the field by name (EC2, RDS, ELBV2, etc.)
			field := reportStatusValue.FieldByName(svc.Name)

			// Skip if the field doesn't exist in the struct
			if !field.IsValid() {
				log.Info("Skipping unknown field in report status", "field", svc.Name)
				continue
			}

			// Convert data to reflect.Value and set it on the field
			dataValue := reflect.ValueOf(data)
			if field.Type() == dataValue.Type() {
				field.Set(dataValue)
			} else {
				log.Info("Type mismatch for field", "field", svc.Name,
					"expectedType", field.Type().String(),
					"gotType", dataValue.Type().String())
			}
		}
	}

	if err := r.Status().Patch(ctx, report, client.MergeFrom(reportCopy)); err != nil {
		log.Error(err, "failed to patch CloudInventoryReport status", "name", report.Name)
		ci.Status.LastFailedTime = &metav1.Time{Time: time.Now()}
		_ = r.Status().Patch(ctx, ci, client.MergeFrom(original))
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	log.Info("✅ CloudInventoryReport created and status patched", "name", report.Name)

	ci.Status.LastFailedTime = nil
	ci.Status.LastRunTime = metav1.Now()
	ci.Status.LastRunSuccess = true
	ci.Status.ItemCount = 0
	for _, cnt := range summary {
		ci.Status.ItemCount += cnt
	}
	ci.Status.Summary = summary
	ci.Status.Message = fmt.Sprintf("AWS Inventory complete for account %s", accountID)

	log.Info("AWS Inventory Summary", "summary", ci.Status.Summary)

	if err := r.Status().Patch(ctx, ci, client.MergeFrom(original)); err != nil {
		log.Error(err, "Failed to patch AWS inventory status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
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
			config.WithRetryer(func() aws.Retryer {
				return awsinventory.NewCustomRetryer()
			}),
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
