package plugins

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"mosn.io/moe/pkg/filtermanager/api"
)

type Plugin interface {
	Config() PluginConfig
	ConfigFactory() api.FilterConfigFactory
	Merge(parent interface{}, child interface{}) interface{}
}

type PluginConfig interface {
	ProtoReflect() protoreflect.Message
	Validate() error
	Init(cb api.ConfigCallbackHandler) error
}
