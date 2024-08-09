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

	"github.com/alibaba/sentinel-golang/core/system"
)

func LoadSystemRule(rule *types.SystemRule) (bool, error) {
	oldRules := system.GetRules()
	newRules := make([]*system.Rule, 0, len(oldRules)+1)
	i := 0
	for _, r := range oldRules {
		tmp := r
		newRules[i] = &tmp
		i++
	}

	strategy := system.BBR
	if rule.Strategy == types.SystemRule_NO_ADAPTIVE {
		strategy = system.NoAdaptive
	}
	newRules[i] = &system.Rule{
		ID:           rule.Id,
		MetricType:   system.MetricType(rule.MetricType),
		TriggerCount: rule.TriggerCount,
		Strategy:     strategy,
	}
	return system.LoadRules(newRules)
}
