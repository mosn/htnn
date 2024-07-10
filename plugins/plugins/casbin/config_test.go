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
			name:  "no required fields",
			input: `{}`,
			err:   "Rule: value is required",
		},
		{
			name: "empty policy",
			input: `{
				"rule": {
					"model": "./config/model.conf",
					"policy": ""
				},
				"token": {
					"source": "HEADER",
					"name": "role"
				}
			}`,
			err: "Policy: value length must be at least 1 runes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &config{}
			err := protojson.Unmarshal([]byte(tt.input), conf)
			if err == nil {
				err = conf.Validate()
			}
			assert.NotNil(t, err)
			assert.ErrorContains(t, err, tt.err)
		})
	}
}

func TestChanged(t *testing.T) {
	setChanged(true)
	assert.True(t, getChanged())
}
