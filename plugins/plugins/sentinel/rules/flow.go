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

    "github.com/alibaba/sentinel-golang/core/flow"
)

func LoadFlowRule(rule *types.FlowRule) (bool, error) {
    oldRules := flow.GetRules()
    newRules := make([]*flow.Rule, 0, len(oldRules)+1)
    i := 0
    for _, r := range oldRules {
        tmp := r
        newRules[i] = &tmp
        i++
    }

    newRules[i] = &flow.Rule{
        ID:                     rule.Id,
        Resource:               rule.Resource,
        TokenCalculateStrategy: flow.TokenCalculateStrategy(rule.TokenCalculateStrategy),
        ControlBehavior:        flow.ControlBehavior(rule.ControlBehavior),
        Threshold:              rule.Threshold,
        RelationStrategy:       flow.RelationStrategy(rule.RelationStrategy),
        RefResource:            rule.RefResource,
        MaxQueueingTimeMs:      rule.MaxQueueingTimeMs,
        WarmUpPeriodSec:        rule.WarmUpPeriodSec,
        WarmUpColdFactor:       rule.WarmUpColdFactor,
        StatIntervalInMs:       rule.StatIntervalInMs,
        LowMemUsageThreshold:   rule.LowMemUsageThreshold,
        HighMemUsageThreshold:  rule.HighMemUsageThreshold,
        MemLowWaterMarkBytes:   rule.MemLowWaterMarkBytes,
        MemHighWaterMarkBytes:  rule.MemHighWaterMarkBytes,
    }
    return flow.LoadRules(newRules)
}
