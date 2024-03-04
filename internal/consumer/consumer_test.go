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

package consumer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"mosn.io/htnn/pkg/filtermanager/model"
	_ "mosn.io/htnn/plugins/key_auth" // for test
	_ "mosn.io/htnn/plugins/opa"      // for test
)

func TestConsumerInitConfigs(t *testing.T) {
	var tests = []struct {
		name     string
		consumer Consumer
		err      string
	}{
		{
			name: "ok",
			consumer: Consumer{
				Auth: map[string]string{
					"keyAuth": `{"key": "test", "unknown_fields":"should be ignored"}`,
				},
				Filters: map[string]*model.FilterConfig{
					"opa": {
						Config: map[string]interface{}{
							"remote": map[string]interface{}{
								"url":            "http://opa:8181",
								"policy":         "t",
								"unknown_fields": "should be ignored",
							},
						},
					},
				},
			},
		},
		{
			name: "not consumer plugin",
			consumer: Consumer{
				Auth: map[string]string{
					"demo": "{\"key\": \"test\"}",
				},
			},
			err: "plugin demo is not for consumer",
		},
		{
			name: "failed to validate",
			consumer: Consumer{
				Auth: map[string]string{
					"keyAuth": "{\"key2\": \"test\"}",
				},
			},
			err: "invalid ConsumerConfig.Key",
		},
		{
			name: "unknown plugin",
			consumer: Consumer{
				Filters: map[string]*model.FilterConfig{
					"opax": {
						Config: []byte(""),
					},
				},
			},
			err: "plugin opax not found",
		},
		{
			name: "failed to validate filters",
			consumer: Consumer{
				Filters: map[string]*model.FilterConfig{
					"opa": {
						Config: []byte(""),
					},
				},
			},
			err: "during parsing plugin opa in consumer",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Consumer
			err := c.Unmarshal(tt.consumer.Marshal())
			require.NoError(t, err)

			err = c.InitConfigs()
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
