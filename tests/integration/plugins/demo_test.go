package plugins

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/moe/tests/integration/plugins/data_plane"
)

func TestDemo(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t)
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	tests := []struct {
		name   string
		config map[string]interface{}
		expect func(t *testing.T, resp *http.Response)
	}{
		{
			name: "happy path",
			config: map[string]interface{}{
				"host_name": "Tom",
			},
			expect: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, "hello,", resp.Header.Get("Resp-Tom"), resp)
			},
		},
		{
			name: "change config",
			config: map[string]interface{}{
				"host_name": "Mike",
			},

			expect: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, "hello,", resp.Header.Get("Resp-Mike"), resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(tt.config)
			resp, err := dp.Get("/echo", nil)
			assert.Nil(t, err)
			tt.expect(t, resp)
		})
	}
}
