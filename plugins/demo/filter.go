package demo

import (
	"fmt"

	"mosn.io/moe/pkg/filtermanager/api"
)

// configFactory returns a factory that produces per-request Filter.
// You can use it to bind the configuration and do per-request initialization.
func configFactory(c interface{}) api.FilterFactory {
	conf := c.(*config)
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &filter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}

type filter struct {
	// PassThroughFilter is the base class of filter which provides the default implementation
	// to Filter methods - do nothing.
	api.PassThroughFilter

	// callbacks provides the API we can use to implement filter's feature
	callbacks api.FilterCallbackHandler
	config    *config
}

// The doc of each API can be found in package pkg/filtermanager/api

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.ResultAction {
	header.Set(f.config.HostName, f.hello())
	return api.Continue
}

func (f *filter) hello() string {
	name := f.callbacks.StreamInfo().FilterState().GetString("guest_name")
	api.LogInfo("hello")
	return fmt.Sprintf("hello, %s", name)
}
