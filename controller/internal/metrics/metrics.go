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

package metrics

import (
	"fmt"

	"mosn.io/htnn/controller/pkg/component"
)

const (
	FP                      = "htnn_filterpolicy"
	Consumer                = "htnn_consumer"
	SR                      = "htnn_service_registry"
	DC                      = "htnn_dynamic_config"
	TranslateDurationSuffix = "translate_duration_seconds"
	ReconcileDurationSuffix = "reconcile_duration_seconds"
)

type voidMetric struct {
}

func (m *voidMetric) Record(value float64) {}

var (
	FPTranslateDurationDistribution              component.Distribution = &voidMetric{}
	FPReconcileDurationDistribution              component.Distribution = &voidMetric{}
	ConsumerReconcileDurationDistribution        component.Distribution = &voidMetric{}
	ServiceRegistryReconcileDurationDistribution component.Distribution = &voidMetric{}
	DynamicConfigReconcileDurationDistribution   component.Distribution = &voidMetric{}
)

func InitMetrics(provider component.MetricProvider) {
	FPTranslateDurationDistribution = provider.NewDistribution(fmt.Sprintf("%s_%s", FP, TranslateDurationSuffix),
		"How long in seconds HTNN translates FilterPolicy in a batch.",
		// minimal: 100 microseconds
		[]float64{1e-4, 1e-3, 0.01, 0.1, 1, 10},
	)
	FPReconcileDurationDistribution = provider.NewDistribution(fmt.Sprintf("%s_%s", FP, ReconcileDurationSuffix),
		"How long in seconds HTNN reconciles FilterPolicy.",
		// Reconciliation time = Fetch resource time + Translate time + Write Envoy Filter to config store time
		// minimal: 100 microseconds
		[]float64{1e-4, 1e-3, 0.01, 0.1, 1, 10},
	)
	ConsumerReconcileDurationDistribution = provider.NewDistribution(fmt.Sprintf("%s_%s", Consumer, ReconcileDurationSuffix),
		"How long in seconds HTNN reconciles Consumer.",
		// minimal: 100 microseconds
		[]float64{1e-4, 1e-3, 0.01, 0.1, 1, 10},
	)
	ServiceRegistryReconcileDurationDistribution = provider.NewDistribution(fmt.Sprintf("%s_%s", SR, ReconcileDurationSuffix),
		"How long in seconds HTNN reconciles ServiceRegistry.",
		// minimal: 100 microseconds
		[]float64{1e-4, 1e-3, 0.01, 0.1, 1, 10},
	)
	DynamicConfigReconcileDurationDistribution = provider.NewDistribution(fmt.Sprintf("%s_%s", DC, ReconcileDurationSuffix),
		"How long in seconds HTNN reconciles DynamicConfig.",
		// minimal: 100 microseconds
		[]float64{1e-4, 1e-3, 0.01, 0.1, 1, 10},
	)
}
