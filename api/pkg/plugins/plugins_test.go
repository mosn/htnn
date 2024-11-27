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

	"mosn.io/htnn/api/pkg/filtermanager/api"
	_ "mosn.io/htnn/api/plugins/tests/pkg/envoy" // for log implementation
)

func TestIteratePluginType(t *testing.T) {
	plugin := &MockPlugin{}
	RegisterPlugin("test", plugin)
	RegisterPluginType("test2", plugin)

	names := []string{}
	IteratePluginType(func(name string, p Plugin) bool {
		names = append(names, name)
		assert.Equal(t, p, plugin)
		return true
	})
	assert.Contains(t, names, "test")
	assert.Contains(t, names, "test2")

	names = []string{}
	IteratePluginType(func(name string, p Plugin) bool {
		names = append(names, name)
		return false
	})
	assert.Equal(t, 1, len(names))
	// the order is not guaranteed, it can be "test" or "test2"
}

func TestIteratePlugin(t *testing.T) {
	plugin := &MockPlugin{}
	RegisterPlugin("test", plugin)

	IteratePlugin(func(name string, p Plugin) bool {
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

			res, err := cp.Parse(c.input)
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

type goPluginOrderWrapper struct {
	GoPlugin

	order PluginOrder
}

func (p *goPluginOrderWrapper) Order() PluginOrder {
	return p.order
}

type consumerPluginWrapper struct {
	ConsumerPlugin

	order PluginOrder
}

func (p *consumerPluginWrapper) Factory() api.FilterFactory {
	return p.ConsumerPlugin.(*MockConsumerPlugin).Factory()
}

func (p *consumerPluginWrapper) Order() PluginOrder {
	return p.order
}

type nativePluginWrapper struct {
	NativePlugin

	order PluginOrder
}

func (p *nativePluginWrapper) Order() PluginOrder {
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
		RegisterPlugin(name, &goPluginOrderWrapper{
			GoPlugin: plugin,
			order:    po,
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

func TestRejectBadPluginDef(t *testing.T) {
	type pluginWrapper struct {
		Plugin
	}

	cases := []struct {
		name  string
		input Plugin
		err   string
	}{
		{
			name: "unknown type",
			input: &pluginWrapper{
				Plugin: &MockPlugin{},
			},
			err: errUnknownPluginType,
		},
		{
			name: "nil plugin",
			err:  errNilPlugin,
		},
		{
			name: "invalid Go plugin order",
			input: &goPluginOrderWrapper{
				GoPlugin: &MockPlugin{},
				order: PluginOrder{
					Position: OrderPositionInner,
				},
			},
			err: errInvalidGoPluginOrder,
		},
		{
			name: "invalid Native plugin order",
			input: &nativePluginWrapper{
				NativePlugin: &MockNativePlugin{},
				order: PluginOrder{
					Position: OrderPositionAuthz,
				},
			},
			err: errInvalidNativePluginOrder,
		},
		{
			name: "invalid Consumer plugin order",
			input: &consumerPluginWrapper{
				ConsumerPlugin: &MockConsumerPlugin{},
				order: PluginOrder{
					Position: OrderPositionAuthz,
				},
			},
			err: errInvalidConsumerPluginOrder,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.PanicsWithValue(t, c.err, func() {
				RegisterPlugin(c.name, c.input)
			})
		})
	}
}

func TestRegisterPluginWithType(t *testing.T) {
	RegisterPlugin("mock", &MockPlugin{})
	assert.NotNil(t, LoadPlugin("mock"))
	assert.NotNil(t, LoadPluginType("mock"))
}
