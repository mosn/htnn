package nacos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	regType := &RegistryType{}
	config := regType.Config()
	assert.NotNil(t, config)
}
