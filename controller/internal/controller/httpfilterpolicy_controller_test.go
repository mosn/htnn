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

	mosniov1 "mosn.io/moe/controller/api/v1"
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
			Namespace:     "ns",
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
		Namespace: "ns",
		Name:      "name",
	}, reqs[0].NamespacedName)
}
