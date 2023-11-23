package demo

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/moe/plugins/tests/pkg/envoy"
)

func TestHello(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	info := &envoy.StreamInfo{}
	info.SetFilterState(envoy.NewFilterState(map[string]string{
		"guest_name": "Jack",
	}))
	cb.SetStreamInfo(info)
	f := configFactory(&config{
		Config: Config{
			HostName: "Tom",
		},
	})(cb).(*filter)
	assert.Equal(t, "hello, Jack", f.hello())
}

func TestDecodeHeaders(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	info := &envoy.StreamInfo{}
	info.SetFilterState(envoy.NewFilterState(map[string]string{
		"guest_name": "Jack",
	}))
	cb.SetStreamInfo(info)
	f := configFactory(&config{
		Config: Config{
			HostName: "Tom",
		},
	})(cb)
	hdr := envoy.NewRequestHeaderMap(http.Header{})
	f.DecodeHeaders(hdr, true)
	v, _ := hdr.Get("Tom")
	assert.Equal(t, "hello, Jack", v)
}
