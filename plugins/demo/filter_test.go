package demo

import (
	"net/http"
	"testing"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/stretchr/testify/assert"

	"mosn.io/moe/pkg/test/envoy"
)

func TestHello(t *testing.T) {
	f := &filter{}
	assert.Equal(t, "hello", f.hello())
}

func TestDecodeHeaders(t *testing.T) {
	cb := &envoy.FiterCallbackHandler{}
	info := &envoy.StreamInfo{}
	info.SetFilterState(envoy.NewFilterState(map[string]string{
		"header_name": "hdr",
	}))
	cb.SetStreamInfo(info)
	f := configFactory(&config{})(cb)
	hdr := envoy.NewRequestHeaderMap(http.Header{})
	assert.Equal(t, api.Continue, f.DecodeHeaders(hdr, true))
	v, _ := hdr.Get("hdr")
	assert.Equal(t, "hello", v)
}
