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

package plugins

import (
	"encoding/json"
	"sync"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/moe/pkg/filtermanager"
	"mosn.io/moe/pkg/log"
)

var (
	logger      = log.DefaultLogger.WithName("plugins")
	httpPlugins = sync.Map{}
)

func RegisterHttpPlugin(name string, plugin Plugin) {
	if plugin == nil {
		panic("plugin should not be nil")
	}

	logger.Info("register plugin", "name", name)
	if goPlugin, ok := plugin.(GoPlugin); ok {
		filtermanager.RegisterHttpFilterConfigFactoryAndParser(name,
			goPlugin.ConfigFactory(),
			NewPluginConfigParser(goPlugin))
	}

	httpPlugins.Store(name, plugin)
}

func LoadHttpPlugin(name string) Plugin {
	res, ok := httpPlugins.Load(name)
	if !ok {
		return nil
	}
	return res.(Plugin)
}

func IterateHttpPlugin(f func(key string, value Plugin) bool) {
	httpPlugins.Range(func(k, v any) bool {
		return f(k.(string), v.(Plugin))
	})
}

type PluginConfigParser struct {
	GoPlugin
}

func NewPluginConfigParser(parser GoPlugin) *PluginConfigParser {
	return &PluginConfigParser{
		GoPlugin: parser,
	}
}

func (cp *PluginConfigParser) Parse(any interface{}, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	conf := cp.Config()
	if any != nil {
		data, err := json.Marshal(any)
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(data, conf)
		if err != nil {
			return nil, err
		}
	}

	err := conf.Validate()
	if err != nil {
		return nil, err
	}

	if initer, ok := conf.(Initer); ok {
		err = initer.Init(callbacks)
		if err != nil {
			return nil, err
		}
	}
	return conf, nil
}

// PluginMethodDefaultImpl provides reasonable implementation for optional methods
type PluginMethodDefaultImpl struct{}

func (p *PluginMethodDefaultImpl) Type() PluginType {
	return TypeGeneral
}

func (p *PluginMethodDefaultImpl) Order() PluginOrder {
	return PluginOrder{
		Position:  OrderPositionUnspecified,
		Operation: OrderOperationNop,
	}
}

func (p *PluginMethodDefaultImpl) Merge(parent interface{}, child interface{}) interface{} {
	return child
}

var (
	nameToOrder     = map[string]PluginOrder{}
	nameToOrderInit = sync.Once{}
)

// The caller should ganrantee the a, b are valid plugin name.
func ComparePluginOrder(a, b string) bool {
	nameToOrderInit.Do(func() {
		IterateHttpPlugin(func(key string, value Plugin) bool {
			nameToOrder[key] = value.Order()
			return true
		})
	})

	aOrder := nameToOrder[a]
	bOrder := nameToOrder[b]
	if aOrder.Position != bOrder.Position {
		return aOrder.Position < bOrder.Position
	}
	if aOrder.Operation != bOrder.Operation {
		return aOrder.Operation < bOrder.Operation
	}
	return a < b
}
