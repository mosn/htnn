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
	"errors"
	"runtime/debug"

	"mosn.io/htnn/api/internal/proto"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("plugins")

	httpPluginTypes            = map[string]Plugin{}
	httpPlugins                = map[string]Plugin{}
	httpFilterFactoryAndParser = map[string]*FilterFactoryAndParser{}
)

// Here we introduce extra struct to avoid cyclic import between pkg/filtermanager and pkg/plugins
type FilterConfigParser interface {
	Parse(input interface{}) (interface{}, error)
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

const (
	errNilPlugin                  = "plugin should not be nil"
	errUnknownPluginType          = "a plugin should be either Go plugin or Native plugin"
	errInvalidGoPluginOrder       = "invalid plugin order position: Go plugin should not use OrderPositionOuter or OrderPositionInner"
	errInvalidNativePluginOrder   = "invalid plugin order position: Native plugin should use OrderPositionOuter or OrderPositionInner"
	errInvalidConsumerPluginOrder = "invalid plugin order position: Consumer plugin should use OrderPositionAuthn"
	errAuthnPluginOrder           = "Authn plugin should run in the DecodeHeaders phase"
	errDecodeRequestUnsatified    = "DecodeRequest is run only after DecodeHeaders returns WaitAllData. So DecodeHeaders should be defined in this plugin."
	errEncodeResponseUnsatified   = "EncodeResponse is run only after EncodeHeaders returns WaitAllData. So EncodeHeaders should be defined in this plugin."
)

func RegisterHttpPluginType(name string, plugin Plugin) {
	// override plugin is allowed so that we can patch plugin with bugfix if upgrading
	// the whole htnn is not available
	httpPluginTypes[name] = plugin
}

func LoadHttpPluginType(name string) Plugin {
	return httpPluginTypes[name]
}

// We separate the plugin type storage and plugin storage, to avoid plugin type overrides the plugin by accident.

func RegisterHttpPlugin(name string, plugin Plugin) {
	if plugin == nil {
		panic(errNilPlugin)
	}

	logger.Info("register plugin", "name", name)

	order := plugin.Order()
	if goPlugin, ok := plugin.(GoPlugin); ok {
		if order.Position == OrderPositionOuter || order.Position == OrderPositionInner {
			panic(errInvalidGoPluginOrder)
		}
		RegisterHttpFilterFactoryAndParser(name,
			goPlugin.Factory(),
			NewPluginConfigParser(goPlugin))
	} else if _, ok := plugin.(NativePlugin); ok {
		if order.Position != OrderPositionOuter && order.Position != OrderPositionInner {
			panic(errInvalidNativePluginOrder)
		}
	} else {
		panic(errUnknownPluginType)
	}

	if _, ok := plugin.(ConsumerPlugin); ok {
		if order.Position != OrderPositionAuthn {
			panic(errInvalidConsumerPluginOrder)
		}
	}

	// override plugin is allowed so that we can patch plugin with bugfix if upgrading
	// the whole htnn is not available
	httpPlugins[name] = plugin

	// We don't force developer to divide their plugin into two parts for better DX.
	httpPluginTypes[name] = plugin
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

func (cp *PluginConfigParser) Parse(any interface{}) (res interface{}, err error) {
	defer func() {
		if p := recover(); p != nil {
			api.LogErrorf("panic: %v\n%s", p, debug.Stack())
			err = errors.New("plugin config parser panic")
		}
	}()

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

	err = conf.Validate()
	if err != nil {
		return nil, err
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
