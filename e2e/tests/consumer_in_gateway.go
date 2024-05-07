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

	"github.com/stretchr/testify/require"

	"mosn.io/htnn/e2e/pkg/suite"
)

func init() {
	suite.Register(suite.Test{
		Manifests: []string{"base/httproute.yml"},
		Run: func(t *testing.T, suite *suite.Suite) {
			rsp, err := suite.Get("/echo", hdrWithKey("rick"))
			require.NoError(t, err)
			require.Equal(t, 200, rsp.StatusCode)
			req, _, err := suite.Capture(rsp)
			require.NoError(t, err)
			require.Equal(t, 1, len(req.Headers["Rick"]), req)
			require.Equal(t, "hello,", req.Headers["Rick"][0])
			rsp, _ = suite.Get("/echo", hdrWithKey("rick"))
			require.Equal(t, 429, rsp.StatusCode)

			rsp, _ = suite.Get("/echo", hdrWithKey("morty"))
			require.Equal(t, 200, rsp.StatusCode)
			// No limit req
			rsp, _ = suite.Get("/echo", hdrWithKey("morty"))
			require.Equal(t, 200, rsp.StatusCode)
			req, _, err = suite.Capture(rsp)
			require.NoError(t, err)
			// use the demo from the route
			require.Equal(t, 1, len(req.Headers["Doraemon"]), req)

			rsp, _ = suite.Get("/echo", hdrWithKey("doraemon"))
			require.Equal(t, 401, rsp.StatusCode)
		},
	})
}
