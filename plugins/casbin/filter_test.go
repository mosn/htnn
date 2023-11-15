package casbin

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/moe/pkg/filtermanager/api"
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
			cb := envoy.NewFilterCallbackHandler()
			c := &config{
				Config: Config{
					Rule: &Config_Rule{
						Model:  "./testdata/model.conf",
						Policy: "./testdata/policy.csv",
					},
					Token: &Config_Token{
						Name: "user",
					},
				},
			}
			c.Init(nil)
			f := configFactory(c)(cb)
			hdr := envoy.NewRequestHeaderMap(tt.header)

			wg := sync.WaitGroup{}
			for i := 0; i < 3; i++ {
				wg.Add(1)
				go func() {
					// ensure the lock takes effect
					lr, ok := f.DecodeHeaders(hdr, true).(*api.LocalResponse)
					if !ok {
						assert.Equal(t, tt.status, 0)
					} else {
						assert.Equal(t, tt.status, lr.Code)
					}
					wg.Done()
				}()
			}
			wg.Wait()
		})
	}
}
