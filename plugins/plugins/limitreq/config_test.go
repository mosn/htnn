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

package limitreq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		err      string
		maxDelay time.Duration
	}{
		{
			name:  "average is required",
			input: `{}`,
			err:   "invalid Config.Average: value must be greater than 0",
		},
		{
			name:  "invalid average",
			input: `{"average":0}`,
			err:   "invalid Config.Average: value must be greater than 0",
		},
		{
			name:  "invalid burst",
			input: `{"average":1,"burst":-1}`,
			err:   "invalid value for uint32 field",
		},
		{
			name:  "bad expr",
			input: `{"average":1,"key":"request.header"}`,
			err:   "unexpected failed resolution",
		},
		{
			name:     "pass",
			input:    `{"average":1}`,
			maxDelay: 500 * time.Millisecond,
		},
		{
			name:     "1min",
			input:    `{"average":30, "period":"60s"}`,
			maxDelay: 500 * time.Millisecond,
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

				err = conf.Init(nil)
				assert.Nil(t, err)
				assert.Equal(t, tt.maxDelay, conf.maxDelay)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}
