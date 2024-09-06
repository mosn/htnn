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

package dynamicconfig

import (
	"testing"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	"mosn.io/htnn/api/internal/proto"
	_ "mosn.io/htnn/api/plugins/tests/pkg/envoy" // mock log
)

func TestParse(t *testing.T) {
	ts := xds.TypedStruct{}
	ts.Value, _ = structpb.NewStruct(map[string]interface{}{})
	any1 := proto.MessageToAny(&ts)
	any2 := proto.MessageToAny(&xds.TypedStruct{})
	tsNoCb := xds.TypedStruct{}
	tsNoCb.Value, _ = structpb.NewStruct(map[string]interface{}{
		"name":   "unknown",
		"config": map[string]interface{}{},
	})
	any3 := proto.MessageToAny(&tsNoCb)

	cases := []struct {
		name  string
		input *anypb.Any
		err   string
	}{
		{
			name:  "happy path without config",
			input: &anypb.Any{},
		},
		{
			name: "error UnmarshalTo",
			input: &anypb.Any{
				TypeUrl: "aaa",
			},
			err: "mismatched message type",
		},
		{
			name:  "invalid value",
			input: any1,
			err:   "invalid dynamic config format",
		},
		{
			name:  "empty value",
			input: any2,
			err:   "bad TypedStruct format",
		},
		{
			name:  "unknown value",
			input: any3,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parser := &DynamicConfigParser{}

			_, err := parser.Parse(c.input, nil)
			if c.err != "" {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), c.err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
