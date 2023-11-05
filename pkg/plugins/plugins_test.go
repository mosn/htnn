package plugins

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
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
	any1 := map[string]interface{}{
		"pet": "cat",
	}

	cfg := "this is plugin conf"
	cases := []struct {
		name    string
		input   interface{}
		checker func(t *testing.T, cp *PluginConfigParser) func()
		wantErr bool
	}{
		{
			name:  "happy path",
			input: any1,
			checker: func(t *testing.T, cp *PluginConfigParser) func() {
				patches := gomonkey.ApplyMethodFunc(cp.ConfigParser, "Validate", func(data []byte) (interface{}, error) {
					assert.Equal(t, `{"pet":"cat"}`, string(data))
					return cfg, nil
				})
				patches.ApplyMethodFunc(cp.ConfigParser, "Handle", func(config interface{}, cb api.ConfigCallbackHandler) (interface{}, error) {
					assert.Equal(t, cfg, config)
					return cfg, nil
				})
				return func() {
					patches.Reset()
				}
			},
			wantErr: false,
		},
		{
			name:  "error validate",
			input: &anypb.Any{},
			checker: func(t *testing.T, cp *PluginConfigParser) func() {
				patches := gomonkey.ApplyMethodReturn(cp.ConfigParser, "Validate", nil, errors.New("ouch"))
				return func() {
					patches.Reset()
				}
			},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cp := NewPluginConfigParser(&MockConfigParser{})
			cln := c.checker(t, cp)
			defer cln()

			res, err := cp.Parse(c.input, nil)
			if c.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, cfg, res)
			}
		})
	}
}

type Merger struct {
	MockConfigParser
}

func (m *Merger) Merge(parentConfig interface{}, childConfig interface{}) interface{} {
	return parentConfig
}

func TestMerge(t *testing.T) {
	cp := NewPluginConfigParser(&MockConfigParser{})
	res := cp.Merge("parent", "child")
	assert.Equal(t, "child", res)

	cp = NewPluginConfigParser(&Merger{})
	res = cp.Merge("parent", "child")
	assert.Equal(t, "parent", res)
}
