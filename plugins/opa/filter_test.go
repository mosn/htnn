package opa

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"mosn.io/moe/tests/pkg/envoy"
)

func TestOpaRemote(t *testing.T) {
	cb := &envoy.FiterCallbackHandler{}
	cli := http.DefaultClient
	f := configFactory(&config{
		Config: &Config{
			ConfigType: &Config_Remote{
				Remote: &Remote{
					Url:    "http://127.0.0.1:8181",
					Policy: "httpapi/authz",
				},
			},
		},
		client: cli,
	})(cb)
	hdr := envoy.NewRequestHeaderMap(http.Header(map[string][]string{
		":path": {"/?a=1"},
	}))

	tests := []struct {
		name       string
		status     int
		checkInput func(input map[string]interface{})
		resp       string
		respErr    error
	}{
		{
			name: "happy path",
			resp: `{"result":{"allow":true}}`,
			checkInput: func(input map[string]interface{}) {
				assert.Equal(t, map[string]interface{}{
					"method":   "GET",
					"scheme":   "http",
					"host":     "localhost",
					"path":     "/",
					"query":    "a=1",
					"protocol": "HTTP/1.1",
				}, input["request"])
			},
		},
		{
			name:   "reject",
			status: 403,
			resp:   `{"result":{"allow":false}}`,
		},
		{
			name:   "bad resp",
			status: 503,
			resp:   `{"result":{"`,
		},
		{
			name:    "bad resp2",
			status:  503,
			respErr: io.ErrUnexpectedEOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{}
			resp.Body = io.NopCloser(bytes.NewReader([]byte(tt.resp)))
			patches := gomonkey.ApplyMethodFunc(cli, "Post",
				func(url, contentType string, body io.Reader) (*http.Response, error) {
					if tt.checkInput != nil {
						input := map[string]interface{}{}
						data, _ := io.ReadAll(body)
						_ = json.Unmarshal(data, &input)
						tt.checkInput(input)
					}
					return resp, tt.respErr
				})
			defer patches.Reset()

			f.DecodeHeaders(hdr, true)
			assert.Equal(t, tt.status, cb.LocalResponseCode())
		})
	}
}
