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

package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"

	api "mosn.io/htnn/types/plugins/api/v1"
)

func TestStringMatcher(t *testing.T) {
	tests := []struct {
		name       string
		cfg        string
		matched    []string
		mismatched []string
	}{
		{
			name:       "exact",
			cfg:        `{"exact": "foo"}`,
			matched:    []string{"foo"},
			mismatched: []string{"fo", "fooo"},
		},
		{
			name:    "exact, ignore_case",
			cfg:     `{"exact": "foo", "ignore_case": true}`,
			matched: []string{"Foo", "foo"},
		},
		{
			name:       "prefix, ignore_case",
			cfg:        `{"prefix": "/p", "ignore_case": true}`,
			matched:    []string{"/P", "/p", "/pa", "/Pa"},
			mismatched: []string{"/"},
		},
		{
			name:       "prefix",
			cfg:        `{"prefix": "/p"}`,
			matched:    []string{"/p", "/pa"},
			mismatched: []string{"/P"},
		},
		{
			name:       "prefix, ignore_case",
			cfg:        `{"prefix": "/p", "ignore_case": true}`,
			matched:    []string{"/P", "/p", "/pa", "/Pa"},
			mismatched: []string{"/"},
		},
		{
			name:       "suffix",
			cfg:        `{"suffix": "foo"}`,
			matched:    []string{"foo", "0foo"},
			mismatched: []string{"fo", "fooo", "aFoo"},
		},
		{
			name:       "suffix, ignore_case",
			cfg:        `{"suffix": "foo", "ignore_case": true}`,
			matched:    []string{"aFoo", "foo"},
			mismatched: []string{"fo", "fooo"},
		},
		{
			name:       "contains",
			cfg:        `{"contains": "foo"}`,
			matched:    []string{"foo", "0foo", "fooo"},
			mismatched: []string{"fo", "aFoo"},
		},
		{
			name:       "contains, ignore_case",
			cfg:        `{"contains": "foo", "ignore_case": true}`,
			matched:    []string{"aFoo", "foo", "FoO"},
			mismatched: []string{"fo"},
		},
		{
			name:       "regex",
			cfg:        `{"regex": "fo{2}"}`,
			matched:    []string{"foo", "0foo", "fooo"},
			mismatched: []string{"aFoo", "fo"},
		},
		{
			name:       "regex, ignore_case",
			cfg:        `{"regex": "fo{2}", "ignore_case": true}`,
			matched:    []string{"foo", "0foo", "fooo", "aFoo"},
			mismatched: []string{"fo"},
		},
		{
			name:       "regex, ignore_case & case insensitive specified in regex",
			cfg:        `{"regex": "(?i)fo{2}", "ignore_case": true}`,
			matched:    []string{"foo", "0foo", "fooo", "aFoo"},
			mismatched: []string{"fo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &api.StringMatcher{}
			protojson.Unmarshal([]byte(tt.cfg), m)
			built, _ := BuildStringMatcher(m)
			for _, s := range tt.matched {
				assert.True(t, built.Match(s))
			}
			for _, s := range tt.mismatched {
				assert.False(t, built.Match(s))
			}
		})
	}
}

func TestBuildRepeatedStringMatcherIgnoreCase(t *testing.T) {
	cfgs := []string{
		`{"exact":"foo"}`,
		`{"prefix":"pre"}`,
		`{"regex":"^Cache"}`,
	}
	matched := []string{"Foo", "foO", "foo", "PreA", "cache-control", "Cache-Control"}
	mismatched := []string{"afoo", "fo"}
	ms := []*api.StringMatcher{}
	for _, cfg := range cfgs {
		m := &api.StringMatcher{}
		protojson.Unmarshal([]byte(cfg), m)
		ms = append(ms, m)
	}
	built, _ := BuildRepeatedStringMatcherIgnoreCase(ms)
	for _, s := range matched {
		assert.True(t, built.Match(s))
	}
	for _, s := range mismatched {
		assert.False(t, built.Match(s))
	}
}

func TestPassOutRegexCompileErr(t *testing.T) {
	cfg := `{"regex":"(?!)aa"}`
	m := &api.StringMatcher{}
	protojson.Unmarshal([]byte(cfg), m)
	_, err := BuildRepeatedStringMatcher([]*api.StringMatcher{m})
	assert.NotNil(t, err)
}
