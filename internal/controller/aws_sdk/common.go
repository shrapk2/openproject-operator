package awsinventory

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
)

// DefaultTags defines the tags to extract from AWS resources
var DefaultTags = []string{"RPO(Hours)", "RTO(Hours)", "RPO", "RTO", "ClientName", "Customer"}

// GetRegion chooses the best region source (spec > secret > default)
func GetRegion(specRegion, secretRegion string) string {
	if specRegion != "" {
		return specRegion
	}
	if secretRegion != "" {
		return secretRegion
	}
	return "us-east-1"
}

func NewCustomRetryer() aws.Retryer {
	return retry.NewStandard(func(o *retry.StandardOptions) {
		o.MaxAttempts = 3

		o.MaxBackoff = 30 * time.Second
	})
}
