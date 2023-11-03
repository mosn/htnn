package demo

import (
	"fmt"

	"mosn.io/moe/pkg/filtermanager/api"
)

func configFactory(c interface{}) api.FilterFactory {
	conf := c.(*Config)
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &filter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *Config
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) {
	header.Set(f.config.HostName, f.hello())
}

func (f *filter) hello() string {
	name := f.callbacks.StreamInfo().FilterState().GetString("guest_name")
	api.LogInfo("hello")
	return fmt.Sprintf("hello, %s", name)
}
