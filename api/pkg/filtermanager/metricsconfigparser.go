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

package filtermanager

import (
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	pkgPlugins "mosn.io/htnn/api/pkg/plugins"
)

type metricsConfigFilter struct {
	capi.PassThroughStreamFilter

	callbacks capi.FilterCallbackHandler
}

func MetricsConfigFactory(_ interface{}, callbacks capi.FilterCallbackHandler) capi.StreamFilter {
	return &metricsConfigFilter{
		callbacks: callbacks,
	}
}

type MetricsConfigParser struct {
}

// MetricsConfigParser is the parser to register metrics only, no real parsing
func (p *MetricsConfigParser) Parse(any *anypb.Any, callbacks capi.ConfigCallbackHandler) (interface{}, error) {
	if callbacks == nil {
		api.LogErrorf("no config callback handler provided")
		// the call back handler to be nil only affects plugin metrics, so we can continue
	}
	counterMetrics := pkgPlugins.GetCounterMetricsForCallback()
	for m := range counterMetrics {
		counterMetrics[m] = callbacks.DefineCounterMetric(m)
		api.LogInfof("initialized counter metrics for %s", m)
	}

	return any, nil
}

// just to satisfy the interface, no real merge
func (p *MetricsConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	return parent
}
