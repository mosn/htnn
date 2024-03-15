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

	istioapi "istio.io/api/networking/v1beta1"

	"mosn.io/htnn/controller/pkg/procession"
	pkgRegistry "mosn.io/htnn/controller/pkg/registry"
)

type serviceEntryStore struct {
	output procession.Output

	lock    sync.RWMutex
	entries map[string]*istioapi.ServiceEntry
}

func newServiceEntryStore(output procession.Output) *serviceEntryStore {
	return &serviceEntryStore{
		output:  output,
		entries: make(map[string]*istioapi.ServiceEntry),
	}
}

// Implement ServiceEntryStore interface

func (store *serviceEntryStore) Update(service string, se *pkgRegistry.ServiceEntryWrapper) {
	store.lock.Lock()
	defer store.lock.Unlock()

	logger.Info("service entry store update", "service", service, "entry", &se.ServiceEntry)
	ctx := context.Background()
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

	logger.Info("service entry store delete", "service", service)
	delete(store.entries, service)
	store.output.FromServiceRegistry(context.Background(), store.entries)
}
