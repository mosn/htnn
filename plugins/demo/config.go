package demo

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"mosn.io/moe/pkg/plugins"
)

const (
	Name = "demo"
)

func init() {
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct{}

func (p *plugin) ConfigFactory() api.StreamFilterConfigFactory {
	return configFactory
}

func (p *plugin) ConfigParser() api.StreamFilterConfigParser {
	return plugins.NewPluginConfigParser(&parser{})
}

type config struct {
}

type parser struct {
}

func (p *parser) Validate(data []byte) (interface{}, error) {
	conf := &config{}
	return conf, nil
}

func (p *parser) Handle(c interface{}, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	conf := c.(config)
	return conf, nil
}

func (p *parser) Merge(parent interface{}, child interface{}) interface{} {
	return child
}
