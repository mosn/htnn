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

package sentinel

import (
	"fmt"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

const (
	Name = "sentinel"
)

func init() {
	plugins.RegisterPluginType(Name, &Plugin{})
}

type Plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *Plugin) Type() plugins.PluginType {
	return plugins.TypeTraffic
}

func (p *Plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionTraffic,
	}
}

func (p *Plugin) Config() api.PluginConfig {
	return &CustomConfig{}
}

type CustomConfig struct {
	Config
}

func (conf *CustomConfig) Validate() error {
	err := conf.Config.Validate()
	if err != nil {
		return err
	}

	if conf.GetFlow() == nil && conf.GetHotSpot() == nil && conf.GetCircuitBreaker() == nil {
		return fmt.Errorf("config must have at least one of 'flow', 'hotSpot', 'circuitBreaker'")
	}

	if err = conf.checkFlow(); err != nil {
		return err
	}

	if err = conf.checkHotSpot(); err != nil {
		return err
	}

	if err = conf.checkCircuitBreaker(); err != nil {
		return err
	}

	return nil
}

func (conf *CustomConfig) checkFlow() error {
	flow := conf.GetFlow()
	if flow == nil {
		return nil
	}

	for _, r := range flow.GetRules() {
		if r == nil {
			continue
		}

		if r.GetMaxQueueingTimeMs() > 0 && r.GetControlBehavior() != ControlBehavior_THROTTLING {
			return fmt.Errorf("wrong config: flow resource %s,  'maxQueueingTimeMs' needs to be set only when 'controlBehavior' == THROTTLING", r.GetResource())
		}

		if r.GetThreshold() == 0 && r.GetRelationStrategy() != FlowRule_ASSOCIATED_RESOURCE {
			return fmt.Errorf("wrong config: flow resource %s, 'threshold' must be greater than 0 or 'relationStrategy' == ASSOCIATED_RESOURCE", r.GetResource())
		}

		if r.GetRelationStrategy() == FlowRule_ASSOCIATED_RESOURCE && r.GetRefResource() == "" {
			return fmt.Errorf("wrong config: flow resource %s,  'refResource' must not be empty when 'relationStrategy' == ASSOCIATED_RESOURCE", r.GetResource())
		}

		if r.GetThreshold() > 0 && r.GetStatIntervalInMs() == 0 {
			r.StatIntervalInMs = 1000
		}

		r.BlockResponse = checkBlockResp(r.GetBlockResponse())
	}

	return nil
}

func (conf *CustomConfig) checkHotSpot() error {
	hs := conf.GetHotSpot()
	if hs == nil {
		return nil
	}

	if hs.GetParams() == nil && hs.GetAttachments() == nil {
		return fmt.Errorf("wrong config: hot spot, 'params' and 'attachments' cannot both be empty")
	}
	for _, r := range hs.GetRules() {
		if r.GetParamKey() != "" && len(hs.GetAttachments()) == 0 {
			return fmt.Errorf("wrong config: hot spot %s, 'attachments' must not be empty when 'paramKey' is set", r.GetResource())
		}

		if r.GetMetricType() == HotSpotRule_QPS && r.GetDurationInSec() == 0 {
			r.DurationInSec = 1
		}

		r.BlockResponse = checkBlockResp(r.GetBlockResponse())
	}

	return nil
}

func (conf *CustomConfig) checkCircuitBreaker() error {
	cb := conf.GetCircuitBreaker()
	if cb == nil {
		return nil
	}

	for _, r := range cb.GetRules() {
		if r.GetRetryTimeoutMs() == 0 {
			r.RetryTimeoutMs = 3000
		}

		if r.GetStatIntervalMs() == 0 {
			r.StatIntervalMs = 1000
		}

		if r.GetTriggeredByStatusCodes() == nil {
			r.TriggeredByStatusCodes = []uint32{500}
		}

		if r.GetStatSlidingWindowBucketCount() > 0 {
			if r.GetStatIntervalMs()%r.GetStatSlidingWindowBucketCount() != 0 {
				return fmt.Errorf("wrong config: circuit breaker %s, must 'statIntervalMs' %% 'statSlidingWindowBucketCount' == 0", r.GetResource())
			}
		}

		r.BlockResponse = checkBlockResp(r.GetBlockResponse())
	}

	return nil
}

func checkBlockResp(br *BlockResponse) *BlockResponse {
	if br == nil {
		return &BlockResponse{
			Message:    "sentinel traffic control",
			StatusCode: 429,
		}
	}
	if br.GetStatusCode() == 0 {
		br.StatusCode = 429
	}
	return br
}
