package casbin

import (
	sync "sync"

	"github.com/casbin/casbin/v2"

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

func (p *plugin) Config() plugins.PluginConfig {
	return &config{}
}

type config struct {
	Config

	lock *sync.RWMutex

	enforcer   *casbin.Enforcer
	modelFile  *file.File
	policyFile *file.File
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	conf.lock = &sync.RWMutex{}

	f, err := file.Stat(conf.Rule.Model)
	if err != nil {
		return err
	}
	conf.modelFile = f

	f, err = file.Stat(conf.Rule.Policy)
	if err != nil {
		return err
	}
	conf.policyFile = f

	e, err := casbin.NewEnforcer(conf.Rule.Model, conf.Rule.Policy)
	if err != nil {
		return err
	}
	conf.enforcer = e
	return nil
}
