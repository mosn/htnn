package plugins

import (
	"net/http"

	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/plugins"
)

const (
	Name = "your_plugin"
)

func init() {
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) ConfigFactory() api.FilterConfigFactory {
	return configFactory
}

func (p *plugin) Config() plugins.PluginConfig {
	return &Config{}
}

func (c *Config) Init(cb api.ConfigCallbackHandler) error {
	return nil
}

func configFactory(c interface{}) api.FilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &filter{
			callbacks: callbacks,
		}
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.ResultAction {
	hdr := http.Header{}
	hdr.Set("content-type", "text/plain")
	return &api.LocalResponse{
		Code:   200,
		Msg:    "Your plugin is run\n",
		Header: hdr,
	}
}
