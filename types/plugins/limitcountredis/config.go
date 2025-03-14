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

package limitcountredis

import (
	"fmt"
	"net"

	"github.com/google/cel-go/cel"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/pkg/expr"
)

const (
	Name = "limitCountRedis"
)

func init() {
	plugins.RegisterPluginType(Name, &Plugin{})
}

type Plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *Plugin) Type() plugins.PluginType {
	return plugins.TypeTraffic
}

func (p *Plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionTraffic,
	}
}

func (p *Plugin) NonBlockingPhases() api.Phase {
	return api.PhaseEncodeHeaders
}

func (p *Plugin) Config() api.PluginConfig {
	return &CustomConfig{}
}

type CustomConfig struct {
	Config
}

func (conf *CustomConfig) Validate() error {
	err := conf.Config.Validate()
	if err != nil {
		return err
	}

	addr := conf.GetAddress()
	if addr != "" {
		_, _, err = net.SplitHostPort(addr)
		if err != nil {
			return fmt.Errorf("bad address %s: %w", addr, err)
		}
	}
	cluster := conf.GetCluster()
	if cluster != nil {
		addrs := cluster.Addresses
		for _, addr := range addrs {
			_, _, err = net.SplitHostPort(addr)
			if err != nil {
				return fmt.Errorf("bad address %s: %w", addr, err)
			}
		}
	}

	for i, rule := range conf.Rules {
		if rule.Key == "" {
			continue
		}
		_, err = expr.CompileCel(rule.Key, cel.StringType)
		if err != nil {
			return fmt.Errorf("bad rule %d: %w", i, err)
		}
	}

	if conf.Username != "" && conf.Password == "" {
		return fmt.Errorf("password is required when username is set")
	}

	return nil
}
