package tests

import (
	"github.com/stretchr/testify/require"
	"mosn.io/htnn/e2e/pkg/suite"
	"testing"
)

func init() {
	suite.Register(suite.Test{
		Manifests: []string{"base/grpc.yml"},
		Run: func(t *testing.T, suite *suite.Suite) {
			rsp, err := suite.Get("/echo", nil)
			require.NoError(t, err)
			require.Equal(t, 200, rsp.StatusCode)
		},
	})
}
