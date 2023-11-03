package plugins

import "mosn.io/moe/pkg/filtermanager/api"

type MockPlugin struct {
}

func (m *MockPlugin) ConfigFactory() api.FilterConfigFactory {
	return nil
}

func (m *MockPlugin) ConfigParser() api.FilterConfigParser {
	return nil
}

type MockConfigParser struct {
}

func (m *MockConfigParser) Validate(encodedJSON []byte) (validated interface{}, err error) {
	return
}

func (m *MockConfigParser) Handle(validated interface{}, cb api.ConfigCallbackHandler) (configInDP interface{}, err error) {
	return
}

func (m *MockConfigParser) Merge(parentConfig interface{}, childConfig interface{}) interface{} {
	return childConfig
}
