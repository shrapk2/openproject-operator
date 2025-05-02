/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ServerConfigRef struct {
	Name string `json:"name"`
	// Optional: Namespace string `json:"namespace,omitempty"`
}

// JSON represents an arbitrary JSON value
// +kubebuilder:validation:Type=object
// +kubebuilder:validation:Schemaless
// +kubebuilder:pruning:PreserveUnknownFields
type JSON struct {
	// Raw is the underlying JSON data
	Raw runtime.RawExtension `json:"-"`
}

// MarshalJSON returns m as the JSON encoding of m.
func (m JSON) MarshalJSON() ([]byte, error) {
	if m.Raw.Raw == nil {
		return []byte("null"), nil
	}
	return m.Raw.Raw, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *JSON) UnmarshalJSON(data []byte) error {
	if m == nil {
		return fmt.Errorf("JSON: UnmarshalJSON on nil pointer")
	}
	m.Raw.Raw = append([]byte(nil), data...)
	return nil
}

// WorkPackagesSpec defines the desired state of WorkPackages
type WorkPackagesSpec struct {
	// +kubebuilder:validation:Required
	// Subject is the title of the ticket
	Subject string `json:"subject"`
	// Description is the markdown content for the ticket
	Description string `json:"description"`

	// ProjectID is the numeric ID of the OpenProject project
	ProjectID int `json:"projectID"`

	// TypeID is the numeric ID of the work package type
	TypeID int `json:"typeID"`

	// EpicID is the parent work package ID (optional)
	// +optional
	EpicID int `json:"epicID,omitempty"`

	// Schedule is a cron expression for when to create the ticket
	Schedule string `json:"schedule"`

	// ServerConfigRef is a reference to the OpenProject server configuration
	ServerConfigRef corev1.LocalObjectReference `json:"serverConfigRef"`

	// AdditionalFields contains extra fields to include in the work package
	// +optional
	AdditionalFields JSON `json:"additionalFields,omitempty"`

	// InventoryRef is an optional reference to a CloudInventory to run/report
	// +optional
	InventoryRef *corev1.LocalObjectReference `json:"inventoryRef,omitempty"`
}

// WorkPackagesStatus defines the observed state of WorkPackages
type WorkPackagesStatus struct {
	CreatedAt   string       `json:"createdAt,omitempty"`
	Status      string       `json:"status,omitempty"`
	Message     string       `json:"message,omitempty"`
	LastRunTime *metav1.Time `json:"lastRunTime,omitempty"`
	NextRunTime *metav1.Time `json:"nextRunTime,omitempty"`
	TicketID    string       `json:"ticketID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WorkPackages is the Schema for the workpackages API
type WorkPackages struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkPackagesSpec   `json:"spec,omitempty"`
	Status WorkPackagesStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkPackagesList contains a list of WorkPackages
type WorkPackagesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkPackages `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkPackages{}, &WorkPackagesList{})
}
