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

	cmModel "mosn.io/htnn/api/pkg/consumer/model"
	fmModel "mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/pkg/plugins"
)

func TestConsumerInitConfigs(t *testing.T) {
	plugins.RegisterPlugin("consumerPluginX", &consumerPlugin{})
	plugins.RegisterPlugin("filterPlugin", &filterPlugin{})

	var tests = []struct {
		name     string
		consumer cmModel.Consumer
		err      string
	}{
		{
			name: "ok",
			consumer: cmModel.Consumer{
				Auth: map[string]string{
					"consumerPluginX": `{"key": "test", "unknown_fields":"should be ignored"}`,
				},
				Filters: map[string]*fmModel.FilterConfig{
					"filterPlugin": {
						Config: map[string]interface{}{
							"url":            "http://opa:8181",
							"unknown_fields": "should be ignored",
						},
					},
				},
			},
		},
		{
			name: "not consumer plugin",
			consumer: cmModel.Consumer{
				Auth: map[string]string{
					"demo": "{\"key\": \"test\"}",
				},
			},
			err: "plugin demo is not for consumer",
		},
		{
			name: "failed to validate",
			consumer: cmModel.Consumer{
				Auth: map[string]string{
					"consumerPluginX": "{\"key2\": \"test\"}",
				},
			},
			err: "invalid ConsumerConfig.Key",
		},
		{
			name: "unknown plugin",
			consumer: cmModel.Consumer{
				Filters: map[string]*fmModel.FilterConfig{
					"opax": {
						Config: []byte(""),
					},
				},
			},
			err: "plugin opax not found",
		},
		{
			name: "failed to validate filters",
			consumer: cmModel.Consumer{
				Filters: map[string]*fmModel.FilterConfig{
					"filterPlugin": {
						Config: []byte(""),
					},
				},
			},
			err: "during parsing plugin filterPlugin in consumer",
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
