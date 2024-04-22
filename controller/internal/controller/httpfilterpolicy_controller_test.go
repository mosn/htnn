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

package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"mosn.io/htnn/controller/internal/controller/component"
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func TestNeedReconcile(t *testing.T) {
	cli := pkg.FakeK8sClient(t)
	output := component.NewK8sOutput(cli)
	rm := component.NewK8sResourceManager(cli)
	r := NewHTTPFilterPolicyReconciler(
		output,
		rm,
	)

	ctx := context.Background()
	policy := mosniov1.HTTPFilterPolicy{}
	route := gwapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "name",
		},
	}
	res := wrapClientObjectToResourceMeta(&route, gwapiv1.GroupVersion.Group, "test")
	// unknown kind
	assert.False(t, r.NeedReconcile(ctx, res))

	r.httpRouteIndexer.Kind = "test"
	r.addIndexer(r.httpRouteIndexer)
	assert.False(t, r.NeedReconcile(ctx, res))

	r.httpRouteIndexer.index = map[string][]*mosniov1.HTTPFilterPolicy{
		"ns/name": {&policy},
	}
	assert.True(t, r.NeedReconcile(ctx, res))
}
