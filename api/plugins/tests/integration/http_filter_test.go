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
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/integration/control_plane"
	"mosn.io/htnn/api/plugins/tests/integration/data_plane"
)

func TestHTTPFilterPlugin(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		Bootstrap: data_plane.Bootstrap().SetHTTPFilterGolang(map[string]interface{}{
			"plugins": []interface{}{
				map[string]interface{}{
					"name": "buffer",
					"config": map[string]interface{}{
						"decode": true,
						"need":   true,
					},
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
			name: "run golang filter from HTTP filter",
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, []string{"buffer"}, resp.Header.Values("Echo-Run"))
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

func TestHTTPFilterMergeIntoRoute(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		LogLevel: "debug",
		Bootstrap: data_plane.Bootstrap().SetHTTPFilterGolang(map[string]interface{}{
			"plugins": []interface{}{
				map[string]interface{}{
					"name": "buffer",
					"config": map[string]interface{}{
						"decode": true,
						"need":   false,
					},
				},
				map[string]interface{}{
					"name":   "init",
					"config": map[string]interface{}{},
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
			name: "merge golang filter from HTTP filter",
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, []string{"no buffer"}, resp.Header.Values("Echo-Run"))
			},
		},
		{
			name: "init should be called only once",
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "1", resp.Header.Get("Echo-InitCounter"))
				resp, _ = dp.Get("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "1", resp.Header.Get("Echo-InitCounter"))
			},
		},
		{
			name:   "init should be called only once (route version)",
			config: control_plane.NewSinglePluinConfig("init", map[string]interface{}{}),
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "1", resp.Header.Get("Echo-InitCounter"))
				resp, _ = dp.Get("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "1", resp.Header.Get("Echo-InitCounter"))
			},
		},
		{
			name: "override",
			config: control_plane.NewSinglePluinConfig("buffer", map[string]interface{}{
				"decode": true,
				"need":   true,
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, []string{"buffer"}, resp.Header.Values("Echo-Run"))
			},
		},
		{
			name: "sort merged plugins",
			config: control_plane.NewPluinConfig([]*model.FilterConfig{
				{
					Name:   "stream",
					Config: map[string]interface{}{},
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, []string{"no buffer", "stream"}, resp.Header.Values("Echo-Run"))
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
