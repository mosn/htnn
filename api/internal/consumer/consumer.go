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
	"encoding/json"
	"fmt"
	sync "sync"

	"mosn.io/htnn/api/internal/proto"
	csModel "mosn.io/htnn/api/pkg/consumer/model"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	fmModel "mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/pkg/log"
	"mosn.io/htnn/api/pkg/plugins"
)

var (
	logger = log.DefaultLogger.WithName("consumer")
)

// Here we put Consumer in the internal/. So that we can access the Consumer's internal definition in other package
// of this repo, while hiding it from the plugin developers.

type Consumer struct {
	csModel.Consumer

	// fields that set in the data plane
	namespace       string
	name            string
	generation      int
	ConsumerConfigs map[string]api.PluginConsumerConfig
	FilterConfigs   map[string]*fmModel.ParsedFilterConfig

	// fields that generated from the configuration
	FilterNames       []string
	InitOnce          sync.Once
	CanSkipMethod     map[string]bool
	CanSkipMethodOnce sync.Once
	CanSyncRunMethod  map[string]bool
	// CanSyncRunMethod share the same sync.Once with CanSkipMethodOnce
}

func (c *Consumer) Unmarshal(s string) error {
	return json.Unmarshal([]byte(s), c)
}

func (c *Consumer) InitConfigs() error {
	logger.Info("init configs for consumer", "name", c.name, "namespace", c.namespace)

	c.ConsumerConfigs = make(map[string]api.PluginConsumerConfig, len(c.Auth))
	for name, data := range c.Auth {
		p, ok := plugins.LoadPlugin(name).(plugins.ConsumerPlugin)
		if !ok {
			return fmt.Errorf("plugin %s is not for consumer", name)
		}

		conf := p.ConsumerConfig()
		err := proto.UnmarshalJSON([]byte(data), conf)
		if err != nil {
			return fmt.Errorf("failed to unmarshal consumer config for plugin %s: %w", name, err)
		}

		err = conf.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate consumer config for plugin %s: %w", name, err)
		}

		c.ConsumerConfigs[name] = conf
	}

	c.FilterConfigs = make(map[string]*fmModel.ParsedFilterConfig, len(c.Filters))
	for name, data := range c.Filters {
		p := plugins.LoadHTTPFilterFactoryAndParser(name)
		if p == nil {
			return fmt.Errorf("plugin %s not found", name)
		}

		conf, err := p.ConfigParser.Parse(data.Config)
		if err != nil {
			return fmt.Errorf("%w during parsing plugin %s in consumer", err, name)
		}

		c.FilterConfigs[name] = &fmModel.ParsedFilterConfig{
			Name:          name,
			ParsedConfig:  conf,
			Factory:       p.Factory,
			SyncRunPhases: p.ConfigParser.NonBlockingPhases(),
		}
	}

	return nil
}

// Implement pkg.filtermanager.api.Consumer
func (c *Consumer) Name() string {
	return c.name
}

func (c *Consumer) PluginConfig(name string) api.PluginConsumerConfig {
	return c.ConsumerConfigs[name]
}
