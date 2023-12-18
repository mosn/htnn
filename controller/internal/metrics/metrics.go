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
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	HTNNSubSystem        = "htnn"
	TranslateDurationKey = "translate_duration_seconds"
)

var (
	TranslateDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: HTNNSubSystem,
		Name:      TranslateDurationKey,
		Help:      "How long in seconds translate in a batch.",
		// minimal: 100 microseconds
		Buckets: prometheus.ExponentialBuckets(10e-5, 10, 10),
	}, []string{"controller"})

	HFPTranslateDurationObserver = TranslateDuration.WithLabelValues("httpfilterpolicy")
)

func init() {
	metrics.Registry.MustRegister(TranslateDuration)
}
