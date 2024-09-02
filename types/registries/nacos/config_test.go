package nacos

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfig(t *testing.T) {
	regType := &RegistryType{}
	config := regType.Config()
	assert.NotNil(t, config)
}
