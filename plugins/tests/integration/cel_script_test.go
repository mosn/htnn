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

	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/plugins/tests/integration/controlplane"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
)

func TestCelScript(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, nil)
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		expect func(t *testing.T, resp *http.Response)
	}{
		{
			name: "allowIf",
			config: controlplane.NewSinglePluginConfig("celScript", map[string]interface{}{
				"allowIf": `request.path() == "/echo" && request.method() == "GET"`,
			}),
			expect: func(t *testing.T, resp *http.Response) {
				require.Equal(t, 200, resp.StatusCode)
			},
		},
		{
			name: "allowIf, reject",
			config: controlplane.NewSinglePluginConfig("celScript", map[string]interface{}{
				"allowIf": `request.path() == "/echo" && request.method() != "GET"`,
			}),
			expect: func(t *testing.T, resp *http.Response) {
				require.Equal(t, 403, resp.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(t, tt.config, dp)
			resp, err := dp.Get("/echo", nil)
			require.Nil(t, err)
			tt.expect(t, resp)
		})
	}
}
