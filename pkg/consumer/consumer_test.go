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
	"google.golang.org/protobuf/types/known/structpb"

	"mosn.io/htnn/pkg/filtermanager/model"
	_ "mosn.io/htnn/plugins/key_auth" // for test
	_ "mosn.io/htnn/plugins/opa"      // for test
)

type consumerTest struct {
	values map[string]interface{}
}

func newConsumerTest() *consumerTest {
	return &consumerTest{
		values: make(map[string]interface{}),
	}
}

func (c *consumerTest) Add(ns string, consumer *Consumer) *consumerTest {
	if c.values[ns] == nil {
		c.values[ns] = make(map[string]interface{})
	}
	idx := c.values[ns].(map[string]interface{})
	idx[consumer.name] = map[string]interface{}{
		"d": consumer.Marshal(),
		"v": consumer.generation,
	}
	return c
}

func (c *consumerTest) Build() *structpb.Struct {
	st, _ := structpb.NewStruct(c.values)
	return st
}

func TestUpdateConsumer(t *testing.T) {
	// clean index
	resourceIndex = make(map[string]map[string]*Consumer)

	auth := map[string]string{
		"key_auth": "{\"key\": \"test\"}",
	}
	c := &Consumer{
		name:       "me",
		generation: 1,
		Auth:       auth,
	}
	v := newConsumerTest().Add("ns", c).Build()
	updateConsumers(v)

	r, _ := LookupConsumer("ns", "key_auth", "test")
	require.NotNil(t, r)
	require.Equal(t, "me", r.Name())

	r, _ = LookupConsumer("ns", "key_auth", "not_found")
	require.Nil(t, r)

	// no change
	c.Auth["key_auth"] = string("{\"key\": \"two\"}")
	v = newConsumerTest().Add("ns", c).Build()
	updateConsumers(v)
	r, _ = LookupConsumer("ns", "key_auth", "test")
	require.Equal(t, "me", r.Name())

	// update
	c.generation = 2
	v = newConsumerTest().Add("ns", c).Build()
	updateConsumers(v)
	r, _ = LookupConsumer("ns", "key_auth", "test")
	require.Nil(t, r)
	r, _ = LookupConsumer("ns", "key_auth", "two")
	require.Equal(t, "me", r.Name())

	// remove
	c.name = "you"
	c.generation = 3
	v = newConsumerTest().Add("ns", c).Build()
	updateConsumers(v)
	r, _ = LookupConsumer("ns", "key_auth", "me")
	require.Nil(t, r)
	r, _ = LookupConsumer("ns", "key_auth", "two")
	require.Equal(t, "you", r.Name())
}

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
					"key_auth": "{\"key\": \"test\"}",
				},
				Filters: map[string]*model.FilterConfig{
					"opa": {
						Config: map[string]interface{}{
							"remote": map[string]interface{}{
								"url":    "http://opa:8181",
								"policy": "t",
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
					"key_auth": "{\"key2\": \"test\"}",
				},
			},
			err: "failed to unmarshal consumer config for plugin key_auth",
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
