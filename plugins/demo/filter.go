package demo

import (
	"fmt"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

func configFactory(c interface{}) api.StreamFilterFactory {
	conf := c.(*Config)
	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &filter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}

type filter struct {
	api.PassThroughStreamFilter

	callbacks api.FilterCallbackHandler
	config    *Config
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	header.Set(f.config.HostName, f.hello())
	return api.Continue
}

func (f *filter) hello() string {
	name := f.callbacks.StreamInfo().FilterState().GetString("guest_name")
	api.LogInfo("hello")
	return fmt.Sprintf("hello, %s", name)
}
