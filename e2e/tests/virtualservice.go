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
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/types"

	"mosn.io/htnn/e2e/pkg/k8s"
	"mosn.io/htnn/e2e/pkg/suite"
	mosniov1 "mosn.io/htnn/types/apis/v1"
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
			req, _, err := suite.Capture(rsp)
			require.NoError(t, err)
			require.Equal(t, "hello,", req.Headers["Doraemon"][0])

			c := suite.K8sClient()
			ctx := context.Background()
			nsName := types.NamespacedName{Name: "policy", Namespace: k8s.IstioRootNamespace}
			var policy mosniov1.FilterPolicy
			err = c.Get(ctx, nsName, &policy)
			require.NoError(t, err)
			st := policy.Status
			cd := st.Conditions[0]
			gen := policy.Generation
			require.Equal(t, gen, cd.ObservedGeneration)
			require.Equal(t, "The policy has been accepted", cd.Message)

			// Ensure the policy is reconciled when the targeted resource is deleted
			nsName = types.NamespacedName{Name: "vs", Namespace: k8s.IstioRootNamespace}
			var route istiov1a3.VirtualService
			err = c.Get(ctx, nsName, &route)
			require.NoError(t, err)
			c.Delete(ctx, &route)
			time.Sleep(time.Second)

			nsName = types.NamespacedName{Name: "policy", Namespace: k8s.IstioRootNamespace}
			err = c.Get(ctx, nsName, &policy)
			require.NoError(t, err)
			st = policy.Status
			cd = st.Conditions[0]
			gen = policy.Generation
			require.Equal(t, gen, cd.ObservedGeneration) // same Generation as the Spec is not changed
			require.Equal(t, "The policy targets non-existent resource", cd.Message)
		},
	})
}
