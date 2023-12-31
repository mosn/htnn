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

package controller

import (
	"context"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	mosniov1 "mosn.io/htnn/controller/api/v1"
)

func TestVirtualServiceIndexer(t *testing.T) {
	r := &mockReader{}
	vsi := &VirtualServiceIndexer{r: r}
	vs := vsi.CustomerResource()
	assert.Equal(t, &istiov1b1.VirtualService{}, vs)
	policies := []*mosniov1.HTTPFilterPolicy{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "xxx",
			},
			Spec: mosniov1.HTTPFilterPolicySpec{
				TargetRef: gwapiv1a2.PolicyTargetReferenceWithSectionName{
					PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
						Group: "networking.istio.io",
						Kind:  "VirtualService",
						Name:  "name",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "name",
			},
			Spec: mosniov1.HTTPFilterPolicySpec{
				TargetRef: gwapiv1a2.PolicyTargetReferenceWithSectionName{
					PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
						Group: "networking.istio.io",
						Kind:  "VirtualService",
						Name:  "vs",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "xxx",
			},
			Spec: mosniov1.HTTPFilterPolicySpec{
				TargetRef: gwapiv1a2.PolicyTargetReferenceWithSectionName{
					PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
						Group: "networking.istio.io",
						Kind:  "Gateway",
						Name:  "name",
					},
				},
			},
		},
	}

	cache := map[string]map[string]client.Object{
		vsi.IndexName(): {},
	}
	for _, po := range policies {
		idx := vsi.Index(po)
		if len(idx) == 0 {
			continue
		}
		cache[vsi.IndexName()][idx[0]] = po
	}

	ctx := context.Background()
	patches := gomonkey.ApplyMethodFunc(r, "List", func(c context.Context, list client.ObjectList, opts ...client.ListOption) error {
		assert.Equal(t, ctx, c)
		policies := list.(*mosniov1.HTTPFilterPolicyList)
		opt := opts[0].(*client.ListOptions)
		assert.Equal(t, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(vsi.IndexName(), "vs"),
		}, opt)

		policy := cache[vsi.IndexName()]["vs"].(*mosniov1.HTTPFilterPolicy)
		policies.Items = []mosniov1.HTTPFilterPolicy{*policy}
		return nil
	})
	defer patches.Reset()

	vs.SetNamespace("ns")
	vs.SetName("vs")
	reqs := vsi.FindAffectedObjects(ctx, vs)
	assert.Equal(t, 1, len(reqs))
	assert.Equal(t, types.NamespacedName{
		Namespace: "",
		Name:      "httpfilterpolicies",
	}, reqs[0].NamespacedName)
}
