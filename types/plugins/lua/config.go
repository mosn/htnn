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

package lua

import (
	lua "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

func init() {
	plugins.RegisterHttpPluginType("outerLua", &OuterPlugin{})
	plugins.RegisterHttpPluginType("innerLua", &InnerPlugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) Config() api.PluginConfig {
	return &lua.LuaPerRoute{}
}

type OuterPlugin struct {
	plugin
}

func (p *OuterPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionOuter,
	}
}

type InnerPlugin struct {
	plugin
}

func (p *InnerPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionInner,
	}
}
