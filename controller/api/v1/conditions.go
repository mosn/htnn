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
