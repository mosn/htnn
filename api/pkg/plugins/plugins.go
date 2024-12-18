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

	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"mosn.io/htnn/api/internal/proto"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("plugins")

	pluginTypes                = map[string]Plugin{}
	plugins                    = map[string]Plugin{}
	httpFilterFactoryAndParser = map[string]*FilterFactoryAndParser{}
	metricsRegister            = map[string]func(capi.ConfigCallbacks){}
)

// Here we introduce extra struct to avoid cyclic import between pkg/filtermanager and pkg/plugins
type FilterConfigParser interface {
	Parse(input interface{}) (interface{}, error)
	Merge(parentConfig interface{}, childConfig interface{}) interface{}
	NonBlockingPhases() api.Phase
}

type FilterFactoryAndParser struct {
	ConfigParser FilterConfigParser
	Factory      api.FilterFactory
}

func RegisterHTTPFilterFactoryAndParser(name string, factory api.FilterFactory, parser FilterConfigParser) {
	if factory == nil {
		panic("config factory should not be nil")
	}
	httpFilterFactoryAndParser[name] = &FilterFactoryAndParser{
		parser,
		factory,
	}
}

func LoadHTTPFilterFactoryAndParser(name string) *FilterFactoryAndParser {
	return httpFilterFactoryAndParser[name]
}

const (
	errNilPlugin                  = "plugin should not be nil"
	errUnknownPluginType          = "a plugin should be either Go plugin or Native plugin"
	errInvalidGoPluginOrder       = "invalid plugin order position: Go plugin should not use OrderPositionOuter or OrderPositionInner"
	errInvalidNativePluginOrder   = "invalid plugin order position: Native plugin should use OrderPositionOuter or OrderPositionInner"
	errInvalidConsumerPluginOrder = "invalid plugin order position: Consumer plugin should use OrderPositionAuthn"
)

func RegisterPluginType(name string, plugin Plugin) {
	if _, ok := pluginTypes[name]; !ok {
		// As RegisterPluginType also calls RegisterPluginType, we only log for the first time.
		// Otherwise, we will log twice for the plugins loaded in the data plane.
		logger.Info("register plugin type", "name", name)
	}
	// override plugin is allowed so that we can patch plugin with bugfix if upgrading
	// the whole htnn is not available
	pluginTypes[name] = plugin
}

func LoadPluginType(name string) Plugin {
	return pluginTypes[name]
}

func IteratePluginType(f func(key string, value Plugin) bool) {
	for k, v := range pluginTypes {
		if !f(k, v) {
			return
		}
	}
}

// We separate the plugin type storage and plugin storage, to avoid plugin type overrides the plugin by accident.

func RegisterPlugin(name string, plugin Plugin) {
	if plugin == nil {
		panic(errNilPlugin)
	}

	logger.Info("register plugin", "name", name)

	order := plugin.Order()
	if goPlugin, ok := plugin.(GoPlugin); ok {
		if order.Position == OrderPositionOuter || order.Position == OrderPositionInner {
			panic(errInvalidGoPluginOrder)
		}
		RegisterHTTPFilterFactoryAndParser(name,
			goPlugin.Factory(),
			NewPluginConfigParser(goPlugin))
	} else if _, ok := plugin.(NativePlugin); ok {
		switch order.Position {
		case OrderPositionOuter, OrderPositionInner, OrderPositionListener, OrderPositionNetwork:
		default:
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
	plugins[name] = plugin

	// We don't force developer to divide their plugin into two parts for better DX.
	RegisterPluginType(name, plugin)
}

func LoadPlugin(name string) Plugin {
	return plugins[name]
}

func IteratePlugin(f func(key string, value Plugin) bool) {
	for k, v := range plugins {
		if !f(k, v) {
			return
		}
	}
}

// This method should be called at startup. There will be race if it's called during runtime.
func DisablePlugin(name string) {
	delete(plugins, name)
	delete(pluginTypes, name)
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

func RegisterMetricsCallback(pluginName string, registerMetricFunc func(capi.ConfigCallbacks)) {
	if registerMetricFunc == nil {
		panic("registerMetricFunc should not be nil")
	}
	if pluginName == "" {
		panic("pluginName should not be empty")
	}
	if _, ok := metricsRegister[pluginName]; ok {
		logger.Error(errors.New("metrics for plugin already registered, overriding"), "name", pluginName)
	}
	metricsRegister[pluginName] = registerMetricFunc
	logger.Info("registered metrics for plugin", "name", pluginName)
}

func LoadMetricsCallback(pluginName string) func(capi.ConfigCallbacks) {
	return metricsRegister[pluginName]
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

func (p *PluginMethodDefaultImpl) NonBlockingPhases() api.Phase {
	return 0
}

func ComparePluginOrder(a, b string) bool {
	return ComparePluginOrderInt(a, b) < 0
}

func ComparePluginOrderInt(a, b string) int {
	pa := pluginTypes[a]
	pb := pluginTypes[b]
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
