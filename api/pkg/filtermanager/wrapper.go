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

package filtermanager

import (
	"time"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/filtermanager/model"
)

type logExecutionFilter struct {
	// Don't inherit the PassThroughFilter
	name      string
	internal  api.Filter
	callbacks api.FilterCallbackHandler
}

func NewLogExecutionFilter(name string, internal api.Filter, callbacks api.FilterCallbackHandler) api.Filter {
	return &logExecutionFilter{
		name:      name,
		internal:  internal,
		callbacks: callbacks,
	}
}

func (f *logExecutionFilter) id() string {
	name := f.callbacks.StreamInfo().GetRouteName()
	if name != "" {
		return "route " + name
	}
	vc, ok := f.callbacks.StreamInfo().VirtualClusterName()
	if ok {
		return "virtual cluster " + vc
	}
	return "filter chain " + f.callbacks.StreamInfo().FilterChainName()
}

func (f *logExecutionFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	api.LogDebugf("%s run plugin %s, method: DecodeHeaders", f.id(), f.name)
	r := f.internal.DecodeHeaders(headers, endStream)
	api.LogDebugf("%s finish running plugin %s, method: DecodeHeaders", f.id(), f.name)
	return r
}

func (f *logExecutionFilter) DecodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogDebugf("%s run plugin %s, method: DecodeData", f.id(), f.name)
	r := f.internal.DecodeData(data, endStream)
	api.LogDebugf("%s finish running plugin %s, method: DecodeData", f.id(), f.name)
	return r
}

func (f *logExecutionFilter) DecodeTrailers(trailers api.RequestTrailerMap) api.ResultAction {
	api.LogDebugf("%s run plugin %s, method: DecodeTrailers", f.id(), f.name)
	r := f.internal.DecodeTrailers(trailers)
	api.LogDebugf("%s finish running plugin %s, method: DecodeTrailers", f.id(), f.name)
	return r
}

func (f *logExecutionFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	api.LogDebugf("%s run plugin %s, method: EncodeHeaders", f.id(), f.name)
	r := f.internal.EncodeHeaders(headers, endStream)
	api.LogDebugf("%s finish running plugin %s, method: EncodeHeaders", f.id(), f.name)
	return r
}

func (f *logExecutionFilter) EncodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogDebugf("%s run plugin %s, method: EncodeData", f.id(), f.name)
	r := f.internal.EncodeData(data, endStream)
	api.LogDebugf("%s finish running plugin %s, method: EncodeData", f.id(), f.name)
	return r
}

func (f *logExecutionFilter) EncodeTrailers(trailers api.ResponseTrailerMap) api.ResultAction {
	api.LogDebugf("%s run plugin %s, method: EncodeTrailers", f.id(), f.name)
	r := f.internal.EncodeTrailers(trailers)
	api.LogDebugf("%s finish running plugin %s, method: EncodeTrailers", f.id(), f.name)
	return r
}

func (f *logExecutionFilter) OnLog(reqHeaders api.RequestHeaderMap, reqTrailers api.RequestTrailerMap,
	respHeaders api.ResponseHeaderMap, respTrailers api.ResponseTrailerMap) {

	api.LogDebugf("%s run plugin %s, method: OnLog", f.id(), f.name)
	f.internal.OnLog(reqHeaders, reqTrailers, respHeaders, respTrailers)
	api.LogDebugf("%s finish running plugin %s, method: OnLog", f.id(), f.name)
}

func (f *logExecutionFilter) DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction {
	api.LogDebugf("%s run plugin %s, method: DecodeRequest", f.id(), f.name)
	r := f.internal.DecodeRequest(headers, data, trailers)
	api.LogDebugf("%s finish running plugin %s, method: DecodeRequest", f.id(), f.name)
	return r
}

func (f *logExecutionFilter) EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	api.LogDebugf("%s run plugin %s, method: EncodeResponse", f.id(), f.name)
	r := f.internal.EncodeResponse(headers, data, trailers)
	api.LogDebugf("%s finish running plugin %s, method: EncodeResponse", f.id(), f.name)
	return r
}

type debugFilter struct {
	// Don't inherit the PassThroughFilter
	name      string
	internal  api.Filter
	callbacks api.FilterCallbackHandler
}

func NewDebugFilter(name string, internal api.Filter, callbacks api.FilterCallbackHandler) api.Filter {
	return &debugFilter{
		name:      name,
		internal:  internal,
		callbacks: callbacks,
	}
}

func (f *debugFilter) recordExecution(start time.Time, method string) {
	duration := time.Since(start)
	executionRecords := f.callbacks.PluginState().Get("debugMode", "executionRecords")
	if executionRecords == nil {
		executionRecords = []model.ExecutionRecord{}
		f.callbacks.PluginState().Set("debugMode", "executionRecords", executionRecords)
	}

	records := executionRecords.([]model.ExecutionRecord)
	for _, record := range records {
		if record.PluginName == f.name {
			record.Record[method] += duration
			return
		}
	}
	f.callbacks.PluginState().Set("debugMode", "executionRecords", append(records, model.ExecutionRecord{
		PluginName: f.name,
		Record:     map[string]time.Duration{method: duration},
	}))
}

func (f *debugFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	defer f.recordExecution(time.Now(), "DecodeHeaders")
	return f.internal.DecodeHeaders(headers, endStream)
}

func (f *debugFilter) DecodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	defer f.recordExecution(time.Now(), "DecodeData")
	return f.internal.DecodeData(data, endStream)
}

func (f *debugFilter) DecodeTrailers(trailers api.RequestTrailerMap) api.ResultAction {
	defer f.recordExecution(time.Now(), "DecodeTrailers")
	return f.internal.DecodeTrailers(trailers)
}

func (f *debugFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	defer f.recordExecution(time.Now(), "EncodeHeaders")
	return f.internal.EncodeHeaders(headers, endStream)
}

func (f *debugFilter) EncodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	defer f.recordExecution(time.Now(), "EncodeData")
	return f.internal.EncodeData(data, endStream)
}

func (f *debugFilter) EncodeTrailers(trailers api.ResponseTrailerMap) api.ResultAction {
	defer f.recordExecution(time.Now(), "EncodeTrailers")
	return f.internal.EncodeTrailers(trailers)
}

func (f *debugFilter) OnLog(reqHeaders api.RequestHeaderMap, reqTrailers api.RequestTrailerMap,
	respHeaders api.ResponseHeaderMap, respTrailers api.ResponseTrailerMap) {

	// The OnLog phase doesn't contribute to the request duration, so we don't need to count it
	f.internal.OnLog(reqHeaders, reqTrailers, respHeaders, respTrailers)
}

func (f *debugFilter) DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction {
	defer f.recordExecution(time.Now(), "DecodeRequest")
	return f.internal.DecodeRequest(headers, data, trailers)
}

func (f *debugFilter) EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	defer f.recordExecution(time.Now(), "EncodeResponse")
	return f.internal.EncodeResponse(headers, data, trailers)
}
