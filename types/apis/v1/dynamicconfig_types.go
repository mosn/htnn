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
	"k8s.io/apimachinery/pkg/runtime"
)

// DynamicConfigSpec defines the desired state of DynamicConfig
type DynamicConfigSpec struct {
	Type   string               `json:"type"`
	Config runtime.RawExtension `json:"config"`
}

// DynamicConfigStatus defines the observed state of DynamicConfig
type DynamicConfigStatus struct {
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

// DynamicConfig is the Schema for the dynamicconfigs API
type DynamicConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynamicConfigSpec   `json:"spec,omitempty"`
	Status DynamicConfigStatus `json:"status,omitempty"`
}

func (c *DynamicConfig) IsSpecChanged() bool {
	if len(c.Status.Conditions) == 0 {
		// newly created
		return true
	}
	for _, cond := range c.Status.Conditions {
		if cond.ObservedGeneration != c.Generation {
			return true
		}
	}
	return false
}

func (c *DynamicConfig) SetAccepted(reason ConditionReason, msg ...string) {
	conds, changed := addOrUpdateAcceptedCondition(c.Status.Conditions, c.Generation, reason, msg...)
	c.Status.Conditions = conds

	if changed {
		c.Status.MarkAsChanged()
	}
}

func (c *DynamicConfig) IsValid() bool {
	for _, cond := range c.Status.Conditions {
		if cond.ObservedGeneration != c.Generation {
			continue
		}
		if cond.Type == string(ConditionAccepted) && cond.Reason == string(ReasonInvalid) {
			return false
		}
	}
	return true
}

//+kubebuilder:object:root=true

// DynamicConfigList contains a list of DynamicConfig
type DynamicConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynamicConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DynamicConfig{}, &DynamicConfigList{})
}
