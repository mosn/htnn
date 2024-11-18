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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	internalConsumer "mosn.io/htnn/api/internal/consumer"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
)

func TestPassThrough(t *testing.T) {
	cb := envoy.NewCAPIFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "passthrough",
			Factory: PassThroughFactory,
		},
	}
	for i := 0; i < 2; i++ {
		m := unwrapFilterManager(FilterManagerFactory(config, cb))
		hdr := envoy.NewRequestHeaderMap(http.Header{})
		m.DecodeHeaders(hdr, false)
		cb.WaitContinued()
		buf := envoy.NewBufferInstance([]byte{})
		m.DecodeData(buf, false)
		cb.WaitContinued()
		trailer := envoy.NewRequestTrailerMap(http.Header{})
		m.DecodeTrailers(trailer)
		cb.WaitContinued()

		respHdr := envoy.NewResponseHeaderMap(http.Header{})
		m.EncodeHeaders(respHdr, false)
		cb.WaitContinued()
		m.EncodeData(buf, false)
		cb.WaitContinued()
		respTrailer := envoy.NewResponseTrailerMap(http.Header{})
		m.EncodeTrailers(respTrailer)
		cb.WaitContinued()

		m.OnLog(hdr, nil, respHdr, nil)
	}
}

func TestLocalReplyJSON_UseReqHeader(t *testing.T) {
	tests := []struct {
		name  string
		hdr   func(hdr http.Header) http.Header
		reply envoy.LocalResponse
	}{
		{
			name: "default",
			hdr: func(h http.Header) http.Header {
				return h
			},
			reply: envoy.LocalResponse{
				Code:    200,
				Headers: map[string][]string{"Content-Type": {"application/json"}},
				Body:    `{"msg":"msg"}`,
			},
		},
		{
			name: "application/json",
			hdr: func(h http.Header) http.Header {
				h.Add("content-type", "application/json")
				return h
			},
			reply: envoy.LocalResponse{
				Code:    200,
				Body:    `{"msg":"msg"}`,
				Headers: map[string][]string{"Content-Type": {"application/json"}},
			},
		},
		{
			name: "no JSON",
			hdr: func(h http.Header) http.Header {
				h.Add("content-type", "text/plain")
				return h
			},
			reply: envoy.LocalResponse{
				Code: 200,
				Body: "msg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			config := initFilterManagerConfig("ns")
			config.parsed = []*model.ParsedFilterConfig{
				{
					Name:    "test",
					Factory: PassThroughFactory,
				},
			}
			m := unwrapFilterManager(FilterManagerFactory(config, cb))
			patches := gomonkey.ApplyMethodReturn(m.filters[0].Filter, "DecodeHeaders", &api.LocalResponse{
				Code: 200,
				Msg:  "msg",
			})
			defer patches.Reset()

			h := http.Header{}
			if tt.hdr != nil {
				h = tt.hdr(h)
			}
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, false)
			cb.WaitContinued()
			lr := cb.LocalResponse()
			assert.Equal(t, tt.reply, lr)
		})
	}
}

func TestLocalReplyJSON_UseRespHeader(t *testing.T) {
	tests := []struct {
		name  string
		hdr   func(hdr http.Header) http.Header
		reply envoy.LocalResponse
	}{
		{
			name: "no content-type",
			hdr: func(h http.Header) http.Header {
				return h
			},
			// do not use the Content-Type from the request
			reply: envoy.LocalResponse{
				Code: 200,
				Body: "msg",
			},
		},
		{
			name: "application/json",
			hdr: func(h http.Header) http.Header {
				h.Add("content-type", "application/json")
				return h
			},
			reply: envoy.LocalResponse{
				Code:    200,
				Body:    `{"msg":"msg"}`,
				Headers: map[string][]string{"Content-Type": {"application/json"}},
			},
		},
		{
			name: "no JSON",
			hdr: func(h http.Header) http.Header {
				h.Add("content-type", "text/plain")
				return h
			},
			reply: envoy.LocalResponse{
				Code: 200,
				Body: "msg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			config := initFilterManagerConfig("ns")
			config.parsed = []*model.ParsedFilterConfig{
				{
					Name:    "test",
					Factory: PassThroughFactory,
				},
			}
			m := unwrapFilterManager(FilterManagerFactory(config, cb))
			patches := gomonkey.ApplyMethodReturn(m.filters[0].Filter, "EncodeHeaders", &api.LocalResponse{
				Code: 200,
				Msg:  "msg",
			})
			defer patches.Reset()

			reqHdr := http.Header{}
			reqHdr.Set("content-type", "application/json")
			hdr := envoy.NewRequestHeaderMap(reqHdr)
			m.DecodeHeaders(hdr, true)
			cb.WaitContinued()

			h := http.Header{}
			if tt.hdr != nil {
				h = tt.hdr(h)
			}
			respHdr := envoy.NewResponseHeaderMap(h)
			m.EncodeHeaders(respHdr, false)
			cb.WaitContinued()

			lr := cb.LocalResponse()
			assert.Equal(t, tt.reply, lr)
		})
	}
}

func TestLocalReplyJSON_DoNotChangeMsgIfContentTypeIsGiven(t *testing.T) {
	cb := envoy.NewCAPIFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "test",
			Factory: PassThroughFactory,
		},
	}
	m := unwrapFilterManager(FilterManagerFactory(config, cb))
	patches := gomonkey.ApplyMethodReturn(m.filters[0].Filter, "DecodeHeaders", &api.LocalResponse{
		Msg:    "msg",
		Header: http.Header(map[string][]string{"Content-Type": {"text/plain"}}),
	})
	defer patches.Reset()

	h := http.Header{}
	h.Set("Content-Type", "application/json")
	hdr := envoy.NewRequestHeaderMap(h)
	m.DecodeHeaders(hdr, false)
	cb.WaitContinued()
	lr := cb.LocalResponse()
	assert.Equal(t, envoy.LocalResponse{
		Code:    200,
		Body:    "msg",
		Headers: map[string][]string{"Content-Type": {"text/plain"}},
	}, lr)
}

func initFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &api.PassThroughFilter{}
}

type initConfig struct {
	count int
	err   error
}

func (c *initConfig) Init(cb api.ConfigCallbackHandler) error {
	c.count++
	return c.err
}

func TestInitFailed(t *testing.T) {
	config := initFilterManagerConfig("ns")
	config.initOnce = &sync.Once{}
	ok := &initConfig{}
	bad := &initConfig{
		err: errors.New("ouch"),
	}
	okParsed := &model.ParsedFilterConfig{
		Name:         "init",
		Factory:      initFactory,
		ParsedConfig: ok,
	}
	badParsed := &model.ParsedFilterConfig{
		Name:         "initFailed",
		Factory:      initFactory,
		ParsedConfig: bad,
	}

	config.parsed = []*model.ParsedFilterConfig{
		okParsed,
		badParsed,
	}
	n := 10
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := FilterManagerFactory(config, cb)
			h := http.Header{}
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, true)
			cb.WaitContinued()
			r := cb.LocalResponse()
			assert.Equal(t, 500, r.Code)

			wg.Done()
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 1, ok.count)
	assert.Equal(t, 1, bad.count)

	config2 := initFilterManagerConfig("from_lds")
	// simulate config inherited from LDS
	config2 = config2.Merge(config)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := FilterManagerFactory(config2, cb)
			h := http.Header{}
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, true)
			cb.WaitContinued()
			r := cb.LocalResponse()
			assert.Equal(t, 500, r.Code)

			wg.Done()
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 1, ok.count)
	assert.Equal(t, 1, bad.count)
}

func onLogFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &onLogFilter{}
}

type onLogFilter struct {
	api.PassThroughFilter
}

func (f *onLogFilter) OnLog(reqHeaders api.RequestHeaderMap, reqTrailers api.RequestTrailerMap,
	respHeaders api.ResponseHeaderMap, respTrailers api.ResponseTrailerMap) {
}

type addReqConf struct {
	hdrName string
}

func addReqFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &addReqFilter{
		conf: c.(addReqConf),
	}
}

type addReqFilter struct {
	api.PassThroughFilter

	conf addReqConf
}

func (f *addReqFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	headers.Set(f.conf.hdrName, "htnn")
	return api.Continue
}

func (f *addReqFilter) DecodeTrailers(trailers api.RequestTrailerMap) api.ResultAction {
	trailers.Set(f.conf.hdrName, "htnn")
	return api.Continue
}

func TestSkipMethodWhenThereAreMultiFilters(t *testing.T) {
	cb := envoy.NewCAPIFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "add_req",
			Factory: addReqFactory,
			ParsedConfig: addReqConf{
				hdrName: "x-htnn-route",
			},
		},
		{
			Name:    "on_log",
			Factory: onLogFactory,
		},
	}

	for i := 0; i < 2; i++ {
		m := unwrapFilterManager(FilterManagerFactory(config, cb))
		assert.Equal(t, false, m.canSkipOnLog)
		assert.Equal(t, false, m.canSkipDecodeHeaders)
		assert.Equal(t, true, m.canSkipDecodeData)
		assert.Equal(t, false, m.canSkipDecodeTrailers)
		assert.Equal(t, true, m.canSkipEncodeTrailers)
	}
}

type addRespConf struct {
	hdrName string
}

func addRespFactory(c interface{}, _ api.FilterCallbackHandler) api.Filter {
	return &addRespFilter{
		conf: c.(addRespConf),
	}
}

type addRespFilter struct {
	api.PassThroughFilter

	conf addRespConf
}

func (f *addRespFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	headers.Set(f.conf.hdrName, "htnn")
	return api.Continue
}

type setConsumerConf struct {
	Consumers map[string]*internalConsumer.Consumer
}

func setConsumerFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &setConsumerFilter{
		callbacks: callbacks,
		conf:      c.(setConsumerConf),
	}
}

type setConsumerFilter struct {
	api.PassThroughFilter
	conf      setConsumerConf
	callbacks api.FilterCallbackHandler
}

func (f *setConsumerFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	key, _ := headers.Get("Consumer")
	c := f.conf.Consumers[key]
	f.callbacks.SetConsumer(c)
	return api.Continue
}

func TestFiltersFromConsumer(t *testing.T) {
	config := initFilterManagerConfig("ns")
	config.consumerFiltersEndAt = 1

	consumers := map[string]*internalConsumer.Consumer{}
	n := 10
	for i := 0; i < n; i++ {
		c := internalConsumer.Consumer{
			FilterConfigs: map[string]*model.ParsedFilterConfig{
				"2_add_req": {
					Name:    "2_add_req",
					Factory: addReqFactory,
					ParsedConfig: addReqConf{
						hdrName: fmt.Sprintf("x-htnn-consumer-%d", i),
					},
				},
				"4_add_resp": {
					Name:    "4_add_resp",
					Factory: addRespFactory,
					ParsedConfig: addRespConf{
						hdrName: fmt.Sprintf("x-htnn-resp-%d", i),
					},
					CanSyncRun: true,
				},
			},
		}
		if i%2 == 0 {
			c.FilterConfigs["3_on_log"] = &model.ParsedFilterConfig{
				Name:    "3_on_log",
				Factory: onLogFactory,
			}
		}
		consumers[strconv.Itoa(i)] = &c
	}
	config.parsed = []*model.ParsedFilterConfig{
		// HTNN will sort the plugins when merging the plugins from the consumer.
		// Here we add number as the prefix to ensure the order.
		{
			Name:    "1_set_consumer",
			Factory: setConsumerFactory,
			ParsedConfig: setConsumerConf{
				Consumers: consumers,
			},
		},
		{
			Name:    "2_add_req",
			Factory: addReqFactory,
			ParsedConfig: addReqConf{
				hdrName: "x-htnn-route",
			},
		},
	}

	var wg sync.WaitGroup
	wg.Add(2 * n)
	for i := 0; i < 2*n; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := unwrapFilterManager(FilterManagerFactory(config, cb))
			assert.Equal(t, true, m.canSkipOnLog)
			assert.Equal(t, 2, len(m.filters))
			h := http.Header{}
			idx := i % n
			h.Add("consumer", strconv.Itoa(idx))
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, true)
			cb.WaitContinued()
			if idx%2 == 0 {
				assert.Equal(t, false, m.canSkipOnLog)
				assert.Equal(t, 4, len(m.filters))
			} else {
				assert.Equal(t, true, m.canSkipOnLog)
				assert.Equal(t, 3, len(m.filters))
			}
			assert.Equal(t, true, m.canSyncRunEncodeHeaders)

			_, ok := hdr.Get("x-htnn-route")
			assert.False(t, ok)
			_, ok = hdr.Get(fmt.Sprintf("x-htnn-consumer-%d", idx))
			assert.True(t, ok)

			wg.Done()
		}(i)
	}
	wg.Wait()
}

func accessFieldOnLogFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &accessFieldOnLogFilter{
		cb: callbacks,
	}
}

type accessFieldOnLogFilter struct {
	api.PassThroughFilter

	cb api.FilterCallbackHandler
}

func (f *accessFieldOnLogFilter) DecodeHeaders(_ api.RequestHeaderMap, _ bool) api.ResultAction {
	f.cb.StreamInfo().DownstreamLocalAddress()
	return api.Continue
}

func (f *accessFieldOnLogFilter) DecodeData(_ api.BufferInstance, _ bool) api.ResultAction {
	f.cb.StreamInfo().DownstreamLocalAddress()
	return api.Continue
}

func (f *accessFieldOnLogFilter) DecodeTrailers(_ api.RequestTrailerMap) api.ResultAction {
	f.cb.StreamInfo().DownstreamLocalAddress()
	return api.Continue
}

func (f *accessFieldOnLogFilter) EncodeHeaders(_ api.ResponseHeaderMap, _ bool) api.ResultAction {
	f.cb.StreamInfo().DownstreamLocalAddress()
	return api.Continue
}

func (f *accessFieldOnLogFilter) EncodeData(_ api.BufferInstance, _ bool) api.ResultAction {
	f.cb.StreamInfo().DownstreamLocalAddress()
	return api.Continue
}

func (f *accessFieldOnLogFilter) EncodeTrailers(_ api.ResponseTrailerMap) api.ResultAction {
	f.cb.StreamInfo().DownstreamLocalAddress()
	return api.Continue
}

func (f *accessFieldOnLogFilter) OnLog(_ api.RequestHeaderMap, _ api.RequestTrailerMap,
	_ api.ResponseHeaderMap, _ api.ResponseTrailerMap) {
	f.cb.StreamInfo().DownstreamLocalAddress()
}

func TestDoNotRecycleInUsedFilterManager(t *testing.T) {
	envoy.DisableLogInTest() // otherwise, there is too much output
	config := initFilterManagerConfig("ns")
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "access_field_on_log",
			Factory: accessFieldOnLogFactory,
		},
	}

	n := 100
	var wg sync.WaitGroup

	// DecodeHeaders
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

	// DecodeData
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := unwrapFilterManager(FilterManagerFactory(config, cb))
			h := http.Header{}
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, false)
			cb.WaitContinued()
			m.DecodeData(nil, true)
			m.OnLog(hdr, nil, nil, nil)
			wg.Done()
		}(i)
	}
	wg.Wait()

	// DecodeTrailers
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := unwrapFilterManager(FilterManagerFactory(config, cb))
			h := http.Header{}
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, false)
			cb.WaitContinued()
			m.DecodeData(nil, true)
			cb.WaitContinued()
			trailer := envoy.NewRequestTrailerMap(h)
			m.DecodeTrailers(trailer)
			m.OnLog(hdr, nil, nil, nil)
			wg.Done()
		}(i)
	}
	wg.Wait()

	// EncodeHeaders
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := unwrapFilterManager(FilterManagerFactory(config, cb))
			h := http.Header{}
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, true)
			cb.WaitContinued()
			hdr2 := envoy.NewResponseHeaderMap(h)
			m.EncodeHeaders(hdr2, true)
			m.OnLog(hdr, nil, hdr2, nil)
			wg.Done()
		}(i)
	}
	wg.Wait()

	// EncodeData
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := unwrapFilterManager(FilterManagerFactory(config, cb))
			h := http.Header{}
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, true)
			cb.WaitContinued()
			hdr2 := envoy.NewResponseHeaderMap(h)
			m.EncodeHeaders(hdr2, true)
			cb.WaitContinued()
			m.EncodeData(nil, true)
			m.OnLog(hdr, nil, hdr2, nil)
			wg.Done()
		}(i)
	}
	wg.Wait()

	// EncodeTrailers
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := unwrapFilterManager(FilterManagerFactory(config, cb))
			h := http.Header{}
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, true)
			cb.WaitContinued()
			hdr2 := envoy.NewResponseHeaderMap(h)
			m.EncodeHeaders(hdr2, true)
			cb.WaitContinued()
			m.EncodeData(nil, true)
			cb.WaitContinued()
			trailer := envoy.NewRequestTrailerMap(h)
			m.EncodeTrailers(trailer)
			m.OnLog(hdr, nil, hdr2, trailer)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestSyncRunWhenThereAreMultiFilters(t *testing.T) {
	cb := envoy.NewCAPIFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "add_req",
			Factory: addReqFactory,
			ParsedConfig: addReqConf{
				hdrName: "x-htnn-route",
			},
			CanSyncRun: false,
		},
		{
			Name:       "access_field_on_log",
			Factory:    accessFieldOnLogFactory,
			CanSyncRun: true,
		},
	}

	for i := 0; i < 2; i++ {
		m := unwrapFilterManager(FilterManagerFactory(config, cb))
		assert.Equal(t, false, m.canSyncRunDecodeHeaders)
		assert.Equal(t, true, m.canSyncRunDecodeData)
		assert.Equal(t, false, m.canSyncRunDecodeTrailers)
		assert.Equal(t, true, m.canSyncRunEncodeHeaders)
		assert.Equal(t, true, m.canSyncRunEncodeData)
		assert.Equal(t, true, m.canSyncRunEncodeTrailers)
	}
}
