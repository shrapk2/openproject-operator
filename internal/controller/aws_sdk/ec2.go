package awsinventory

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// DefaultTags defines the tags to extract for EC2 instances
var DefaultTags = []string{"RPO(Hours)", "RTO(Hours)", "ClientName", "Customer"}

type EC2InstanceInfo struct {
	Name       string
	InstanceID string
	State      string
	Type       string
	AZ         string
	Platform   string
	PublicIP   string
	PrivateDNS string
	PrivateIP  string
	ImageID    string
	VPCID      string
	Tags       map[string]string
}

func InventoryEC2Instances(ctx context.Context, cfg aws.Config, tagFilter string) ([]EC2InstanceInfo, error) {
	client := ec2.NewFromConfig(cfg)

	var filters []types.Filter
	if tagFilter != "" {
		parts := strings.SplitN(tagFilter, "=", 2)
		if len(parts) == 2 {
			filters = []types.Filter{
				{
					Name:   aws.String(fmt.Sprintf("tag:%s", parts[0])),
					Values: []string{parts[1]},
				},
			}
		}
	}

	output, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe EC2 instances: %w", err)
	}

	var results []EC2InstanceInfo
	for _, reservation := range output.Reservations {
		for _, inst := range reservation.Instances {
			tags := make(map[string]string)
			for _, t := range inst.Tags {
				tags[*t.Key] = *t.Value
			}

			instance := EC2InstanceInfo{
				Name:       tags["Name"],
				InstanceID: aws.ToString(inst.InstanceId),
				State:      string(inst.State.Name),
				Type:       string(inst.InstanceType),
				AZ:         aws.ToString(inst.Placement.AvailabilityZone),
				Platform:   aws.ToString(inst.PlatformDetails),
				PublicIP:   aws.ToString(inst.PublicIpAddress),
				PrivateIP:  aws.ToString(inst.PrivateIpAddress),
				PrivateDNS: aws.ToString(inst.PrivateDnsName),
				ImageID:    aws.ToString(inst.ImageId),
				VPCID:      aws.ToString(inst.VpcId),
				Tags: func() map[string]string {
					selectedTags := make(map[string]string)
					for _, key := range DefaultTags {
						if value, exists := tags[key]; exists {
							selectedTags[key] = value
						}
					}
					return selectedTags
				}(),
			}

			results = append(results, instance)
		}
	}

	// Optional: Lookup EIPs here and patch ElasticIP (next step)

	return results, nil
}

// BuildEC2Filters creates EC2 filter objects from tagFilter="Key=Value"
func BuildEC2Filters(tagFilter string) []types.Filter {
	if tagFilter == "" {
		return nil
	}

	parts := strings.SplitN(tagFilter, "=", 2)
	if len(parts) != 2 {
		return nil
	}

	return []types.Filter{
		{
			Name:   aws.String(fmt.Sprintf("tag:%s", parts[0])),
			Values: []string{parts[1]},
		},
	}
}
