package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EC2InstanceInfo struct {
	Name       string            `json:"name"`
	InstanceID string            `json:"instanceId"`
	State      string            `json:"state"`
	Type       string            `json:"type"`
	AZ         string            `json:"az"`
	Platform   string            `json:"platform"`
	PublicIP   string            `json:"publicIp"`
	PrivateDNS string            `json:"privateDns"`
	PrivateIP  string            `json:"privateIp"`
	ImageID    string            `json:"imageId"`
	VPCID      string            `json:"vpcId"`
	Tags       map[string]string `json:"tags"`
}

// CloudInventorySpec defines the desired state of CloudInventory
type CloudInventorySpec struct {
	// Mode specifies the inventory mode: "aws" or "kubernetes"
	// +kubebuilder:validation:Enum=aws;kubernetes
	Mode string `json:"mode"`

	// Filter is an optional label/tag filter
	// e.g., for AWS: "tag:Environment=prod", for K8s: "app=frontend"
	// +optional
	Filter string `json:"filter,omitempty"`

	// AWS-specific inventory options (required if Mode == "aws")
	// +optional
	AWS *AWSInventorySpec `json:"aws,omitempty"`

	// Kubernetes-specific inventory options (required if Mode == "kubernetes")
	// +optional
	Kubernetes *KubernetesInventorySpec `json:"kubernetes,omitempty"`
}

// AWSInventorySpec holds configuration for AWS inventory scanning
type AWSInventorySpec struct {
	CredentialsSecretRef *corev1.SecretKeySelector `json:"credentialsSecretRef,omitempty"`
	AssumeRoleARN        string                    `json:"assumeRoleARN,omitempty"`
	Resources            []string                  `json:"resources"`

	// Global region override (if not provided in secret)
	// +optional
	Region string `json:"region,omitempty"`

	// Optional: tag filter (e.g., "Environment=prod")
	// +optional
	TagFilter string `json:"tagFilter,omitempty"`
}

// KubernetesInventorySpec holds configuration for Kubernetes inventory scanning
type KubernetesInventorySpec struct {
	Namespaces    []string `json:"namespaces,omitempty"`
	LabelSelector string   `json:"labelSelector,omitempty"`

	// Optional: reference to a Secret that contains a kubeconfig for a remote cluster
	KubeconfigSecretRef *corev1.SecretKeySelector `json:"kubeconfigSecretRef,omitempty"`
}

type ContainerImageInfo struct {
	Cluster    string `json:"cluster"`
	Image      string `json:"image"`
	Repository string `json:"repository"`
	Version    string `json:"version"`
	SHA        string `json:"sha,omitempty"`
}

// CloudInventoryStatus defines the observed state of CloudInventory
type CloudInventoryStatus struct {
	// LastRunTime is the last time the inventory job ran
	// +optional
	LastRunTime metav1.Time `json:"lastRunTime,omitempty"`

	// ItemCount is the total number of resources discovered
	// +optional
	ItemCount int `json:"itemCount,omitempty"`

	// Message is a summary or error message
	// +optional
	Message string `json:"message,omitempty"`

	// Summary contains per-service or per-namespace counts
	// +optional
	Summary map[string]int `json:"summary,omitempty"`

	// Images is a deduplicated list of container images (kubernetes mode only)
	// +optional
	ContainerImages []ContainerImageInfo `json:"containerImages,omitempty"`

	EC2 []EC2InstanceInfo `json:"ec2,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CloudInventory is the Schema for the cloudinventories API
type CloudInventory struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudInventorySpec   `json:"spec,omitempty"`
	Status CloudInventoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudInventoryList contains a list of CloudInventory
type CloudInventoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudInventory `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudInventory{}, &CloudInventoryList{})
}
