package filtermanager

import (
	"net/http"
	"testing"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

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
	cb := &envoy.FiterCallbackHandler{}
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
