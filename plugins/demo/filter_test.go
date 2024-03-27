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

package demo

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
)

func TestHello(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	info := &envoy.StreamInfo{}
	info.SetFilterState(envoy.NewFilterState(map[string]string{
		"guest_name": "Jack",
	}))
	cb.SetStreamInfo(info)
	f := factory(&config{
		Config: Config{
			HostName: "Tom",
		},
	}, cb).(*filter)
	assert.Equal(t, "hello, Jack", f.hello())
}

func TestDecodeHeaders(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	info := &envoy.StreamInfo{}
	info.SetFilterState(envoy.NewFilterState(map[string]string{
		"guest_name": "Jack",
	}))
	cb.SetStreamInfo(info)
	f := factory(&config{
		Config: Config{
			HostName: "Tom",
		},
	}, cb)
	hdr := envoy.NewRequestHeaderMap(http.Header{})
	f.DecodeHeaders(hdr, true)
	v, _ := hdr.Get("Tom")
	assert.Equal(t, "hello, Jack", v)
}
