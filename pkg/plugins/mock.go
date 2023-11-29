package plugins

import (
	"mosn.io/moe/pkg/filtermanager/api"
)

type MockPlugin struct {
	PluginMethodDefaultImpl
}

func (m *MockPlugin) ConfigFactory() api.FilterConfigFactory {
	return func(interface{}) api.FilterFactory { return nil }
}

func (m *MockPlugin) Config() PluginConfig {
	return &MockPluginConfig{}
}

func (m *MockPlugin) Merge(parent interface{}, child interface{}) interface{} {
	return child
}

type MockPluginConfig struct {
	Config
}

func (m *MockPluginConfig) Init(cb api.ConfigCallbackHandler) error {
	return nil
}
