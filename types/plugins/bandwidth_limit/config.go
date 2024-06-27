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

package bandwidth_limit

import (
	bandwidth_limit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/bandwidth_limit/v3"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

const (
	Name = "bandwidthLimit"
)

func init() {
	plugins.RegisterPluginType(Name, &Plugin{})
}

type Plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *Plugin) Type() plugins.PluginType {
	return plugins.TypeTraffic
}

func (p *Plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionOuter,
	}
}

func (p *Plugin) Config() api.PluginConfig {
	return &bandwidth_limit.BandwidthLimit{}
}
