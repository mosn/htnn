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

package hmac_auth

import (
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/htnn/pkg/consumer"
	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/plugins/tests/pkg/envoy"
)

func TestHmacAuth(t *testing.T) {
	tests := []struct {
		name     string
		conf     string
		consumer *consumer.Consumer
		hdr      map[string][]string
		status   int
	}{
		{
			name: "default",
			hdr: map[string][]string{
				SignatureHeader: {"1Qx+PybdlxxfRYu5uZXSLSN1C9y5UgE9YkXBhn97FKo="},
				DateHeader:      {"Fri Jan  5 16:10:54 CST 2024"},
				"extra":         {"2", "1"},
			},
			consumer: &consumer.Consumer{
				ConsumerConfigs: map[string]api.PluginConsumerConfig{
					Name: &ConsumerConfig{
						AccessKey: "ak",
						SecretKey: "sk",
						SignedHeaders: []string{
							"extra",
						},
					},
				},
			},
		},
		{
			name:   "consumer not found",
			status: 401,
		},
		{
			name: "sha384",
			hdr: map[string][]string{
				SignatureHeader: {"3QV0rnURMgHkIg6jGJRIgMueAlWMjKnbVX6HhUOw1KtBxbmpe0kyTH/uhxUvaBzb"},
				DateHeader:      {"Fri Jan  5 16:10:54 CST 2024"},
			},
			consumer: &consumer.Consumer{
				ConsumerConfigs: map[string]api.PluginConsumerConfig{
					Name: &ConsumerConfig{
						AccessKey: "ak",
						SecretKey: "sk",
						Algorithm: Algorithm_HMAC_SHA384,
					},
				},
			},
		},
		{
			name: "sha512",
			hdr: map[string][]string{
				SignatureHeader: {"K8cPdrqqcMGkpDP3Oz/6LSOwPUsIS1vMhOgwEh3OPy3Gi1IshAZ38jukAnUR66NWo1/Ela20P7/Bgp/JE7ltKA=="},
				DateHeader:      {"Fri Jan  5 16:10:54 CST 2024"},
			},
			consumer: &consumer.Consumer{
				ConsumerConfigs: map[string]api.PluginConsumerConfig{
					Name: &ConsumerConfig{
						AccessKey: "ak",
						SecretKey: "sk",
						Algorithm: Algorithm_HMAC_SHA512,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := envoy.NewFilterCallbackHandler()
			conf := &Config{}
			if tt.conf != "" {
				protojson.Unmarshal([]byte(tt.conf), conf)
			}
			f := factory(conf, cb)
			defaultHdr := map[string][]string{
				":authority": {"test.local"},
				":method":    {"GET"},
				":path":      {"/echo"},
			}
			httpHdr := http.Header(defaultHdr)
			httpHdr.Set(AccessKeyHeader, "ak")
			for k, v := range tt.hdr {
				for _, vv := range v {
					httpHdr.Add(k, vv)
				}
			}

			if tt.consumer != nil {
				patches := gomonkey.ApplyMethodReturn(cb, "LookupConsumer", tt.consumer, true)
				defer patches.Reset()
			}

			hdr := envoy.NewRequestHeaderMap(httpHdr)
			res := f.DecodeHeaders(hdr, true)
			if tt.status != 0 {
				r, ok := res.(*api.LocalResponse)
				require.True(t, ok)
				assert.Equal(t, tt.status, r.Code)
			} else {
				assert.Equal(t, api.Continue, res)
			}
		})
	}
}
