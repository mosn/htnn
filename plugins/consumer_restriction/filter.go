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

package consumer_restriction

import "mosn.io/htnn/pkg/filtermanager/api"

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

func (f *filter) reject(msg string) api.ResultAction {
	return &api.LocalResponse{Code: 403, Msg: msg}
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	consumer := f.callbacks.GetConsumer()
	if consumer == nil {
		api.LogInfo("consumerRestriction: consumer not found")
		return f.reject("consumer not found")
	}

	_, ok := f.config.consumers[consumer.Name()]
	if ok != f.config.allow {
		api.LogInfof("consumerRestriction: consumer %s not allowed", consumer.Name())
		// don't leak consumer name to the caller
		return f.reject("consumer not allowed")
	}

	return api.Continue
}
