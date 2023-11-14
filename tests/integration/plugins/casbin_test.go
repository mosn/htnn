package plugins

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/moe/pkg/filtermanager"
	"mosn.io/moe/tests/integration/plugins/control_plane"
	"mosn.io/moe/tests/integration/plugins/data_plane"
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
	modelFile := writeTempFile(model)
	policy := `
p, *, /, POST
p, admin, *, POST
g, alice, admin
`
	policyFile := writeTempFile(policy)
	policy2 := `
p, *, /, POST
p, admin, *, POST
g, bob, admin
`
	policyFile2 := writeTempFile(policy2)
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
			controlPlane.UseGoPluginConfig(tt.config)
			hdr := http.Header{}
			hdr.Set("customer", "alice")
			resp, err := dp.Post("/echo", hdr, strings.NewReader("any"))
			assert.Nil(t, err)
			tt.expect(t, resp)
		})
	}
}
