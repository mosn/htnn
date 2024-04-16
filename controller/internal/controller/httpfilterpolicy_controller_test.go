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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"mosn.io/htnn/controller/internal/controller/component"
	pkgComponent "mosn.io/htnn/controller/pkg/component"
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

type FindAffectedObjectsWrapper struct {
	CustomerResourceIndexer

	succ bool
}

func (w *FindAffectedObjectsWrapper) FindAffectedObjects(ctx context.Context, obj pkgComponent.ResourceMeta) []reconcile.Request {
	if w.succ {
		return triggerReconciliation()
	}
	return nil
}

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
	policy.SetGroupVersionKind(schema.GroupVersionKind{
		Group: gwapiv1.GroupVersion.Group,
		Kind:  "test",
	})
	res := wrapClientObjectToResourceMeta(&policy)
	// unknown kind
	assert.False(t, r.NeedReconcile(ctx, res))

	r.addIndexer(&FindAffectedObjectsWrapper{r.httpRouteIndexer, false}, gwapiv1.GroupVersion.Group, "test")
	assert.False(t, r.NeedReconcile(ctx, res))

	r.addIndexer(&FindAffectedObjectsWrapper{r.httpRouteIndexer, true}, gwapiv1.GroupVersion.Group, "test")
	assert.True(t, r.NeedReconcile(ctx, res))
}
