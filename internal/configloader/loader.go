package configloader

import (
	"context"
	"fmt"

	openprojectv1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LoadServerConfig fetches the singleton config in a namespace
func LoadServerConfig(ctx context.Context, c client.Client, namespace string) (*openprojectv1alpha1.ServerConfig, error) {
	var config openprojectv1alpha1.ServerConfig
	name := types.NamespacedName{
		Name:      "openproject",
		Namespace: namespace,
	}

	if err := c.Get(ctx, name, &config); err != nil {
		return nil, fmt.Errorf("failed to get ServerConfig: %w", err)
	}

	return &config, nil
}

func LoadAPIKey(ctx context.Context, c client.Client, config *openprojectv1alpha1.ServerConfig) (string, error) {
	secretRef := config.Spec.APIKeySecretRef
	var secret corev1.Secret

	nsName := types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: config.Namespace,
	}

	if err := c.Get(ctx, nsName, &secret); err != nil {
		return "", fmt.Errorf("failed to get API key secret: %w", err)
	}

	tokenBytes, exists := secret.Data[secretRef.Key]
	if !exists {
		return "", fmt.Errorf("API key not found in secret under key %q", secretRef.Key)
	}

	return string(tokenBytes), nil
}
