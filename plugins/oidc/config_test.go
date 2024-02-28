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

package oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestBadIssuer(t *testing.T) {
	c := config{
		Config: Config{
			Issuer: "http://github.com",
		},
	}
	err := c.Init(nil)
	assert.Error(t, err)
}

func TestDefaultValue(t *testing.T) {
	c := config{
		Config: Config{
			Issuer: "http://github.com",
		},
	}
	// we set default value before communicating with the issuer
	c.Init(nil)
	assert.Equal(t, c.IdTokenHeader, "x-id-token")
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name:  "bad issuer url",
			input: `{"clientId":"a", "clientSecret":"b", "issuer":"google.com"}`,
			err:   "invalid Config.Issuer:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &config{}
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
