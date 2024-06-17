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
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/integration/control_plane"
	"mosn.io/htnn/api/plugins/tests/integration/data_plane"
)

func TestConsumerWithFilter(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		Bootstrap: data_plane.Bootstrap().AddConsumer("marvin", map[string]interface{}{
			"auth": map[string]interface{}{
				"consumer": `{"name":"marvin"}`,
			},
			"filters": map[string]interface{}{
				"localReply": map[string]interface{}{
					"config": `{"decode": true, "headers": true}`,
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
			name:   "authn & exec",
			config: control_plane.NewSinglePluinConfig("consumer", map[string]interface{}{}),
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", http.Header{"Authorization": []string{"marvin"}})
				assert.Equal(t, 206, resp.StatusCode)
				b, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, "{\"msg\":\"ok\"}", string(b))
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

func TestConsumerWithFilterInitFailed(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		Bootstrap: data_plane.Bootstrap().AddConsumer("marvin", map[string]interface{}{
			"auth": map[string]interface{}{
				"consumer": `{"name":"marvin"}`,
			},
			"filters": map[string]interface{}{
				"bad": map[string]interface{}{
					"config": `{"errorInInit": true}`,
				},
			},
		}),
		NoErrorLogCheck: true,
		ExpectLogPattern: []string{
			`error in plugin bad: `,
		},
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
			name:   "authn & exec",
			config: control_plane.NewSinglePluinConfig("consumer", map[string]interface{}{}),
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", http.Header{"Authorization": []string{"marvin"}})
				assert.Equal(t, 500, resp.StatusCode)
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

func TestConsumerWithFilterAndMergeFromHTTPFilter(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		LogLevel: "debug",
		Bootstrap: data_plane.Bootstrap().AddConsumer("marvin", map[string]interface{}{
			"auth": map[string]interface{}{
				"consumer": `{"name":"marvin"}`,
			},
			"filters": map[string]interface{}{
				"localReply": map[string]interface{}{
					"config": `{"decode": true, "headers": true}`,
				},
			},
		}).SetHTTPFilterGolang(map[string]interface{}{
			"plugins": []interface{}{
				map[string]interface{}{
					"name": "buffer",
					"config": map[string]interface{}{
						"decode": true,
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
			name:   "authn & exec",
			config: control_plane.NewSinglePluinConfig("consumer", map[string]interface{}{}),
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", http.Header{"Authorization": []string{"marvin"}})
				assert.Equal(t, 206, resp.StatusCode)
				assert.Equal(t, []string{"no buffer"}, resp.Header.Values("Run"))
				b, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, "{\"msg\":\"ok\"}", string(b))

				resp, _ = dp.Get("/echo", http.Header{"Authorization": []string{"marvin"}})
				assert.Equal(t, 206, resp.StatusCode)
				assert.Equal(t, []string{"no buffer"}, resp.Header.Values("Run"))
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

func TestConsumerFilterNotAfterConsumerRunInLaterPhase(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		Bootstrap: data_plane.Bootstrap().AddConsumer("marvin", map[string]interface{}{
			"auth": map[string]interface{}{
				"consumer": `{"name":"marvin"}`,
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
			config: control_plane.NewPluinConfig([]*model.FilterConfig{
				{
					Name:   "beforeConsumerAndHasOtherMethod",
					Config: map[string]interface{}{},
				},
				{
					Name:   "consumer",
					Config: map[string]interface{}{},
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Get("/echo", http.Header{"Authorization": []string{"marvin"}})
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "beforeConsumerAndHasOtherMethod", resp.Header.Get("Echo-Run"))
				assert.Equal(t, "beforeConsumerAndHasOtherMethod", resp.Header.Get("Run"))
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
