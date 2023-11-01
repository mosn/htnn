package casbin

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"mosn.io/moe/pkg/request"
)

func configFactory(c interface{}) api.StreamFilterFactory {
	conf := c.(*config)
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
	role, _ := header.Get(f.config.Token.Name) // role can be ""
	url := request.GetUrl(header)
	if ok, _ := f.config.enforcer.Enforce(role, url.Path, header.Method()); !ok {
		api.LogInfof("reject forbidden user %s", role)
		f.callbacks.SendLocalReply(403, "", nil, 0, "")
		return api.LocalReply
	}

	return api.Continue
}
