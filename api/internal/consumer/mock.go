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

package consumer

import (
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

type consumerPlugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *consumerPlugin) Type() plugins.PluginType {
	return plugins.TypeAuthn
}

func (p *consumerPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAuthn,
	}
}

func (p *consumerPlugin) Factory() api.FilterFactory {
	return func(interface{}, api.FilterCallbackHandler) api.Filter {
		return &api.PassThroughFilter{}
	}
}

func (p *consumerPlugin) Config() api.PluginConfig {
	return &Config{}
}

func (p *consumerPlugin) ConsumerConfig() api.PluginConsumerConfig {
	return &ConsumerConfig{}
}

func (conf *ConsumerConfig) Index() string {
	return conf.Key
}

type filterPlugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *filterPlugin) Type() plugins.PluginType {
	return plugins.TypeAuthz
}

func (p *filterPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAuthz,
	}
}

func (p *filterPlugin) Factory() api.FilterFactory {
	return func(interface{}, api.FilterCallbackHandler) api.Filter {
		return &api.PassThroughFilter{}
	}
}

func (p *filterPlugin) Config() api.PluginConfig {
	return &Config{}
}

type MockConsumer struct {
}

func (c *MockConsumer) Name() string {
	return "mock"
}

func (c *MockConsumer) PluginConfig(_ string) api.PluginConsumerConfig {
	return &ConsumerConfig{}
}
