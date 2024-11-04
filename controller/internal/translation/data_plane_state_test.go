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

package translation

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func TestHostMatch(t *testing.T) {
	matched := []string{"*", "*.com", "*.test.com", "v.test.com"}
	mismatched := []string{"a.test.com", "*.t.com"}

	for _, m := range matched {
		if !hostMatch(m, "v.test.com") {
			t.Errorf("hostMatch(%s, v.test.com) should be true", m)
		}
	}
	for _, m := range mismatched {
		if hostMatch(m, "v.test.com") {
			t.Errorf("hostMatch(%s, v.test.com) should be false", m)
		}
	}

	// host is a wildcard domain
	matched = []string{"*", "*.com", "*.test.com", "v.test.com", "ab.test.com"}
	mismatched = []string{"*.t.com", "test.com"}

	for _, m := range matched {
		if !hostMatch(m, "*.test.com") {
			t.Errorf("hostMatch(%s, *.test.com) should be true", m)
		}
	}
	for _, m := range mismatched {
		if hostMatch(m, "*.test.com") {
			t.Errorf("hostMatch(%s, *.test.com) should be false", m)
		}
	}

	matched = []string{"*", "*.com", "a.com"}

	for _, m := range matched {
		if !hostMatch(m, "*") {
			t.Errorf("hostMatch(%s, *) should be true", m)
		}
	}
}

func ptrstr(s string) *string {
	return &s
}

func ptrFrom(from gwapiv1.FromNamespaces) *gwapiv1.FromNamespaces {
	return &from
}

func TestAllowRoute(t *testing.T) {
	var tests = []struct {
		name     string
		expected bool
		cond     *gwapiv1.AllowedRoutes
	}{
		{
			name:     "no allowRoute",
			expected: true,
		},
		{
			name: "Kind mismatched",
			cond: &gwapiv1.AllowedRoutes{
				Kinds: []gwapiv1.RouteGroupKind{
					{
						Group: (*gwapiv1.Group)(ptrstr("networking.htnn.io")),
					},
					{
						Kind: gwapiv1.Kind("GRPCRoute"),
					},
				},
			},
		},
		{
			name: "Selector mismatched",
			cond: &gwapiv1.AllowedRoutes{
				Namespaces: &gwapiv1.RouteNamespaces{
					From: ptrFrom(gwapiv1.NamespacesFromSelector),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "in",
						},
					},
				},
			},
		},
		{
			name: "namespace mismatched",
			cond: &gwapiv1.AllowedRoutes{
				Namespaces: &gwapiv1.RouteNamespaces{
					From: ptrFrom(gwapiv1.NamespacesFromSame),
				},
			},
		},
		{
			name: "all namespace",
			cond: &gwapiv1.AllowedRoutes{
				Namespaces: &gwapiv1.RouteNamespaces{
					From: ptrFrom(gwapiv1.NamespacesFromAll),
				},
			},
			expected: true,
		},
		{
			name: "pass",
			cond: &gwapiv1.AllowedRoutes{
				Kinds: []gwapiv1.RouteGroupKind{
					{
						Group: (*gwapiv1.Group)(ptrstr("networking.k8s.io")),
						Kind:  gwapiv1.Kind("HTTPRoute"),
					},
				},
				Namespaces: &gwapiv1.RouteNamespaces{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "ingress",
						},
					},
				},
			},
			expected: true,
		},
	}

	route := &gwapiv1b1.HTTPRoute{
		Spec: gwapiv1b1.HTTPRouteSpec{},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": "ingress",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPRoute",
			APIVersion: "networking.k8s.io/v1",
		},
	}
	gwNsName := &types.NamespacedName{
		Name:      "gw",
		Namespace: "ingress",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := AllowRoute(tt.cond, route, gwNsName)
			if actual != tt.expected {
				t.Errorf("(%s): expected %v, actual %v", tt.name, tt.expected, actual)
			}

		})
	}
}
