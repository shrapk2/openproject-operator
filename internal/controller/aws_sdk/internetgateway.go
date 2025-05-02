package awsinventory

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// InternetGatewayInfo holds Internet Gateway attributes
type InternetGatewayInfo struct {
	InternetGatewayId string            `json:"internetGatewayId"`
	Attachments       []string          `json:"attachments,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
}

// InventoryInternetGateways returns all Internet Gateways in the account
func InventoryInternetGateways(ctx context.Context, cfg aws.Config, tagFilter string) ([]InternetGatewayInfo, error) {
	client := ec2.NewFromConfig(cfg)

	out, err := client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe Internet Gateways: %w", err)
	}

	var results []InternetGatewayInfo
	for _, ig := range out.InternetGateways {
		// Collect tags
		tags := make(map[string]string)
		for _, t := range ig.Tags {
			tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
		}

		// Collect attachment VPC IDs
		var vpcIds []string
		for _, attach := range ig.Attachments {
			vpcIds = append(vpcIds, aws.ToString(attach.VpcId))
		}

		results = append(results, InternetGatewayInfo{
			InternetGatewayId: aws.ToString(ig.InternetGatewayId),
			Attachments:       vpcIds,
			Tags:              tags,
		})
	}

	return results, nil
}
