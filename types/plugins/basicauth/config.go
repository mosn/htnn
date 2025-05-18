package basicauth

import (
    "errors"
    "mosn.io/htnn/api/pkg/filtermanager/api"
    "mosn.io/htnn/api/pkg/plugins"
)

const (
    Name = "basicAuth"
)

func init() {
    plugins.RegisterPluginType(Name, &Plugin{})
}

type Plugin struct {
    plugins.PluginMethodDefaultImpl
}

func (p *Plugin) Type() plugins.PluginType {
    return plugins.TypeAuthn
}

func (p *Plugin) Order() plugins.PluginOrder {
    return plugins.PluginOrder{
        Position: plugins.OrderPositionAuthn,
    }
}

func (p *Plugin) Config() api.PluginConfig {
    return &Config{}
}

// Validate 验证配置的合法性
func (conf *Config) Validate() error {
    if len(conf.Credentials) == 0 {
        return errors.New("at least one username-password pair must be specified")
    }
    return nil
}