// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"mosn.io/moe/e2e/pkg/suite"
)

func init() {
	suite.Register(suite.Test{
		Manifests: []string{"base/virtualservice.yml"},
		Run: func(t *testing.T, suite *suite.Suite) {
			rsp, err := suite.Get("/echo", nil)
			require.NoError(t, err)
			require.Equal(t, 200, rsp.StatusCode)
			rsp, _ = suite.Get("/echo", nil)
			require.Equal(t, 429, rsp.StatusCode)

			time.Sleep(1000 * time.Millisecond)
			rsp, _ = suite.Get("/echo", nil)
			require.Equal(t, 200, rsp.StatusCode)
		},
	})
}
