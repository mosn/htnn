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
	"k8s.io/apimachinery/pkg/types"

	"mosn.io/htnn/e2e/pkg/k8s"
	"mosn.io/htnn/e2e/pkg/suite"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func init() {
	suite.Register(suite.Test{
		Run: func(t *testing.T, suite *suite.Suite) {
			tr := &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
				return net.DialTimeout("tcp", ":18001", 1*time.Second)
			}}
			client := &http.Client{Transport: tr, Timeout: 10 * time.Second}
			// check if deny policy works
			_, err := client.Get("http://default.local:18001/echo")
			require.Error(t, err)

			c := suite.K8sClient()
			ctx := context.Background()
			nsName := types.NamespacedName{Name: "policy", Namespace: k8s.IstioRootNamespace}
			var policy mosniov1.FilterPolicy
			err = c.Get(ctx, nsName, &policy)
			require.NoError(t, err)
			err = c.Delete(ctx, &policy)
			require.NoError(t, err)

			time.Sleep(1 * time.Second)
			rsp, err := client.Get("http://default.local:18001/echo")
			require.NoError(t, err)
			require.Equal(t, 200, rsp.StatusCode)

			// Do the same with Gateway API
			tr = &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
				return net.DialTimeout("tcp", ":10001", 1*time.Second)
			}}
			client = &http.Client{Transport: tr, Timeout: 10 * time.Second}
			_, err = client.Get("http://localhost:10001/echo")
			require.Error(t, err)

			nsName = types.NamespacedName{Name: "policy", Namespace: k8s.DefaultNamespace}
			err = c.Get(ctx, nsName, &policy)
			require.NoError(t, err)
			err = c.Delete(ctx, &policy)
			require.NoError(t, err)

			time.Sleep(1 * time.Second)
			rsp, err = client.Get("http://localhost:10001/echo")
			require.NoError(t, err)
			require.Equal(t, 200, rsp.StatusCode)
		},
	})
}
