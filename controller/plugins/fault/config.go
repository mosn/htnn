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

package fault

import (
	fault "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/fault/v3"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

const (
	Name = "fault"
)

func init() {
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) Type() plugins.PluginType {
	return plugins.TypeGeneral
}

func (p *plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position:  plugins.OrderPositionOuter,
		Operation: plugins.OrderOperationInsertLast,
	}
}

func (p *plugin) Config() api.PluginConfig {
	return &fault.HTTPFault{}
}

func (p *plugin) RouteConfigTypeURL() string {
	return "type.googleapis.com/envoy.extensions.filters.http.fault.v3.HTTPFault"
}

func (p *plugin) HTTPFilterConfigPlaceholder() map[string]interface{} {
	return map[string]interface{}{
		"typed_config": map[string]interface{}{
			"@type": p.RouteConfigTypeURL(),
		},
	}
}
