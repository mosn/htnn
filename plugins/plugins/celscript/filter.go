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

package celscript

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
	config := f.config
	if config.allowIfScript != nil {
		res, err := config.allowIfScript.EvalWithRequest(f.callbacks, headers)
		if err != nil {
			api.LogErrorf("failed to eval script with request: %v", err)
			return &api.LocalResponse{Code: 503}
		}

		allowed := res.(bool)
		if !allowed {
			api.LogInfo("celScript rejects request")
			return &api.LocalResponse{Code: 403}
		}
	}
	return api.Continue
}
