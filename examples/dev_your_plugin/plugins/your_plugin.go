package plugins

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"

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

func (p *plugin) ConfigFactory() api.StreamFilterConfigFactory {
	return configFactory
}

func (p *plugin) ConfigParser() api.StreamFilterConfigParser {
	return plugins.NewPluginConfigParser(&parser{})
}

type parser struct {
}

type config struct {
}

func (p *parser) Validate(data []byte) (interface{}, error) {
	return &config{}, nil
}

func (p *parser) Handle(c interface{}, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	return c, nil
}

func (p *parser) Merge(parent interface{}, child interface{}) interface{} {
	return child
}

func configFactory(c interface{}) api.StreamFilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &filter{
			callbacks: callbacks,
		}
	}
}

type filter struct {
	api.PassThroughStreamFilter

	callbacks api.FilterCallbackHandler
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	f.callbacks.SendLocalReply(200, "Your plugin is run\n", nil, 0, "")
	return api.LocalReply
}
