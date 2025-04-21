package v1alpha1

import (
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudInventorySpec defines the desired state of CloudInventory
type CloudInventorySpec struct {
  // +kubebuilder:validation:Required
  // Region is the cloud region to inventory
  Region string `json:"region"`
  // +optional
  // Filter is an optional label filter for resources
  Filter string `json:"filter,omitempty"`
}

// CloudInventoryStatus defines the observed state of CloudInventory
type CloudInventoryStatus struct {
  // LastRunTime is the last time the inventory ran
  LastRunTime metav1.Time `json:"lastRunTime,omitempty"`
  // ItemCount is how many resources we found
  ItemCount int `json:"itemCount,omitempty"`
  // Message holds any human-readable status info
  Message string `json:"message,omitempty"`
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
