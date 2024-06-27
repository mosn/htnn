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
	"mosn.io/htnn/api/pkg/filtermanager/api"
)

type MockPlugin struct {
	PluginMethodDefaultImpl
}

func (m *MockPlugin) Factory() api.FilterFactory {
	return func(interface{}, api.FilterCallbackHandler) api.Filter { return nil }
}

func (m *MockPlugin) Config() api.PluginConfig {
	return &MockPluginConfig{}
}

func (m *MockPlugin) Merge(parent interface{}, child interface{}) interface{} {
	return child
}

var _ GoPlugin = &MockPlugin{}

type MockPluginConfig struct {
	Config
}

type MockConsumerPlugin struct {
	MockPlugin
}

func (m *MockConsumerPlugin) Order() PluginOrder {
	return PluginOrder{
		Position: OrderPositionAuthn,
	}
}

func (m *MockConsumerPlugin) ConsumerConfig() api.PluginConsumerConfig {
	return nil
}

var _ ConsumerPlugin = &MockConsumerPlugin{}

type MockNativePlugin struct {
	PluginMethodDefaultImpl
}

func (m *MockNativePlugin) Config() api.PluginConfig {
	return &MockPluginConfig{}
}

func (m *MockNativePlugin) Order() PluginOrder {
	return PluginOrder{
		Position: OrderPositionOuter,
	}
}

func (m *MockNativePlugin) ConfigTypeURL() string {
	return ""
}

var _ NativePlugin = &MockNativePlugin{}

type MockHTTPNativePlugin struct {
	MockNativePlugin
}

func (m *MockHTTPNativePlugin) FilterConfigPlaceholder() map[string]interface{} {
	return nil
}

var _ HTTPNativePlugin = &MockHTTPNativePlugin{}

var _ NativePlugin = &MockNativePlugin{}

type MockNetworkNativePlugin struct {
	MockNativePlugin
}

func (m *MockNetworkNativePlugin) Order() PluginOrder {
	return PluginOrder{
		Position: OrderPositionNetwork,
	}
}

var _ NativePlugin = &MockNativePlugin{}
