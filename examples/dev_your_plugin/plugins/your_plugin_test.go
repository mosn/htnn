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

package plugins

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name:  "config can be empty",
			input: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &Config{}
			err := protojson.Unmarshal([]byte(tt.input), conf)
			if err == nil {
				err = conf.Validate()
			}
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}

func TestDecodeHeaders(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	f := factory(&Config{}, cb)
	hdr := envoy.NewRequestHeaderMap(http.Header{})
	res := f.DecodeHeaders(hdr, true)
	reply := res.(*api.LocalResponse)
	assert.Equal(t, 200, reply.Code)
	assert.Equal(t, "text/plain", reply.Header.Get("content-type"))
}
