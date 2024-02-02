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

package opa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestBadConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name:  "at least one config type is required",
			input: `{}`,
			err:   "value is required",
		},
		{
			name: "empty url in remote",
			input: `{
				"remote": {
					"url": "",
					"policy": "authz"
				}
			}`,
			err: "invalid Remote.Url: value must be absolute",
		},
		{
			name: "empty policy in remote",
			input: `{
				"remote": {
					"url": "http://127.0.0.1:8181",
					"policy": ""
				}
			}`,
			err: "invalid Remote.Policy: value length must be at least 1 runes",
		},
		{
			name: "bad url in remote",
			input: `{
				"remote": {
					"url": "127.0.0.1:8181",
					"policy": "test"
				}
			}`,
			err: "invalid Remote.Url: value must be a valid URI",
		},
		{
			name: "empty text in local",
			input: `{
				"local": {
					"text": ""
				}
			}`,
			err: "invalid Local.Text: value length must be at least 1 runes",
		},
		{
			name: "bad text in local",
			input: `{
				"local": {
					"text": "package a/b"
				}
			}`,
			err: "invalid Local.Text: bad package name",
		},
		{
			name: "bad rego syntax",
			input: `{
				"local": {
					"text": "package ab\nimport"
				}
			}`,
			err: "rego_parse_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &config{}
			protojson.Unmarshal([]byte(tt.input), conf)
			err := conf.Validate()
			assert.NotNil(t, err)
			assert.ErrorContains(t, err, tt.err)
		})
	}
}
