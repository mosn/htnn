package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	p := &parser{}
	data := []byte(`{"guest_name":"Jack"}`)
	ty, err := p.Validate(data)
	assert.Nil(t, err)
	c := ty.(*Config)
	assert.Equal(t, "Jack", c.GuestName)

	parentConfig := &Config{}
	childConfig := &Config{}
	assert.Equal(t, childConfig, p.Merge(parentConfig, childConfig))
}

func TestBadConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "no guest_name",
			input: `{}`,
		},
		{
			name:  "empty guest_name",
			input: `{"guest_name":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&parser{}).Validate([]byte(tt.input))
			assert.NotNil(t, err)
		})
	}
}
