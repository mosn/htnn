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

package casbin

import (
	sync "sync"

	"github.com/casbin/casbin/v2"

	"mosn.io/htnn/pkg/file"
	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/plugins"
)

const (
	Name = "casbin"
)

func init() {
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) Type() plugins.PluginType {
	return plugins.TypeAuthz
}

func (p *plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAuthz,
	}
}

func (p *plugin) ConfigFactory() api.FilterConfigFactory {
	return configFactory
}

func (p *plugin) Config() plugins.PluginConfig {
	return &config{}
}

type config struct {
	Config

	lock *sync.RWMutex

	enforcer   *casbin.Enforcer
	modelFile  *file.File
	policyFile *file.File
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	conf.lock = &sync.RWMutex{}

	f, err := file.Stat(conf.Rule.Model)
	if err != nil {
		return err
	}
	conf.modelFile = f

	f, err = file.Stat(conf.Rule.Policy)
	if err != nil {
		return err
	}
	conf.policyFile = f

	e, err := casbin.NewEnforcer(conf.Rule.Model, conf.Rule.Policy)
	if err != nil {
		return err
	}
	conf.enforcer = e
	return nil
}
