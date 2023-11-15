package plugins

import (
	"encoding/json"
	"sync"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/moe/pkg/filtermanager"
)

var httpPlugins = sync.Map{}

func RegisterHttpPlugin(name string, plugin Plugin) {
	if plugin == nil {
		panic("plugin should not be nil")
	}

	api.LogInfof("register plugin %s", name)
	filtermanager.RegisterHttpFilterConfigFactoryAndParser(name,
		plugin.ConfigFactory(),
		NewPluginConfigParser(plugin))

	httpPlugins.Store(name, plugin)
}

func IterateHttpPlugin(f func(key string, value Plugin) bool) {
	httpPlugins.Range(func(k, v any) bool {
		return f(k.(string), v.(Plugin))
	})
}

type PluginConfigParser struct {
	Plugin
}

func NewPluginConfigParser(parser Plugin) *PluginConfigParser {
	return &PluginConfigParser{
		Plugin: parser,
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

	err = conf.Init(callbacks)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

// PluginMethodDefaultImpl provides reasonable implementation for optional methods
type PluginMethodDefaultImpl struct{}

func (p *PluginMethodDefaultImpl) Handle(c interface{}, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	return c, nil
}

func (p *PluginMethodDefaultImpl) Merge(parent interface{}, child interface{}) interface{} {
	return child
}
