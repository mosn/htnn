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

//go:build !envoy1.29 && !envoydev

package filtermanager

import (
	"runtime/debug"

	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"mosn.io/htnn/api/pkg/filtermanager/api"
)

func (cb *filterManagerCallbackHandler) RefreshRouteCache() {
	api.LogErrorf("RefreshRouteCache is not implemented: %s", debug.Stack())
}

type decoderFilterCallbackHandlerWrapper struct {
	capi.DecoderFilterCallbacks
}

func NewDecoderFilterCallbackHandlerWrapper(h capi.DecoderFilterCallbacks) api.DecoderFilterCallbacks {
	return &decoderFilterCallbackHandlerWrapper{DecoderFilterCallbacks: h}
}

func (w *decoderFilterCallbackHandlerWrapper) AddData([]byte, bool) {
	api.LogErrorf("AddData is not implemented: %s", debug.Stack())
}

func (cb *filterManagerCallbackHandler) DecoderFilterCallbacks() api.DecoderFilterCallbacks {
	return NewDecoderFilterCallbackHandlerWrapper(cb.FilterCallbackHandler.DecoderFilterCallbacks())
}

type encoderFilterCallbackHandlerWrapper struct {
	capi.EncoderFilterCallbacks
}

func NewEncoderFilterCallbackHandlerWrapper(h capi.EncoderFilterCallbacks) api.EncoderFilterCallbacks {
	return &encoderFilterCallbackHandlerWrapper{EncoderFilterCallbacks: h}
}

func (w *encoderFilterCallbackHandlerWrapper) AddData([]byte, bool) {
	api.LogErrorf("AddData is not implemented: %s", debug.Stack())
}

func (cb *filterManagerCallbackHandler) EncoderFilterCallbacks() api.EncoderFilterCallbacks {
	return NewEncoderFilterCallbackHandlerWrapper(cb.FilterCallbackHandler.EncoderFilterCallbacks())
}

func (cb *filterManagerCallbackHandler) Continue(st capi.StatusType, decoding bool) {
	if decoding {
		cb.FilterCallbackHandler.DecoderFilterCallbacks().Continue(st)
	} else {
		cb.FilterCallbackHandler.EncoderFilterCallbacks().Continue(st)
	}
}
