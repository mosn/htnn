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
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/e2e/pkg/k8s"
	"mosn.io/htnn/e2e/pkg/suite"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func registerInstance(suite *suite.Suite, name string, ip string, port string, metadata map[string]any) error {
	params := url.Values{}
	params.Set("serviceName", name)
	params.Set("ip", ip)
	params.Set("port", port)

	if metadata != nil {
		b, _ := json.Marshal(metadata)
		params.Set("metadata", string(b))
	}

	path := "/nacos/v1/ns/instance?" + params.Encode()

	resp, err := suite.Post(path, nil, strings.NewReader(""))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("register instance failed, status code: %d", resp.StatusCode)
	}
	return nil
}

func deregisterInstance(suite *suite.Suite, name string, ip string, port string) error {
	params := url.Values{}
	params.Set("serviceName", name)
	params.Set("ip", ip)
	params.Set("port", port)

	path := "/nacos/v1/ns/instance?" + params.Encode()

	resp, err := suite.Delete(path, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("deregister instance failed, status code: %d", resp.StatusCode)
	}
	return nil
}

func init() {
	suite.Register(suite.Test{
		Run: func(t *testing.T, suite *suite.Suite) {
			c := suite.K8sClient()
			ctx := context.Background()
			nsName := types.NamespacedName{Name: "backend", Namespace: k8s.DefaultNamespace}
			var service corev1.Service
			err := c.Get(ctx, nsName, &service)
			require.NoError(t, err)
			ip := service.Spec.ClusterIP
			require.NoError(t, registerInstance(suite, "backend", ip, "8080", nil))

			time.Sleep(2 * time.Second) // wait for service registry to sync
			rsp, err := suite.Get("/echo", nil)
			require.NoError(t, err)
			req, _, err := suite.Capture(rsp)
			require.NoError(t, err)
			require.Equal(t, "hello,", req.Headers["Doraemon"][0])

			// service change
			require.NoError(t, deregisterInstance(suite, "backend", ip, "8080"))
			// to invalid ip
			require.NoError(t, registerInstance(suite, "backend", "127.0.0.1", "8080", nil))
			time.Sleep(2 * time.Second) // wait for service registry to sync
			rsp, err = suite.Get("/echo", nil)
			require.NoError(t, err)
			require.Equal(t, 503, rsp.StatusCode)

			// test CRD status
			nsName = types.NamespacedName{Name: "default", Namespace: k8s.DefaultNamespace}
			var res mosniov1.ServiceRegistry
			err = c.Get(ctx, nsName, &res)
			require.NoError(t, err)

			st := res.Status
			cd := st.Conditions[0]
			gen := res.Generation
			require.Equal(t, gen, cd.ObservedGeneration)
			require.Equal(t, metav1.ConditionTrue, cd.Status)
			require.Equal(t, "Accepted", cd.Type)
			require.Equal(t, "The resource has been accepted", cd.Message)
			require.Equal(t, "Accepted", cd.Reason)

			// test webhook
			base := client.MergeFrom(res.DeepCopy())
			res.Spec.Config = runtime.RawExtension{
				Raw: []byte(`{"rubbish":"invalid"}`),
			}
			err = c.Patch(ctx, &res, base)
			require.Error(t, err)
			require.True(t, strings.HasPrefix(err.Error(), "admission webhook"))

			// service remove
			err = c.Delete(ctx, &res)
			require.NoError(t, err)
			time.Sleep(2 * time.Second) // wait for service registry to remove
			rsp, err = suite.Get("/echo", nil)
			require.NoError(t, err)
			require.Equal(t, 500, rsp.StatusCode) // cluster not found
		},
	})
}
