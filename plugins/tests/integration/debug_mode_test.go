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
	"time"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/integration/controlplane"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
)

func TestDebugModeSlowLog(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		NoErrorLogCheck: true,
		ExpectLogPattern: []string{
			`slow log report:.+"executed_plugins":\[.+"name":"limitReq","per_phase_cost_seconds":\{"DecodeHeaders":.+`,
		},
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	config := controlplane.NewPluginConfig([]*model.FilterConfig{
		{
			Name: "debugMode",
			Config: map[string]interface{}{
				"slowLog": map[string]interface{}{
					"threshold": "0.02s",
				},
			},
		},
		{
			Name: "limitReq",
			Config: map[string]interface{}{
				"average": 1,
				"period":  "0.1s",
			},
		},
	})
	controlPlane.UseGoPluginConfig(t, config, dp)

	resp, _ := dp.Head("/echo", nil)
	assert.Equal(t, 200, resp.StatusCode)

	time.Sleep(50 * time.Millisecond) // trigger delay
	now := time.Now()
	resp, _ = dp.Head("/echo", nil)
	pass := time.Since(now)
	assert.Equal(t, 200, resp.StatusCode)
	// delay time plus the req time
	assert.True(t, pass < 60*time.Millisecond, pass)
}

func TestDebugModeSlowLogNoPlugin(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		NoErrorLogCheck: true,
		ExpectNoLogPattern: []string{
			`slow log report:.+"executed_plugins":\[.+"name":"limitReq","per_phase_cost_seconds":\{"DecodeHeaders":.+`,
		},
		ExpectLogPattern: []string{
			`slow log report:.+"server":\["envoy"\]`,
		},
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	config := controlplane.NewPluginConfig([]*model.FilterConfig{
		{
			Name: "debugMode",
			Config: map[string]interface{}{
				"slowLog": map[string]interface{}{
					"threshold": "0.0001s",
				},
			},
		},
	})
	controlPlane.UseGoPluginConfig(t, config, dp)
	resp, _ := dp.Head("/echo", nil)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDebugModeSlowLogNoEncodeHeaders(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		LogLevel:        "debug",
		NoErrorLogCheck: true,
		ExpectNoLogPattern: []string{
			`slow log report:.+"server":\["envoy"\]`,
		},
		ExpectLogPattern: []string{
			`slow log report:.+"headers":\{":authority"`,
		},
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	config := controlplane.NewPluginConfig([]*model.FilterConfig{
		{
			Name: "debugMode",
			Config: map[string]interface{}{
				"slowLog": map[string]interface{}{
					"threshold": "0.0002s",
				},
			},
		},
		{
			Name: "limitReq",
			Config: map[string]interface{}{
				"average": 1,
				"period":  "0.1s",
			},
		},
	})
	controlPlane.UseGoPluginConfig(t, config, dp)
	err = dp.SendAndCancelRequest("/echo", 1*time.Millisecond)
	assert.Nil(t, err)
}

func TestDebugModeSlowLogNotEmit(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{})
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
			name: "not emit",
			config: controlplane.NewPluginConfig([]*model.FilterConfig{
				{
					Name: "debugMode",
					Config: map[string]interface{}{
						"slowLog": map[string]interface{}{
							"threshold": "0.1s",
						},
					},
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Head("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
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

func TestDebugModeSlowLogWithFiltersFromConsumer(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		Bootstrap: dataplane.Bootstrap().AddConsumer("rick", map[string]interface{}{
			"auth": map[string]interface{}{
				"keyAuth": `{"key":"rick"}`,
			},
			"filters": map[string]interface{}{
				"limitReq": map[string]interface{}{
					"config": `{"average": 1, "period": "0.1s"}`,
				},
			},
		}),
		LogLevel:        "debug",
		NoErrorLogCheck: true,
		ExpectLogPattern: []string{
			`slow log report:.+"executed_plugins":\[.+"name":"limitReq","per_phase_cost_seconds":\{"DecodeHeaders":.+`,
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
			name: "sanity",
			config: controlplane.NewPluginConfig([]*model.FilterConfig{
				{
					Name: "debugMode",
					Config: map[string]interface{}{
						"slowLog": map[string]interface{}{
							"threshold": "0.02s",
						},
					},
				},
				{
					Name: "keyAuth",
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
				hdr := http.Header{"Authorization": []string{"rick"}}
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)

				time.Sleep(50 * time.Millisecond) // trigger delay
				now := time.Now()
				resp, _ = dp.Head("/echo", hdr)
				pass := time.Since(now)
				assert.Equal(t, 200, resp.StatusCode)
				// delay time plus the req time
				assert.True(t, pass < 60*time.Millisecond, pass)
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
