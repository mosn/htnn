package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	p := &parser{}
	data := []byte(`{"host_name":"Jack"}`)
	ty, err := p.Validate(data)
	assert.Nil(t, err)
	c := ty.(*Config)
	assert.Equal(t, "Jack", c.HostName)
	res, err := p.Handle(c, nil)
	assert.Nil(t, err)
	assert.Equal(t, c, res)

	parentConfig := &Config{}
	childConfig := &Config{}
	assert.Equal(t, childConfig, p.Merge(parentConfig, childConfig))
}

func TestBadConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name:  "no host_name",
			input: `{}`,
			err:   "invalid Config.HostName: value length must be at least 1 runes",
		},
		{
			name:  "empty host_name",
			input: `{"host_name":""}`,
			err:   "invalid Config.HostName: value length must be at least 1 runes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&parser{}).Validate([]byte(tt.input))
			assert.NotNil(t, err)
			assert.ErrorContains(t, err, tt.err)
		})
	}
}
