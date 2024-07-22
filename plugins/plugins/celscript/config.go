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

package celscript

import (
	"github.com/google/cel-go/cel"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/pkg/expr"
	"mosn.io/htnn/types/plugins/celscript"
)

func init() {
	plugins.RegisterPlugin(celscript.Name, &plugin{})
}

type plugin struct {
	celscript.Plugin
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	celscript.CustomConfig

	allowIfScript expr.Script
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	if conf.AllowIf != "" {
		s, _ := expr.CompileCel(conf.AllowIf, cel.BoolType)
		conf.allowIfScript = s
	}
	return nil
}
