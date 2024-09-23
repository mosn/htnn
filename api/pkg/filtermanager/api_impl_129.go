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

//go:build envoy1.29

package filtermanager

import (
	"runtime/debug"

	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"mosn.io/htnn/api/pkg/filtermanager/api"
)

func (s *filterManagerStreamInfo) WorkerID() uint32 {
	api.LogErrorf("WorkerID is not implemented: %s", debug.Stack())
	return 0
}

func (cb *filterManagerCallbackHandler) ClearRouteCache() {
	api.LogErrorf("ClearRouteCache is not implemented: %s", debug.Stack())
}

func (cb *filterManagerCallbackHandler) DecoderFilterCallbacks() api.DecoderFilterCallbacks {
	return cb.FilterCallbackHandler
}

func (cb *filterManagerCallbackHandler) EncoderFilterCallbacks() api.EncoderFilterCallbacks {
	return cb.FilterCallbackHandler
}

func (cb *filterManagerCallbackHandler) Continue(st capi.StatusType, _ bool) {
	cb.FilterCallbackHandler.Continue(st)
}
