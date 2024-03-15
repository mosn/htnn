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
	"fmt"

	"github.com/go-logr/logr"
	"google.golang.org/protobuf/proto"
	istioapi "istio.io/api/networking/v1beta1"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/model"
	"mosn.io/htnn/controller/pkg/procession"
	"mosn.io/htnn/pkg/log"
)

type k8sOutput struct {
	client.Client
	logger logr.Logger

	serviceEntrySyncer *serviceEntrySyncer
}

func NewK8sOutput(c client.Client) procession.Output {
	o := &k8sOutput{
		Client: c,
		logger: log.DefaultLogger.WithName("k8s output"),
	}
	o.serviceEntrySyncer = newServiceEntrySyncer(c, &o.logger)
	return o
}

func fillEnvoyFilterMeta(ef *istiov1a3.EnvoyFilter) {
	ef.Namespace = config.RootNamespace()
	if ef.Labels == nil {
		ef.Labels = map[string]string{}
	}
	ef.Labels[model.LabelCreatedBy] = "HTTPFilterPolicy"
}

func (o *k8sOutput) FromHTTPFilterPolicy(ctx context.Context, generatedEnvoyFilters map[string]*istiov1a3.EnvoyFilter) error {
	logger := o.logger

	var envoyfilters istiov1a3.EnvoyFilterList
	if err := o.List(ctx, &envoyfilters,
		client.MatchingLabels{model.LabelCreatedBy: "HTTPFilterPolicy"},
	); err != nil {
		return fmt.Errorf("failed to list EnvoyFilter: %w", err)
	}

	preEnvoyFilterMap := make(map[string]*istiov1a3.EnvoyFilter, len(envoyfilters.Items))
	for _, e := range envoyfilters.Items {
		if _, ok := generatedEnvoyFilters[e.Name]; !ok || e.Namespace != config.RootNamespace() {
			logger.Info("delete EnvoyFilter", "name", e.Name, "namespace", e.Namespace)
			if err := o.Delete(ctx, e); err != nil {
				return fmt.Errorf("failed to delete EnvoyFilter: %w, namespacedName: %v",
					err, types.NamespacedName{Name: e.Name, Namespace: e.Namespace})
			}
		} else {
			preEnvoyFilterMap[e.Name] = e
		}
	}

	for _, ef := range generatedEnvoyFilters {
		envoyfilter, ok := preEnvoyFilterMap[ef.Name]
		if !ok {
			logger.Info("create EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)
			fillEnvoyFilterMeta(ef)

			if err := o.Create(ctx, ef); err != nil {
				nsName := types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace}
				return fmt.Errorf("failed to create EnvoyFilter: %w, namespacedName: %v", err, nsName)
			}

		} else {
			if proto.Equal(&envoyfilter.Spec, &ef.Spec) {
				continue
			}

			logger.Info("update EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)
			fillEnvoyFilterMeta(ef)
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

func (o *k8sOutput) FromConsumer(ctx context.Context, ef *istiov1a3.EnvoyFilter) error {
	logger := o.logger

	nsName := types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace}
	var envoyfilters istiov1a3.EnvoyFilterList
	if err := o.List(ctx, &envoyfilters, client.MatchingLabels{model.LabelCreatedBy: "Consumer"}); err != nil {
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
