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

	"github.com/alibaba/sentinel-golang/core/flow"

	types "mosn.io/htnn/types/plugins/sentinel"
)

func LoadFlowRules(f *types.Flow, m map[string]*types.FlowRule) (bool, error) {
	if f == nil {
		return true, nil
	}

	rs := f.GetRules()
	if len(rs) == 0 {
		return true, nil
	}

	news := make([]*flow.Rule, 0, len(rs))
	for _, r := range rs {
		res := r.GetResource()
		if res == "" || r == nil {
			continue
		}

		if _, exist := m[res]; exist {
			return false, fmt.Errorf("duplicate flow rule for resource %s", res)
		}
		m[res] = r

		news = append(news, &flow.Rule{
			ID:                     r.GetId(),
			Resource:               r.GetResource(),
			TokenCalculateStrategy: flow.TokenCalculateStrategy(r.GetTokenCalculateStrategy()),
			ControlBehavior:        flow.ControlBehavior(r.GetControlBehavior()),
			Threshold:              r.GetThreshold(),
			RelationStrategy:       flow.RelationStrategy(r.GetRelationStrategy()),
			RefResource:            r.GetRefResource(),
			MaxQueueingTimeMs:      r.GetMaxQueueingTimeMs(),
			WarmUpPeriodSec:        r.GetWarmUpPeriodSec(),
			WarmUpColdFactor:       r.GetWarmUpColdFactor(),
			StatIntervalInMs:       r.GetStatIntervalInMs(),
			LowMemUsageThreshold:   r.GetLowMemUsageThreshold(),
			HighMemUsageThreshold:  r.GetHighMemUsageThreshold(),
			MemLowWaterMarkBytes:   r.GetMemLowWaterMarkBytes(),
			MemHighWaterMarkBytes:  r.GetMemHighWaterMarkBytes(),
		})
	}

	return flow.LoadRules(news)
}
