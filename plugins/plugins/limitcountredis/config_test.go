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

package limitcountredis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name:  "rules are required",
			input: `{"address":"127.0.0.1:6379"}`,
			err:   "invalid Config.Rules",
		},
		{
			name:  "prefix is required",
			input: `{"address":"127.9.1.1:6379", "rules":[{"count":1,"timeWindow":"1s"}]}`,
			err:   "invalid Config.Prefix",
		},
		{
			name:  "invalid address",
			input: `{"address":"12::0:1", "prefix":"test", "rules":[{"count":1,"timeWindow":"1s"}]}`,
			err:   "bad address 12::0:1",
		},
		{
			name:  "address is required",
			input: `{"prefix":"test", "rules":[{"count":1, "timeWindow":"1s"}]}`,
			err:   "invalid Config.Source: value is required",
		},
		{
			name:  "validate rule",
			input: `{"address":"127.0.0.1:6479", "prefix":"test", "rules":[{"count":1}]}`,
			err:   "invalid Rule.TimeWindow",
		},
		{
			name:  "bad expr",
			input: `{"address":"127.0.0.1:6479", "prefix":"test", "rules":[{"count":1,"timeWindow":"1s","key":"request.headers"}]}`,
			err:   "unexpected failed resolution",
		},
		{
			name:  "passwd",
			input: `{"address":"127.0.0.1:6479", "prefix":"test", "rules":[{"count":1,"timeWindow":"1s"}], "username":"user"}`,
			err:   "password is required when username is set",
		},
		{
			name:  "pass",
			input: `{"address":"127.0.0.1:6479", "rules":[{"count":1,"timeWindow":"1s"}], "prefix":"test"}`,
		},
		{
			name:  "disable x-envoy-ratelimited header",
			input: `{"address":"127.0.0.1:6479", "rules":[{"count":1,"timeWindow":"1s"}], "prefix":"test", "disable_x_envoy_ratelimited_header": true}`,
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
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}
