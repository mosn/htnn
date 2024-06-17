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
	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	internalConsumer "mosn.io/htnn/api/internal/consumer"
	"mosn.io/htnn/api/internal/proto"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
)

func TestParse(t *testing.T) {
	ts := xds.TypedStruct{}
	ts.Value, _ = structpb.NewStruct(map[string]interface{}{})
	any1 := proto.MessageToAny(&ts)

	cases := []struct {
		name    string
		input   *anypb.Any
		wantErr bool
	}{
		{
			name:    "happy path",
			input:   any1,
			wantErr: false,
		},
		{
			name:    "happy path without config",
			input:   &anypb.Any{},
			wantErr: false,
		},
		{
			name: "error UnmarshalTo",
			input: &anypb.Any{
				TypeUrl: "aaa",
			},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parser := &FilterManagerConfigParser{}

			_, err := parser.Parse(c.input, nil)
			if c.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

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
		m := FilterManagerFactory(config)(cb).(*filterManager)
		hdr := envoy.NewRequestHeaderMap(http.Header{})
		m.DecodeHeaders(hdr, false)
		cb.WaitContinued()
		buf := envoy.NewBufferInstance([]byte{})
		m.DecodeData(buf, true)
		cb.WaitContinued()
		respHdr := envoy.NewResponseHeaderMap(http.Header{})
		m.EncodeHeaders(respHdr, false)
		cb.WaitContinued()
		m.EncodeData(buf, true)
		cb.WaitContinued()
		m.OnLog()
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
			m := FilterManagerFactory(config)(cb).(*filterManager)
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
			// use the Content-Type from the request
			reply: envoy.LocalResponse{
				Code:    200,
				Body:    `{"msg":"msg"}`,
				Headers: map[string][]string{"Content-Type": {"application/json"}},
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
			m := FilterManagerFactory(config)(cb).(*filterManager)
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
	m := FilterManagerFactory(config)(cb).(*filterManager)
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
			m := FilterManagerFactory(config)(cb).(*filterManager)
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

	config = initFilterManagerConfig("from_lds")
	// simulate config inherited from LDS
	config.parsed = []*model.ParsedFilterConfig{
		okParsed,
		badParsed,
	}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := FilterManagerFactory(config)(cb).(*filterManager)
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

func (f *onLogFilter) OnLog() {
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
		m := FilterManagerFactory(config)(cb).(*filterManager)
		assert.Equal(t, false, m.canSkipOnLog)
		assert.Equal(t, true, m.canSkipDecodeData)
	}
}

func TestFiltersFromConsumer(t *testing.T) {
	config := initFilterManagerConfig("ns")
	config.consumerFiltersEndAt = 1

	consumers := map[string]*internalConsumer.Consumer{}
	for i := 0; i < 10; i++ {
		c := internalConsumer.Consumer{
			FilterConfigs: map[string]*model.ParsedFilterConfig{
				"add_req": {
					Name:    "add_req",
					Factory: addReqFactory,
					ParsedConfig: addReqConf{
						hdrName: fmt.Sprintf("x-htnn-consumer-%d", i),
					},
				},
			},
		}
		if i%2 == 0 {
			c.FilterConfigs["on_log"] = &model.ParsedFilterConfig{
				Name:    "on_log",
				Factory: onLogFactory,
			}
		}
		consumers[strconv.Itoa(i)] = &c
	}
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "set_consumer",
			Factory: setConsumerFactory,
			ParsedConfig: setConsumerConf{
				Consumers: consumers,
			},
		},
		{
			Name:    "add_req",
			Factory: addReqFactory,
			ParsedConfig: addReqConf{
				hdrName: "x-htnn-route",
			},
		},
	}

	var wg sync.WaitGroup
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func(i int) {
			cb := envoy.NewCAPIFilterCallbackHandler()
			m := FilterManagerFactory(config)(cb).(*filterManager)
			assert.Equal(t, true, m.canSkipOnLog)
			assert.Equal(t, 1, len(m.filters))
			h := http.Header{}
			idx := i % 10
			h.Add("consumer", strconv.Itoa(idx))
			hdr := envoy.NewRequestHeaderMap(h)
			m.DecodeHeaders(hdr, true)
			cb.WaitContinued()
			if idx%2 == 0 {
				assert.Equal(t, false, m.canSkipOnLog)
				assert.Equal(t, 2, len(m.filters))
			} else {
				assert.Equal(t, true, m.canSkipOnLog)
				assert.Equal(t, 1, len(m.filters))
			}

			_, ok := hdr.Get("x-htnn-route")
			assert.False(t, ok)
			_, ok = hdr.Get(fmt.Sprintf("x-htnn-consumer-%d", idx))
			assert.True(t, ok)

			wg.Done()
		}(i)
	}
	wg.Wait()
}

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
