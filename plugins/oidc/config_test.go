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
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestBadIssuer(t *testing.T) {
	c := config{
		Config: Config{
			Issuer:  "http://1.1.1.1",
			Timeout: &durationpb.Duration{Seconds: 1}, // quick fail
		},
	}
	err := c.Init(nil)
	assert.Error(t, err)
}

func TestDefaultValue(t *testing.T) {
	c := config{
		Config: Config{
			Issuer:  "http://1.1.1.1",
			Timeout: &durationpb.Duration{Seconds: 1}, // quick fail
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
		{
			name:  "leeway can be 0s",
			input: `{"clientId":"a", "clientSecret":"b", "issuer":"https://google.com", "redirectUrl":"http://127.0.0.1:10000/echo", "accessTokenRefreshLeeway":"0s"}`,
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
