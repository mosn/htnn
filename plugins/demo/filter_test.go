package demo

import (
	"net/http"
	"testing"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/stretchr/testify/assert"

	"mosn.io/moe/tests/pkg/envoy"
)

func TestHello(t *testing.T) {
	cb := &envoy.FiterCallbackHandler{}
	info := &envoy.StreamInfo{}
	info.SetFilterState(envoy.NewFilterState(map[string]string{
		"guest_name": "Jack",
	}))
	cb.SetStreamInfo(info)
	f := configFactory(&Config{
		HostName: "Tom",
	})(cb).(*filter)
	assert.Equal(t, "hello, Jack", f.hello())
}

func TestDecodeHeaders(t *testing.T) {
	cb := &envoy.FiterCallbackHandler{}
	info := &envoy.StreamInfo{}
	info.SetFilterState(envoy.NewFilterState(map[string]string{
		"guest_name": "Jack",
	}))
	cb.SetStreamInfo(info)
	f := configFactory(&Config{
		HostName: "Tom",
	})(cb)
	hdr := envoy.NewRequestHeaderMap(http.Header{})
	assert.Equal(t, api.Continue, f.DecodeHeaders(hdr, true))
	v, _ := hdr.Get("Tom")
	assert.Equal(t, "hello, Jack", v)
}
