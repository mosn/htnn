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

package sentinel

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
			name:  "resource are required",
			input: `{}`,
			err:   "invalid Config.Resource: value is required",
		},
		{
			name:  "one of flow, hotSpot, circuitBreaker is required",
			input: `{"resource": {"from": "HEADER", "key": "test"}}`,
			err:   "config must have at least one of 'flow', 'hotSpot', 'circuitBreaker'",
		},
		{
			name:  "flow err: threshold must be greater than 0",
			input: `{"resource": {"from": "HEADER", "key": "test"}, "flow": {"rules": [{"resource": "flow"}]}}`,
			err:   "'threshold' must be greater than 0",
		},
		{
			name:  "flow ok: current resource",
			input: `{"resource": {"from": "HEADER", "key": "test"}, "flow": {"rules": [{"resource": "flow", "threshold": 10}]}}`,
			err:   "",
		},
		{
			name:  "flow ok: related resource",
			input: `{"resource": {"from": "HEADER", "key": "test"}, "flow": {"rules": [{"resource": "f2", "relationStrategy": "ASSOCIATED_RESOURCE", "refResource": "f1"}]}}`,
			err:   "",
		},
		{
			name:  "hot spot err: one of params, attachments is required",
			input: `{"resource": {"from": "HEADER", "key": "test"}, "hotSpot": {}}`,
			err:   "'params' and 'attachments' cannot both be empty",
		},
		{
			name:  "hot spot err: threshold must be greater than 0",
			input: `{"resource": {"from": "HEADER", "key": "test"}, "hotSpot": {"params": ["test"], "rules": [{"resource": "hs"}]}}`,
			err:   "invalid HotSpotRule.Threshold: value must be greater than 0",
		},
		{
			name:  "hot spot ok",
			input: `{"resource": {"from": "HEADER", "key": "test"}, "hotSpot": {"params": ["test"], "rules": [{"resource": "hs", "metricType": "QPS", "threshold": 10}]}}`,
			err:   "",
		},
		{
			name:  "circuit breaker err: threshold must be greater than 0",
			input: `{"resource": {"from": "HEADER", "key": "test"}, "circuitBreaker": {"rules": [{"resource": "cb"}]}}`,
			err:   "invalid CircuitBreakerRule.Threshold: value must be greater than 0",
		},
		{
			name:  "circuit breaker ok",
			input: `{"resource": {"from": "HEADER", "key": "test"}, "circuitBreaker": {"rules": [{"resource": "cb", "threshold": 10}]}}`,
			err:   "",
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
