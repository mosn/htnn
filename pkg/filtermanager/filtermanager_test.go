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
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	pkgConsumer "mosn.io/htnn/pkg/consumer"
	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/filtermanager/model"
	"mosn.io/htnn/pkg/proto"
	"mosn.io/htnn/plugins/tests/pkg/envoy"
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
	cb := envoy.NewFilterCallbackHandler()
	m := FilterManagerConfigFactory(&filterManagerConfig{
		current: []*model.ParsedFilterConfig{
			{
				Name:          "passthrough",
				ConfigFactory: PassThroughFactory,
			},
		},
	})(cb)
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
			cb := envoy.NewFilterCallbackHandler()
			m := FilterManagerConfigFactory(&filterManagerConfig{
				current: []*model.ParsedFilterConfig{
					{
						Name:          "test",
						ConfigFactory: PassThroughFactory,
					},
				},
			})(cb).(*filterManager)
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
			cb := envoy.NewFilterCallbackHandler()
			m := FilterManagerConfigFactory(&filterManagerConfig{
				current: []*model.ParsedFilterConfig{
					{
						Name:          "test",
						ConfigFactory: PassThroughFactory,
					},
				},
			})(cb).(*filterManager)
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
	cb := envoy.NewFilterCallbackHandler()
	m := FilterManagerConfigFactory(&filterManagerConfig{
		current: []*model.ParsedFilterConfig{
			{
				Name:          "test",
				ConfigFactory: PassThroughFactory,
			},
		},
	})(cb).(*filterManager)
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

func setConsumerFactory(interface{}) api.FilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &setConsumerFilter{
			callbacks: callbacks,
		}
	}
}

type setConsumerFilter struct {
	api.PassThroughFilter
	callbacks api.FilterCallbackHandler
}

func (f *setConsumerFilter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.ResultAction {
	f.callbacks.SetConsumer(&pkgConsumer.Consumer{
		FilterConfigs: map[string]*model.ParsedFilterConfig{
			"on_log": {
				Name:          "on_log",
				ConfigFactory: onLogFactory,
			},
			"add_req": {
				Name:          "add_req",
				ConfigFactory: addReqFactory,
				ParsedConfig: addReqConf{
					hdrName: "x-htnn-consumer",
				},
			},
		},
	})
	return api.Continue
}

func onLogFactory(interface{}) api.FilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &onLogFilter{}
	}
}

type onLogFilter struct {
	api.PassThroughFilter
}

func (f *onLogFilter) OnLog() {
}

type addReqConf struct {
	hdrName string
}

func addReqFactory(c interface{}) api.FilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &addReqFilter{
			conf: c.(addReqConf),
		}
	}
}

type addReqFilter struct {
	api.PassThroughFilter

	conf addReqConf
}

func (f *addReqFilter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.ResultAction {
	header.Set(f.conf.hdrName, "htnn")
	return api.Continue
}

func TestFiltersFromConsumer(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	m := FilterManagerConfigFactory(&filterManagerConfig{
		authnFiltersEndAt: 1,
		current: []*model.ParsedFilterConfig{
			{
				Name:          "set_consumer",
				ConfigFactory: setConsumerFactory,
			},
			{
				Name:          "add_req",
				ConfigFactory: addReqFactory,
				ParsedConfig: addReqConf{
					hdrName: "x-htnn-route",
				},
			},
		},
	})(cb).(*filterManager)
	assert.Equal(t, true, m.canSkipOnLog)
	assert.Equal(t, 1, len(m.filters))
	hdr := envoy.NewRequestHeaderMap(http.Header{})
	m.DecodeHeaders(hdr, true)
	cb.WaitContinued()
	assert.Equal(t, false, m.canSkipOnLog)
	assert.Equal(t, 2, len(m.filters))

	_, ok := hdr.Get("x-htnn-route")
	assert.False(t, ok)
	_, ok = hdr.Get("x-htnn-consumer")
	assert.True(t, ok)
}
