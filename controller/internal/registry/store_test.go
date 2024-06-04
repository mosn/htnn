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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"
	istioapi "istio.io/api/networking/v1alpha3"

	"mosn.io/htnn/controller/internal/controller/component"
	pkgRegistry "mosn.io/htnn/controller/pkg/registry"
	"mosn.io/htnn/controller/tests/pkg"
)

func TestStoreRepeatedUpdate(t *testing.T) {
	client := pkg.FakeK8sClient(t)
	out := component.NewK8sOutput(client)
	counter := 0

	patches := gomonkey.ApplyMethodFunc(out, "FromServiceRegistry", func(ctx interface{}, serviceEntries map[string]*istioapi.ServiceEntry) {
		counter++
	})
	defer patches.Reset()

	store := newServiceEntryStore(out)
	sew := &pkgRegistry.ServiceEntryWrapper{
		ServiceEntry: istioapi.ServiceEntry{
			Hosts: []string{"test.default-group.public.earth.nacos"},
		},
	}
	store.Update("test", sew)
	sew2 := &pkgRegistry.ServiceEntryWrapper{
		ServiceEntry: istioapi.ServiceEntry{
			Hosts: []string{"test.default-group.public.earth.nacos"},
		},
	}
	store.Update("test", sew2)

	require.Equal(t, 1, counter)
}
