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
	data_plane2 "mosn.io/htnn/api/plugins/tests/integration/data_plane"
)

func TestConsumerWithFilter(t *testing.T) {
	dp, err := data_plane2.StartDataPlane(t, &data_plane2.Option{
		Bootstrap: data_plane2.Bootstrap().AddConsumer("with_filter", map[string]interface{}{
			"auth": map[string]interface{}{
				"keyAuth": `{"key":"marvin"}`,
			},
			"filters": map[string]interface{}{
				"demo": map[string]interface{}{
					"config": `{"hostName": "Mike"}`,
				},
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
			name: "authn & exec",
			config: control_plane.NewSinglePluinConfig("keyAuth", map[string]interface{}{
				"keys": []interface{}{
					map[string]interface{}{
						"name": "Authorization",
					},
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", http.Header{"Authorization": []string{"marvin"}})
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "hello,", resp.Header.Get("Echo-Mike"), resp)
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
