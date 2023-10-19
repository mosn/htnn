package demo

import (
	"bytes"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/golang/protobuf/jsonpb"

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

type parser struct {
}

func (p *parser) Validate(data []byte) (interface{}, error) {
	conf := &Config{}
	marshaler := &jsonpb.Unmarshaler{}
	err := marshaler.Unmarshal(bytes.NewReader(data), conf)
	if err != nil {
		return nil, err
	}

	if err = conf.Validate(); err != nil {
		return nil, err
	}
	return conf, nil
}

func (p *parser) Handle(c interface{}, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	conf := c.(*Config)
	return conf, nil
}

func (p *parser) Merge(parent interface{}, child interface{}) interface{} {
	return child
}
