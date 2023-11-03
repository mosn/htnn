package filtermanager

import "mosn.io/moe/pkg/filtermanager/api"

func PassThroughFactory(interface{}) api.FilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &api.PassThroughFilter{}
	}
}
