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

package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func TestAddOrUpdate(t *testing.T) {
	changed := false
	p := &FilterPolicy{}
	c := metav1.Condition{
		Type:               string(gwapiv1a2.PolicyConditionAccepted),
		Reason:             "blah",
		LastTransitionTime: metav1.NewTime(time.Now()),
	}
	p.Status.Conditions, changed = addOrUpdateCondition(p.Status.Conditions, c)
	assert.Equal(t, 1, len(p.Status.Conditions))
	assert.Equal(t, c, p.Status.Conditions[0])
	assert.True(t, changed)

	p.Status.Conditions, changed = addOrUpdateCondition(p.Status.Conditions, c)
	assert.Equal(t, 1, len(p.Status.Conditions))
	assert.False(t, changed)

	update := metav1.Condition{
		Type:               string(gwapiv1a2.PolicyConditionAccepted),
		Reason:             string(gwapiv1a2.PolicyReasonInvalid),
		LastTransitionTime: metav1.NewTime(time.Now()),
	}
	p.Status.Conditions, changed = addOrUpdateCondition(p.Status.Conditions, update)
	assert.Equal(t, 1, len(p.Status.Conditions))
	assert.Equal(t, update, p.Status.Conditions[0])
	assert.True(t, changed)
}
