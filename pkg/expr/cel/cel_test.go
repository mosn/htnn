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

package cel

import (
	"net/http"
	"sync"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/plugins/tests/pkg/envoy"
)

func TestCompile(t *testing.T) {
	cases := []struct {
		name string
		expr string
	}{
		{
			name: "bad expr",
			expr: `req`,
		},
		{
			name: "bad arguments",
			expr: `request.header()`,
		},
		{
			name: "bad return type",
			expr: `1 + 2`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(tt.expr, cel.StringType)
			require.Error(t, err)
		})
	}
}

func TestCustomType(t *testing.T) {
	ct := &request{}
	cp := ct
	require.True(t, ct.Equal(cp).Value().(bool))
	require.False(t, ct.Equal(types.String("")).Value().(bool))
}

func TestCel(t *testing.T) {
	s, err := Compile(`request.host()`, cel.StringType)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func() {
			_, err := Compile(`request.host()`, cel.StringType)
			require.NoError(t, err)
			wg.Done()
		}()
	}
	wg.Wait()

	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func() {
			hdr := http.Header{}
			hdr.Set(":authority", "t.local")
			res, err := EvalRequest(s, nil, envoy.NewRequestHeaderMap(hdr))
			require.NoError(t, err)
			require.Equal(t, "t.local", res)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestCelWithRequest(t *testing.T) {
	hdr := http.Header{}
	hdr.Set(":authority", "t.local")
	hdr.Set(":method", "PUT")
	hdr.Set(":path", "/x?a=1&b=2&b=")
	hdr.Add("single", "a")
	hdr.Add("multi", "a")
	hdr.Add("multi", "b")

	tests := []struct {
		name   string
		code   string
		expect func(t *testing.T, res any)
	}{
		{
			name: "path",
			code: `request.path()`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "/x?a=1&b=2&b=", res)
			},
		},
		{
			name: "url_path",
			code: `request.url_path()`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "/x", res)
			},
		},
		{
			name: "host",
			code: `request.host()`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "t.local", res)
			},
		},
		{
			name: "scheme",
			code: `request.scheme()`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "http", res)
			},
		},
		{
			name: "method",
			code: `request.method()`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "PUT", res)
			},
		},
		{
			name: "single header",
			code: `request.header("single")`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "a", res)
			},
		},
		{
			name: "multi header",
			code: `request.header("multi")`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "a,b", res)
			},
		},
		{
			name: "header not found",
			code: `request.header("x")`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "", res)
			},
		},
		{
			name: "query_path",
			code: `request.query_path()`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "a=1&b=2&b=", res)
			},
		},
		{
			name: "single query arg",
			code: `request.query("a")`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "1", res)
			},
		},
		{
			name: "multi query arg",
			code: `request.query("b")`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "2,", res)
			},
		},
		{
			name: "query arg not found",
			code: `request.query("x")`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "", res)
			},
		},
		{
			name: "id",
			code: `request.id()`,
			expect: func(t *testing.T, res any) {
				require.Equal(t, "property.request.id", res)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := Compile(tt.code, cel.StringType)
			require.NoError(t, err)
			cb := envoy.NewFilterCallbackHandler()
			patches := gomonkey.ApplyMethodFunc(cb, "GetProperty", func(s string) (string, error) {
				return "property." + s, nil
			})
			defer patches.Reset()
			res, err := EvalRequest(s, cb, envoy.NewRequestHeaderMap(hdr))
			require.NoError(t, err)
			tt.expect(t, res)
		})
	}
}
