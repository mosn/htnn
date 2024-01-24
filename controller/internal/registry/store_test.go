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

package registry

import (
	"context"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	istioapi "istio.io/api/networking/v1beta1"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pkgRegistry "mosn.io/htnn/controller/pkg/registry"
)

func fakeClient(t *testing.T) client.Client {
	cfg := &rest.Config{}
	k8sClient, err := client.New(cfg, client.Options{})
	require.NoError(t, err)
	return k8sClient
}

func TestSync(t *testing.T) {
	store := newServiceEntryStore(fakeClient(t))

	var created, updated, deleted bool

	patches := gomonkey.ApplyMethodFunc(store.client, "List", func(c context.Context, list client.ObjectList, opts ...client.ListOption) error {
		serviceEntries := list.(*istiov1b1.ServiceEntryList)
		serviceEntries.Items = []*istiov1b1.ServiceEntry{
			// To delete
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "delete",
				},
			},
			// To update
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "update",
				},
				Spec: istioapi.ServiceEntry{
					Hosts: []string{"before"},
				},
			},
		}
		return nil
	})
	patches.ApplyMethodFunc(store.client, "Create", func(c context.Context, obj client.Object, opts ...client.CreateOption) error {
		created = true
		return nil
	})
	patches.ApplyMethodFunc(store.client, "Update", func(c context.Context, obj client.Object, opts ...client.UpdateOption) error {
		updated = true
		return nil
	})
	patches.ApplyMethodFunc(store.client, "Delete", func(c context.Context, obj client.Object, opts ...client.DeleteOption) error {
		deleted = true
		return nil
	})
	defer patches.Reset()

	store.entries = map[string]*pkgRegistry.ServiceEntryWrapper{
		// To update
		"update": {
			ServiceEntry: istioapi.ServiceEntry{
				Hosts: []string{"after"},
			},
		},
		// To add
		"add": {
			ServiceEntry: istioapi.ServiceEntry{
				Hosts: []string{"add"},
			},
		},
	}

	store.sync()
	assert.True(t, created)
	assert.True(t, updated)
	assert.True(t, deleted)
}
