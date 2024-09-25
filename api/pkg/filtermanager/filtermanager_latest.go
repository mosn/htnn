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

//go:build !envoydev

package filtermanager

import (
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"mosn.io/htnn/api/pkg/filtermanager/api"
)

func (m *filterManager) OnLog(_ capi.RequestHeaderMap, _ capi.RequestTrailerMap, _ capi.ResponseHeaderMap, _ capi.ResponseTrailerMap) {
	if m.canSkipOnLog {
		return
	}

	var reqHdr api.RequestHeaderMap
	m.hdrLock.Lock()
	reqHdr = m.reqHdr
	m.hdrLock.Unlock()
	var rspHdr api.ResponseHeaderMap
	m.hdrLock.Lock()
	rspHdr = m.rspHdr
	m.hdrLock.Unlock()

	m.runOnLogPhase(reqHdr, rspHdr)
}

type filterManagerWrapper struct {
	*filterManager
}

func (w *filterManagerWrapper) OnLog() {
	w.filterManager.OnLog(nil, nil, nil, nil)
}

// we will get rid of this wrapper once Envoy 1.32 is released

func wrapFilterManager(fm *filterManager) capi.StreamFilter {
	return &filterManagerWrapper{fm}
}

// This method is test only
func unwrapFilterManager(wrapper capi.StreamFilter) *filterManager {
	fmw, _ := wrapper.(*filterManagerWrapper)
	return fmw.filterManager
}
