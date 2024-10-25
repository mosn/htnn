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

package debugmode

import (
	"encoding/json"
	"time"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/types/plugins/debugmode"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &filter{
		callbacks: callbacks,
		config:    c.(*debugmode.Config),
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *debugmode.Config
}

type executionPlugin struct {
	Name        string  `json:"name"`
	CostSeconds float64 `json:"cost_seconds"`
}

type SlowLogReport struct {
	TotalSeconds float64 `json:"total_seconds"`

	Request struct {
		Headers map[string][]string `json:"headers"`
	} `json:"request"`
	Response struct {
		Headers map[string][]string `json:"headers,omitempty"`
	} `json:"response"`
	StreamInfo struct {
		DownstreamRemoteAddress string `json:"downstream_remote_address"`
		UpstreamRemoteAddress   string `json:"upstream_remote_address,omitempty"`
	} `json:"stream_info"`

	// Note: the ExecutedPlugins don't contain plugins executed in OnLog phase
	ExecutedPlugins []executionPlugin `json:"executed_plugins,omitempty"`
}

func (f *filter) OnLog(reqHeaders api.RequestHeaderMap, reqTrailers api.RequestTrailerMap,
	respHeaders api.ResponseHeaderMap, respTrailers api.ResponseTrailerMap) {

	config := f.config

	slowLog := config.GetSlowLog()
	if slowLog != nil {
		dur, err := f.callbacks.GetProperty("request.duration")
		if err != nil {
			api.LogErrorf("unexpected err when getting request duration: %v", err)
			return
		}

		d, err := time.ParseDuration(dur)
		if err != nil {
			api.LogErrorf("unexpected err when parsing request duration: %v", err)
			return
		}

		if d > slowLog.GetThreshold().AsDuration() {
			report := &SlowLogReport{
				TotalSeconds: d.Seconds(),
			}

			report.StreamInfo.DownstreamRemoteAddress = f.callbacks.StreamInfo().DownstreamRemoteAddress()
			report.StreamInfo.UpstreamRemoteAddress, _ = f.callbacks.StreamInfo().UpstreamRemoteAddress()

			report.Request.Headers = make(map[string][]string)
			reqHeaders.Range(func(key, value string) bool {
				report.Request.Headers[key] = append(report.Request.Headers[key], value)
				return true
			})

			if respHeaders != nil {
				report.Response.Headers = make(map[string][]string)
				respHeaders.Range(func(key, value string) bool {
					report.Response.Headers[key] = append(report.Response.Headers[key], value)
					return true
				})
			}

			// This is a private API and we don't guarantee its stablibity
			r := f.callbacks.PluginState().Get("debugMode", "executionRecords")
			if r != nil {
				executionRecords := r.([]*model.ExecutionRecord)
				for _, record := range executionRecords {
					p := executionPlugin{
						Name:        record.PluginName,
						CostSeconds: record.Record.Seconds(),
					}
					report.ExecutedPlugins = append(report.ExecutedPlugins, p)
				}
			}

			b, _ := json.Marshal(report)
			api.LogErrorf("slow log report: %s", b)
		}
	}
}
