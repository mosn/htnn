package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	istioapi "istio.io/api/networking/v1beta1"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	mosniov1 "mosn.io/moe/controller/api/v1"
	"mosn.io/moe/pkg/plugins"
)

func TestValidateHTTPFilterPolicy(t *testing.T) {
	plugins.RegisterHttpPlugin("animal", &plugins.MockPlugin{})

	tests := []struct {
		name   string
		policy *mosniov1.HTTPFilterPolicy
		err    string
	}{
		{
			name: "ok",
			policy: &mosniov1.HTTPFilterPolicy{
				Spec: mosniov1.HTTPFilterPolicySpec{
					TargetRef: gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
						},
					},
					Filters: map[string]runtime.RawExtension{
						"animal": {
							Raw: []byte(`{"pet":"cat"}`),
						},
					},
				},
			},
		},
		{
			name: "unknown",
			policy: &mosniov1.HTTPFilterPolicy{
				Spec: mosniov1.HTTPFilterPolicySpec{
					TargetRef: gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
						},
					},
					Filters: map[string]runtime.RawExtension{
						"property": {
							Raw: []byte(`{"pet":"cat"}`),
						},
					},
				},
			},
			err: "unknown http filter: property",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPFilterPolicy(tt.policy)
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}

func TestValidateVirtualService(t *testing.T) {
	tests := []struct {
		name string
		vs   *istiov1b1.VirtualService
		err  string
	}{
		{
			name: "delegate not supported",
			err:  "not supported",
			vs: &istiov1b1.VirtualService{
				Spec: istioapi.VirtualService{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVirtualService(tt.vs)
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}
