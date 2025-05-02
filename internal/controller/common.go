package controller

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	v1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
	awsinventory "github.com/shrapk2/openproject-operator/internal/controller/aws_sdk"
)

// ServiceInfo defines metadata about a cloud service for inventory reporting
type ServiceInfo struct {
	Name      string
	GetCount  func(*v1alpha1.CloudInventoryReport) int
	LogName   string
	Inventory func(ctx context.Context, cfg aws.Config, tagFilter string) (interface{}, error)
	Count     func(items interface{}) int
	Convert   func(items interface{}) interface{}
}

var (
	// Debug mode configuration
	debugEnabled = os.Getenv("DEBUG") == "true"
)

// monitoredServices defines all cloud services monitored by the operator
var monitoredServices = []ServiceInfo{
	{
		Name:     "EC2",
		GetCount: func(r *v1alpha1.CloudInventoryReport) int { return len(r.Status.EC2) },
		LogName:  "ec2Count",
		// Add inventory operations
		Inventory: func(ctx context.Context, cfg aws.Config, tagFilter string) (interface{}, error) {
			return awsinventory.InventoryEC2Instances(ctx, cfg, tagFilter)
		},
		Count: func(items interface{}) int {
			return len(items.([]awsinventory.EC2InstanceInfo))
		},
		Convert: func(items interface{}) interface{} {
			rawItems := items.([]awsinventory.EC2InstanceInfo)
			result := make([]v1alpha1.EC2InstanceInfo, len(rawItems))
			for i, item := range rawItems {
				result[i] = v1alpha1.EC2InstanceInfo(item)
			}
			return result
		},
	},
	{
		Name:     "RDS",
		GetCount: func(r *v1alpha1.CloudInventoryReport) int { return len(r.Status.RDS) },
		LogName:  "rdsCount",
		// Add inventory operations
		Inventory: func(ctx context.Context, cfg aws.Config, tagFilter string) (interface{}, error) {
			return awsinventory.InventoryRDSInstances(ctx, cfg, tagFilter)
		},
		Count: func(items interface{}) int {
			return len(items.([]awsinventory.RDSInstanceInfo))
		},
		Convert: func(items interface{}) interface{} {
			rawItems := items.([]awsinventory.RDSInstanceInfo)
			result := make([]v1alpha1.RDSInstanceInfo, len(rawItems))
			for i, item := range rawItems {
				result[i] = v1alpha1.RDSInstanceInfo(item)
			}
			return result
		},
	},
	{
		Name:     "ELBV2",
		GetCount: func(r *v1alpha1.CloudInventoryReport) int { return len(r.Status.ELBV2) },
		LogName:  "elbv2Count",
		// Add inventory operations
		Inventory: func(ctx context.Context, cfg aws.Config, tagFilter string) (interface{}, error) {
			return awsinventory.InventoryALBs(ctx, cfg, tagFilter)
		},
		Count: func(items interface{}) int {
			return len(items.([]awsinventory.ELBV2Info))
		},
		Convert: func(items interface{}) interface{} {
			rawItems := items.([]awsinventory.ELBV2Info)
			result := make([]v1alpha1.ELBV2Info, len(rawItems))
			for i, item := range rawItems {
				result[i] = v1alpha1.ELBV2Info(item)
			}
			return result
		},
	},
	{
		Name:     "S3",
		GetCount: func(r *v1alpha1.CloudInventoryReport) int { return len(r.Status.S3) },
		LogName:  "s3Count",
		Inventory: func(ctx context.Context, cfg aws.Config, tagFilter string) (interface{}, error) {
			return awsinventory.InventoryS3Buckets(ctx, cfg, tagFilter)
		},
		Count: func(items interface{}) int {
			return len(items.([]awsinventory.S3BucketInfo))
		},
		Convert: func(items interface{}) interface{} {
			rawItems := items.([]awsinventory.S3BucketInfo)
			result := make([]v1alpha1.S3BucketInfo, len(rawItems))
			for i, item := range rawItems {
				result[i] = v1alpha1.S3BucketInfo(item)
			}
			return result
		},
	},
	{
		Name:     "EIP",
		GetCount: func(r *v1alpha1.CloudInventoryReport) int { return len(r.Status.EIP) },
		LogName:  "eipCount",
		Inventory: func(ctx context.Context, cfg aws.Config, tagFilter string) (interface{}, error) {
			return awsinventory.InventoryEIPs(ctx, cfg, tagFilter)
		},
		Count: func(items interface{}) int {
			return len(items.([]awsinventory.EIPInfo))
		},
		Convert: func(items interface{}) interface{} {
			raw := items.([]awsinventory.EIPInfo)
			out := make([]v1alpha1.EIPInfo, len(raw))
			for i, v := range raw {
				out[i] = v1alpha1.EIPInfo(v)
			}
			return out
		},
	},
	{
		Name:     "ECR",
		GetCount: func(r *v1alpha1.CloudInventoryReport) int { return len(r.Status.ECR) },
		LogName:  "ecrCount",
		Inventory: func(ctx context.Context, cfg aws.Config, tagFilter string) (interface{}, error) {
			return awsinventory.InventoryECRRepositories(ctx, cfg, tagFilter)
		},
		Count: func(items interface{}) int {
			return len(items.([]awsinventory.ECRRegistryInfo))
		},
		Convert: func(items interface{}) interface{} {
			raw := items.([]awsinventory.ECRRegistryInfo)
			out := make([]v1alpha1.ECRRegistryInfo, len(raw))
			for i, v := range raw {
				out[i] = v1alpha1.ECRRegistryInfo(v)
			}
			return out
		},
	},
	{
		Name:     "NATGateways",
		GetCount: func(r *v1alpha1.CloudInventoryReport) int { return len(r.Status.NATGateways) },
		LogName:  "natGatewayCount",
		Inventory: func(ctx context.Context, cfg aws.Config, tagFilter string) (interface{}, error) {
			return awsinventory.InventoryNATGateways(ctx, cfg, tagFilter)
		},
		Count: func(items interface{}) int {
			return len(items.([]awsinventory.NATGatewayInfo))
		},
		Convert: func(items interface{}) interface{} {
			raw := items.([]awsinventory.NATGatewayInfo)
			out := make([]v1alpha1.NATGatewayInfo, len(raw))
			for i, v := range raw {
				out[i] = v1alpha1.NATGatewayInfo{
					NatGatewayId: v.NatGatewayID, // map ID→Id
					VpcId:        v.VpcID,
					SubnetId:     v.SubnetID,
					State:        v.State,
					Tags:         v.Tags,
				}
			}
			return out
		},
	},
	{
		Name:     "InternetGateways",
		GetCount: func(r *v1alpha1.CloudInventoryReport) int { return len(r.Status.InternetGateways) },
		LogName:  "internetGatewayCount",
		Inventory: func(ctx context.Context, cfg aws.Config, tagFilter string) (interface{}, error) {
			return awsinventory.InventoryInternetGateways(ctx, cfg, tagFilter)
		},
		Count: func(items interface{}) int {
			return len(items.([]awsinventory.InternetGatewayInfo))
		},
		Convert: func(items interface{}) interface{} {
			raw := items.([]awsinventory.InternetGatewayInfo)
			out := make([]v1alpha1.InternetGatewayInfo, len(raw))
			for i, v := range raw {
				out[i] = v1alpha1.InternetGatewayInfo{
					InternetGatewayId: v.InternetGatewayId,
					Attachments:       v.Attachments,
					Tags:              v.Tags,
				}
			}
			return out
		},
	},
}
