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

package opa

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/plugins/tests/pkg/envoy"
)

func TestOpaRemote(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	cli := http.DefaultClient
	f := configFactory(&config{
		Config: Config{
			ConfigType: &Config_Remote{
				Remote: &Remote{
					Url:    "http://127.0.0.1:8181",
					Policy: "httpapi/authz",
				},
			},
		},
		client: cli,
	})(cb)
	hdr := envoy.NewRequestHeaderMap(http.Header(map[string][]string{
		":path": {"/?a=1&b&c=true&c=foo"},
		"pet":   {"cat"},
		"fruit": {"apple", "banana"},
	}))

	tests := []struct {
		name       string
		status     int
		checkInput func(input map[string]interface{})
		resp       string
		respErr    error
	}{
		{
			name: "happy path",
			resp: `{"result":{"allow":true}}`,
			checkInput: func(input map[string]interface{}) {
				assert.Equal(t, map[string]interface{}{
					"method": "GET",
					"scheme": "http",
					"host":   "localhost",
					"path":   "/",
					"query": map[string]interface{}{
						"a": []interface{}{"1"},
						"b": []interface{}{""},
						"c": []interface{}{"true", "foo"},
					},
					"headers": map[string]interface{}{
						"pet":   []interface{}{"cat"},
						"fruit": []interface{}{"apple", "banana"},
					},
				}, input["input"].(map[string]interface{})["request"])
			},
		},
		{
			name:   "reject",
			status: 403,
			resp:   `{"result":{"allow":false}}`,
		},
		{
			name:   "bad resp",
			status: 503,
			resp:   `{"result":{"`,
		},
		{
			name:    "bad resp2",
			status:  503,
			respErr: io.ErrUnexpectedEOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{}
			resp.Body = io.NopCloser(bytes.NewReader([]byte(tt.resp)))
			patches := gomonkey.ApplyMethodFunc(cli, "Post",
				func(url, contentType string, body io.Reader) (*http.Response, error) {
					if tt.checkInput != nil {
						input := map[string]interface{}{}
						data, _ := io.ReadAll(body)
						_ = json.Unmarshal(data, &input)
						tt.checkInput(input)
					}
					return resp, tt.respErr
				})
			defer patches.Reset()

			lr, ok := f.DecodeHeaders(hdr, true).(*api.LocalResponse)
			if !ok {
				assert.Equal(t, tt.status, 0)
			} else {
				assert.Equal(t, tt.status, lr.Code)
			}
		})
	}
}

func TestOpaLocal(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	hdr := envoy.NewRequestHeaderMap(http.Header(map[string][]string{
		":path": {"/?a=1&b&c=true&c=foo"},
		"fruit": {"apple", "banana"},
	}))

	tests := []struct {
		name   string
		status int
		text   string
	}{
		{
			name: "happy path",
			text: `default allow = true`,
		},
		{
			name: "check input",
			text: `import input.request
				import future.keywords
				default allow = false
				allow {
					request.method == "GET"
					request.path == "/"
					some "apple" in request.headers.fruit
					some "true" in request.query.c
				}`,
		},
		{
			name: "reject",
			text: `import input.request
				import future.keywords
				default allow = false
				allow {
					some true in request.query.c
				}`,
			status: 403,
		},
		{
			name:   "bad result",
			text:   `import input.request`,
			status: 503,
		},
		{
			name:   "no bool result",
			text:   `default allow = "a"`,
			status: 503,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{
				Config: Config{
					ConfigType: &Config_Local{
						Local: &Local{
							Text: "package test\n" + tt.text,
						},
					},
				},
			}
			err := c.Init(nil)
			require.NoError(t, err)
			f := configFactory(c)(cb)
			lr, ok := f.DecodeHeaders(hdr, true).(*api.LocalResponse)
			if !ok {
				assert.Equal(t, tt.status, 0)
			} else {
				assert.Equal(t, tt.status, lr.Code)
			}
		})
	}
}
