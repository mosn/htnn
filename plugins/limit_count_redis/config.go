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

package limit_count_redis

import (
	"fmt"
	"net"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"mosn.io/htnn/pkg/expr"
	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/plugins"
)

const (
	Name = "limitCountRedis"
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

func (p *plugin) ConfigFactory() api.FilterConfigFactory {
	return configFactory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	Config

	client      *redis.Client
	limiters    []*Limiter
	quotaPolicy string
}

type Limiter struct {
	script     expr.Script
	count      uint32
	timeWindow int64
	prefix     string
}

func (conf *config) Validate() error {
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

	for i, rule := range conf.Rules {
		if rule.Key == "" {
			continue
		}
		_, err = expr.CompileCel(rule.Key, cel.StringType)
		if err != nil {
			return fmt.Errorf("bad rule %d: %w", i, err)
		}
	}

	return nil
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	addr := conf.GetAddress()
	conf.client = redis.NewClient(&redis.Options{
		Addr: addr,
	})

	prefix := uuid.NewString()[:8] // enough for millions configurations
	api.LogInfof("limitCountRedis filter uses %s as prefix, config: %v", prefix, &conf.Config)

	conf.limiters = make([]*Limiter, len(conf.Rules))
	quotaPolicy := make([]string, len(conf.Rules))
	for i, rule := range conf.Rules {
		conf.limiters[i] = &Limiter{
			count:      rule.Count,
			timeWindow: rule.TimeWindow.Seconds,
			prefix:     fmt.Sprintf("%s|%d", prefix, i),
		}
		quotaPolicy[i] = fmt.Sprintf("%d;w=%d", rule.Count, rule.TimeWindow.Seconds)

		if rule.Key == "" {
			continue
		}
		script, _ := expr.CompileCel(rule.Key, cel.StringType)
		conf.limiters[i].script = script
	}
	conf.quotaPolicy = strings.Join(quotaPolicy, ", ")

	return nil
}
