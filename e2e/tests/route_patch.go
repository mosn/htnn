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
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"mosn.io/htnn/e2e/pkg/suite"
)

func init() {
	suite.Register(suite.Test{
		Run: func(t *testing.T, suite *suite.Suite) {
			tr := &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
				return net.DialTimeout("tcp", ":18000", 1*time.Second)
			}}
			client := &http.Client{Transport: tr, Timeout: 10 * time.Second}
			rsp, err := client.Get("http://default.local:18000/echo")
			require.NoError(t, err)
			require.Equal(t, 403, rsp.StatusCode)
			rsp, err = client.Get("http://default.local:18000/echo2")
			require.NoError(t, err)
			require.Equal(t, 403, rsp.StatusCode)
			rsp, err = client.Get("http://default.local:18000/")
			require.NoError(t, err)
			require.Equal(t, 405, rsp.StatusCode)
		},
	})
}
