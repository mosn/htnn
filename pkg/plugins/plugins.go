package plugins

import (
	"encoding/json"
	"sync"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

var httpPlugins = sync.Map{}

func RegisterHttpPlugin(name string, plugin Plugin) {
	if plugin == nil {
		panic("plugin should not be nil")
	}
	httpPlugins.Store(name, plugin)
}

func IterateHttpPlugin(f func(key string, value Plugin) bool) {
	httpPlugins.Range(func(k, v any) bool {
		return f(k.(string), v.(Plugin))
	})
}

type PluginConfigParser struct {
	ConfigParser
}

func NewPluginConfigParser(parser ConfigParser) *PluginConfigParser {
	return &PluginConfigParser{
		ConfigParser: parser,
	}
}

func (cp *PluginConfigParser) Parse(any interface{}, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	data, err := json.Marshal(any)
	if err != nil {
		return nil, err
	}

	c, err := cp.Validate(data)
	if err != nil {
		return nil, err
	}

	return cp.Handle(c, callbacks)
}

type merger interface {
	Merge(parent interface{}, child interface{}) interface{}
}

func (cp *PluginConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	if merger, ok := cp.ConfigParser.(merger); ok {
		return merger.Merge(parent, child)
	}
	return child
}

// PluginMethodDefaultImpl provides reasonable implementation for optional methods
type PluginMethodDefaultImpl struct{}
