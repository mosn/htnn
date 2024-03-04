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
		"keyAuth": "{\"key\": \"test\"}",
	}
	c := &Consumer{
		name:       "me",
		generation: 1,
		Auth:       auth,
	}
	v := newConsumerTest().Add("ns", c).Build()
	UpdateConsumers(v)

	r, _ := LookupConsumer("ns", "keyAuth", "test")
	require.NotNil(t, r)
	require.Equal(t, "me", r.Name())

	r, _ = LookupConsumer("ns", "keyAuth", "not_found")
	require.Nil(t, r)

	// no change
	c.Auth["keyAuth"] = string("{\"key\": \"two\"}")
	v = newConsumerTest().Add("ns", c).Build()
	UpdateConsumers(v)
	r, _ = LookupConsumer("ns", "keyAuth", "test")
	require.Equal(t, "me", r.Name())

	// update
	c.generation = 2
	v = newConsumerTest().Add("ns", c).Build()
	UpdateConsumers(v)
	r, _ = LookupConsumer("ns", "keyAuth", "test")
	require.Nil(t, r)
	r, _ = LookupConsumer("ns", "keyAuth", "two")
	require.Equal(t, "me", r.Name())

	// remove
	c.name = "you"
	c.generation = 3
	v = newConsumerTest().Add("ns", c).Build()
	UpdateConsumers(v)
	r, _ = LookupConsumer("ns", "keyAuth", "me")
	require.Nil(t, r)
	r, _ = LookupConsumer("ns", "keyAuth", "two")
	require.Equal(t, "you", r.Name())
}
