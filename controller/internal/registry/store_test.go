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

	"github.com/stretchr/testify/assert"
	istioapi "istio.io/api/networking/v1beta1"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/controller/tests/pkg"
)

type syncTestClient struct {
	client.Client
	created, updated, deleted bool
}

func (cli *syncTestClient) List(c context.Context, list client.ObjectList, opts ...client.ListOption) error {
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
}

func (cli *syncTestClient) Create(c context.Context, obj client.Object, opts ...client.CreateOption) error {
	cli.created = true
	return nil
}

func (cli *syncTestClient) Update(c context.Context, obj client.Object, opts ...client.UpdateOption) error {
	cli.updated = true
	return nil
}

func (cli *syncTestClient) Delete(c context.Context, obj client.Object, opts ...client.DeleteOption) error {
	cli.deleted = true
	return nil
}

func TestSync(t *testing.T) {
	cli := &syncTestClient{
		Client: pkg.FakeK8sClient(t),
	}
	store := newServiceEntryStore(cli)
	store.entries = map[string]*istiov1b1.ServiceEntry{
		// To update
		"update": {
			Spec: istioapi.ServiceEntry{
				Hosts: []string{"after"},
			},
		},
		// To add
		"add": {
			Spec: istioapi.ServiceEntry{
				Hosts: []string{"add"},
			},
		},
	}

	store.sync()
	assert.True(t, cli.created)
	assert.True(t, cli.updated)
	assert.True(t, cli.deleted)
}
