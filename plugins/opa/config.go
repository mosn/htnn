package opa

import (
	"net/http"
	"time"

	"mosn.io/moe/pkg/filtermanager/api"
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

func (p *plugin) ConfigFactory() api.FilterConfigFactory {
	return configFactory
}

func (p *plugin) Config() plugins.PluginConfig {
	return &config{}
}

type config struct {
	Config

	client *http.Client
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	conf.client = &http.Client{Timeout: 10 * time.Second}
	return nil
}
