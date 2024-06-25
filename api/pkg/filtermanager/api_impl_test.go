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
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
)

// Most of APIs are tests in the test of their caller.

func setPluginStateFilterFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &setPluginStateFilter{
		callbacks: callbacks,
	}
}

type setPluginStateFilter struct {
	api.PassThroughFilter
	callbacks api.FilterCallbackHandler
}

func (f *setPluginStateFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	f.callbacks.PluginState().Set("test", "key", "value")
	return api.Continue
}

func getPluginStateFilterFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &getPluginStateFilter{
		callbacks: callbacks,
	}
}

type getPluginStateFilter struct {
	api.PassThroughFilter
	callbacks api.FilterCallbackHandler
}

func (f *getPluginStateFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	v := f.callbacks.PluginState().Get("test", "key")
	headers.Set("x-htnn-v", v.(string))
	return api.Continue
}

func TestPluginState(t *testing.T) {
	cb := envoy.NewCAPIFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "alice",
			Factory: setPluginStateFilterFactory,
		},
		{
			Name:    "bob",
			Factory: getPluginStateFilterFactory,
		},
	}
	m := FilterManagerFactory(config)(cb).(*filterManager)
	h := http.Header{}
	hdr := envoy.NewRequestHeaderMap(h)
	m.DecodeHeaders(hdr, true)
	cb.WaitContinued()
	v, _ := hdr.Get("x-htnn-v")
	assert.Equal(t, "value", v)
}

func accessCacheFieldsFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &accessCacheFieldsFilter{
		callbacks: callbacks,
	}
}

type accessCacheFieldsFilter struct {
	api.PassThroughFilter
	callbacks api.FilterCallbackHandler
}

func (f *accessCacheFieldsFilter) do(headers api.RequestHeaderMap) api.ResultAction {
	// Maybe we can relax the concurreny requirement for header modification?
	// Update headers in OnLog is meaningless. Anyway, add lock for now.
	headers.Set("Cookie", "k=v")
	p := headers.Url().Path
	headers.Add("Cookie", fmt.Sprintf("k=%s", p))
	headers.Cookie("k")
	headers.Del("Cookie")

	f.callbacks.PluginState().Set("ns", "k", f.callbacks.StreamInfo().DownstreamRemoteAddress())
	st := f.callbacks.PluginState()
	st.Get("ns", "ip")
	st.Set("ns", "ip", f.callbacks.StreamInfo().DownstreamRemoteParsedAddress())
	st.Get("ns", "ip")
	return api.Continue
}

func (f *accessCacheFieldsFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	f.do(headers)
	return api.Continue
}

func (f *accessCacheFieldsFilter) OnLog(reqHeaders api.RequestHeaderMap, reqTrailers api.RequestTrailerMap,
	respHeaders api.ResponseHeaderMap, respTrailers api.ResponseTrailerMap) {
	f.do(reqHeaders)
}

func TestAccessCacheFieldsConcurrently(t *testing.T) {
	config := initFilterManagerConfig("ns")
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "access_cache_fields",
			Factory: accessCacheFieldsFactory,
		},
	}
	// FIXME: remove this once we get request headers from the OnLog directly
	config.enableDebugMode = true // let m.reqHdr not be nil

	n := 10
	var wg sync.WaitGroup

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := FilterManagerFactory(config)(cb).(*filterManager)
			h := http.Header{}
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, true)
			m.OnLog()
			wg.Done()
		}(i)
	}
	wg.Wait()
}
