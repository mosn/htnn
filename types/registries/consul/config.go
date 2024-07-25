package consul

import "mosn.io/htnn/types/pkg/registry"

const (
	Name = "consul"
)

func init() {
	registry.AddRegistryType(Name, &RegistryType{})
}

type RegistryType struct {
}

func (reg *RegistryType) Config() registry.RegistryConfig {
	return &Config{}
}
