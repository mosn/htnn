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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/e2e/pkg/k8s"
	"mosn.io/htnn/e2e/pkg/suite"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func init() {
	suite.Register(suite.Test{
		Manifests: []string{"base/httproute.yml"},
		Run: func(t *testing.T, suite *suite.Suite) {
			rsp, err := suite.Get("/echo", nil)
			require.NoError(t, err)
			require.Equal(t, 200, rsp.StatusCode)
			require.Equal(t, 1, len(rsp.Header["Demokey"]), rsp)
			require.Equal(t, "value", rsp.Header["Demokey"][0])

			c := suite.K8sClient()
			ctx := context.Background()
			nsName := types.NamespacedName{Name: "demo", Namespace: k8s.DefaultNamespace}
			var dynamicConfig mosniov1.DynamicConfig
			err = c.Get(ctx, nsName, &dynamicConfig)
			require.NoError(t, err)

			// check status
			st := dynamicConfig.Status
			cd := st.Conditions[0]
			gen := dynamicConfig.Generation
			require.Equal(t, gen, cd.ObservedGeneration)
			require.Equal(t, metav1.ConditionTrue, cd.Status)
			require.Equal(t, "Accepted", cd.Type)
			require.Equal(t, "The resource has been accepted", cd.Message)
			require.Equal(t, "Accepted", cd.Reason)

			// update
			base := client.MergeFrom(dynamicConfig.DeepCopy())
			dynamicConfig.Spec.Config.Raw = []byte(`{"key":"value2"}`)
			err = c.Patch(ctx, &dynamicConfig, base)
			require.NoError(t, err)

			time.Sleep(1 * time.Second)
			rsp, _ = suite.Get("/echo", nil)
			require.Equal(t, 1, len(rsp.Header["Demokey"]), rsp)
			require.Equal(t, "value2", rsp.Header["Demokey"][0])

			// test webhook
			base = client.MergeFrom(dynamicConfig.DeepCopy())
			dynamicConfig.Spec.Config.Raw = []byte(`{"key":""}`)
			err = c.Patch(ctx, &dynamicConfig, base)
			require.Error(t, err)
			require.True(t, strings.HasPrefix(err.Error(), "admission webhook"))

			// remove
			err = c.Delete(ctx, &dynamicConfig)
			require.NoError(t, err)

			time.Sleep(1 * time.Second)
			rsp, _ = suite.Get("/echo", nil)
			// delete the resource won't trigger the OnUpdate callback
			require.Equal(t, 1, len(rsp.Header["Demokey"]), rsp)
			require.Equal(t, "value2", rsp.Header["Demokey"][0])
		},
	})
}
