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

import types "mosn.io/htnn/types/plugins/sentinel"

type Rule interface {
}

func Load(configType types.Config_Type, rule interface{}) (bool, error) {
	switch configType {
	case types.Config_FLOW:
		flowRule, ok := rule.(types.FlowRule)
		if !ok {
			return false, nil
		}
		return LoadFlowRule(&flowRule)
	case types.Config_HOT_SPOT:
		hotSpotRule, ok := rule.(types.HotSpotRule)
		if !ok {
			return false, nil
		}
		return LoadHotSpotRule(&hotSpotRule)
	case types.Config_ISOLATION:
		isolationRule, ok := rule.(types.IsolationRule)
		if !ok {
			return false, nil
		}
		return LoadIsolationRule(&isolationRule)
	case types.Config_CIRCUIT_BREAKER:
		circuitBreakerRule, ok := rule.(types.CircuitBreakerRule)
		if !ok {
			return false, nil
		}
		return LoadCircuitBreakerRule(&circuitBreakerRule)
	case types.Config_SYSTEM:
		systemRule, ok := rule.(types.SystemRule)
		if !ok {
			return false, nil
		}
		return LoadSystemRule(&systemRule)
	default:
		return false, nil
	}
}
