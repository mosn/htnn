package consul

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/types/pkg/registry"
)

func TestConfig(t *testing.T) {
	regType := &RegistryType{}
	config := regType.Config()

	assert.NotNil(t, config)

	_, ok := config.(registry.RegistryConfig)
	require.True(t, ok)
}
