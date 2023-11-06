package casbin

import (
	"github.com/casbin/casbin/v2"

	"mosn.io/moe/pkg/file"
	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/request"
)

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
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *config
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) {
	role, _ := header.Get(f.config.Token.Name) // role can be ""
	url := request.GetUrl(header)

	f.config.lock.RLock()
	ok, _ := f.config.enforcer.Enforce(role, url.Path, header.Method())
	f.config.lock.RUnlock()

	if !ok {
		api.LogInfof("reject forbidden user %s", role)
		f.callbacks.SendLocalReply(403, "", nil, 0, "")
		return
	}
}

func (f *filter) OnLog() {
	conf := f.config

	conf.lock.RLock()
	ok := file.IsChanged(conf.modelFile, conf.policyFile)
	conf.lock.RUnlock()
	if ok {
		conf.lock.Lock()
		defer conf.lock.Unlock()

		e, err := casbin.NewEnforcer(conf.Rule.Model, conf.Rule.Policy)
		if err != nil {
			api.LogErrorf("failed to update Enforcer: %v", err)
			return
		}
		conf.enforcer = e

		file.Update(conf.modelFile, conf.policyFile)
	}
}
