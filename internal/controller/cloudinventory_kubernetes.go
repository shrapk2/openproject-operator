package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
)

func (r *CloudInventoryReconciler) reconcileKubernetes(ctx context.Context, ci *v1alpha1.CloudInventory, log logr.Logger) (ctrl.Result, error) {
	// Determine if we use local client or build remote client
	var restConfig *rest.Config
	var clientset *kubernetes.Clientset
	var err error
	var clusterName = "local"

	if ci.Spec.Kubernetes.KubeconfigSecretRef != nil {
		// Remote cluster
		restConfig, clusterName, err = r.buildRemoteKubeConfig(ctx, ci, log)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// Local cluster
		restConfig, err = ctrl.GetConfig()
		if err != nil {
			log.Error(err, "unable to load local kubeconfig")
			return ctrl.Result{}, err
		}
	}

	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Error(err, "failed to create Kubernetes client")
		return ctrl.Result{}, err
	}

	// List pods (all namespaces or scoped)
	ns := ""
	if len(ci.Spec.Kubernetes.Namespaces) == 1 {
		ns = ci.Spec.Kubernetes.Namespaces[0]
	}

	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list pods")
		return ctrl.Result{}, err
	}

	var images []v1alpha1.ContainerImageInfo
	imageMap := make(map[string]v1alpha1.ContainerImageInfo)

	for _, pod := range pods.Items {
		statusMap := make(map[string]string)
		for _, status := range pod.Status.ContainerStatuses {
			statusMap[status.Name] = status.ImageID
		}

		for _, container := range pod.Spec.Containers {
			image := container.Image
			imageID := statusMap[container.Name] // safe even if not found

			parsed := parseContainerImageWithDigest(image, imageID)
			parsed.Cluster = clusterName

			if _, exists := imageMap[parsed.Image]; !exists {
				imageMap[parsed.Image] = parsed
			}
		}
	}

	// Convert map to slice
	for _, val := range imageMap {
		images = append(images, val)
	}

	summary := map[string]int{
		"pods":   len(pods.Items),
		"images": len(images),
	}

	if err := r.patchCloudInventoryStatus(ctx, ci, "Kubernetes inventory completed", len(pods.Items), summary, nil); err != nil {
		log.Error(err, "failed to update Kubernetes inventory status")
		return ctrl.Result{}, err
	}

	// Optional: patch detailed image info too
	original := ci.DeepCopy()
	ci.Status.ContainerImages = images
	if err := r.Status().Patch(ctx, ci, client.MergeFrom(original)); err != nil {
		log.Error(err, "failed to patch container image details")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

func parseContainerImageWithDigest(image, imageID string) v1alpha1.ContainerImageInfo {
	var repo, version, sha string

	// Parse the image repo and tag
	tagParts := strings.Split(image, ":")
	repo = tagParts[0]
	if len(tagParts) > 1 {
		version = tagParts[1]
	}

	// Extract SHA from ImageID if available
	if strings.Contains(imageID, "@sha256:") {
		parts := strings.Split(imageID, "@")
		if len(parts) > 1 {
			sha = parts[1]
		}
	}

	return v1alpha1.ContainerImageInfo{
		Image:      image,
		Repository: repo,
		Version:    version,
		SHA:        sha,
	}
}

// buildRemoteKubeConfig reads kubeconfig from Secret and returns a *rest.Config
func (r *CloudInventoryReconciler) buildRemoteKubeConfig(ctx context.Context, ci *v1alpha1.CloudInventory, log logr.Logger) (*rest.Config, string, error) {
	ref := ci.Spec.Kubernetes.KubeconfigSecretRef
	secret := &corev1.Secret{}

	if err := r.Get(ctx, client.ObjectKey{Namespace: ci.Namespace, Name: ref.Name}, secret); err != nil {
		return nil, "", fmt.Errorf("failed to get kubeconfig secret: %w", err)
	}

	raw, ok := secret.Data[ref.Key]
	if !ok {
		return nil, "", fmt.Errorf("kubeconfig secret missing key %q", ref.Key)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(raw)
	if err != nil {
		return nil, "", fmt.Errorf("invalid kubeconfig: %w", err)
	}

	clusterName := parseClusterNameFromKubeconfig(raw)
	return config, clusterName, nil
}

// parseClusterNameFromKubeconfig returns the name of the first cluster found
func parseClusterNameFromKubeconfig(kubeconfig []byte) string {
	config, err := clientcmd.Load(kubeconfig)
	if err != nil || len(config.Clusters) == 0 {
		return "remote"
	}

	for name := range config.Clusters {
		return name
	}
	return "remote"
}

// parseContainerImage parses image into repository, version, and sha
func parseContainerImage(image string) v1alpha1.ContainerImageInfo {
	var repo, version, sha string

	// Format: repo[:tag][@sha256:...]
	mainParts := strings.Split(image, "@")
	namePart := mainParts[0]

	if len(mainParts) > 1 {
		sha = mainParts[1]
	}

	tagParts := strings.Split(namePart, ":")
	repo = tagParts[0]
	if len(tagParts) > 1 {
		version = tagParts[1]
	}

	return v1alpha1.ContainerImageInfo{
		Image:      image,
		Repository: repo,
		Version:    version,
		SHA:        sha,
	}
}
