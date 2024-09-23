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
	"fmt"

	"github.com/alibaba/sentinel-golang/core/circuitbreaker"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	types "mosn.io/htnn/types/plugins/sentinel"
)

func LoadCircuitBreakerRules(cb *types.CircuitBreaker, m map[string]*types.CircuitBreakerRule) (bool, error) {
	if cb == nil {
		return true, nil
	}

	rs := cb.GetRules()
	if len(rs) == 0 {
		return true, nil
	}

	news := make([]*circuitbreaker.Rule, 0, len(rs))
	for _, r := range rs {
		res := r.GetResource()
		if res == "" || r == nil {
			continue
		}

		if _, exist := m[res]; exist {
			return false, fmt.Errorf("duplicate circuit breaker rule for resource %s", res)
		}
		m[res] = r

		if _, exist := listeners[res]; !exist {
			lsn := &debugListener{res}
			listeners[res] = lsn
			circuitbreaker.RegisterStateChangeListeners(lsn)
			// api.LogDebugf("registered state change listener for resource: %s", res)
		}

		news = append(news, &circuitbreaker.Rule{
			Id:                           r.GetId(),
			Resource:                     r.GetResource(),
			Strategy:                     circuitbreaker.Strategy(r.GetStrategy()),
			RetryTimeoutMs:               r.GetRetryTimeoutMs(),
			MinRequestAmount:             r.GetMinRequestAmount(),
			StatIntervalMs:               r.GetStatIntervalMs(),
			StatSlidingWindowBucketCount: r.GetStatSlidingWindowBucketCount(),
			MaxAllowedRtMs:               r.GetMaxAllowedRtMs(),
			Threshold:                    r.GetThreshold(),
			ProbeNum:                     r.GetProbeNum(),
		})
	}

	return circuitbreaker.LoadRules(news)
}

var listeners = make(debugListenerMap)

type debugListenerMap map[string]*debugListener

type debugListener struct {
	res string
}

func (s *debugListener) OnTransformToClosed(prev circuitbreaker.State, rule circuitbreaker.Rule) {
	api.LogDebugf("[circuitbreaker state change] resource: %s, steategy: %+v, %s -> Closed", s.res, rule.Strategy, prev.String())
}

func (s *debugListener) OnTransformToOpen(prev circuitbreaker.State, rule circuitbreaker.Rule, snapshot interface{}) {
	api.LogDebugf("[circuitbreaker state change] resource: %s, steategy: %+v, %s -> Open, failed times: %d", s.res, rule.Strategy, prev.String(), snapshot)
}

func (s *debugListener) OnTransformToHalfOpen(prev circuitbreaker.State, rule circuitbreaker.Rule) {
	api.LogDebugf("[circuitbreaker state change] resource: %s, steategy: %+v, %s -> Half-Open", s.res, rule.Strategy, prev.String())
}
