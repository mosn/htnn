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
				return net.DialTimeout("tcp", ":10000", 1*time.Second)
			}}
			client := &http.Client{Transport: tr, Timeout: 10 * time.Second}
			rsp, err := client.Get("http://localhost:10000/echo")
			require.NoError(t, err)
			req, _, err := suite.Capture(rsp)
			require.NoError(t, err, rsp)
			require.Equal(t, 1, len(req.Headers["Doraemon"]), req)
			require.Equal(t, "hello,", req.Headers["Doraemon"][0])

			// Same host, in different gateway of different namespace
			tr = &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
				return net.DialTimeout("tcp", ":10100", 1*time.Second)
			}}
			client = &http.Client{Transport: tr, Timeout: 10 * time.Second}
			rsp, err = client.Get("http://localhost:10000/echo")
			require.NoError(t, err)
			req, _, err = suite.Capture(rsp)
			require.NoError(t, err, rsp)
			require.Equal(t, 1, len(req.Headers["Nobi-Nobita"]), req)
			require.Equal(t, "hello,", req.Headers["Nobi-Nobita"][0])
		},
	})
}
