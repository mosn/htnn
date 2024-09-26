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
	"strings"
	"sync"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/internal/consumer"
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
	m := FilterManagerFactory(config, cb)
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
	headers.Set("Cookie", "k=v")
	p := headers.URL().Path
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
			m := unwrapFilterManager(FilterManagerFactory(config, cb))
			h := http.Header{}
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, true)
			m.OnLog(hdr, nil, nil, nil)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func testLogFilterFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &testLogFilter{
		callbacks: callbacks,
	}
}

type testLogFilter struct {
	api.PassThroughFilter
	callbacks api.FilterCallbackHandler
}

func (f *testLogFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	cb := f.callbacks.WithLogArg("k1", 1)
	for _, fu := range []func(string){
		f.callbacks.LogTrace, f.callbacks.LogDebug, f.callbacks.LogInfo, f.callbacks.LogWarn, f.callbacks.LogError,
	} {
		fu("testLog: msg")
	}
	cb.WithLogArg("k2", 2)
	for _, fu := range []func(string, ...any){
		f.callbacks.LogTracef, f.callbacks.LogDebugf, f.callbacks.LogInfof, f.callbacks.LogWarnf, f.callbacks.LogErrorf,
	} {
		fu("testLog: out: %s", 0)
	}
	return api.Continue
}

func TestLogWithArgs(t *testing.T) {
	fmtStr := map[string]string{}
	fmtArgs := map[string][]any{}

	patch := gomonkey.NewPatches()
	for _, s := range []struct {
		level string
		logf  func(string, ...any)
		log   func(string)
	}{
		{"Trace", api.LogTracef, api.LogTrace},
		{"Debug", api.LogDebugf, api.LogDebug},
		{"Info", api.LogInfof, api.LogInfo},
		{"Warn", api.LogWarnf, api.LogWarn},
		{"Error", api.LogErrorf, api.LogError},
	} {
		level := s.level

		patch.ApplyFunc(s.logf, func(format string, args ...any) {
			if !strings.HasPrefix(format, "testLog: ") {
				return
			}
			fmtStr[level+"f"] = strings.Clone(format)
			fmtArgs[level+"f"] = args
		})
		patch.ApplyFunc(s.log, func(msg string) {
			if !strings.HasPrefix(msg, "testLog: ") {
				return
			}
			fmtStr[level] = strings.Clone(msg)
		})
	}
	defer patch.Reset()

	cb := envoy.NewCAPIFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "log",
			Factory: testLogFilterFactory,
		},
	}
	m := FilterManagerFactory(config, cb)
	h := http.Header{}
	hdr := envoy.NewRequestHeaderMap(h)
	m.DecodeHeaders(hdr, true)
	cb.WaitContinued()
	for _, level := range []string{"Trace", "Debug", "Info", "Warn", "Error"} {
		assert.Equal(t, "testLog: msg, k1: 1", fmtStr[level])
		assert.Equal(t, "testLog: out: %s, k1: %v, k2: %v", fmtStr[level+"f"])
		assert.Equal(t, []any{0, 1, 2}, fmtArgs[level+"f"])
	}
}

func TestReset(t *testing.T) {
	cb := &filterManagerCallbackHandler{
		FilterCallbackHandler: envoy.NewCAPIFilterCallbackHandler(),
	}
	cb.SetConsumer(&consumer.MockConsumer{})
	cb.PluginState()
	cb.StreamInfo()
	cb.WithLogArg("k", "v")

	assert.NotNil(t, cb.consumer)
	assert.NotNil(t, cb.pluginState)
	assert.NotNil(t, cb.streamInfo)
	assert.NotEqual(t, "", cb.logArgNames)
	assert.NotNil(t, cb.logArgs)

	cb.Reset()
	assert.Nil(t, cb.consumer)
	assert.Nil(t, cb.pluginState)
	assert.Nil(t, cb.streamInfo)
	assert.Equal(t, "", cb.logArgNames)
	assert.Nil(t, cb.logArgs)
}
