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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HTTPFilterPolicySpec defines the desired state of HTTPFilterPolicy
type HTTPFilterPolicySpec struct {
	// TargetRef is the name of the resource this policy is being attached to.
	// This Policy and the TargetRef MUST be in the same namespace.
	// HTTPFilterPolicy in embedded mode can have no targetRef.
	//
	// +optional
	TargetRef *gwapiv1a2.PolicyTargetReferenceWithSectionName `json:"targetRef"`

	// Filters is a map of filter names to filter configurations.
	Filters map[string]HTTPPlugin `json:"filters,omitempty"`

	// SubPolicies is an array of sub-policies to specific section name.
	// If the specific section name is not found, the HTTPFilterPolicy will still be
	// treated as accepted.
	// If the SectionName field in TargetRef is given, the SubPolicies won't take effect.
	//
	// +listType=map
	// +listMapKey=sectionName
	SubPolicies []HTTPFilterSubPolicy `json:"subPolicies,omitempty"`
}

// HTTPFilterSubPolicy defines the sub-policy
type HTTPFilterSubPolicy struct {
	// SectionName is the name of a section within the target resource.
	SectionName gwapiv1.SectionName `json:"sectionName"`
	// Filters is a map of filter names to filter configurations.
	Filters map[string]HTTPPlugin `json:"filters,omitempty"`
}

// HTTPFilterPolicyStatus defines the observed state of HTTPFilterPolicy
type HTTPFilterPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions describe the current conditions.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	ChangeDetector `json:",inline"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:metadata:labels=gateway.networking.k8s.io/policy=direct

// HTTPFilterPolicy is the Schema for the httpfilterpolicies API
type HTTPFilterPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HTTPFilterPolicySpec   `json:"spec,omitempty"`
	Status HTTPFilterPolicyStatus `json:"status,omitempty"`
}

func (p *HTTPFilterPolicy) IsSpecChanged() bool {
	if len(p.Status.Conditions) == 0 {
		// newly created
		return true
	}
	for _, cond := range p.Status.Conditions {
		if cond.ObservedGeneration != p.Generation {
			return true
		}
	}
	return false
}

func (p *HTTPFilterPolicy) SetAccepted(reason gwapiv1a2.PolicyConditionReason, msg ...string) {
	c := metav1.Condition{
		Type:               string(gwapiv1a2.PolicyConditionAccepted),
		Reason:             string(reason),
		LastTransitionTime: metav1.NewTime(time.Now()),
		ObservedGeneration: p.Generation,
	}
	switch reason {
	case gwapiv1a2.PolicyReasonAccepted:
		c.Status = metav1.ConditionTrue
		c.Message = "The policy has been accepted"
	case gwapiv1a2.PolicyReasonInvalid:
		c.Status = metav1.ConditionFalse
		if len(msg) > 0 {
			c.Message = msg[0]
		} else {
			c.Message = "The policy is invalid"
		}
	case gwapiv1a2.PolicyReasonTargetNotFound:
		c.Status = metav1.ConditionFalse
		if len(msg) > 0 {
			c.Message = msg[0]
		} else {
			c.Message = "The policy targets non-existent resource"
		}
	}
	conds, changed := addOrUpdateCondition(p.Status.Conditions, c)
	p.Status.Conditions = conds

	if changed {
		p.Status.MarkAsChanged()
	}
}

func (p *HTTPFilterPolicy) IsValid() bool {
	for _, cond := range p.Status.Conditions {
		if cond.ObservedGeneration != p.Generation {
			continue
		}
		if cond.Type == string(gwapiv1a2.PolicyConditionAccepted) && cond.Reason == string(gwapiv1a2.PolicyReasonInvalid) {
			return false
		}
	}
	return true
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
