// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package casbin

import (
	"github.com/casbin/casbin/v2"

	"mosn.io/htnn/pkg/file"
	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/request"
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

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.ResultAction {
	role, _ := header.Get(f.config.Token.Name) // role can be ""
	url := request.GetUrl(header)

	f.config.lock.RLock()
	ok, _ := f.config.enforcer.Enforce(role, url.Path, header.Method())
	f.config.lock.RUnlock()

	if !ok {
		api.LogInfof("reject forbidden user %s", role)
		return &api.LocalResponse{
			Code: 403,
		}
	}
	return api.Continue
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
