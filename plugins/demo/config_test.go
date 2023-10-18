package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestParser(t *testing.T) {
	p := &parser{}
	cfg := &anypb.Any{}
	_, err := p.Parse(cfg, nil)
	assert.Nil(t, err)

	parentConfig := &config{}
	childConfig := &config{}
	assert.Equal(t, childConfig, p.Merge(parentConfig, childConfig))
}
