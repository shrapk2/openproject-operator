package awsinventory

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// NATGatewayInfo holds NAT Gateway attributes
type NATGatewayInfo struct {
	NatGatewayID string            `json:"natGatewayId"`
	VpcID        string            `json:"vpcId"`
	SubnetID     string            `json:"subnetId"`
	State        string            `json:"state"`
	Tags         map[string]string `json:"tags"`
}

// InventoryNATGateways returns all NAT Gateways in the account
func InventoryNATGateways(ctx context.Context, cfg aws.Config, tagFilter string) ([]NATGatewayInfo, error) {
	client := ec2.NewFromConfig(cfg)
	var results []NATGatewayInfo

	// Create a paginator for NAT Gateways
	paginator := ec2.NewDescribeNatGatewaysPaginator(client, &ec2.DescribeNatGatewaysInput{})

	// Process each page of results
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe NAT gateways: %w", err)
		}

		for _, nat := range page.NatGateways {
			// Collect tags
			tags := make(map[string]string)
			for _, t := range nat.Tags {
				tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
			}

			// Apply tag filter if specified
			if tagFilter != "" {
				parts := strings.SplitN(tagFilter, "=", 2)
				if len(parts) == 2 {
					if val, ok := tags[parts[0]]; !ok || val != parts[1] {
						continue // Skip this NAT Gateway if it doesn't match the filter
					}
				}
			}

			results = append(results, NATGatewayInfo{
				NatGatewayID: aws.ToString(nat.NatGatewayId),
				VpcID:        aws.ToString(nat.VpcId),
				SubnetID:     aws.ToString(nat.SubnetId),
				State:        string(nat.State),
				Tags:         tags,
			})
		}
	}

	return results, nil
}
