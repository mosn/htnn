package plugins

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/moe/pkg/filtermanager"
	"mosn.io/moe/tests/integration/plugins/control_plane"
	"mosn.io/moe/tests/integration/plugins/data_plane"
)

func TestOpa(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	waitServiceUp(t, ":8181",
		"OPA service is unavailble. Please run `docker-compose up opa` under ci/ and ensure it is started")

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		run    func(t *testing.T)
	}{
		{
			name: "happy path",
			config: control_plane.NewSinglePluinConfig("opa", map[string]interface{}{
				"remote": map[string]string{
					"url":    "http://opa:8181",
					"policy": "test",
				},
			}),
			run: func(t *testing.T) {
				resp, err := dp.Get("/echo", nil)
				assert.Nil(t, err)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Get("/x", nil)
				assert.Equal(t, 403, resp.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(tt.config)
			tt.run(t)
		})
	}
}
