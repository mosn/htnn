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

	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/plugins"
)

func init() {
	plugins.RegisterHttpPlugin("preLua", &prePlugin{})
	plugins.RegisterHttpPlugin("postLua", &postPlugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) Config() api.PluginConfig {
	return &lua.LuaPerRoute{}
}

func (p *plugin) RouteConfigTypeURL() string {
	return "type.googleapis.com/envoy.extensions.filters.http.lua.v3.LuaPerRoute"
}

func (p *plugin) DefaultHTTPFilterConfig() map[string]interface{} {
	return map[string]interface{}{
		"typed_config": map[string]interface{}{
			"@type": "type.googleapis.com/envoy.extensions.filters.http.lua.v3.Lua",
		},
	}
}

type prePlugin struct {
	plugin
}

func (p *prePlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionPre,
	}
}

type postPlugin struct {
	plugin
}

func (p *postPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionPost,
	}
}
