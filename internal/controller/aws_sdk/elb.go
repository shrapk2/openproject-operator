package awsinventory

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	// "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

// ELBV2Info holds details about AWS Application Load Balancers
// Fields map LoadBalancer attributes and selected tags.
type ELBV2Info struct {
	Name           string            `json:"name" yaml:"name"`
	ARN            string            `json:"arn" yaml:"arn"`
	DNSName        string            `json:"dnsName" yaml:"dnsName"`
	Scheme         string            `json:"scheme" yaml:"scheme"`
	Type           string            `json:"type" yaml:"type"`
	VPCID          string            `json:"vpcId" yaml:"vpcId"`
	State          string            `json:"state" yaml:"state"`
	IPAddressType  string            `json:"ipAddressType" yaml:"ipAddressType"`
	SecurityGroups []string          `json:"securityGroups" yaml:"securityGroups"`
	Subnets        []string          `json:"subnets" yaml:"subnets"`
	Tags           map[string]string `json:"tags" yaml:"tags"`
}

// InventoryALBs returns all ALBs matching an optional tag filter.
// tagFilter is of the form "Key=Value"; if empty, returns all.
func InventoryALBs(ctx context.Context, cfg aws.Config, tagFilter string) ([]ELBV2Info, error) {
	client := elbv2.NewFromConfig(cfg)

	// 1) Describe all load balancers
	lbsOut, err := client.DescribeLoadBalancers(ctx, &elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe ALBs: %w", err)
	}

	// Collect ARNs for tag lookup
	var arns []string
	for _, lb := range lbsOut.LoadBalancers {
		arns = append(arns, aws.ToString(lb.LoadBalancerArn))
	}

	// 2) Describe tags for all ARNs
	tagsOut, err := client.DescribeTags(ctx, &elbv2.DescribeTagsInput{ResourceArns: arns})
	if err != nil {
		return nil, fmt.Errorf("failed to describe ALB tags: %w", err)
	}

	// Map ARN -> tags map
	tagMap := make(map[string]map[string]string, len(tagsOut.TagDescriptions))
	for _, td := range tagsOut.TagDescriptions {
		m := make(map[string]string)
		for _, t := range td.Tags {
			m[aws.ToString(t.Key)] = aws.ToString(t.Value)
		}
		tagMap[aws.ToString(td.ResourceArn)] = m
	}

	// Parse tagFilter
	var fk, fv string
	if tagFilter != "" {
		parts := strings.SplitN(tagFilter, "=", 2)
		if len(parts) == 2 {
			fk, fv = parts[0], parts[1]
		}
	}

	// 3) Build result list
	var result []ELBV2Info
	for _, lb := range lbsOut.LoadBalancers {
		arn := aws.ToString(lb.LoadBalancerArn)
		tags := tagMap[arn]

		// Apply filter if provided
		if fk != "" {
			if val, ok := tags[fk]; !ok || val != fv {
				continue
			}
		}

		// Gather security groups
		sgs := make([]string, len(lb.SecurityGroups))
		copy(sgs, lb.SecurityGroups)

		// Gather subnet IDs
		var subs []string
		for _, az := range lb.AvailabilityZones {
			subs = append(subs, aws.ToString(az.SubnetId))
		}

		// Select default tags
		selTags := make(map[string]string)
		for _, key := range DefaultTags {
			if v, ok := tags[key]; ok {
				selTags[key] = v
			}
		}

		info := ELBV2Info{
			Name:           aws.ToString(lb.LoadBalancerName),
			ARN:            arn,
			DNSName:        aws.ToString(lb.DNSName),
			Scheme:         string(lb.Scheme),
			Type:           string(lb.Type),
			VPCID:          aws.ToString(lb.VpcId),
			State:          string(lb.State.Code),
			IPAddressType:  string(lb.IpAddressType),
			SecurityGroups: sgs,
			Subnets:        subs,
			Tags:           selTags,
		}
		result = append(result, info)
	}

	return result, nil
}
