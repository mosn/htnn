package casbin

import (
	"github.com/casbin/casbin/v2"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/plugins"
)

const (
	Name = "casbin"
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

func (p *plugin) ConfigParser() api.FilterConfigParser {
	return plugins.NewPluginConfigParser(&parser{})
}

type parser struct {
}

type config struct {
	*Config
	enforcer *casbin.Enforcer
}

func (p *parser) Validate(data []byte) (interface{}, error) {
	conf := &Config{}
	err := protojson.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}
	return conf, nil
}

func (p *parser) Handle(c interface{}, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	cfg := c.(*Config)
	conf := &config{
		Config: cfg,
	}

	// TODO: record the mtime of Model/Policy files and check if it's up to date in OnLog
	e, err := casbin.NewEnforcer(conf.Rule.Model, conf.Rule.Policy)
	if err != nil {
		return nil, err
	}
	conf.enforcer = e
	return conf, nil
}

func (p *parser) Merge(parent interface{}, child interface{}) interface{} {
	return child
}
