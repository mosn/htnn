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

package casbin

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/types/plugins/casbin"
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
				Config: casbin.Config{
					Rule: &casbin.Config_Rule{
						Model:  "./testdata/model.conf",
						Policy: "./testdata/policy.csv",
					},
					Token: &casbin.Config_Token{
						Name: "user",
					},
				},
			}
			c.Init(nil)
			f := factory(c, cb)
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
						assert.False(t, Changed)
					}
					wg.Done()
				}()
			}
			wg.Wait()
		})
	}
}
