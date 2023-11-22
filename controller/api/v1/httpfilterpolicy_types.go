/*
Copyright 2023 The HTNN Authors.

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
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HTTPFilterPolicySpec defines the desired state of HTTPFilterPolicy
type HTTPFilterPolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:XValidation:rule="self.group in ['', 'networking.istio.io', 'gateway.networking.k8s.io']", message="unsupported targetRef.group"
	// +kubebuilder:validation:XValidation:rule="self.kind in ['Namespace', 'VirtualService', 'Gateway', 'HTTPRoute', 'GRPCRoute']", message="unsupported targetRef.kind"

	// TargetRef is the name of the resource this policy is being attached to.
	// This Policy and the TargetRef MUST be in the same namespace.
	TargetRef gwapiv1a2.PolicyTargetReferenceWithSectionName `json:"targetRef"`

	// Filters is a map of filter names to filter configurations.
	Filters map[string]runtime.RawExtension `json:"filters,omitempty"`
}

// HTTPFilterPolicyStatus defines the observed state of HTTPFilterPolicy
type HTTPFilterPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions describe the current conditions of the SecurityPolicy.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HTTPFilterPolicy is the Schema for the httpfilterpolicies API
type HTTPFilterPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HTTPFilterPolicySpec   `json:"spec,omitempty"`
	Status HTTPFilterPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HTTPFilterPolicyList contains a list of HTTPFilterPolicy
type HTTPFilterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HTTPFilterPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HTTPFilterPolicy{}, &HTTPFilterPolicyList{})
}
