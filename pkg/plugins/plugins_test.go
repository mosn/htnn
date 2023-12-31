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

package plugins

import (
	"errors"
	"sort"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"

	_ "mosn.io/htnn/plugins/tests/pkg/envoy"
)

func TestIterateHttpPlugin(t *testing.T) {
	plugin := &MockPlugin{}
	RegisterHttpPlugin("test", plugin)

	IterateHttpPlugin(func(name string, p Plugin) bool {
		assert.Equal(t, "test", name)
		assert.Equal(t, p, plugin)
		return true
	})
}

func TestParse(t *testing.T) {
	cat := "cat"
	any1 := map[string]interface{}{
		"pet": cat,
	}

	cases := []struct {
		name    string
		input   interface{}
		checker func(t *testing.T, cp *PluginConfigParser) func()
		wantErr bool
		pet     string
	}{
		{
			name:    "happy path",
			input:   any1,
			wantErr: false,
			pet:     "cat",
		},
		{
			name:    "no input",
			wantErr: false,
			pet:     "", // use default value
		},
		{
			name:  "error validate",
			input: &anypb.Any{},
			checker: func(t *testing.T, cp *PluginConfigParser) func() {
				conf := &MockPluginConfig{}
				patches := gomonkey.ApplyMethodReturn(conf, "Validate", errors.New("ouch"))
				patches.ApplyMethodReturn(cp.GoPlugin, "Config", conf)
				return func() {
					patches.Reset()
				}
			},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cp := NewPluginConfigParser(&MockPlugin{})
			if c.checker != nil {
				cln := c.checker(t, cp)
				defer cln()
			}

			res, err := cp.Parse(c.input, nil)
			if c.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, c.pet, res.(*MockPluginConfig).Pet)
			}
		})
	}
}

type Merger struct {
	MockPlugin
}

func (m *Merger) Merge(parentConfig interface{}, childConfig interface{}) interface{} {
	return parentConfig
}

func TestMerge(t *testing.T) {
	cp := NewPluginConfigParser(&MockPlugin{})
	res := cp.Merge("parent", "child")
	assert.Equal(t, "child", res)

	cp = NewPluginConfigParser(&Merger{})
	res = cp.Merge("parent", "child")
	assert.Equal(t, "parent", res)
}

type pluginOrderWrapper struct {
	Plugin

	order PluginOrder
}

func (p *pluginOrderWrapper) Order() PluginOrder {
	return p.order
}

func TestComparePluginOrder(t *testing.T) {
	plugin := &MockPlugin{}

	pluginOrders := map[string]PluginOrder{
		"authz_first": {
			Position:  OrderPositionAuthz,
			Operation: OrderOperationInsertFirst,
		},
		"authz_second": {
			Position: OrderPositionAuthz,
		},
		"authz_third": {
			Position: OrderPositionAuthz,
		},
		"authz_last": {
			Position:  OrderPositionAuthz,
			Operation: OrderOperationInsertLast,
		},
		"authn": {
			Position: OrderPositionAuthn,
		},
	}
	for name, po := range pluginOrders {
		RegisterHttpPlugin(name, &pluginOrderWrapper{
			Plugin: plugin,
			order:  po,
		})
	}

	plugins := []string{
		"authn",
		"authz_third",
		"authz_last",
		"authz_second",
		"authz_first",
	}
	sort.Slice(plugins, func(i, j int) bool {
		return ComparePluginOrder(plugins[i], plugins[j])
	})
	assert.Equal(t, []string{
		"authn",
		"authz_first",
		"authz_second",
		"authz_third",
		"authz_last",
	}, plugins)
}
