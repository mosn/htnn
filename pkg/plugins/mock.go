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

package plugins

import (
	"mosn.io/moe/pkg/filtermanager/api"
)

type MockPlugin struct {
	PluginMethodDefaultImpl
}

func (m *MockPlugin) ConfigFactory() api.FilterConfigFactory {
	return func(interface{}) api.FilterFactory { return nil }
}

func (m *MockPlugin) Config() PluginConfig {
	return &MockPluginConfig{}
}

func (m *MockPlugin) Merge(parent interface{}, child interface{}) interface{} {
	return child
}

type MockPluginConfig struct {
	Config
}

func (m *MockPluginConfig) Init(cb api.ConfigCallbackHandler) error {
	return nil
}
