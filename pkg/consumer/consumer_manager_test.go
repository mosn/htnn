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

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	"mosn.io/htnn/pkg/proto"
)

func TestParse(t *testing.T) {
	ts := xds.TypedStruct{}
	ts.Value, _ = structpb.NewStruct(map[string]interface{}{})
	any1 := proto.MessageToAny(&ts)
	any2 := proto.MessageToAny(&xds.TypedStruct{})

	cases := []struct {
		name    string
		input   *anypb.Any
		wantErr bool
	}{
		{
			name:    "happy path",
			input:   any1,
			wantErr: false,
		},
		{
			name:    "happy path without config",
			input:   &anypb.Any{},
			wantErr: false,
		},
		{
			name: "error UnmarshalTo",
			input: &anypb.Any{
				TypeUrl: "aaa",
			},
			wantErr: true,
		},
		{
			name:    "empty value",
			input:   any2,
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parser := &ConsumerManagerConfigParser{}

			_, err := parser.Parse(c.input, nil)
			if c.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
