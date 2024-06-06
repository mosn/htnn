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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	ConditionAccepted ConditionType = "Accepted"
)

type ConditionReason string

const (
	ReasonAccepted ConditionReason = "Accepted"
	ReasonInvalid  ConditionReason = "Invalid"
)

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

func addOrUpdateAcceptedCondition(conditions []metav1.Condition,
	observedGeneration int64, reason ConditionReason, msg ...string) ([]metav1.Condition, bool) {

	c := metav1.Condition{
		Type:               string(ConditionAccepted),
		Reason:             string(reason),
		LastTransitionTime: metav1.NewTime(time.Now()),
		ObservedGeneration: observedGeneration,
	}
	switch reason {
	case ReasonAccepted:
		c.Status = metav1.ConditionTrue
		c.Message = "The resource has been accepted"
	case ReasonInvalid:
		c.Status = metav1.ConditionFalse
		if len(msg) > 0 {
			c.Message = msg[0]
		} else {
			c.Message = "The resource is invalid"
		}
	}
	return addOrUpdateCondition(conditions, c)
}

type ChangeDetector struct {
	changed bool
}

func (s *ChangeDetector) IsChanged() bool {
	return s.changed
}

func (s *ChangeDetector) MarkAsChanged() {
	s.changed = true
}

func (s *ChangeDetector) Reset() {
	s.changed = false
}
