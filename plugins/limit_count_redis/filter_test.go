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

package limit_count_redis

import (
	"net/http"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/pkg/expr"
)

func TestGetKey(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	f := factory(&config{}, cb).(*filter)
	h := http.Header{}
	h.Set("pet", "cat")
	hdr := envoy.NewRequestHeaderMap(h)
	s, err := expr.CompileCel(`request.header("pet")`, cel.StringType)
	require.NoError(t, err)
	s2, err := expr.CompileCel(`request.header("food")`, cel.StringType)
	require.NoError(t, err)

	tests := []struct {
		name   string
		script expr.Script
		key    string
	}{
		{
			name: "default",
			key:  "183.128.130.43",
		},
		{
			name:   "use expr",
			script: s,
			key:    "cat",
		},
		{
			name:   "fallback",
			script: s2,
			key:    "183.128.130.43",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := f.getKey(tt.script, hdr)
			assert.Equal(t, tt.key, key)
		})
	}
}
