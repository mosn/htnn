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

package plugin_state

import (
	"mosn.io/htnn/pkg/filtermanager/api"
)

type pluginState struct {
	store map[string]map[string]any
}

func NewPluginState() api.PluginState {
	return &pluginState{
		store: make(map[string]map[string]any),
	}
}

func (p *pluginState) Get(pluginName string, key string) any {
	if pluginStore, ok := p.store[pluginName]; ok {
		return pluginStore[key]
	}
	return nil
}

func (p *pluginState) Set(pluginName string, key string, value any) {
	pluginStore, ok := p.store[pluginName]
	if !ok {
		pluginStore = make(map[string]any)
		p.store[pluginName] = pluginStore
	}
	pluginStore[key] = value
}
