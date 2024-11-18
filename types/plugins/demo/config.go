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

package demo

import (
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

const (
	Name = "demo"
)

func init() {
	// Register the plugin type with the given name
	plugins.RegisterPluginType(Name, &Plugin{})
}

type Plugin struct {
	// PluginMethodDefaultImpl is the base class of plugin which provides the default implementation
	// to Plugin methods.
	plugins.PluginMethodDefaultImpl
}

// Type returns type of the plugin, default to TypeGeneral
func (p *Plugin) Type() plugins.PluginType {
	// If a plugin doesn't claim its type, it will have type general.
	return plugins.TypeGeneral
}

// Order returns the order of the plugin, default to OrderPositionUnspecified
func (p *Plugin) Order() plugins.PluginOrder {
	// If a plugin doesn't claim its order, it will be put into OrderPositionUnspecified group.
	// The order of plugins in the group is decided by the operation. For plugins which have
	// same operation, they are sorted by alphabetical order.
	return plugins.PluginOrder{
		Position:  plugins.OrderPositionUnspecified,
		Operation: plugins.OrderOperationNop,
	}
}

// NonBlockingPhases returns the phases of the plugin which can be run non-blockingly, default to 0.
// If the plugin's filter doesn't contain any blocking operation, it should return true.
// A blocking operation can be:
// 1. I/O operation
// 2. Sleep
// 3. Blocking syscall
// 4. Context switch like waiting on a channel
// and more.
//
// If a phase only contains non-blocking plugins, it will be executed synchorously, which is
// more effective.
//
// Phase OnLog is always be executed synchorously so we don't need to specify it here.
func (p *Plugin) NonBlockingPhases() api.Phase {
	return api.PhaseDecodeHeaders | api.PhaseEncodeHeaders
}

// Config returns api.PluginConfig's implementation used during configuration processing
func (p *Plugin) Config() api.PluginConfig {
	return &Config{}
}
