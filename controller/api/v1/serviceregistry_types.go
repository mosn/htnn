/*
Copyright The HTNN Authors.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServiceRegistrySpec defines the desired state of ServiceRegistry
type ServiceRegistrySpec struct {
	// Type is the type of the service registry.
	Type string `json:"type,omitempty"`
	// Config is the configuration of the corresponding service registry.
	Config runtime.RawExtension `json:"config,omitempty"`
}

// ServiceRegistryStatus defines the observed state of ServiceRegistry
type ServiceRegistryStatus struct {
	// Conditions describe the current conditions.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ServiceRegistry is the Schema for the serviceregistries API
type ServiceRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceRegistrySpec   `json:"spec,omitempty"`
	Status ServiceRegistryStatus `json:"status,omitempty"`
}

func (r *ServiceRegistry) SetAccepted(reason ConditionReason, msg ...string) {
	conds, _ := addOrUpdateAcceptedCondition(r.Status.Conditions, r.Generation, reason, msg...)
	r.Status.Conditions = conds
}

//+kubebuilder:object:root=true

// ServiceRegistryList contains a list of ServiceRegistry
type ServiceRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceRegistry{}, &ServiceRegistryList{})
}
