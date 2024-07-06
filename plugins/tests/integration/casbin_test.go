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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/plugins/tests/integration/control_plane"
	"mosn.io/htnn/api/plugins/tests/integration/data_plane"
	"mosn.io/htnn/api/plugins/tests/integration/helper"
)

func TestCasbin(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, nil)
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	model := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (g(r.sub, p.sub) || keyMatch(r.sub, p.sub)) && keyMatch(r.obj, p.obj) && keyMatch(r.act, p.act)
`
	modelFile := helper.WriteTempFile(model)
	policy := `
p, *, /, POST
p, admin, *, POST
g, alice, admin
`
	policyFile := helper.WriteTempFile(policy)
	policy2 := `
p, *, /, POST
p, admin, *, POST
g, bob, admin
`
	policyFile2 := helper.WriteTempFile(policy2)
	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		expect func(t *testing.T, resp *http.Response)
	}{
		{
			name: "happy path",
			config: control_plane.NewSinglePluinConfig("casbin", map[string]interface{}{
				"rule": map[string]string{
					"model":  modelFile.Name(),
					"policy": policyFile.Name(),
				},
				"token": map[string]string{
					"name": "customer",
				},
			}),
			expect: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, 200, resp.StatusCode)
			},
		},
		{
			name: "change config",
			config: control_plane.NewSinglePluinConfig("casbin", map[string]interface{}{
				"rule": map[string]string{
					"model":  modelFile.Name(),
					"policy": policyFile2.Name(),
				},
				"token": map[string]string{
					"name": "customer",
				},
			}),
			expect: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, 403, resp.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(t, tt.config, dp)
			hdr := http.Header{}
			hdr.Set("customer", "alice")
			resp, err := dp.Post("/echo", hdr, strings.NewReader("any"))
			require.Nil(t, err)
			tt.expect(t, resp)
		})
	}

	// configuration is not changed, but file changed
	err = os.WriteFile(policyFile2.Name(), []byte(policy), 0755)
	require.Nil(t, err)

	hdr := http.Header{}
	hdr.Set("customer", "alice")

	assert.Eventually(t, func() bool {
		resp, _ := dp.Post("/echo", hdr, strings.NewReader("any"))
		return resp != nil && resp.StatusCode == 200
	}, 10*time.Second, 1*time.Second)
}
