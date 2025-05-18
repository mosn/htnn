package acl

import (
	"errors"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

const (
	Name = "acl"
)

func init() {
	plugins.RegisterPluginType(Name, &Plugin{})
}

type Plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *Plugin) Type() plugins.PluginType {
	return plugins.TypeAuthz
}

func (p *Plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAuthz,
	}
}

func (p *Plugin) Config() api.PluginConfig {
	return &Config{} // 使用重命名后的结构体
}


// Validate 验证配置的合法性
func (conf *Config) Validate() error {
	if len(conf.AllowList) == 0 && len(conf.DenyList) == 0 {
		return errors.New("at least one of allow_list or deny_list must be specified")
	}
	return nil
}
