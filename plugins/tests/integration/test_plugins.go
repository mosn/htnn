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

package integration

import (
	"net/http"
	"runtime/debug"
	"strings"

	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/plugins"
)

type basePlugin struct {
}

func (p basePlugin) Config() plugins.PluginConfig {
	return &Config{}
}

type streamPlugin struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func streamConfigFactory(c interface{}) api.FilterFactory {
	conf := c.(*Config)
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &streamFilter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}

type streamFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *Config
}

func (f *streamFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "stream")
	return api.Continue
}

func (f *streamFilter) DecodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Decode {
		data.AppendString("stream\n")
	}
	return api.Continue
}

func (f *streamFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "stream")
	headers.Del("content-length")
	return api.Continue
}

func (f *streamFilter) EncodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Encode {
		data.AppendString("stream\n")
	}
	return api.Continue
}

func (p *streamPlugin) ConfigFactory() api.FilterConfigFactory {
	return streamConfigFactory
}

type bufferPlugin struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func bufferConfigFactory(c interface{}) api.FilterFactory {
	conf := c.(*Config)
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &bufferFilter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}

type bufferFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *Config
}

func (f *bufferFilter) DecodeRequest(headers api.RequestHeaderMap, buf api.BufferInstance, trailer api.RequestTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "buffer")
	if buf != nil && f.config.Decode {
		buf.AppendString("buffer\n")
	}
	return api.Continue
}

func (f *bufferFilter) EncodeResponse(headers api.ResponseHeaderMap, buf api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "buffer")
	headers.Del("content-length")
	if buf != nil && f.config.Encode {
		buf.AppendString("buffer\n")
	}
	return api.Continue
}

func (f *bufferFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	_, ok := headers.Get("stream")
	if !ok && f.config.Need {
		return api.WaitAllData
	}
	headers.Add("run", "no buffer")
	return api.Continue
}

func (f *bufferFilter) DecodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Decode {
		data.AppendString("no buffer\n")
	}
	return api.Continue
}

func (f *bufferFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	_, ok := headers.Get("stream")
	if !ok && f.config.Need {
		return api.WaitAllData
	}
	headers.Del("content-length")
	headers.Add("run", "no buffer")
	return api.Continue
}

func (f *bufferFilter) EncodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Encode {
		data.AppendString("no buffer\n")
	}
	return api.Continue
}

func (p *bufferPlugin) ConfigFactory() api.FilterConfigFactory {
	return bufferConfigFactory
}

type localReplyPlugin struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func localReplyConfigFactory(c interface{}) api.FilterFactory {
	conf := c.(*Config)
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &localReplyFilter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}

type localReplyFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *Config
	reqHdr    api.RequestHeaderMap
}

func (f *localReplyFilter) NewLocalResponse(reply string) *api.LocalResponse {
	hdr := http.Header{}
	hdr.Set("local", reply)

	runFilters := f.reqHdr.Values("run")
	if len(runFilters) > 0 {
		hdr.Set("order", strings.Join(runFilters, "|"))
	}
	return &api.LocalResponse{Code: 206, Msg: "ok", Header: hdr}
}

func (f *localReplyFilter) DecodeRequest(headers api.RequestHeaderMap, buf api.BufferInstance, trailer api.RequestTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	f.reqHdr = headers
	if f.config.Decode {
		return f.NewLocalResponse("reply")
	}
	return api.Continue
}

func (f *localReplyFilter) EncodeResponse(headers api.ResponseHeaderMap, buf api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Encode {
		r, _ := headers.Get("echo-from")
		return f.NewLocalResponse(r)
	}
	return api.Continue
}

func (f *localReplyFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Need {
		return api.WaitAllData
	}
	f.reqHdr = headers
	if f.config.Decode && f.config.Headers {
		return f.NewLocalResponse("reply")
	}
	return api.Continue
}

func (f *localReplyFilter) DecodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Decode && f.config.Data {
		return f.NewLocalResponse("reply")
	}
	return api.Continue
}

func (f *localReplyFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Need {
		return api.WaitAllData
	}
	if f.config.Encode && f.config.Headers {
		r, _ := headers.Get("echo-from")
		return f.NewLocalResponse(r)
	}
	return api.Continue
}

func (f *localReplyFilter) EncodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Encode && f.config.Data {
		return f.NewLocalResponse("reply")
	}
	return api.Continue
}

func (p *localReplyPlugin) ConfigFactory() api.FilterConfigFactory {
	return localReplyConfigFactory
}

func init() {
	plugins.RegisterHttpPlugin("stream", &streamPlugin{})
	plugins.RegisterHttpPlugin("buffer", &bufferPlugin{})
	plugins.RegisterHttpPlugin("localReply", &localReplyPlugin{})
}
