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
	"errors"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

type basePlugin struct {
}

func (p basePlugin) Config() api.PluginConfig {
	return &Config{}
}

type streamPlugin struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func streamFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &streamFilter{
		callbacks: callbacks,
		config:    c.(*Config),
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

func (p *streamPlugin) Factory() api.FilterFactory {
	return streamFactory
}

type bufferPlugin struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func bufferFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &bufferFilter{
		callbacks: callbacks,
		config:    c.(*Config),
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

func (p *bufferPlugin) Factory() api.FilterFactory {
	return bufferFactory
}

type localReplyPlugin struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func localReplyFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &localReplyFilter{
		callbacks: callbacks,
		config:    c.(*Config),
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

	msg := "ok"
	if f.config.ReplyMsg != "" {
		msg = f.config.ReplyMsg
	}
	return &api.LocalResponse{Code: 206, Msg: msg, Header: hdr}
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

func (p *localReplyPlugin) Factory() api.FilterFactory {
	return localReplyFactory
}

type badPlugin struct {
	plugins.PluginMethodDefaultImpl
}

type badPluginConfig struct {
	BadPluginConfig
}

func (c *badPluginConfig) Validate() error {
	if c.PanicInParse {
		panic("panic in parse")
	}
	return nil
}

func (c *badPluginConfig) Init(cb api.ConfigCallbackHandler) error {
	if c.ErrorInInit {
		return errors.New("ouch")
	}
	return nil
}

func (p *badPlugin) Config() api.PluginConfig {
	return &badPluginConfig{}
}

func (p *badPlugin) Factory() api.FilterFactory {
	return badFactory
}

func badFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	cfg := c.(*badPluginConfig)
	if cfg.PanicInFactory {
		panic("panic in factory")
	}
	return &badFilter{
		callbacks: callbacks,
	}
}

type badFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
}

func (f *badFilter) DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction {
	return api.Continue
}

func (f *badFilter) EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	return api.Continue
}

type consumerPlugin struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func (p *consumerPlugin) Factory() api.FilterFactory {
	return consumerFactory
}

func (p *consumerPlugin) Type() plugins.PluginType {
	return plugins.TypeAuthn
}

func (p *consumerPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAuthn,
	}
}

func (p *consumerPlugin) ConsumerConfig() api.PluginConsumerConfig {
	return &ConsumerConfig{}
}

func (conf *ConsumerConfig) Index() string {
	return conf.Name
}

func consumerFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &consumerFilter{
		callbacks: callbacks,
		config:    c.(*Config),
	}
}

type consumerFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *Config
}

func (f *consumerFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	h, _ := headers.Get("authorization")
	c, ok := f.callbacks.LookupConsumer("consumer", h)
	if !ok {
		return &api.LocalResponse{Code: 401, Msg: "invalid key"}
	}

	f.callbacks.SetConsumer(c)
	return api.Continue
}

type initConfig struct {
	Config

	initCounter int
}

func (c *initConfig) Init(cb api.ConfigCallbackHandler) error {
	api.LogInfof("init at %s", string(debug.Stack()))
	c.initCounter++
	return nil
}

var _ plugins.Initer = &initConfig{}

type initPlugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *initPlugin) Config() api.PluginConfig {
	return &initConfig{}
}

func (p *initPlugin) Factory() api.FilterFactory {
	return initFactory
}

func initFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &initFilter{
		callbacks: callbacks,
		config:    c.(*initConfig),
	}
}

type initFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *initConfig
}

func (f *initFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	headers.Add("InitCounter", strconv.Itoa(f.config.initCounter))
	return api.Continue
}

type benchmarkPlugin struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func (p *benchmarkPlugin) Factory() api.FilterFactory {
	return benchmarkFactory
}

func benchmarkFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &benchmarkFilter{
		callbacks: callbacks,
		config:    c.(*Config),
	}
}

type benchmarkFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *Config
}

func (f *benchmarkFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	return api.Continue
}

type beforeConsumerAndHasOtherMethodPlugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *beforeConsumerAndHasOtherMethodPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAccess,
	}
}

func (p *beforeConsumerAndHasOtherMethodPlugin) Config() api.PluginConfig {
	return &Config{}
}

func (p *beforeConsumerAndHasOtherMethodPlugin) Factory() api.FilterFactory {
	return beforeConsumerAndHasOtherMethodFactory
}

func beforeConsumerAndHasOtherMethodFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &beforeConsumerAndHasOtherMethodFilter{
		callbacks: callbacks,
		config:    c.(*Config),
	}
}

type beforeConsumerAndHasOtherMethodFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *Config
}

func (f *beforeConsumerAndHasOtherMethodFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	headers.Add("run", "beforeConsumerAndHasOtherMethod")
	return api.Continue
}

func (f *beforeConsumerAndHasOtherMethodFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	headers.Add("run", "beforeConsumerAndHasOtherMethod")
	return api.Continue
}

func init() {
	plugins.RegisterHttpPlugin("stream", &streamPlugin{})
	plugins.RegisterHttpPlugin("buffer", &bufferPlugin{})
	plugins.RegisterHttpPlugin("localReply", &localReplyPlugin{})
	plugins.RegisterHttpPlugin("bad", &badPlugin{})
	plugins.RegisterHttpPlugin("consumer", &consumerPlugin{})
	plugins.RegisterHttpPlugin("init", &initPlugin{})
	plugins.RegisterHttpPlugin("benchmark", &benchmarkPlugin{})
	plugins.RegisterHttpPlugin("benchmark2", &benchmarkPlugin{})
	plugins.RegisterHttpPlugin("beforeConsumerAndHasOtherMethod", &beforeConsumerAndHasOtherMethodPlugin{})
}
