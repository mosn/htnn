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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"mosn.io/htnn/e2e/pkg/suite"
)

func init() {
	suite.Register(suite.Test{
		Manifests: []string{"base/httproute.yml"},
		Run: func(t *testing.T, suite *suite.Suite) {
			// port-forwarder will spend some time for the first request.
			suite.Post("/echo", nil, strings.NewReader(""))

			s := strings.Repeat("a", 1024)
			// start to count
			now := time.Now()
			rsp, err := suite.Post("/echo", nil, strings.NewReader(s))
			cost := time.Since(now)
			require.NoError(t, err)
			require.Equal(t, 200, rsp.StatusCode)
			// the delta is relative to the bandwidthLimit's fillInterval
			require.Truef(t, cost > 900*time.Millisecond, "cost: %v", cost)
			require.Truef(t, cost < 1100*time.Millisecond, "cost: %v", cost)
		},
	})
}
