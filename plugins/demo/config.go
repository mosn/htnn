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

const (
	Name = "demo"
)

func init() {
	// Register the plugin with the given name
	plugins.RegisterHttpPlugin(Name, &plugin{})
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

// Init allows the initialization of non-generated fields during configuration processing.
// This method is run in Envoy's main thread, so it doesn't block the request processing.
// This method is optional.
func (c *config) Init(cb api.ConfigCallbackHandler) error {
	c.client = http.DefaultClient
	return nil
}
