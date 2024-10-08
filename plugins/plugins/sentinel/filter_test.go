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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/types/plugins/sentinel"
)

func TestFilter(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	f := factory(&config{}, cb).(*filter)
	h := http.Header{}
	h.Set("X-Sentinel", "test1")
	h.Add("X-Multi", "a")
	h.Add("X-Multi", "b")
	h.Add("X-Multi", "c")
	h.Set(":path", "/echo?query=test2")
	hdr := envoy.NewRequestHeaderMap(h)

	tests := []struct {
		name     string
		source   *sentinel.Source
		expected string
	}{
		{
			name: "from header",
			source: &sentinel.Source{
				From: sentinel.Source_HEADER,
				Key:  "X-Sentinel",
			},
			expected: "test1",
		},
		{
			name: "from header: multi val",
			source: &sentinel.Source{
				From: sentinel.Source_HEADER,
				Key:  "X-Multi",
			},
			expected: "a",
		},
		{
			name: "from header: empty",
			source: &sentinel.Source{
				From: sentinel.Source_HEADER,
				Key:  "X-Empty",
			},
			expected: "",
		},
		{
			name: "from query",
			source: &sentinel.Source{
				From: sentinel.Source_QUERY,
				Key:  "query",
			},
			expected: "test2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := f.getSource(tt.source, hdr)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
