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

package plugins

import (
	"net/http"
	"strings"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

const (
	Name = "yourPlugin"
)

func init() {
	plugins.RegisterPlugin(Name, &plugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &Config{}
}

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &filter{
		callbacks: callbacks,
		config:    c.(*Config),
	}
}

type filter struct {
	api.PassThroughFilter

	config    *Config
	callbacks api.FilterCallbackHandler
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	hdr := http.Header{}
	hdr.Set("content-type", "text/plain")
	allHeaders := headers.GetAllHeaders()
	msg := "Headers:\n"
	for k, v := range allHeaders {
		msg += k + ": " + strings.Join(v, ",") + "\n"
	}
	return &api.LocalResponse{
		Code:   200,
		Msg:    "Your plugin is running\n" + msg,
		Header: hdr,
	}
}

// DecodeData is for processing request body with streaming API.
// The endStream is true when handling the last piece of the body.
func (f *filter) DecodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	// DecodeData won't be called if return api.WaitAllData in DecodeHeaders.
	api.LogInfof("Decode partial request body: %s", data.String())

	// Please note return api.WaitAllData is only allowed to be used in DecodeHeaders.
	return api.Continue
}

// DecodeRequest is for processing the full request body with buffered data.
func (f *filter) DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction {
	// DecodeRequest is called if DecodeHeaders has return api.WaitAllData, which means the full request body is buffered.
	api.LogInfof("Decode full request body: %s", data.String())

	// Please note return api.WaitAllData is only allowed to be used in DecodeHeaders.
	return api.Continue
}

// EncodeHeaders processes response headers. The endStream is true if the response doesn't have body
func (f *filter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	headers.Add("my-plugin", "running")
	return api.Continue
}

// EncodeData might be called multiple times during handling the response body for scenarios like streaming API.
// The endStream is true when handling the last piece of the body.
func (f *filter) EncodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	// EncodeData won't be called if return api.WaitAllData in EncodeHeaders.
	body := data.String()
	// do something with the response body
	if strings.Contains(body, "error") {
		return &api.LocalResponse{
			Code: 500,
			Msg:  "error",
		}
	}
	api.LogInfof("Encode partial response body: %s", data.String())
	// Please note return api.WaitAllData is only allowed to be used in EncodeHeaders.
	return api.Continue
}

func (f *filter) EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	// EncodeResponse is called if EncodeHeaders has return api.WaitAllData, which means the full response body is buffered.
	api.LogInfof("Encode full response body: %s", data.String())

	// Please note return api.WaitAllData is only allowed to be used in EncodeHeaders.
	return api.Continue
}

// OnLog is called when the HTTP stream is ended on HTTP Connection Manager filter.
func (f *filter) OnLog(reqHeaders api.RequestHeaderMap, reqTrailers api.RequestTrailerMap,
	respHeaders api.ResponseHeaderMap, respTrailers api.ResponseTrailerMap) {
	api.LogInfo("this is my plugin log")
}
