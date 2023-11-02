package plugins

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

type Plugin interface {
	ConfigFactory() api.StreamFilterConfigFactory
	ConfigParser() api.StreamFilterConfigParser
}

// We split the interface into Validate & Handle, so that
// 1. the Validate can be used separately elsewhere
// 2. we can put common logic before Validate, like exacting configuration from xDS
type ConfigParser interface {
	Validate(encodedJSON []byte) (validated interface{}, err error)
	Handle(validated interface{}, cb api.ConfigCallbackHandler) (configInDP interface{}, err error)
}
