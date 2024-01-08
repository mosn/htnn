package hmac_auth

import (
	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/plugins"
)

const (
	Name = "hmac_auth"
)

func init() {
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) Type() plugins.PluginType {
	return plugins.TypeAuthn
}

func (p *plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAuthn,
	}
}

func (p *plugin) ConfigFactory() api.FilterConfigFactory {
	return configFactory
}

func (p *plugin) Config() plugins.PluginConfig {
	return &Config{}
}

func (p *plugin) ConsumerConfig() plugins.PluginConsumerConfig {
	return &ConsumerConfig{}
}

func (conf *ConsumerConfig) Index() string {
	return conf.AccessKey
}
