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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	csModel "mosn.io/htnn/api/pkg/consumer/model"
	fmModel "mosn.io/htnn/api/pkg/filtermanager/model"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ConsumerPlugin defines the authentication plugin configuration used in the consumer
type ConsumerPlugin struct {
	Config runtime.RawExtension `json:"config"`
}

// ConsumerSpec defines the desired state of Consumer
type ConsumerSpec struct {
	// Auth is a map of authentication plugin names to plugin configurations.
	//
	// +kubebuilder:validation:MinProperties=1
	Auth map[string]ConsumerPlugin `json:"auth"`

	// Filters is a map of filter names to filter configurations.
	//
	// +optional
	Filters map[string]Plugin `json:"filters,omitempty"`

	// Name is the name of consumer, which is used in the data plane matching.
	// If this field is not set, the name of the consumer CustomResource will be used.
	//
	// +optional
	Name string `json:"name,omitempty"`
}

// ConsumerStatus defines the observed state of Consumer
type ConsumerStatus struct {
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

// Consumer is the Schema for the consumers API
type Consumer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConsumerSpec   `json:"spec,omitempty"`
	Status ConsumerStatus `json:"status,omitempty"`
}

func (c *Consumer) Marshal() string {
	auth := make(map[string]string, len(c.Spec.Auth))
	for k, v := range c.Spec.Auth {
		auth[k] = string(v.Config.Raw)
	}

	consumer := &csModel.Consumer{
		Auth: auth,
	}

	if len(c.Spec.Filters) > 0 {
		filters := make(map[string]*fmModel.FilterConfig, len(c.Spec.Filters))
		for k, v := range c.Spec.Filters {
			var config interface{}
			// we use interface{} here because we will introduce configuration merging one day
			_ = json.Unmarshal(v.Config.Raw, &config)
			filters[k] = &fmModel.FilterConfig{
				Config: config,
			}
		}
		consumer.Filters = filters
	}

	return consumer.Marshal()
}

func (c *Consumer) IsSpecChanged() bool {
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

func (c *Consumer) SetAccepted(reason ConditionReason, msg ...string) {
	conds, changed := addOrUpdateAcceptedCondition(c.Status.Conditions, c.Generation, reason, msg...)
	c.Status.Conditions = conds

	if changed {
		c.Status.MarkAsChanged()
	}
}

func (c *Consumer) IsValid() bool {
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

// ConsumerList contains a list of Consumer
type ConsumerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Consumer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Consumer{}, &ConsumerList{})
}
