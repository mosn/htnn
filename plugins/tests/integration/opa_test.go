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
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/plugins/tests/integration/controlplane"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
	"mosn.io/htnn/api/plugins/tests/integration/helper"
)

func TestOpa(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	helper.WaitServiceUp(t, ":8181", "opa")

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		run    func(t *testing.T)
	}{
		{
			name: "happy path",
			config: controlplane.NewSinglePluinConfig("opa", map[string]interface{}{
				"remote": map[string]string{
					"url":    "http://opa:8181",
					"policy": "test",
				},
			}),
			run: func(t *testing.T) {
				resp, err := dp.Get("/echo", nil)
				require.Nil(t, err)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Get("/x", nil)
				assert.Equal(t, 403, resp.StatusCode)
			},
		},
		{
			name: "local",
			config: controlplane.NewSinglePluinConfig("opa", map[string]interface{}{
				"local": map[string]string{
					"text": `package test
						import input.request
						default allow = false
						allow {
							request.method == "GET"
							startswith(request.path, "/echo")
						}`,
				},
			}),
			run: func(t *testing.T) {
				resp, err := dp.Get("/echo", nil)
				require.Nil(t, err)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Get("/x", nil)
				assert.Equal(t, 403, resp.StatusCode)
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
