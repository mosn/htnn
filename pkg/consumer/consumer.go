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

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	"mosn.io/htnn/pkg/log"
	"mosn.io/htnn/pkg/plugins"
)

var (
	logger = log.DefaultLogger.WithName("consumer")

	resourceIndex = make(map[string]map[string]*Consumer)
	scopeIndex    map[string]map[string]map[string]*Consumer
)

type Consumer struct {
	Name string

	Auth map[string][]byte

	// fields that set in the data plane
	ResourceVersion string                                  `json:"-"`
	ConsumerConfigs map[string]plugins.PluginConsumerConfig `json:"-"`
}

func NewConsumer(resource *mosniov1.Consumer) *Consumer {
	auth := make(map[string][]byte, len(resource.Spec.Auth))
	for k, v := range resource.Spec.Auth {
		auth[k] = v.Raw
	}
	return &Consumer{
		Name: resource.Name,
		Auth: auth,
	}
}

func (c *Consumer) Marshal() string {
	// Consumer is defined to be marshalled to JSON, so err must be nil
	b, _ := json.Marshal(c)
	return string(b)
}

func (c *Consumer) Unmarshal(s string) error {
	err := json.Unmarshal([]byte(s), c)
	if err != nil {
		return err
	}

	c.ConsumerConfigs = make(map[string]plugins.PluginConsumerConfig, len(c.Auth))
	for name, data := range c.Auth {
		p, ok := plugins.LoadHttpPlugin(name).(plugins.ConsumerPlugin)
		if !ok {
			return fmt.Errorf("plugin %s is not for consumer", name)
		}

		conf := p.ConsumerConfig()
		err = protojson.Unmarshal(data, conf)
		if err != nil {
			return fmt.Errorf("failed to unmarshal consumer config for plugin %s: %w", name, err)
		}

		err := conf.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate consumer config for plugin %s: %w", name, err)
		}

		c.ConsumerConfigs[name] = conf
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
			v := fields["v"].GetStringValue()

			currValue, ok := currIdx[name]
			if !ok || currValue.ResourceVersion != v {
				s := fields["d"].GetStringValue()
				var c Consumer
				err := c.Unmarshal(s)
				if err != nil {
					logger.Error(err, "failed to unmarshal", "consumer", s, "name", name, "namespace", ns)
					continue
				}

				c.ResourceVersion = v
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
				pluginScopeIdx[cfg.Index()] = value
			}
		}
		scopeIndex[ns] = nsScopeIdx
	}
}

// GetConsumer returns the consumer config for the given namespace, plugin name and key.
func GetConsumer(ns, pluginName, key string) *Consumer {
	if nsIdx, ok := scopeIndex[ns]; ok {
		if pluginIdx, ok := nsIdx[pluginName]; ok {
			return pluginIdx[key]
		}
	}
	return nil
}
