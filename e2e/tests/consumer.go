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
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/e2e/pkg/k8s"
	"mosn.io/htnn/e2e/pkg/suite"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func hdrWithKey(name string) http.Header {
	hdr := http.Header{}
	hdr.Set("Authorization", name)
	return hdr
}

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
			rsp, _ = suite.Get("/echo", hdrWithKey("morty"))
			require.Equal(t, 200, rsp.StatusCode)

			rsp, _ = suite.Get("/echo", hdrWithKey("doraemon"))
			require.Equal(t, 401, rsp.StatusCode)

			c := suite.K8sClient()
			ctx := context.Background()
			nsName := types.NamespacedName{Name: "morty", Namespace: k8s.DefaultNamespace}
			var consumer mosniov1.Consumer
			err = c.Get(ctx, nsName, &consumer)
			require.NoError(t, err)
			base := client.MergeFrom(consumer.DeepCopy())
			consumer.Spec.Filters = map[string]mosniov1.HTTPPlugin{
				"limitReq": {
					Config: runtime.RawExtension{
						Raw: []byte(`{"average":1}`),
					},
				},
			}
			err = c.Patch(ctx, &consumer, base)
			require.NoError(t, err)
			nsName = types.NamespacedName{Name: "rick", Namespace: k8s.DefaultNamespace}
			err = c.Get(ctx, nsName, &consumer)
			require.NoError(t, err)
			err = c.Delete(ctx, &consumer)
			require.NoError(t, err)

			time.Sleep(1 * time.Second)

			rsp, _ = suite.Get("/echo", hdrWithKey("morty"))
			require.Equal(t, 200, rsp.StatusCode)
			rsp, _ = suite.Get("/echo", hdrWithKey("morty"))
			require.Equal(t, 429, rsp.StatusCode)

			rsp, _ = suite.Get("/echo", hdrWithKey("rick"))
			require.Equal(t, 401, rsp.StatusCode)
		},
	})
}
