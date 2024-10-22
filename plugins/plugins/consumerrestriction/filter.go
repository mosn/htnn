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

package consumerrestriction

import (
	"mosn.io/htnn/api/pkg/filtermanager/api"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &filter{
		callbacks: callbacks,
		config:    c.(*config),
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *config
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	consumer := f.callbacks.GetConsumer()
	if consumer == nil {
		api.LogInfo("consumerRestriction: consumer not found")
		return &api.LocalResponse{Code: 401, Msg: "consumer not found"}
	}

	if f.config.GetDenyIfNoConsumer() {
		return api.Continue
	}

	consumerName := consumer.Name()
	rule, ok := f.config.consumers[consumerName]
	methodMatched := ok && (len(rule.Methods) == 0 || rule.Methods[headers.Method()])

	if methodMatched != f.config.allow {
		api.LogInfof("consumerRestriction: consumer %s not allowed", consumerName)
		return &api.LocalResponse{Code: 403, Msg: "consumer not allowed"}
	}

	return api.Continue
}
