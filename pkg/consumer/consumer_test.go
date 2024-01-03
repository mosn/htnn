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

	_ "mosn.io/htnn/plugins/key_auth"
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
	idx[consumer.Name] = map[string]interface{}{
		"d": consumer.Marshal(),
		"v": consumer.ResourceVersion,
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

	auth := map[string][]byte{
		"key_auth": []byte("{\"key\": \"test\"}"),
	}
	c := &Consumer{
		Name:            "me",
		ResourceVersion: "1",
		Auth:            auth,
	}
	v := newConsumerTest().Add("ns", c).Build()
	updateConsumers(v)

	r := GetConsumer("ns", "key_auth", "test")
	require.NotNil(t, r)
	require.Equal(t, "me", r.Name)

	r = GetConsumer("ns", "key_auth", "not_found")
	require.Nil(t, r)

	// no change
	c.Auth["key_auth"] = []byte("{\"key\": \"two\"}")
	v = newConsumerTest().Add("ns", c).Build()
	updateConsumers(v)
	r = GetConsumer("ns", "key_auth", "test")
	require.Equal(t, "me", r.Name)

	// update
	c.ResourceVersion = "2"
	v = newConsumerTest().Add("ns", c).Build()
	updateConsumers(v)
	r = GetConsumer("ns", "key_auth", "test")
	require.Nil(t, r)
	r = GetConsumer("ns", "key_auth", "two")
	require.Equal(t, "me", r.Name)

	// remove
	c.Name = "you"
	c.ResourceVersion = "3"
	v = newConsumerTest().Add("ns", c).Build()
	updateConsumers(v)
	r = GetConsumer("ns", "key_auth", "me")
	require.Nil(t, r)
	r = GetConsumer("ns", "key_auth", "two")
	require.Equal(t, "you", r.Name)
}

func TestConsumerUnmarshal(t *testing.T) {
	var tests = []struct {
		name     string
		consumer Consumer
		wantErr  bool
	}{
		{
			name: "not consumer plugin",
			consumer: Consumer{
				Auth: map[string][]byte{
					"not_consumer": []byte("{\"key\": \"test\"}"),
				},
			},
			wantErr: true,
		},
		{
			name: "failed to validate",
			consumer: Consumer{
				Auth: map[string][]byte{
					"key_auth": []byte("{\"key2\": \"test\"}"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Consumer
			err := c.Unmarshal(tt.consumer.Marshal())
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
