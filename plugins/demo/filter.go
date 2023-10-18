package demo

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

func configFactory(c interface{}) api.StreamFilterFactory {
	conf, ok := c.(*config)
	if !ok {
		panic("unexpected config type")
	}
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
	config    *config
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	header.Set(f.callbacks.StreamInfo().FilterState().GetString("header_name"),
		f.hello())
	return api.Continue
}

func (f *filter) hello() string {
	api.LogInfo("hello")
	return "hello"
}
