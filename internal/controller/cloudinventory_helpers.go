package controller

import (
	"context"
	"time"

	v1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// patchCloudInventoryStatus updates key CloudInventory status fields safely
func (r *CloudInventoryReconciler) patchCloudInventoryStatus(
	ctx context.Context,
	ci *v1alpha1.CloudInventory,
	message string,
	itemCount int,
	summary map[string]int,
	containerImages []string,
) error {
	original := ci.DeepCopy()

	ci.Status.LastRunTime = metav1.NewTime(time.Now())
	ci.Status.ItemCount = itemCount
	ci.Status.Message = message
	ci.Status.Summary = summary
	ci.Status.ContainerImages = make([]v1alpha1.ContainerImageInfo, len(containerImages))
	for i, img := range containerImages {
		ci.Status.ContainerImages[i] = v1alpha1.ContainerImageInfo{Image: img}
	}

	return r.Status().Patch(ctx, ci, client.MergeFrom(original))
}

// utility: deduplicateStrings returns a set of unique strings
func deduplicateStrings(values []string) []string {
	set := make(map[string]struct{})
	for _, val := range values {
		set[val] = struct{}{}
	}

	unique := make([]string, 0, len(set))
	for val := range set {
		unique = append(unique, val)
	}
	return unique
}
