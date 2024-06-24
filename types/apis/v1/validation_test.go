// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"mosn.io/htnn/api/pkg/plugins"
	_ "mosn.io/htnn/types/plugins"    // register plugin types
	_ "mosn.io/htnn/types/registries" // register registry types
)

func TestValidateHTTPFilterPolicy(t *testing.T) {
	plugins.RegisterHttpPluginType("animal", &plugins.MockPlugin{})
	namespace := gwapiv1.Namespace("ns")
	sectionName := gwapiv1.SectionName("test")

	tests := []struct {
		name      string
		policy    *HTTPFilterPolicy
		err       string
		strictErr string
	}{
		{
			name: "ok, VirtualService",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
						},
					},
					Filters: map[string]HTTPPlugin{
						"animal": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat"}`),
							},
						},
					},
				},
			},
		},
		{
			name: "ok, VirtualService with sectionName",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
						},
						SectionName: &sectionName,
					},
					Filters: map[string]HTTPPlugin{
						"animal": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat"}`),
							},
						},
					},
				},
			},
		},
		{
			name: "ok, embedded VirtualService",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					Filters: map[string]HTTPPlugin{
						"animal": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat"}`),
							},
						},
					},
				},
			},
		},
		{
			name: "unknown fields, VirtualService",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
						},
					},
					Filters: map[string]HTTPPlugin{
						"animal": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat", "unknown_fields":"should be ignored"}`),
							},
						},
					},
				},
			},
			strictErr: "unknown field \"unknown_fields\"",
		},
		{
			name: "ok, HTTPRoute",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "gateway.networking.k8s.io",
							Kind:  "HTTPRoute",
						},
					},
					Filters: map[string]HTTPPlugin{
						"animal": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat"}`),
							},
						},
					},
				},
			},
		},
		{
			name: "unsupported, HTTPRoute with sectionName",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "gateway.networking.k8s.io",
							Kind:  "HTTPRoute",
						},
						SectionName: &sectionName,
					},
					Filters: map[string]HTTPPlugin{
						"animal": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat"}`),
							},
						},
					},
				},
			},
			err: "targetRef.SectionName is not supported for HTTPRoute",
		},
		{
			name: "unknown fields, HTTPRoute",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "gateway.networking.k8s.io",
							Kind:  "HTTPRoute",
						},
					},
					Filters: map[string]HTTPPlugin{
						"animal": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat", "unknown_fields":"should be ignored"}`),
							},
						},
					},
				},
			},
			strictErr: "unknown field \"unknown_fields\"",
		},
		{
			name: "unknown",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
						},
					},
					Filters: map[string]HTTPPlugin{
						"property": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat"}`),
							},
						},
					},
				},
			},
			strictErr: "unknown http filter: property",
		},
		{
			name: "cross namespace",
			policy: &HTTPFilterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace",
				},
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Namespace: &namespace,
							Group:     "networking.istio.io",
							Kind:      "VirtualService",
						},
					},
				},
			},
			err: "namespace in TargetRef doesn't match HTTPFilterPolicy's namespace",
		},
		{
			name: "Filters in SubPolicies",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
						},
					},
					SubPolicies: []HTTPFilterSubPolicy{
						{
							SectionName: sectionName,
							Filters: map[string]HTTPPlugin{
								"property": {
									Config: runtime.RawExtension{
										Raw: []byte(`{"pet":"cat"}`),
									},
								},
							},
						},
					},
				},
			},
			strictErr: "unknown http filter: property",
		},
		{
			name: "targetRef.SectionName and SubPolicies can not be used together",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
						},
						SectionName: &sectionName,
					},
					SubPolicies: []HTTPFilterSubPolicy{
						{
							SectionName: sectionName,
						},
					},
				},
			},
			err: "targetRef.SectionName and SubPolicies can not be used together",
		},
		{
			name: "bad configuration",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
						},
					},
					Filters: map[string]HTTPPlugin{
						"localRatelimit": {
							Config: runtime.RawExtension{
								Raw: []byte(`{}`),
							},
						},
					},
				},
			},
			err: "invalid LocalRateLimit.StatPrefix: value length must be at least 1 runes",
		},
		{
			name: "ok, Istio Gateway",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "Gateway",
						},
					},
					Filters: map[string]HTTPPlugin{
						"animal": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat"}`),
							},
						},
					},
				},
			},
		},
		{
			name: "not implemented, Istio Gateway with Native Plugin",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "Gateway",
						},
					},
					Filters: map[string]HTTPPlugin{
						"localRatelimit": {
							Config: runtime.RawExtension{
								Raw: []byte(`{}`),
							},
						},
					},
				},
			},
			err: "configure native plugins to the Gateway is not implemented",
		},
		{
			name: "ok, k8s Gateway",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "gateway.networking.k8s.io",
							Kind:  "Gateway",
						},
					},
					Filters: map[string]HTTPPlugin{
						"animal": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat"}`),
							},
						},
					},
				},
			},
		},
		{
			name: "not implemented, k8s Gateway with Native Plugin",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: &gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "gateway.networking.k8s.io",
							Kind:  "Gateway",
						},
					},
					Filters: map[string]HTTPPlugin{
						"localRatelimit": {
							Config: runtime.RawExtension{
								Raw: []byte(`{}`),
							},
						},
					},
				},
			},
			err: "configure native plugins to the Gateway is not implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHTTPFilterPolicy(tt.policy)
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}

			err = ValidateHTTPFilterPolicyStrictly(tt.policy)
			if tt.strictErr == "" && tt.err == "" {
				assert.Nil(t, err)
			} else {
				exp := tt.strictErr
				if exp == "" {
					exp = tt.err
				}
				assert.ErrorContains(t, err, exp)
			}
		})
	}
}

func TestValidateVirtualService(t *testing.T) {
	tests := []struct {
		name string
		vs   *istiov1a3.VirtualService
		err  string
	}{
		{
			name: "empty route name not allowed",
			err:  "route name is empty",
			vs: &istiov1a3.VirtualService{
				Spec: istioapi.VirtualService{
					Http: []*istioapi.HTTPRoute{
						{
							Route: []*istioapi.HTTPRouteDestination{},
						},
					},
				},
			},
		},
		{
			name: "only http route is supported",
			err:  "only http route is supported",
			vs: &istiov1a3.VirtualService{
				Spec: istioapi.VirtualService{},
			},
		},
		{
			name: "delegate not supported",
			err:  "not supported",
			vs: &istiov1a3.VirtualService{
				Spec: istioapi.VirtualService{
					Http: []*istioapi.HTTPRoute{
						{
							Name: "test",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVirtualService(tt.vs)
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}

func TestValidateConsumer(t *testing.T) {
	tests := []struct {
		name     string
		consumer *Consumer
		err      string
	}{
		{
			name: "ok",
			consumer: &Consumer{
				Spec: ConsumerSpec{
					Auth: map[string]ConsumerPlugin{
						"keyAuth": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"key":"cat", "unknown_fields":"should be ignored"}`),
							},
						},
					},
					Filters: map[string]HTTPPlugin{
						"animal": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat", "unknown_fields":"should be ignored"}`),
							},
						},
					},
				},
			},
		},
		{
			name: "unknown",
			consumer: &Consumer{
				Spec: ConsumerSpec{
					Auth: map[string]ConsumerPlugin{
						"property": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"pet":"cat"}`),
							},
						},
					},
				},
			},
			err: "unknown authn filter: property",
		},
		{
			name: "bad configuration",
			consumer: &Consumer{
				Spec: ConsumerSpec{
					Auth: map[string]ConsumerPlugin{
						"keyAuth": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"key":1}`),
							},
						},
					},
				},
			},
			err: "invalid value for string type",
		},
		{
			name: "invalid config for filter",
			consumer: &Consumer{
				Spec: ConsumerSpec{
					Auth: map[string]ConsumerPlugin{
						"keyAuth": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"key":"cat"}`),
							},
						},
					},
					Filters: map[string]HTTPPlugin{
						"opa": {
							Config: runtime.RawExtension{
								Raw: []byte(`{}`),
							},
						},
					},
				},
			},
			err: "invalid config for filter opa",
		},
		{
			name: "invalid filter",
			consumer: &Consumer{
				Spec: ConsumerSpec{
					Auth: map[string]ConsumerPlugin{
						"keyAuth": {
							Config: runtime.RawExtension{
								Raw: []byte(`{"key":"cat"}`),
							},
						},
					},
					Filters: map[string]HTTPPlugin{
						"keyAuth": {
							Config: runtime.RawExtension{
								Raw: []byte(`{}`),
							},
						},
					},
				},
			},
			err: "this http filter can not be added by the consumer: keyAuth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConsumer(tt.consumer)
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}

func TestValidateServiceRegistry(t *testing.T) {
	tests := []struct {
		name     string
		registry *ServiceRegistry
		err      string
	}{
		{
			name: "ok",
			registry: &ServiceRegistry{
				Spec: ServiceRegistrySpec{
					Type: "nacos",
					Config: runtime.RawExtension{
						Raw: []byte(`{"serverUrl":"http://nacos.io", "unknown_fields":"should be ignored"}`),
					},
				},
			},
		},
		{
			name: "unknown",
			registry: &ServiceRegistry{
				Spec: ServiceRegistrySpec{
					Type: "unknown",
					Config: runtime.RawExtension{
						Raw: []byte(`{"serverUrl":"http://nacos.io"}`),
					},
				},
			},
			err: "unknown registry type: unknown",
		},
		{
			name: "bad configuration",
			registry: &ServiceRegistry{
				Spec: ServiceRegistrySpec{
					Type: "nacos",
					Config: runtime.RawExtension{
						Raw: []byte(`{"serverUrl":""}`),
					},
				},
			},
			err: "value must be absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceRegistry(tt.registry)
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}
