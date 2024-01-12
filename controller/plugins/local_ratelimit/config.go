// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package local_ratelimit

import (
	local_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"

	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/plugins"
)

const (
	Name = "local_ratelimit"
)

func init() {
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) Type() plugins.PluginType {
	return plugins.TypeTraffic
}

func (p *plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionPre,
	}
}

// Each Native plugin need to implement the methods below

func (p *plugin) Config() api.PluginConfig {
	return &local_ratelimit.LocalRateLimit{}
}

// RouteConfigTypeURL returns the type url of per-route config
func (p *plugin) RouteConfigTypeURL() string {
	return "type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit"
}

// DefaultHTTPFilterConfig returns the placeholder config for http filter
func (p *plugin) DefaultHTTPFilterConfig() map[string]interface{} {
	return map[string]interface{}{
		"typed_config": map[string]interface{}{
			"@type":      p.RouteConfigTypeURL(),
			"statPrefix": "http_local_rate_limiter",
		},
	}
}
