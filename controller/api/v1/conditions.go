package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func needUpdateCondition(a, b metav1.Condition) bool {
	return (a.Status != b.Status) ||
		(a.Reason != b.Reason) ||
		(a.Message != b.Message) ||
		(a.ObservedGeneration != b.ObservedGeneration)
}

func addOrUpdateCondition(conditions []metav1.Condition, one metav1.Condition) ([]metav1.Condition, bool) {
	add := true
	changed := false
	for i, cond := range conditions {
		if cond.Type == one.Type {
			add = false
			if needUpdateCondition(cond, one) {
				changed = true
				conditions[i].Status = one.Status
				conditions[i].Reason = one.Reason
				conditions[i].Message = one.Message
				conditions[i].ObservedGeneration = one.ObservedGeneration
				conditions[i].LastTransitionTime = one.LastTransitionTime
				break
			}
		}
	}
	if add {
		return append(conditions, one), true
	}
	return conditions, changed
}
