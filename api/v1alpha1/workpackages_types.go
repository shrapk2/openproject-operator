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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ServerConfigRef struct {
	Name string `json:"name"`
	// Optional: Namespace string `json:"namespace,omitempty"`
}

// WorkPackagesSpec defines the desired state of WorkPackages
type WorkPackagesSpec struct {
	// +kubebuilder:validation:Required
	ServerConfigRef corev1.LocalObjectReference `json:"serverConfigRef"`
	Subject         string                      `json:"subject"`
	Description     string                      `json:"description"`
	Schedule        string                      `json:"schedule,omitempty"` // Optional cron
	ProjectID       int                         `json:"projectID"`
	TypeID          int                         `json:"typeID"`
	EpicID          int                         `json:"epicID,omitempty"` // Optional parent
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
