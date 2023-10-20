package plugins

import (
	"sync"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"
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

func (cp *PluginConfigParser) Parse(any *anypb.Any, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	configStruct := &xds.TypedStruct{}
	var data []byte
	var err error
	if any.GetTypeUrl() == "" {
		// no config
		data = []byte(`{}`)
	} else {
		if err := any.UnmarshalTo(configStruct); err != nil {
			return nil, err
		}

		v := configStruct.Value
		data, err = v.MarshalJSON()
		if err != nil {
			return nil, err
		}
	}

	c, err := cp.Validate(data)
	if err != nil {
		return nil, err
	}

	return cp.Handle(c, callbacks)
}
