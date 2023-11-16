package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestPlugin(t *testing.T) {
	data := []byte(`{"host_name":"Jack"}`)
	c := &config{}
	protojson.Unmarshal(data, c)
	err := c.Validate()
	assert.Nil(t, err)
	assert.Equal(t, "Jack", c.HostName)
	err = c.Init(nil)
	assert.Nil(t, err)
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
			conf := &Config{}
			protojson.Unmarshal([]byte(tt.input), conf)
			err := conf.Validate()
			assert.NotNil(t, err)
			assert.ErrorContains(t, err, tt.err)
		})
	}
}
