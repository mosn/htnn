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

	"google.golang.org/protobuf/types/known/structpb"

	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/filtermanager/model"
	"mosn.io/htnn/pkg/log"
	"mosn.io/htnn/pkg/plugins"
	"mosn.io/htnn/pkg/proto"
)

var (
	logger = log.DefaultLogger.WithName("consumer")

	resourceIndex = make(map[string]map[string]*Consumer)
	scopeIndex    map[string]map[string]map[string]*Consumer
)

type Consumer struct {
	Auth    map[string]string              `json:"auth"`
	Filters map[string]*model.FilterConfig `json:"filters,omitempty"`

	// fields that set in the data plane
	namespace       string
	name            string
	generation      int
	ConsumerConfigs map[string]api.PluginConsumerConfig  `json:"-"`
	FilterConfigs   map[string]*model.ParsedFilterConfig `json:"-"`

	// fields that generated from the configuration
	CanSkipMethod  map[string]bool
	FilterNames    []string
	FilterWrappers []*model.FilterWrapper
}

func (c *Consumer) Marshal() string {
	// Consumer is defined to be marshalled to JSON, so err must be nil
	b, _ := json.Marshal(c)
	return string(b)
}

func (c *Consumer) Unmarshal(s string) error {
	return json.Unmarshal([]byte(s), c)
}

func (c *Consumer) InitConfigs() error {
	logger.Info("init configs for consumer", "name", c.name, "namespace", c.namespace)

	c.ConsumerConfigs = make(map[string]api.PluginConsumerConfig, len(c.Auth))
	for name, data := range c.Auth {
		p, ok := plugins.LoadHttpPlugin(name).(plugins.ConsumerPlugin)
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

	c.FilterConfigs = make(map[string]*model.ParsedFilterConfig, len(c.Filters))
	for name, data := range c.Filters {
		p := plugins.LoadHttpFilterFactoryAndParser(name)
		if p == nil {
			return fmt.Errorf("plugin %s not found", name)
		}

		conf, err := p.ConfigParser.Parse(data.Config, nil)
		if err != nil {
			return fmt.Errorf("%w during parsing plugin %s in consumer", err, name)
		}

		c.FilterConfigs[name] = &model.ParsedFilterConfig{
			Name:         name,
			ParsedConfig: conf,
			Factory:      p.Factory,
		}
	}

	return nil
}

func updateConsumers(value *structpb.Struct) {
	// build the idx for syncing with the control plane
	for ns, nsValue := range value.GetFields() {
		currIdx := resourceIndex[ns]
		if currIdx == nil {
			currIdx = make(map[string]*Consumer)
		}

		newIdx := map[string]*Consumer{}
		for name, value := range nsValue.GetStructValue().GetFields() {
			fields := value.GetStructValue().GetFields()
			v := int(fields["v"].GetNumberValue())

			currValue, ok := currIdx[name]
			if !ok || currValue.generation != v {
				s := fields["d"].GetStringValue()
				var c Consumer
				err := c.Unmarshal(s)
				if err != nil {
					logger.Error(err, "failed to unmarshal", "consumer", s, "name", name, "namespace", ns)
					continue
				}

				c.name = name
				c.namespace = ns

				err = c.InitConfigs()
				if err != nil {
					logger.Error(err, "failed to init", "consumer", s, "name", name, "namespace", ns)
					continue
				}

				c.generation = v
				newIdx[name] = &c
			} else {
				newIdx[name] = currValue
			}
		}
		resourceIndex[ns] = newIdx
	}

	// build the idx for matching in the data plane
	scopeIndex = make(map[string]map[string]map[string]*Consumer)
	for ns, nsValue := range resourceIndex {
		nsScopeIdx := make(map[string]map[string]*Consumer)
		for _, value := range nsValue {
			for pluginName, cfg := range value.ConsumerConfigs {
				pluginScopeIdx := nsScopeIdx[pluginName]
				if pluginScopeIdx == nil {
					pluginScopeIdx = make(map[string]*Consumer)
					nsScopeIdx[pluginName] = pluginScopeIdx
				}

				idx := cfg.Index()
				if pluginScopeIdx[idx] != nil {
					// TODO: find an effective way to detect collision in the control plane
					err := fmt.Errorf("duplicate index %s", value.name)
					logger.Error(err, fmt.Sprintf("ignore consumer %s for plugin %s", pluginName, idx),
						"namespace", ns, "existing consumer", pluginScopeIdx[idx].name)
					continue
				}
				pluginScopeIdx[idx] = value
			}
		}
		scopeIndex[ns] = nsScopeIdx
	}
}

// LookupConsumer returns the consumer config for the given namespace, plugin name and key.
func LookupConsumer(ns, pluginName, key string) (api.Consumer, bool) {
	if nsIdx, ok := scopeIndex[ns]; ok {
		if pluginIdx, ok := nsIdx[pluginName]; ok {
			// return extra bool to indicate whether the key exists so user doesn't need to
			// distinguish nil interface.
			// An interface in Go is nil only when both its type and value are nil.
			c, ok := pluginIdx[key]
			return c, ok
		}
	}
	return nil, false
}

// Implement pkg.filtermanager.api.Consumer
func (c *Consumer) Name() string {
	return c.name
}

func (c *Consumer) PluginConfig(name string) api.PluginConsumerConfig {
	return c.ConsumerConfigs[name]
}
