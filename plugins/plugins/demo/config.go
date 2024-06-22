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
	"net/http"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/plugins/demo"
)

func init() {
	// Register the plugin with the given name
	plugins.RegisterHttpPlugin(demo.Name, &plugin{})
}

type plugin struct {
	// A plugin should embed the type definition in the correndsponding type package.
	demo.Plugin
}

// Each Go plugin need to implement the two methods below

// Factory returns api.Factory's implementation used during request processing
func (p *plugin) Factory() api.FilterFactory {
	return factory
}

// Factory returns api.PluginConfig's implementation used during configuration processing
func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	// Config is generated from `config.proto` in the correndsponding type package.
	// The returned implementation should embed the Config.
	demo.Config

	client *http.Client
}

// Init allows the initialization of non-generated fields.
// This method is not called until the configuration is used, for example,
// when the first request is received. It's nonblocking so we can do IO
// operations in this method.
//
// If the plugin is configured for a gateway/consumer, it's only run once during
// processing the first request to the route which is using the gateway/consumer.
//
// So far, if multiple plugins are configured for a gateway/route/consumer, and if
// one of their configuration is changed, all the plugins will be re-initialized.
// Because as an extension, the plugin doesn't have its own lifecycle and it is
// created or destroyed at the same time as its parent. For example, assume we have
// a Route which has Plugin A & B,
// Plugin A conf changed -> Route conf changed -> Route re-init -> Plugin B also re-init
//
// This method is optional.
func (c *config) Init(cb api.ConfigCallbackHandler) error {
	c.client = http.DefaultClient
	return nil
}
