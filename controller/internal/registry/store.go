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
	"time"

	"google.golang.org/protobuf/proto"
	istioapi "istio.io/api/networking/v1beta1"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/internal/model"
	pkgRegistry "mosn.io/htnn/controller/pkg/registry"
)

type serviceEntryStore struct {
	client client.Client

	lock    sync.RWMutex
	entries map[string]*istiov1b1.ServiceEntry

	syncInterval time.Duration
}

func newServiceEntryStore(client client.Client) *serviceEntryStore {
	return &serviceEntryStore{
		client:       client,
		entries:      make(map[string]*istiov1b1.ServiceEntry),
		syncInterval: 20 * time.Second,
	}
}

func (store *serviceEntryStore) Update(service string, se *pkgRegistry.ServiceEntryWrapper) {
	store.lock.Lock()
	defer store.lock.Unlock()

	ctx := context.Background()
	var obj istiov1b1.ServiceEntry
	var latestServiceEntry *istiov1b1.ServiceEntry

	if prev, ok := store.entries[service]; !ok {
		if err := store.getFromK8s(ctx, service, &obj); err != nil {
			if !apierrors.IsNotFound(err) {
				log.Errorf("failed to get service entry from k8s, err: %v, service: %s", err, service)
				return
			}

			latestServiceEntry = store.addToK8s(ctx, service, &se.ServiceEntry)
		} else {
			latestServiceEntry = store.updateToK8s(ctx, &obj, &se.ServiceEntry)
		}
	} else {
		latestServiceEntry = store.updateToK8s(ctx, prev, &se.ServiceEntry)
	}

	store.entries[service] = latestServiceEntry
}

// Implement ServiceEntryStore interface

func (store *serviceEntryStore) Delete(service string) {
	store.lock.Lock()
	defer store.lock.Unlock()
	if _, ok := store.entries[service]; !ok {
		// a service is registered without hosts, which will trigger a delete event
		return
	}

	delete(store.entries, service)

	ctx := context.Background()
	var se istiov1b1.ServiceEntry
	if err := store.getFromK8s(ctx, service, &se); err != nil {
		log.Errorf("failed to get service entry from k8s, err: %v, service: %s", err, service)
		return
	}
	store.deleteFromK8s(ctx, &se)
}

func (store *serviceEntryStore) getFromK8s(ctx context.Context, service string, se *istiov1b1.ServiceEntry) error {
	err := store.client.Get(ctx, client.ObjectKey{
		Namespace: config.RootNamespace(),
		Name:      service,
	}, se)
	return err
}

func (store *serviceEntryStore) deleteFromK8s(ctx context.Context, se *istiov1b1.ServiceEntry) {
	c := store.client
	log.Infof("delete ServiceEntry name %s, namespace %s", se.Name, se.Namespace)
	err := c.Delete(ctx, se)
	if err != nil {
		log.Errorf("failed to delete service entry from k8s, err: %v, service: %s", err, se.Name)
		return
	}
}

func (store *serviceEntryStore) addToK8s(ctx context.Context, service string, entry *istioapi.ServiceEntry) *istiov1b1.ServiceEntry {
	c := store.client
	ns := config.RootNamespace()
	se := istiov1b1.ServiceEntry{
		Spec: *entry.DeepCopy(),
	}
	se.Namespace = ns
	if se.Labels == nil {
		se.Labels = map[string]string{}
	}
	se.Labels[model.LabelCreatedBy] = "ServiceRegistry"
	se.Name = service

	log.Infof("create ServiceEntry name: %s, namespace: %s", service, ns)
	err := c.Create(ctx, &se)
	if err != nil {
		log.Errorf("failed to create service entry to k8s, err: %v, service: %s", err, service)
	}

	return &se
}

func (store *serviceEntryStore) updateToK8s(ctx context.Context, se *istiov1b1.ServiceEntry, entry *istioapi.ServiceEntry) *istiov1b1.ServiceEntry {
	if proto.Equal(&se.Spec, entry) {
		return se
	}

	c := store.client
	log.Infof("update ServiceEntry name %s, namespace %s", se.Name, se.Namespace)
	se.SetResourceVersion(se.ResourceVersion)
	se.Spec = *entry.DeepCopy()
	if err := c.Update(ctx, se); err != nil {
		log.Errorf("failed to update service entry to k8s, err: %v, service: %s", err, se.Name)
		return se
	}

	return se
}

func (store *serviceEntryStore) sync() {
	store.lock.Lock()
	defer store.lock.Unlock()

	c := store.client
	ctx := context.Background()
	var serviceEntries istiov1b1.ServiceEntryList
	err := c.List(ctx, &serviceEntries, client.MatchingLabels{model.LabelCreatedBy: "ServiceRegistry"})
	if err != nil {
		log.Errorf("failed to list service entries, err: %v", err)
		return
	}

	persisted := make(map[string]*istiov1b1.ServiceEntry, len(serviceEntries.Items))
	for _, se := range serviceEntries.Items {
		if _, ok := store.entries[se.Name]; !ok {
			store.deleteFromK8s(ctx, se)
		} else {
			persisted[se.Name] = se
		}
	}

	for service, wrp := range store.entries {
		entry := &wrp.Spec
		if se, ok := persisted[service]; !ok {
			store.addToK8s(ctx, service, entry)
		} else {
			store.updateToK8s(ctx, se, entry)
		}
	}
}

func (store *serviceEntryStore) Sync() {
	// We sync the service entries so we can retry if something wrong happened
	ticker := time.NewTicker(store.syncInterval)
	// For now we don't release the ticker
	for range ticker.C {
		store.sync()
	}
}
