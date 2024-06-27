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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"mosn.io/htnn/controller/pkg/constant"
	"mosn.io/htnn/e2e/pkg/k8s"
	"mosn.io/htnn/e2e/pkg/suite"
)

func init() {
	suite.Register(suite.Test{
		CleanUp: func(t *testing.T, suite *suite.Suite) {
			c := suite.K8sClient()
			ctx := context.Background()
			nsName := types.NamespacedName{Name: "default-embedded", Namespace: k8s.IstioRootNamespace}
			var gw istiov1a3.Gateway
			err := c.Get(ctx, nsName, &gw)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					require.NoError(t, err)
				}
				return
			}
			require.NoError(t, c.Delete(ctx, &gw))
		},
		Run: func(t *testing.T, suite *suite.Suite) {
			// wait for gateway to be ready
			require.Eventually(t, func() bool {
				tr := &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
					return net.DialTimeout("tcp", ":18000", 1*time.Second)
				}}
				client := &http.Client{Transport: tr, Timeout: 1 * time.Second}
				_, err := client.Get("http://default.local:18000/echo")
				return err == nil
			}, 30*time.Second, 3*time.Second)

			tr := &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
				return net.DialTimeout("tcp", ":18000", 1*time.Second)
			}}
			client := &http.Client{Transport: tr, Timeout: 10 * time.Second}
			rsp, err := client.Get("http://default.local:18000/echo")
			require.NoError(t, err)
			req, _, err := suite.Capture(rsp)
			require.NoError(t, err)
			require.Equal(t, "hello,", req.Headers["Micky"][0])

			c := suite.K8sClient()
			ctx := context.Background()
			nsName := types.NamespacedName{Name: "default-embedded", Namespace: k8s.IstioRootNamespace}
			var gw istiov1a3.Gateway

			err = c.Get(ctx, nsName, &gw)
			require.NoError(t, err)
			ann := gw.GetAnnotations()
			gw.SetAnnotations(nil)
			err = c.Update(ctx, &gw)
			require.NoError(t, err)
			time.Sleep(1 * time.Second)
			rsp, err = client.Get("http://default.local:18000/echo")
			require.NoError(t, err)
			req, _, err = suite.Capture(rsp)
			require.NoError(t, err)
			// Should not generate EnvoyFilter
			require.Equal(t, 0, len(req.Headers["Micky"]))

			gw.SetAnnotations(ann)
			err = c.Update(ctx, &gw)
			require.NoError(t, err)
			time.Sleep(1 * time.Second)
			rsp, err = client.Get("http://default.local:18000/echo")
			require.NoError(t, err)
			req, _, err = suite.Capture(rsp)
			require.NoError(t, err)
			// Should generate EnvoyFilter again
			require.Equal(t, "hello,", req.Headers["Micky"][0])

			// test webhook
			gw.SetAnnotations(map[string]string{constant.AnnotationHTTPFilterPolicy: "invalid"})
			err = c.Update(ctx, &gw)
			require.ErrorContains(t, err, "configuration is invalid: cannot unmarshal HTTPFilterPolicy: ")

			incorrectPolicy := `{"apiVersion":"htnn.mosn.io/v1","kind":"HTTPFilterPolicy","metadata":{"name":"policy"},"spec":{"filters":{"demo":{"config":{"hostName":"micky"}}},"subPolicies":[{"sectionName":"gw","filters":{"demo":{"config":{"hostName":["doraemon"]}}}}]}}`
			gw.SetAnnotations(map[string]string{constant.AnnotationHTTPFilterPolicy: incorrectPolicy})
			err = c.Update(ctx, &gw)
			require.ErrorContains(t, err, "configuration is invalid: invalid HTTPFilterPolicy: ")

			incorrectWhenValidateStrictlyPolicy := `{"apiVersion":"htnn.mosn.io/v1","kind":"HTTPFilterPolicy","metadata":{"name":"policy"},"spec":{"filters":{"demo":{"config":{"hostName":"micky"}},"unknown":{"config":{}}}}}`
			gw.SetAnnotations(map[string]string{constant.AnnotationHTTPFilterPolicy: incorrectWhenValidateStrictlyPolicy})
			err = c.Update(ctx, &gw)
			require.ErrorContains(t, err, "configuration is invalid: invalid HTTPFilterPolicy: ")
		},
	})
}
