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

	pkgConsumer "mosn.io/htnn/pkg/consumer"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ConsumerPlugin struct {
	Config runtime.RawExtension `json:"config"`
}

// ConsumerSpec defines the desired state of Consumer
type ConsumerSpec struct {
	Auth map[string]ConsumerPlugin `json:"auth"`
}

// ConsumerStatus defines the observed state of Consumer
type ConsumerStatus struct {
	// Conditions describe the current conditions of the SecurityPolicy.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

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
	auth := make(map[string][]byte, len(c.Spec.Auth))
	for k, v := range c.Spec.Auth {
		auth[k] = v.Config.Raw
	}
	consumer := &pkgConsumer.Consumer{
		ConsumerName: c.Name,
		Auth:         auth,
	}
	return consumer.Marshal()
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
