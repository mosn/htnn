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

package rules

import (
	types "mosn.io/htnn/types/plugins/sentinel"

	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
)

func LoadCircuitBreakerRule(rule *types.CircuitBreakerRule) (bool, error) {
	oldRules := circuitbreaker.GetRules()
	newRules := make([]*circuitbreaker.Rule, 0, len(oldRules)+1)
	i := 0
	for _, r := range oldRules {
		tmp := r
		newRules[i] = &tmp
		i++
	}

	newRules[i] = &circuitbreaker.Rule{
		Id:                           rule.Id,
		Resource:                     rule.Resource,
		Strategy:                     circuitbreaker.Strategy(rule.Strategy),
		RetryTimeoutMs:               rule.RetryTimeoutMs,
		MinRequestAmount:             rule.MinRequestAmount,
		StatIntervalMs:               rule.StatIntervalMs,
		StatSlidingWindowBucketCount: rule.StatSlidingWindowBucketCount,
		MaxAllowedRtMs:               rule.MaxAllowedRtMs,
		Threshold:                    rule.Threshold,
		ProbeNum:                     rule.ProbeNum,
	}
	return circuitbreaker.LoadRules(newRules)
}
