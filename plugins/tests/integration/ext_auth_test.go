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
	_ "embed"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/pkg/filtermanager"
	"mosn.io/htnn/plugins/tests/integration/control_plane"
	"mosn.io/htnn/plugins/tests/integration/data_plane"
)

var (
	//go:embed ext_auth_route.yml
	extAuthRoute string
)

func TestExtAuth(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		Bootstrap: data_plane.Bootstrap().AddBackendRoute(extAuthRoute),
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
			name: "default",
			config: control_plane.NewSinglePluinConfig("ext_auth", map[string]interface{}{
				"http_service": map[string]interface{}{
					"url": "http://127.0.0.1:10001/ext_auth",
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Set("Authorization", "Basic amFjazIwMjE6MTIzNDU2")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Post("/echo", hdr, strings.NewReader("any"))
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Post("/echo", nil, strings.NewReader("any"))
				assert.Equal(t, 403, resp.StatusCode)
				assert.Equal(t, "not matched", resp.Header.Get("reason"))
			},
		},
		{
			name: "failed to ext auth",
			config: control_plane.NewSinglePluinConfig("ext_auth", map[string]interface{}{
				"http_service": map[string]interface{}{
					"url":             "http://127.0.0.1:2023/ext_auth",
					"status_on_error": 401,
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Post("/echo", nil, strings.NewReader("any"))
				assert.Equal(t, 401, resp.StatusCode)
			},
		},
		{
			name: "with body",
			config: control_plane.NewSinglePluinConfig("ext_auth", map[string]interface{}{
				"http_service": map[string]interface{}{
					"url":               "http://127.0.0.1:10001/ext_auth",
					"with_request_body": true,
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				body := strings.NewReader("any")
				resp, _ := dp.Post("/echo", hdr, body)
				assert.Equal(t, 403, resp.StatusCode)
				assert.Equal(t, "any", resp.Header.Get("body"))
				emptyBody := strings.NewReader("")
				resp, _ = dp.Post("/echo", hdr, emptyBody)
				assert.Equal(t, 403, resp.StatusCode)
				assert.Equal(t, "", resp.Header.Get("body"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(tt.config, dp)
			tt.run(t)
		})
	}
}
