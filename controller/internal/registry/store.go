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
	"sync"

	"google.golang.org/protobuf/proto"
	istioapi "istio.io/api/networking/v1alpha3"

	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/pkg/component"
	pkgRegistry "mosn.io/htnn/controller/pkg/registry"
)

type serviceEntryStore struct {
	output component.Output

	lock    sync.RWMutex
	entries map[string]*istioapi.ServiceEntry
}

func newServiceEntryStore(output component.Output) *serviceEntryStore {
	return &serviceEntryStore{
		output:  output,
		entries: make(map[string]*istioapi.ServiceEntry),
	}
}

// Implement ServiceEntryStore interface

func (store *serviceEntryStore) Update(service string, se *pkgRegistry.ServiceEntryWrapper) {
	store.lock.Lock()
	defer store.lock.Unlock()

	log.Infof("service entry store updates service: %s, entry: %v", service, &se.ServiceEntry)
	ctx := context.Background()

	if prev, ok := store.entries[service]; ok {
		// Some registry SDKs may send the same service entry multiple times. For example, at least in
		// nacos-sdk-go 1.1.4, when the service is first subscribed, the SDK will run the callback
		// twice. Here we decide to deduplicate in the store.
		if proto.Equal(&se.ServiceEntry, prev) {
			log.Infof("service %s not changed in service entry store, ignored", service)
			return
		}
	}
	store.entries[service] = &se.ServiceEntry

	store.output.FromServiceRegistry(ctx, store.entries)
}

func (store *serviceEntryStore) Delete(service string) {
	store.lock.Lock()
	defer store.lock.Unlock()
	if _, ok := store.entries[service]; !ok {
		// a service is registered without hosts, which will trigger a delete event
		return
	}

	log.Infof("service entry store deletes service: %s", service)
	delete(store.entries, service)
	store.output.FromServiceRegistry(context.Background(), store.entries)
}
