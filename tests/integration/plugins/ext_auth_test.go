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

func TestExtAuth(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, nil)
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
	}{
		{
			name: "default",
			config: control_plane.NewSinglePluinConfig("ext_auth", map[string]interface{}{
				"http_service": map[string]interface{}{
					"url": "http://127.0.0.1:10001/ext_auth",
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(tt.config)
			hdr := http.Header{}
			hdr.Set("Authorization", "Basic amFjazIwMjE6MTIzNDU2")
			resp, _ := dp.Post("/echo", hdr, strings.NewReader("any"))
			assert.Equal(t, 200, resp.StatusCode)
			resp, _ = dp.Post("/echo", nil, strings.NewReader("any"))
			assert.Equal(t, 403, resp.StatusCode)
		})
	}
}
