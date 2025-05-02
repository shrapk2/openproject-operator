package awsinventory

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type EIPInfo struct {
	AllocationID       string
	PublicIP           string
	Domain             string
	InstanceID         string
	NetworkInterfaceID string
	PrivateIP          string
	Tags               map[string]string
}

func InventoryEIPs(ctx context.Context, cfg aws.Config, tagFilter string) ([]EIPInfo, error) {
	client := ec2.NewFromConfig(cfg)

	out, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe EIPs: %w", err)
	}

	var results []EIPInfo
	for _, addr := range out.Addresses {
		results = append(results, EIPInfo{
			AllocationID:       aws.ToString(addr.AllocationId),
			PublicIP:           aws.ToString(addr.PublicIp),
			Domain:             string(addr.Domain),
			InstanceID:         aws.ToString(addr.InstanceId),
			NetworkInterfaceID: aws.ToString(addr.NetworkInterfaceId),
			PrivateIP:          aws.ToString(addr.PrivateIpAddress),
			Tags:               map[string]string{},
		})
	}
	return results, nil
}
