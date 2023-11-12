package filtermanager

import (
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/proto"
	"mosn.io/moe/tests/pkg/envoy"
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
		current: []*filterConfig{
			{
				// fallback to PassThroughFilter
				Name: "unknown",
			},
		},
	})(cb)
	hdr := envoy.NewRequestHeaderMap(http.Header{})
	m.DecodeHeaders(hdr, false)
	buf := envoy.NewBufferInstance([]byte{})
	m.DecodeData(buf, true)
	respHdr := envoy.NewResponseHeaderMap(http.Header{})
	m.EncodeHeaders(respHdr, false)
	m.EncodeData(buf, true)
	m.OnLog()
}

func TestLocalReplyJSON_UseReqHeader(t *testing.T) {
	tests := []struct {
		name string
		hdr  func(hdr http.Header) http.Header
		body string
	}{
		{
			name: "default",
			hdr: func(h http.Header) http.Header {
				return h
			},
			body: `{"msg":"msg"}`,
		},
		{
			name: "application/json",
			hdr: func(h http.Header) http.Header {
				h.Add("content-type", "application/json")
				return h
			},
			body: `{"msg":"msg"}`,
		},
		{
			name: "no JSON",
			hdr: func(h http.Header) http.Header {
				h.Add("content-type", "text/plain")
				return h
			},
			body: `msg`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := envoy.NewFilterCallbackHandler()
			m := FilterManagerConfigFactory(&filterManagerConfig{
				current: []*filterConfig{
					{
						Name: "test",
					},
				},
			})(cb).(*filterManager)
			patches := gomonkey.ApplyMethodReturn(m.filters[0], "DecodeHeaders", &api.LocalResponse{
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
			assert.Equal(t, tt.body, lr.Body)
		})
	}
}

func TestLocalReplyJSON_UseRespHeader(t *testing.T) {
	tests := []struct {
		name string
		hdr  func(hdr http.Header) http.Header
		body string
	}{
		{
			name: "no content-type",
			hdr: func(h http.Header) http.Header {
				return h
			},
			body: `{"msg":"msg"}`,
		},
		{
			name: "application/json",
			hdr: func(h http.Header) http.Header {
				h.Add("content-type", "application/json")
				return h
			},
			body: `{"msg":"msg"}`,
		},
		{
			name: "no JSON",
			hdr: func(h http.Header) http.Header {
				h.Add("content-type", "text/plain")
				return h
			},
			body: `msg`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := envoy.NewFilterCallbackHandler()
			m := FilterManagerConfigFactory(&filterManagerConfig{
				current: []*filterConfig{
					{
						Name: "test",
					},
				},
			})(cb).(*filterManager)
			patches := gomonkey.ApplyMethodReturn(m.filters[0], "EncodeHeaders", &api.LocalResponse{
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
			assert.Equal(t, tt.body, lr.Body)
		})
	}
}
