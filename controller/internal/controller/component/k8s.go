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

package component

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"google.golang.org/protobuf/proto"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/pkg/component"
	"mosn.io/htnn/controller/pkg/constant"
)

type k8sOutput struct {
	client.Client
	logger logr.Logger

	serviceEntrySyncer *serviceEntrySyncer
}

func NewK8sOutput(c client.Client) component.Output {
	o := &k8sOutput{
		Client: c,
		logger: log.Logger(),
	}
	o.serviceEntrySyncer = newServiceEntrySyncer(c, &o.logger)
	return o
}

func (o *k8sOutput) FromFilterPolicy(ctx context.Context, generatedEnvoyFilters map[component.EnvoyFilterKey]*istiov1a3.EnvoyFilter) error {
	return o.diffGeneratedEnvoyFilters(ctx, "FilterPolicy", generatedEnvoyFilters)
}

func (o *k8sOutput) FromConsumer(ctx context.Context, ef *istiov1a3.EnvoyFilter) error {
	return o.diffGeneratedEnvoyFilter(ctx, "Consumer", ef)
}

func (o *k8sOutput) FromDynamicConfig(ctx context.Context, efs map[component.EnvoyFilterKey]*istiov1a3.EnvoyFilter) error {
	return o.diffGeneratedEnvoyFilters(ctx, "DynamicConfig", efs)
}

func (o *k8sOutput) diffGeneratedEnvoyFilters(ctx context.Context, creator string, generatedEnvoyFilters map[component.EnvoyFilterKey]*istiov1a3.EnvoyFilter) error {
	logger := o.logger

	var envoyfilters istiov1a3.EnvoyFilterList
	if err := o.List(ctx, &envoyfilters,
		client.MatchingLabels{constant.LabelCreatedBy: creator},
	); err != nil {
		return fmt.Errorf("failed to list EnvoyFilter: %w", err)
	}

	preEnvoyFilterMap := make(map[component.EnvoyFilterKey]*istiov1a3.EnvoyFilter, len(envoyfilters.Items))
	for _, e := range envoyfilters.Items {
		key := component.EnvoyFilterKey{
			Namespace: e.Namespace,
			Name:      e.Name,
		}
		if _, ok := generatedEnvoyFilters[key]; !ok {
			logger.Info("delete EnvoyFilter", "name", e.Name, "namespace", e.Namespace)
			if err := o.Delete(ctx, e); err != nil {
				return fmt.Errorf("failed to delete EnvoyFilter: %w, namespacedName: %v",
					err, types.NamespacedName{Name: e.Name, Namespace: e.Namespace})
			}
		} else {
			preEnvoyFilterMap[key] = e
		}
	}

	for key, ef := range generatedEnvoyFilters {
		envoyfilter, ok := preEnvoyFilterMap[key]
		if !ok {
			logger.Info("create EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)

			if err := o.Create(ctx, ef); err != nil {
				nsName := types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace}
				return fmt.Errorf("failed to create EnvoyFilter: %w, namespacedName: %v", err, nsName)
			}

		} else {
			if proto.Equal(&envoyfilter.Spec, &ef.Spec) {
				continue
			}

			logger.Info("update EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)
			// Address metadata.resourceVersion: Invalid value: 0x0 error
			ef.SetResourceVersion(envoyfilter.ResourceVersion)
			if err := o.Update(ctx, ef); err != nil {
				nsName := types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace}
				return fmt.Errorf("failed to update EnvoyFilter: %w, namespacedName: %v", err, nsName)
			}
		}
	}

	return nil
}

func (o *k8sOutput) diffGeneratedEnvoyFilter(ctx context.Context, creator string, ef *istiov1a3.EnvoyFilter) error {
	logger := o.logger

	nsName := types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace}
	var envoyfilters istiov1a3.EnvoyFilterList
	if err := o.List(ctx, &envoyfilters, client.MatchingLabels{constant.LabelCreatedBy: creator}); err != nil {
		return fmt.Errorf("failed to list EnvoyFilter: %w", err)
	}

	var envoyfilter *istiov1a3.EnvoyFilter
	for _, e := range envoyfilters.Items {
		if e.Namespace != nsName.Namespace || e.Name != nsName.Name {
			logger.Info("delete EnvoyFilter", "name", e.Name, "namespace", e.Namespace)

			if err := o.Delete(ctx, e); err != nil {
				return fmt.Errorf("failed to delete EnvoyFilter: %w, namespacedName: %v",
					err, types.NamespacedName{Name: e.Name, Namespace: e.Namespace})
			}
		} else {
			envoyfilter = e
		}
	}

	if envoyfilter == nil {
		logger.Info("create EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)

		if err := o.Create(ctx, ef.DeepCopy()); err != nil {
			return fmt.Errorf("failed to create EnvoyFilter: %w, namespacedName: %v", err, nsName)
		}
	} else {
		logger.Info("update EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)

		ef = ef.DeepCopy()
		ef.SetResourceVersion(envoyfilter.ResourceVersion)
		if err := o.Update(ctx, ef); err != nil {
			return fmt.Errorf("failed to update EnvoyFilter: %w, namespacedName: %v", err, nsName)
		}
	}

	return nil
}

func (o *k8sOutput) FromServiceRegistry(ctx context.Context, serviceEntries map[string]*istioapi.ServiceEntry) {
	o.serviceEntrySyncer.Update(ctx, serviceEntries)
}

type resourceManager struct {
	client.Client
}

func (r *resourceManager) Get(ctx context.Context, key client.ObjectKey, out client.Object) error {
	return r.Client.Get(ctx, key, out)
}

func (r *resourceManager) List(ctx context.Context, list client.ObjectList) error {
	return r.Client.List(ctx, list)
}

func (r *resourceManager) UpdateStatus(ctx context.Context, obj client.Object, status any) error {
	return r.Client.Status().Update(ctx, obj)
}

func NewK8sResourceManager(c client.Client) component.ResourceManager {
	return &resourceManager{c}
}
