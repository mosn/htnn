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

package sentinel

import (
	sentinelApi "github.com/alibaba/sentinel-golang/api"
	sentinelConf "github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/logging"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/plugins/plugins/sentinel/rules"
	"mosn.io/htnn/types/plugins/sentinel"
)

func init() {
	plugins.RegisterPlugin(sentinel.Name, &plugin{})
}

type plugin struct {
	sentinel.Plugin
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	sentinel.CustomConfig

	params      sentinelApi.EntryOption
	attachments []*sentinel.Source
	m           *res2RuleMap
}

type res2RuleMap struct {
	f  map[string]*sentinel.FlowRule
	hs map[string]*sentinel.HotSpotRule
	cb map[string]*sentinel.CircuitBreakerRule
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	sc := sentinelConf.NewDefaultConfig()

	// Sentinel-golang logs come in two types: metric and record the log.
	// See https://sentinelguard.io/zh-cn/docs/golang/logging.html.
	sc.Sentinel.Log.Dir = conf.GetLogDir()
	if conf.GetLogDir() == "" {
		// When we want the log output to the console, set Dir = "" and Logger = logging.NewConsoleLogger() to
		// output the record log to the console as expected, but the metric log will still be output to the default
		// directory (~/logs/csp), so we should set Metric.FlushIntervalSec == 0 to disable metric log.
		sc.Sentinel.Log.Logger = logging.NewConsoleLogger()
		sc.Sentinel.Log.Metric.FlushIntervalSec = 0
	}

	if err := sentinelApi.InitWithConfig(sc); err != nil {
		return err
	}

	if err := loadRules(conf); err != nil {
		return err
	}

	return nil
}

func loadRules(conf *config) error {
	conf.m = &res2RuleMap{
		f:  make(map[string]*sentinel.FlowRule),
		hs: make(map[string]*sentinel.HotSpotRule),
		cb: make(map[string]*sentinel.CircuitBreakerRule),
	}

	conf.params = sentinelApi.WithArgs()
	conf.attachments = make([]*sentinel.Source, 0)
	hs := conf.GetHotSpot()
	if hs != nil {
		args := make([]interface{}, len(hs.GetParams()))
		for i, p := range hs.GetParams() {
			args[i] = p
		}
		conf.params = sentinelApi.WithArgs(args...)
		conf.attachments = hs.GetAttachments()
	}

	if err := rules.LoadFlowRules(conf.GetFlow(), conf.m.f); err != nil {
		return err
	}

	if err := rules.LoadHotSpotRules(hs, conf.m.hs); err != nil {
		return err
	}

	if err := rules.LoadCircuitBreakerRules(conf.GetCircuitBreaker(), conf.m.cb); err != nil {
		return err
	}

	return nil
}
