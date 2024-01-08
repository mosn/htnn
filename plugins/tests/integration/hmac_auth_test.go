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

	"mosn.io/htnn/pkg/filtermanager"
	"mosn.io/htnn/plugins/tests/integration/control_plane"
	"mosn.io/htnn/plugins/tests/integration/data_plane"
)

func TestHmacAuth(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, nil)
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
			name: "sanity",
			config: control_plane.NewSinglePluinConfig("hmac_auth", map[string]interface{}{
				"signature_header":  "x-sign-hdr",
				"access_key_header": "x-ak",
				"date_header":       "x-date",
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Set("x-sign-hdr", "E6m5y84WIu/XeeIox2VZes/+xd/8QPRSMKqo+lp3cAo=")
				hdr.Set("x-ak", "ak")
				hdr.Set("x-date", "Fri Jan  5 16:10:54 CST 2024")
				hdr.Set("x-custom-a", "test")
				resp, _ := dp.Get("/echo?age=36&address=&title=ops&title=dev", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Get("/echo?age=36&title=ops&title=dev", hdr)
				assert.Equal(t, 401, resp.StatusCode)
			},
		},
		{
			name: "bypass if no credential",
			config: control_plane.NewPluinConfig([]*filtermanager.FilterConfig{
				{
					Name: "hmac_auth",
					Config: map[string]interface{}{
						"signature_header":  "x-sign-hdr",
						"access_key_header": "x-ak",
					},
				},
				{
					Name: "key_auth",
					Config: map[string]interface{}{
						"keys": []interface{}{
							map[string]interface{}{
								"name": "Authorization",
							},
						},
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("Authorization", "rick")
				resp, _ := dp.Get("/echo?age=36&address=&title=ops&title=dev", hdr)
				assert.Equal(t, 200, resp.StatusCode)
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
