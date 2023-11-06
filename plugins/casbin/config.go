package casbin

import (
	sync "sync"

	"github.com/casbin/casbin/v2"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/moe/pkg/file"
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

	lock *sync.RWMutex

	enforcer   *casbin.Enforcer
	modelFile  *file.File
	policyFile *file.File
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
		lock:   &sync.RWMutex{},
	}

	f, err := file.Stat(conf.Rule.Model)
	if err != nil {
		return nil, err
	}
	conf.modelFile = f

	f, err = file.Stat(conf.Rule.Policy)
	if err != nil {
		return nil, err
	}
	conf.policyFile = f

	e, err := casbin.NewEnforcer(conf.Rule.Model, conf.Rule.Policy)
	if err != nil {
		return nil, err
	}
	conf.enforcer = e
	return conf, nil
}
