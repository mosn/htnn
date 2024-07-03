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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/e2e/pkg/k8s"
	"mosn.io/htnn/e2e/pkg/suite"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func init() {
	suite.Register(suite.Test{
		Run: func(t *testing.T, suite *suite.Suite) {
			// check if deny policy works
			rsp, err := suite.Get("/echo", nil)
			require.Error(t, err)

			c := suite.K8sClient()
			ctx := context.Background()
			nsName := types.NamespacedName{Name: "policy", Namespace: k8s.DefaultNamespace}
			var policy mosniov1.FilterPolicy
			err = c.Get(ctx, nsName, &policy)
			require.NoError(t, err)
			prevName := policy.Spec.TargetRef.Name
			base := client.MergeFrom(policy.DeepCopy())
			// let the deny policy point to nowhere
			policy.Spec.TargetRef.Name = "nowhere"
			err = c.Patch(ctx, &policy, base)
			require.NoError(t, err)

			time.Sleep(1 * time.Second)
			rsp, err = suite.Get("/echo", nil)
			require.NoError(t, err)
			require.Equal(t, 200, rsp.StatusCode)

			base = client.MergeFrom(policy.DeepCopy())
			// restore the policy
			policy.Spec.TargetRef.Name = prevName
			err = c.Patch(ctx, &policy, base)
			require.NoError(t, err)

			time.Sleep(1 * time.Second)
			rsp, err = suite.Get("/echo", nil)
			require.Error(t, err)
		},
	})
}
