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
	"cmp"
	"encoding/json"

	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/log"
	"mosn.io/htnn/pkg/proto"
)

var (
	logger = log.DefaultLogger.WithName("plugins")

	httpPlugins                = map[string]Plugin{}
	httpFilterFactoryAndParser = map[string]*FilterFactoryAndParser{}
)

// Here we introduce extra struct to avoid cyclic import between pkg/filtermanager and pkg/plugins
type FilterConfigParser interface {
	Parse(input interface{}, callbacks api.ConfigCallbackHandler) (interface{}, error)
	Merge(parentConfig interface{}, childConfig interface{}) interface{}
}

type FilterFactoryAndParser struct {
	ConfigParser FilterConfigParser
	Factory      api.FilterFactory
}

func RegisterHttpFilterFactoryAndParser(name string, factory api.FilterFactory, parser FilterConfigParser) {
	if factory == nil {
		panic("config factory should not be nil")
	}
	httpFilterFactoryAndParser[name] = &FilterFactoryAndParser{
		parser,
		factory,
	}
}

func LoadHttpFilterFactoryAndParser(name string) *FilterFactoryAndParser {
	return httpFilterFactoryAndParser[name]
}

func RegisterHttpPlugin(name string, plugin Plugin) {
	if plugin == nil {
		panic("plugin should not be nil")
	}

	logger.Info("register plugin", "name", name)

	if goPlugin, ok := plugin.(GoPlugin); ok {
		order := plugin.Order()
		if order.Position == OrderPositionOuter || order.Position == OrderPositionInner {
			panic("invalid plugin order position: Go plugin should not use OrderPositionOuter or OrderPositionInner")
		}
		RegisterHttpFilterFactoryAndParser(name,
			goPlugin.Factory(),
			NewPluginConfigParser(goPlugin))
	}
	if _, ok := plugin.(NativePlugin); ok {
		order := plugin.Order()
		if order.Position != OrderPositionOuter && order.Position != OrderPositionInner {
			panic("invalid plugin order position: Native plugin should use OrderPositionOuter or OrderPositionInner")
		}
	}
	if _, ok := plugin.(ConsumerPlugin); ok {
		order := plugin.Order()
		if order.Position != OrderPositionAuthn {
			panic("invalid plugin order position: Consumer plugin should use OrderPositionAuthn")
		}
	}

	// override plugin is allowed so that we can patch plugin with bugfix if upgrading
	// the whole htnn is not available
	httpPlugins[name] = plugin
}

func LoadHttpPlugin(name string) Plugin {
	return httpPlugins[name]
}

func IterateHttpPlugin(f func(key string, value Plugin) bool) {
	for k, v := range httpPlugins {
		if !f(k, v) {
			return
		}
	}
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

		err = proto.UnmarshalJSON(data, conf)
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

func ComparePluginOrder(a, b string) bool {
	return ComparePluginOrderInt(a, b) < 0
}

func ComparePluginOrderInt(a, b string) int {
	pa := httpPlugins[a]
	pb := httpPlugins[b]
	if pa == nil || pb == nil {
		// The caller should guarantee the a, b are valid plugin name, so this case only happens
		// in test.
		return cmp.Compare(a, b)
	}

	aOrder := pa.Order()
	bOrder := pb.Order()

	if aOrder.Position != bOrder.Position {
		return int(aOrder.Position - bOrder.Position)
	}
	if aOrder.Operation != bOrder.Operation {
		return int(aOrder.Operation - bOrder.Operation)
	}
	return cmp.Compare(a, b)
}
