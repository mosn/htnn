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

package filtermanager

import (
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"mosn.io/htnn/api/pkg/filtermanager/api"
)

type internalErrorFilter struct {
	api.PassThroughFilter

	plugin string
	err    error
}

func (f *internalErrorFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	api.LogErrorf("error in plugin %s: %s", f.plugin, f.err)
	return &api.LocalResponse{
		Code: 500,
	}
}

func NewInternalErrorFactory(plugin string, err error) api.FilterFactory {
	return func(interface{}, api.FilterCallbackHandler) api.Filter {
		return &internalErrorFilter{
			plugin: plugin,
			err:    err,
		}
	}
}

type internalErrorFilterForCAPI struct {
	capi.PassThroughStreamFilter

	callbacks capi.FilterCallbacks
}

func (f *internalErrorFilterForCAPI) DecodeHeaders(headers capi.RequestHeaderMap, endStream bool) capi.StatusType {
	f.callbacks.SendLocalReply(500, "", nil, 0, "")
	return capi.LocalReply
}

func InternalErrorFactoryForCAPI(cfg interface{}, callbacks capi.FilterCallbackHandler) capi.StreamFilter {
	return &internalErrorFilterForCAPI{
		callbacks: callbacks,
	}
}
