package demo

import (
	"net/http"

	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/plugins"
)

const (
	Name = "demo"
)

func init() {
	// Register the plugin with the given name
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct {
	// PluginMethodDefaultImpl is the base class of plugin which provides the default implementation
	// to Plugin methods.
	plugins.PluginMethodDefaultImpl
}

// Each plugin need to implement the two methods below

// ConfigFactory returns api.ConfigFactory's implementation used during request processing
func (p *plugin) ConfigFactory() api.FilterConfigFactory {
	return configFactory
}

// ConfigFactory returns plugins.PluginConfig's implementation used during configuration processing
func (p *plugin) Config() plugins.PluginConfig {
	return &config{}
}

type config struct {
	// Config is generated from `config.proto`.
	// The returned implementation should embed the Config.
	Config

	client *http.Client
}

// Init allows the initialization of non-generated fields during configuration processing.
// This function is run in Envoy's main thread, so it doesn't block the request processing.
func (c *config) Init(cb api.ConfigCallbackHandler) error {
	c.client = http.DefaultClient
	return nil
}
