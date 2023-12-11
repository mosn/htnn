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
	p := &HTTPFilterPolicy{}
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
