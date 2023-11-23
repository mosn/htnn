package plugins

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"

	_ "mosn.io/moe/plugins/tests/pkg/envoy"
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
				patches.ApplyMethodReturn(cp.Plugin, "Config", conf)
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
