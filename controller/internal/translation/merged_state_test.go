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

package translation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func TestSortFilterPolicy(t *testing.T) {
	ps := []*FilterPolicyWrapper{
		{FilterPolicy: &mosniov1.FilterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gateway-policy",
			},
		}, scope: PolicyScopeGateway},
		{FilterPolicy: &mosniov1.FilterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "route-policy",
				CreationTimestamp: metav1.Time{
					Time: metav1.Now().Add(-1000),
				},
			},
		}, scope: PolicyScopeRoute},
		{FilterPolicy: &mosniov1.FilterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "route-policy-embeded",
			},
		}, scope: PolicyScopeRoute},
		{FilterPolicy: &mosniov1.FilterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "route-policy-latest",
				CreationTimestamp: metav1.Time{
					Time: metav1.Now().Add(-1),
				},
			},
		}, scope: PolicyScopeRoute},
		{FilterPolicy: &mosniov1.FilterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "route-policy-different-name",
				CreationTimestamp: metav1.Time{
					Time: metav1.Now().Add(-1000),
				},
			},
		}, scope: PolicyScopeRoute},
	}
	sortFilterPolicy(ps)

	assert.Equal(t, "route-policy-embeded", ps[0].GetName())
	assert.Equal(t, "route-policy", ps[1].GetName())
	assert.Equal(t, "route-policy-different-name", ps[2].GetName())
	assert.Equal(t, "route-policy-latest", ps[3].GetName())
	assert.Equal(t, "gateway-policy", ps[4].GetName())
}
