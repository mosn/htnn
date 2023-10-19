package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	p := &parser{}
	data := []byte("{}")
	_, err := p.Validate(data)
	assert.Nil(t, err)

	parentConfig := &config{}
	childConfig := &config{}
	assert.Equal(t, childConfig, p.Merge(parentConfig, childConfig))
}
