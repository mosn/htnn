package plugins

import (
	"mosn.io/moe/pkg/filtermanager/api"
)

type Plugin interface {
	ConfigFactory() api.FilterConfigFactory
	ConfigParser() api.FilterConfigParser
}

// We split the Parse method into Validate & Handle, so that
// 1. the Validate can be used separately elsewhere
// 2. we can put common logic before Validate, like exacting configuration from xDS
type ConfigParser interface {
	Validate(encodedJSON []byte) (validated interface{}, err error)
	Handle(validated interface{}, cb api.ConfigCallbackHandler) (configInDP interface{}, err error)
	Merge(parentConfig interface{}, childConfig interface{}) interface{}
}
