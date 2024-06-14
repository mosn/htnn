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

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/plugins/tests/integration/control_plane"
	"mosn.io/htnn/api/plugins/tests/integration/data_plane"
)

func TestKeyAuth(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		Bootstrap: data_plane.Bootstrap().AddConsumer("rick", map[string]interface{}{
			"auth": map[string]interface{}{
				"keyAuth":  `{"key":"rick"}`,
				"hmacAuth": `{"accessKey":"ak","secretKey":"sk","signedHeaders":["x-custom-a"],"algorithm":"HMAC_SHA256"}`,
			},
		}).AddConsumer("tom", map[string]interface{}{
			"auth": map[string]interface{}{
				"keyAuth": `{"key":"tom"}`,
			},
		}),
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		run    func(t *testing.T)
	}{
		{
			name: "key in the header",
			config: control_plane.NewSinglePluinConfig("keyAuth", map[string]interface{}{
				"keys": []interface{}{
					map[string]interface{}{
						"name": "Authorization",
					},
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", http.Header{"Authorization": []string{"rick"}})
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, 0, len(resp.Header.Values("Echo-Authorization")))
				resp, _ = dp.Get("/echo", http.Header{"Authorization": []string{"morty"}})
				assert.Equal(t, 401, resp.StatusCode)
				resp, _ = dp.Get("/echo", nil)
				assert.Equal(t, 401, resp.StatusCode)
				resp, _ = dp.Get("/echo", http.Header{"Authorization": []string{"rick", "morty"}})
				assert.Equal(t, 401, resp.StatusCode)
			},
		},
		{
			name: "key in the query",
			config: control_plane.NewSinglePluinConfig("keyAuth", map[string]interface{}{
				"keys": []interface{}{
					map[string]interface{}{
						"name":   "Authorization",
						"source": "HEADER",
					},
					map[string]interface{}{
						"name":   "ak",
						"source": "QUERY",
					},
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo?ak=rick&other=Key", nil)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "/echo?other=Key", resp.Header.Get("Echo-Path"))
				resp, _ = dp.Get("/echo?ak=morty", nil)
				assert.Equal(t, 401, resp.StatusCode)
				resp, _ = dp.Get("/echo", nil)
				assert.Equal(t, 401, resp.StatusCode)
				resp, _ = dp.Get("/echo?ak=rick&ak=morty", nil)
				assert.Equal(t, 401, resp.StatusCode)
				resp, _ = dp.Get("/echo?ak=rick", http.Header{"Authorization": []string{"morty"}})
				assert.Equal(t, 401, resp.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(t, tt.config, dp)
			tt.run(t)
		})
	}
}
