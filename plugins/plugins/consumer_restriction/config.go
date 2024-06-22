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

package consumer_restriction

import (
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/plugins/consumer_restriction"
)

func init() {
	plugins.RegisterHttpPlugin(consumer_restriction.Name, &plugin{})
}

type plugin struct {
	consumer_restriction.Plugin
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	consumer_restriction.Config

	allow     bool
	consumers map[string]*Rule
}

type Rule struct {
	Name    string
	Methods map[string]bool
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	rules := conf.GetDeny()
	if rules == nil {
		rules = conf.GetAllow()
		conf.allow = true
	}

	conf.consumers = make(map[string]*Rule, len(rules.Rules))
	for _, r := range rules.Rules {
		methods := make(map[string]bool)
		for _, method := range r.GetMethods() {
			methods[method] = true
		}
		conf.consumers[r.Name] = &Rule{
			Name:    r.Name,
			Methods: methods,
		}
	}
	return nil
}
