package awsinventory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

type RDSInstanceInfo struct {
	DBInstanceIdentifier string
	Engine               string
	EngineVersion        string
	InstanceClass        string
	AvailabilityZone     string
	Status               string
	MultiAZ              bool
	PubliclyAccessible   bool
	StorageType          string
	AllocatedStorage     int32
	VPCID                string
	Tags                 map[string]string
}

func InventoryRDSInstances(ctx context.Context, cfg aws.Config, tagFilter string) ([]RDSInstanceInfo, error) {
	client := rds.NewFromConfig(cfg)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

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

	out, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe EC2 instances: %w", err)
	}

	var result []RDSInstanceInfo
	for _, db := range out.DBInstances {
		vpcID := ""
		if db.DBSubnetGroup != nil {
			vpcID = aws.ToString(db.DBSubnetGroup.VpcId)
		}
		tags := make(map[string]string)
		for _, t := range db.TagList {
			tags[*t.Key] = *t.Value
		}

		result = append(result, RDSInstanceInfo{
			DBInstanceIdentifier: aws.ToString(db.DBInstanceIdentifier),
			Engine:               aws.ToString(db.Engine),
			EngineVersion:        aws.ToString(db.EngineVersion),
			InstanceClass:        aws.ToString(db.DBInstanceClass),
			AvailabilityZone:     aws.ToString(db.AvailabilityZone),
			Status:               aws.ToString(db.DBInstanceStatus),
			MultiAZ:              aws.ToBool(db.MultiAZ),
			PubliclyAccessible:   aws.ToBool(db.PubliclyAccessible),
			StorageType:          aws.ToString(db.StorageType),
			AllocatedStorage:     aws.ToInt32(db.AllocatedStorage),
			VPCID:                vpcID,
			Tags: func() map[string]string {
				selectedTags := make(map[string]string)
				for _, key := range DefaultTags {
					if value, exists := tags[key]; exists {
						selectedTags[key] = value
					}
				}
				return selectedTags
			}(),
		})
	}

	return result, nil
}

func BuildRDSFilters(tagFilter string) []types.Filter {
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
