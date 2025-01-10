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

package dataplane

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	//go:embed testdata/listener.yml
	listener string
)

func TestInsertHTTPFilterBeforeRouter(t *testing.T) {
	dp := &DataPlane{
		t:   t,
		opt: &Option{},
	}
	b := Bootstrap()
	b.SetDataPlane(dp)
	b.InsertHTTPFilter(map[string]interface{}{
		"name": "test.filter.buffer",
		"typed_config": map[string]interface{}{
			"@type":             "type.googleapis.com/envoy.extensions.filters.http.buffer.v3.Buffer",
			"max_request_bytes": 1024,
		},
	}, HTTPFilterInsertOperationBeforeRouter)
	buf := bytes.NewBuffer([]byte(""))
	err := b.WriteTo(buf)
	assert.Nil(t, err)
	assert.Regexp(t, `- name: test.filter.buffer\s+typed_config:[^-]+- name: envoy.filters.http.router`, buf.String())
}

func TestAddListener(t *testing.T) {
	dp := &DataPlane{
		t:   t,
		opt: &Option{},
	}
	b := Bootstrap()
	b.SetDataPlane(dp)
	b.AddListener(listener)
	buf := bytes.NewBuffer([]byte(""))
	err := b.WriteTo(buf)
	assert.Nil(t, err)
	s := buf.String()
	assert.Contains(t, s, "name: listener_proxy")
	assert.Contains(t, s, "cluster: x_backend")
}
