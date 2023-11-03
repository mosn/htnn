package casbin

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/moe/tests/pkg/envoy"
)

func TestCasbin(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		status int
	}{
		{
			name:   "pass",
			header: http.Header{"User": []string{"alice"}, ":path": []string{"/"}},
		},
		{
			name:   "pass, any path",
			header: http.Header{"User": []string{"alice"}, ":path": []string{"/other"}},
		},
		{
			name:   "token not found",
			header: http.Header{":path": []string{"/"}},
		},
		{
			name:   "token not found, any path",
			header: http.Header{":path": []string{"/other"}},
			status: 403,
		},
		{
			name:   "normal user",
			header: http.Header{"User": []string{"bob"}, ":path": []string{"/"}},
		},
		{
			name:   "normal user, any path",
			header: http.Header{"User": []string{"bob"}, ":path": []string{"/other"}},
			status: 403,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := &envoy.FiterCallbackHandler{}
			p := &parser{}
			cfg := &Config{
				Rule: &Config_Rule{
					Model:  "./testdata/model.conf",
					Policy: "./testdata/policy.csv",
				},
				Token: &Config_Token{
					Name: "user",
				},
			}
			config, err := p.Handle(cfg, nil)
			assert.Nil(t, err)
			f := configFactory(config)(cb)
			hdr := envoy.NewRequestHeaderMap(tt.header)
			f.DecodeHeaders(hdr, true)
			assert.Equal(t, tt.status, cb.LocalResponseCode())
		})
	}
}
