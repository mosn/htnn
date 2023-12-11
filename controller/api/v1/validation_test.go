package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	istioapi "istio.io/api/networking/v1beta1"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"mosn.io/moe/pkg/plugins"
)

func TestValidateHTTPFilterPolicy(t *testing.T) {
	plugins.RegisterHttpPlugin("animal", &plugins.MockPlugin{})
	namespace := gwapiv1a2.Namespace("ns")

	tests := []struct {
		name   string
		policy *HTTPFilterPolicy
		err    string
	}{
		{
			name: "ok",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
					TargetRef: gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
						},
					},
					Filters: map[string]runtime.RawExtension{
						"animal": {
							Raw: []byte(`{"config":{"pet":"cat"}}`),
						},
					},
				},
			},
		},
		{
			name: "unknown",
			policy: &HTTPFilterPolicy{
				Spec: HTTPFilterPolicySpec{
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
		{
			name: "cross namespace",
			policy: &HTTPFilterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace",
				},
				Spec: HTTPFilterPolicySpec{
					TargetRef: gwapiv1a2.PolicyTargetReferenceWithSectionName{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHTTPFilterPolicy(tt.policy)
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
			name: "empty route name not allowed",
			err:  "route name is empty",
			vs: &istiov1b1.VirtualService{
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
			name: "delegate not supported",
			err:  "not supported",
			vs: &istiov1b1.VirtualService{
				Spec: istioapi.VirtualService{},
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
