package integration

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/moe/pkg/filtermanager"
	"mosn.io/moe/plugins/tests/integration/control_plane"
	"mosn.io/moe/plugins/tests/integration/data_plane"
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
		run    func(t *testing.T)
	}{
		{
			name: "default",
			config: control_plane.NewSinglePluinConfig("ext_auth", map[string]interface{}{
				"http_service": map[string]interface{}{
					"url": "http://127.0.0.1:10001/ext_auth",
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Set("Authorization", "Basic amFjazIwMjE6MTIzNDU2")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Post("/echo", hdr, strings.NewReader("any"))
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Post("/echo", nil, strings.NewReader("any"))
				assert.Equal(t, 403, resp.StatusCode)
				assert.Equal(t, "not matched", resp.Header.Get("reason"))
			},
		},
		{
			name: "failed to ext auth",
			config: control_plane.NewSinglePluinConfig("ext_auth", map[string]interface{}{
				"http_service": map[string]interface{}{
					"url":             "http://127.0.0.1:2023/ext_auth",
					"status_on_error": 401,
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Post("/echo", nil, strings.NewReader("any"))
				assert.Equal(t, 401, resp.StatusCode)
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
