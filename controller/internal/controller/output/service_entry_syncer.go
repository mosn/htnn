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

package output

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/protobuf/proto"
	istioapi "istio.io/api/networking/v1beta1"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/model"
)

type serviceEntrySyncer struct {
	client client.Client
	logger *logr.Logger

	lock         sync.RWMutex
	entries      map[string]*istiov1b1.ServiceEntry
	syncInterval time.Duration
}

func newServiceEntrySyncer(c client.Client, logger *logr.Logger) *serviceEntrySyncer {
	s := &serviceEntrySyncer{
		client:       c,
		logger:       logger,
		entries:      make(map[string]*istiov1b1.ServiceEntry),
		syncInterval: 20 * time.Second,
	}
	go s.Sync()
	return s
}

func (syncer *serviceEntrySyncer) getFromK8s(ctx context.Context, service string, se *istiov1b1.ServiceEntry) error {
	err := syncer.client.Get(ctx, client.ObjectKey{
		Namespace: config.RootNamespace(),
		Name:      service,
	}, se)
	return err
}

func (syncer *serviceEntrySyncer) deleteFromK8s(ctx context.Context, se *istiov1b1.ServiceEntry) {
	c := syncer.client
	syncer.logger.Info("delete ServiceEntry", "name", se.Name, "namespace", se.Namespace)
	err := c.Delete(ctx, se)
	if err != nil {
		syncer.logger.Error(err, "failed to delete service entry from k8s", "service", se.Name)
		return
	}
}

func (syncer *serviceEntrySyncer) addToK8s(ctx context.Context, service string, entry *istioapi.ServiceEntry) *istiov1b1.ServiceEntry {
	c := syncer.client
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

	syncer.logger.Info("create ServiceEntry", "name", service, "namespace", ns)
	err := c.Create(ctx, &se)
	if err != nil {
		syncer.logger.Error(err, "failed to create service entry to k8s", "service", service)
	}

	return &se
}

func (syncer *serviceEntrySyncer) updateToK8s(ctx context.Context, se *istiov1b1.ServiceEntry, entry *istioapi.ServiceEntry) *istiov1b1.ServiceEntry {
	if proto.Equal(&se.Spec, entry) {
		return se
	}

	c := syncer.client
	syncer.logger.Info("update ServiceEntry", "name", se.Name, "namespace", se.Namespace)
	se.SetResourceVersion(se.ResourceVersion)
	se.Spec = *entry.DeepCopy()
	if err := c.Update(ctx, se); err != nil {
		syncer.logger.Error(err, "failed to update service entry to k8s", "service", se.Name)
		return se
	}

	return se
}

func (syncer *serviceEntrySyncer) Update(ctx context.Context, entries map[string]*istioapi.ServiceEntry) {
	syncer.lock.Lock()
	defer syncer.lock.Unlock()

	var obj istiov1b1.ServiceEntry
	for service, se := range syncer.entries {
		if _, ok := entries[service]; !ok {
			syncer.deleteFromK8s(ctx, se)
			delete(syncer.entries, service)
		}
	}

	var latestServiceEntry *istiov1b1.ServiceEntry
	for service, se := range entries {
		if prev, ok := syncer.entries[service]; !ok {
			if err := syncer.getFromK8s(ctx, service, &obj); err != nil {
				if !apierrors.IsNotFound(err) {
					syncer.logger.Error(err, "failed to get service entry from k8s", "service", service)
					return
				}

				latestServiceEntry = syncer.addToK8s(ctx, service, se)
			} else {
				latestServiceEntry = syncer.updateToK8s(ctx, &obj, se)
			}
		} else {
			latestServiceEntry = syncer.updateToK8s(ctx, prev, se)
		}

		syncer.entries[service] = latestServiceEntry
	}
}

func (syncer *serviceEntrySyncer) sync() {
	syncer.lock.Lock()
	defer syncer.lock.Unlock()

	c := syncer.client
	ctx := context.Background()
	var serviceEntries istiov1b1.ServiceEntryList
	err := c.List(ctx, &serviceEntries, client.MatchingLabels{model.LabelCreatedBy: "ServiceRegistry"})
	if err != nil {
		syncer.logger.Error(err, "failed to list service entries")
		return
	}

	persisted := make(map[string]*istiov1b1.ServiceEntry, len(serviceEntries.Items))
	for _, se := range serviceEntries.Items {
		if _, ok := syncer.entries[se.Name]; !ok {
			syncer.deleteFromK8s(ctx, se)
		} else {
			persisted[se.Name] = se
		}
	}

	for service, wrp := range syncer.entries {
		entry := &wrp.Spec
		if se, ok := persisted[service]; !ok {
			syncer.addToK8s(ctx, service, entry)
		} else {
			syncer.updateToK8s(ctx, se, entry)
		}
	}
}

func (syncer *serviceEntrySyncer) Sync() {
	// We sync the service entries so we can retry if something wrong happened
	ticker := time.NewTicker(syncer.syncInterval)
	// For now we don't release the ticker
	for range ticker.C {
		syncer.sync()
	}
}
