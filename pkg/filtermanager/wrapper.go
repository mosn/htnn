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
	"mosn.io/htnn/pkg/filtermanager/api"
)

type logExecutionFilter struct {
	// Don't inherit the PassThroughFilter
	name     string
	internal api.Filter
}

func NewLogExecutionFilter(name string, internal api.Filter) api.Filter {
	return &logExecutionFilter{
		name:     name,
		internal: internal,
	}
}

func (f *logExecutionFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	api.LogDebugf("run plugin %s, method: DecodeHeaders", f.name)
	return f.internal.DecodeHeaders(headers, endStream)
}

func (f *logExecutionFilter) DecodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogDebugf("run plugin %s, method: DecodeData", f.name)
	return f.internal.DecodeData(data, endStream)
}

func (f *logExecutionFilter) DecodeTrailers(trailers api.RequestTrailerMap) api.ResultAction {
	api.LogDebugf("run plugin %s, method: DecodeTrailers", f.name)
	return f.internal.DecodeTrailers(trailers)
}

func (f *logExecutionFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	api.LogDebugf("run plugin %s, method: EncodeHeaders", f.name)
	return f.internal.EncodeHeaders(headers, endStream)
}

func (f *logExecutionFilter) EncodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogDebugf("run plugin %s, method: EncodeData", f.name)
	return f.internal.EncodeData(data, endStream)
}

func (f *logExecutionFilter) EncodeTrailers(trailers api.ResponseTrailerMap) api.ResultAction {
	api.LogDebugf("run plugin %s, method: EncodeTrailers", f.name)
	return f.internal.EncodeTrailers(trailers)
}

func (f *logExecutionFilter) OnLog() {
	api.LogDebugf("run plugin %s, method: OnLog", f.name)
	f.internal.OnLog()
}

func (f *logExecutionFilter) DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction {
	api.LogDebugf("run plugin %s, method: DecodeRequest", f.name)
	return f.internal.DecodeRequest(headers, data, trailers)
}

func (f *logExecutionFilter) EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	api.LogDebugf("run plugin %s, method: EncodeResponse", f.name)
	return f.internal.EncodeResponse(headers, data, trailers)
}
