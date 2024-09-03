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
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"mosn.io/htnn/e2e/pkg/k8s"
	"mosn.io/htnn/e2e/pkg/suite"
)

func init() {
	suite.Register(suite.Test{
		Run: func(t *testing.T, suite *suite.Suite) {
			hdr := http.Header{}
			hdr.Add("Connection", "close")
			rsp, err := suite.Get("/echo", hdr)
			require.NoError(t, err)
			require.Equal(t, 200, rsp.StatusCode)

			time.Sleep(100 * time.Millisecond)
			require.Eventually(t, func() bool {
				namespace := k8s.DefaultNamespace

				b, err := suite.GetLog(namespace, "default-istio-")
				if err != nil {
					t.Logf("unexpected error %v", err)
					return false
				}
				return bytes.Contains(b, []byte("added access log: 127.0.0.1:10000"))
			}, 10*time.Second, 100*time.Millisecond)
		},
	})
}
