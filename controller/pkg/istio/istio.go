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

package istio

// This module stores API used by the istio patches
import (
	"context"
	"fmt"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/controller"
	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/internal/metrics"
	"mosn.io/htnn/controller/internal/registry"
	"mosn.io/htnn/controller/pkg/component"
)

type Reconciler interface {
	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}

type FilterPolicyReconciler interface {
	Reconciler

	NeedReconcile(ctx context.Context, meta component.ResourceMeta) bool
}

func NewFilterPolicyReconciler(output component.Output, manager component.ResourceManager) FilterPolicyReconciler {
	return controller.NewFilterPolicyReconciler(
		output,
		manager,
	)
}

type ConsumerReconciler interface {
	Reconciler
}

func NewConsumerReconciler(output component.Output, manager component.ResourceManager) ConsumerReconciler {
	return &controller.ConsumerReconciler{
		Output:          output,
		ResourceManager: manager,
	}
}

type ServiceRegistryReconciler interface {
	Reconciler
}

func NewServiceRegistryReconciler(output component.Output, manager component.ResourceManager) ServiceRegistryReconciler {
	registry.InitRegistryManager(&registry.RegistryManagerOption{
		Output: output,
	})
	return controller.NewServiceRegistryReconciler(
		manager,
	)
}

func SetLogger(logger component.CtrlLogger) {
	log.SetLogger(logger)
}

func InitConfig(enableGatewayAPI bool, rootNamespace string) {
	scopeGatewayToNamespace := os.Getenv("PILOT_SCOPE_GATEWAY_TO_NAMESPACE")
	if scopeGatewayToNamespace != "true" {
		// HTNN will generate the EnvoyFilter according to the gateway's namespace,
		// so we require the namespace of workload matches the namespace of gateway.
		log.Info("Env variable PILOT_SCOPE_GATEWAY_TO_NAMESPACE is not set to true. In this case we need to follow the convention that the namespace of workload matches the namespace of gateway.")
	}

	os.Setenv("HTNN_ENABLE_GATEWAY_API", fmt.Sprintf("%t", enableGatewayAPI))
	os.Setenv("HTNN_ISTIO_ROOT_NAMESPACE", rootNamespace)
	config.Init()
}

func InitMetrics(provider component.MetricProvider) {
	metrics.InitMetrics(provider)
}
