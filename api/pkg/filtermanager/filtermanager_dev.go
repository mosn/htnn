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

//go:build envoydev

package filtermanager

import (
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

const (
	// FIXME: set header/buffer fails because of "invalid phase" error. We need to fix it.
	supportBufferingWithTrailers = false
	supportGettingHeadersOnLog   = true
)

func (m *filterManager) OnLog(reqHdr capi.RequestHeaderMap, reqTrailer capi.RequestTrailerMap, rspHdr capi.ResponseHeaderMap, rspTrailer capi.ResponseTrailerMap) {
	if m.canSkipOnLog {
		return
	}

	m.hdrLock.Lock()
	if m.reqHdr == nil {
		m.reqHdr = &filterManagerRequestHeaderMap{
			RequestHeaderMap: reqHdr,
		}
	} else {
		// In our benchmark BenchmarkFilterManagerRegular, reuse the request header wrapper is 5% faster than create a new one,
		// even the reusage requires holding the lock though it is running on fast path.
		h, _ := m.reqHdr.(*filterManagerRequestHeaderMap)
		h.RequestHeaderMap = reqHdr
	}
	m.hdrLock.Unlock()
	m.runOnLogPhase(m.reqHdr, reqTrailer, rspHdr, rspTrailer)
}

func wrapFilterManager(fm *filterManager) capi.StreamFilter {
	return fm
}

// This method is test only
func unwrapFilterManager(wrapper capi.StreamFilter) *filterManager {
	return wrapper.(*filterManager)
}
