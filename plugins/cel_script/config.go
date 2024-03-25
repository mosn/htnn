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

package cel_script

import (
	"github.com/google/cel-go/cel"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/pkg/expr"
)

const (
	Name = "celScript"
)

func init() {
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) Type() plugins.PluginType {
	return plugins.TypeTraffic
}

func (p *plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionTraffic,
	}
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	Config

	allowIfScript expr.Script
}

func (conf *config) Validate() error {
	err := conf.Config.Validate()
	if err != nil {
		return err
	}

	if conf.AllowIf != "" {
		_, err = expr.CompileCel(conf.AllowIf, cel.BoolType)
		if err != nil {
			return err
		}
	}
	return nil
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	if conf.AllowIf != "" {
		s, _ := expr.CompileCel(conf.AllowIf, cel.BoolType)
		conf.allowIfScript = s
	}
	return nil
}
