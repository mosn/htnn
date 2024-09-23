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

	"github.com/alibaba/sentinel-golang/core/hotspot"

	types "mosn.io/htnn/types/plugins/sentinel"
)

func LoadHotSpotRules(hs *types.HotSpot, m map[string]*types.HotSpotRule) (bool, error) {
	if hs == nil {
		return true, nil
	}

	rs := hs.GetRules()
	if len(rs) == 0 {
		return true, nil
	}

	news := make([]*hotspot.Rule, 0, len(rs))
	for _, r := range rs {
		res := r.GetResource()
		if res == "" || r == nil {
			continue
		}

		if _, exist := m[res]; exist {
			return false, fmt.Errorf("duplicate hot spot rule for resource %s", res)
		}
		m[res] = r

		sis := make(map[interface{}]int64)
		for k, v := range r.SpecificItems {
			sis[k] = v
		}

		news = append(news, &hotspot.Rule{
			ID:                r.GetId(),
			Resource:          r.GetResource(),
			MetricType:        hotspot.MetricType(r.GetMetricType()),
			ControlBehavior:   hotspot.ControlBehavior(r.GetControlBehavior()),
			ParamIndex:        int(r.GetParamIndex()),
			ParamKey:          r.GetParamKey(),
			Threshold:         r.GetThreshold(),
			MaxQueueingTimeMs: r.GetMaxQueueingTimeMs(),
			BurstCount:        r.GetBurstCount(),
			DurationInSec:     r.GetDurationInSec(),
			ParamsMaxCapacity: r.GetParamsMaxCapacity(),
			SpecificItems:     sis,
		})
	}

	return hotspot.LoadRules(news)
}
