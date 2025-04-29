package awsinventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// S3BucketInfo holds S3 bucket attributes.
type S3BucketInfo struct {
	Name                 string            `json:"name" yaml:"name"`
	Region               string            `json:"region" yaml:"region"`
	BlockAllPublicAccess bool              `json:"blockAllPublicAccess" yaml:"blockAllPublicAccess"`
	Tags                 map[string]string `json:"tags" yaml:"tags"`
}

// InventoryS3Buckets returns all S3 buckets matching an optional tagFilter ("Key=Value").
func InventoryS3Buckets(ctx context.Context, cfg aws.Config, tagFilter string) ([]S3BucketInfo, error) {
	// Parent timeout for the entire S3 inventory operation
	opCtx, opCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer opCancel()

	client := s3.NewFromConfig(cfg)

	// List all buckets with a 30s timeout
	listCtx, listCancel := context.WithTimeout(opCtx, 30*time.Second)
	listOut, err := client.ListBuckets(listCtx, &s3.ListBucketsInput{})
	listCancel()
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 buckets: %w", err)
	}

	var buckets []S3BucketInfo
	// per-bucket timeout for sub-operations
	bucketTimeout := 15 * time.Second

	for _, b := range listOut.Buckets {
		name := aws.ToString(b.Name)

		// Determine region
		locCtx, locCancel := context.WithTimeout(opCtx, bucketTimeout)
		locOut, err := client.GetBucketLocation(locCtx, &s3.GetBucketLocationInput{Bucket: &name})
		locCancel()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// this bucket timed out—skip it
				continue
			}
			fmt.Printf("Warning: couldn't determine location for bucket %s: %v\n", name, err)
			continue
		}
		region := string(locOut.LocationConstraint)
		if region == "" {
			region = cfg.Region
		}

		// Use a region-specific client for the next calls
		regionalCfg := cfg.Copy()
		regionalCfg.Region = region
		regionalClient := s3.NewFromConfig(regionalCfg)

		// Check PublicAccessBlock config (assume blocked by default)
		blockEnabled := true

		pabCtx, pabCancel := context.WithTimeout(opCtx, bucketTimeout)
		pab, err := regionalClient.GetPublicAccessBlock(pabCtx, &s3.GetPublicAccessBlockInput{Bucket: &name})
		pabCancel()
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchPublicAccessBlockConfiguration" {
				// no bucket-level config → "Block all public access" is off
				blockEnabled = false
			} else {
				fmt.Printf("Warning: couldn't check PublicAccessBlock for %s: %v\n", name, err)
			}
		} else {
			cfg := pab.PublicAccessBlockConfiguration
			blockEnabled = aws.ToBool(cfg.BlockPublicAcls) &&
				aws.ToBool(cfg.BlockPublicPolicy) &&
				aws.ToBool(cfg.IgnorePublicAcls) &&
				aws.ToBool(cfg.RestrictPublicBuckets)
		}

		// If not fully blocked, do an ACL check (15s timeout)
		if !blockEnabled {
			aclCtx, aclCancel := context.WithTimeout(opCtx, bucketTimeout)
			aclOut, aclErr := regionalClient.GetBucketAcl(aclCtx, &s3.GetBucketAclInput{Bucket: &name})
			aclCancel()
			if aclErr == nil {
				for _, grant := range aclOut.Grants {
					if grant.Grantee.URI != nil &&
						aws.ToString(grant.Grantee.URI) == "http://acs.amazonaws.com/groups/global/AllUsers" {
						// bucket is public
						break
					}
				}
			}
		}

		// Fetch tags (15s timeout)
		tags := map[string]string{}
		tagCtx, tagCancel := context.WithTimeout(opCtx, bucketTimeout)
		tagOut, tagErr := regionalClient.GetBucketTagging(tagCtx, &s3.GetBucketTaggingInput{Bucket: &name})
		tagCancel()
		if tagErr == nil {
			for _, t := range tagOut.TagSet {
				tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
			}
		}

		// Apply tag filter
		if tagFilter != "" {
			parts := strings.SplitN(tagFilter, "=", 2)
			if len(parts) == 2 {
				if val, ok := tags[parts[0]]; !ok || val != parts[1] {
					continue
				}
			}
		}

		buckets = append(buckets, S3BucketInfo{
			Name:                 name,
			Region:               region,
			BlockAllPublicAccess: blockEnabled,
			Tags:                 tags,
		})
	}

	return buckets, nil
}
