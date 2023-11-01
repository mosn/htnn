package opa

import (
	"net/http"
	"time"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/moe/pkg/plugins"
)

const (
	Name = "opa"
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
	*Config

	client *http.Client
}

func (p *parser) Validate(data []byte) (interface{}, error) {
	conf := &Config{}
	err := protojson.Unmarshal(data, conf)
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

	client := &http.Client{Timeout: 10 * time.Second}
	return &config{
		Config: conf,
		client: client,
	}, nil
}

func (p *parser) Merge(parent interface{}, child interface{}) interface{} {
	return child
}
