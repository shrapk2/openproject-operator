package awsinventory

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

// ECRRegistryInfo mirrors the API type for conversion
type ECRRegistryInfo struct {
	RegistryID        string
	RepositoryName    string
	LatestImageTag    string
	LatestImageDigest string
}

// InventoryECRRepositories lists all ECR repositories and finds each latest tagged image
func InventoryECRRepositories(ctx context.Context, cfg aws.Config, tagFilter string) ([]ECRRegistryInfo, error) {
	client := ecr.NewFromConfig(cfg)
	// Increase timeout to 60 seconds - ECR operations can be slow
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// List all repositories (private)
	var repos []types.Repository
	p := ecr.NewDescribeRepositoriesPaginator(client, &ecr.DescribeRepositoriesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list ECR repos: %w", err)
		}
		repos = append(repos, page.Repositories...)
	}

	var result []ECRRegistryInfo
	for _, r := range repos {
		// Create a repo-level context with shorter timeout to avoid hanging on one repo
		repoCtx, repoCancel := context.WithTimeout(ctx, 10*time.Second)
		defer repoCancel()

		// Get repository name safely
		repoName := aws.ToString(r.RepositoryName)

		// Optimize: Just get the most recent image directly instead of listing all tags
		imgList, err := client.DescribeImages(repoCtx, &ecr.DescribeImagesInput{
			RepositoryName: r.RepositoryName,
			MaxResults:     aws.Int32(10), // Just get a few recent images
			Filter: &types.DescribeImagesFilter{
				TagStatus: types.TagStatusTagged,
			},
		})

		if err != nil {
			// Log error but continue with next repository
			fmt.Printf("Warning: couldn't fetch images for %s: %v\n", repoName, err)
			continue
		}

		// Find the most recent image from the batch
		var latestTag, latestDigest string
		var latestTime time.Time

		for _, detail := range imgList.ImageDetails {
			if detail.ImagePushedAt != nil && detail.ImagePushedAt.After(latestTime) {
				latestTime = aws.ToTime(detail.ImagePushedAt)
				if len(detail.ImageTags) > 0 {
					latestTag = detail.ImageTags[0]
				}
				latestDigest = aws.ToString(detail.ImageDigest)
			}
		}

		result = append(result, ECRRegistryInfo{
			RegistryID:        aws.ToString(r.RegistryId),
			RepositoryName:    repoName,
			LatestImageTag:    latestTag,
			LatestImageDigest: latestDigest,
		})
	}

	return result, nil
}
