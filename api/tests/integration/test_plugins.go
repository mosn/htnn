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

	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

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

func (f *streamFilter) DecodeTrailers(trailers api.RequestTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	trailers.Add("run", "stream")
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

func (f *streamFilter) EncodeTrailers(trailers api.ResponseTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	trailers.Add("run", "stream")
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

func (f *bufferFilter) DecodeRequest(headers api.RequestHeaderMap, buf api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "buffer")
	if buf != nil && f.config.Decode {
		buf.AppendString("buffer\n")
	}
	if trailers != nil && f.config.Decode {
		trailers.Add("run", "buffer")
	}
	return api.Continue
}

func (f *bufferFilter) EncodeResponse(headers api.ResponseHeaderMap, buf api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "buffer")
	headers.Del("content-length")
	if buf != nil && f.config.Encode && !f.config.InGrpcMode {
		buf.AppendString("buffer\n")
	}
	if trailers != nil && f.config.Encode {
		trailers.Add("run", "buffer")
	}
	return api.Continue
}

func (f *bufferFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	_, ok := headers.Get("stream")
	if !ok && f.config.NeedBuffer {
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

func (f *bufferFilter) DecodeTrailers(trailers api.RequestTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Decode {
		trailers.Add("run", "no buffer")
	}
	return api.Continue
}

func (f *bufferFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	_, ok := headers.Get("stream")
	if !ok && f.config.NeedBuffer {
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

func (f *bufferFilter) EncodeTrailers(trailers api.ResponseTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Encode {
		trailers.Add("run", "no buffer")
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

	reqHdr     api.RequestHeaderMap
	runFilters []string
}

func (f *localReplyFilter) NewLocalResponse(reply string, decoding bool) *api.LocalResponse {
	hdr := http.Header{}
	hdr.Set("local", reply)

	if decoding {
		f.runFilters = f.reqHdr.Values("run")
	}
	if len(f.runFilters) > 0 {
		hdr.Set("order", strings.Join(f.runFilters, "|"))
	}

	msg := "ok"
	if f.config.ReplyMsg != "" {
		msg = f.config.ReplyMsg
	}
	return &api.LocalResponse{Code: 206, Msg: msg, Header: hdr, Details: "custom_details"}
}

func (f *localReplyFilter) DecodeRequest(headers api.RequestHeaderMap, buf api.BufferInstance, trailer api.RequestTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	f.reqHdr = headers
	f.runFilters = headers.Values("run")
	if f.config.Decode {
		return f.NewLocalResponse("reply", true)
	}
	return api.Continue
}

func (f *localReplyFilter) EncodeResponse(headers api.ResponseHeaderMap, buf api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Encode {
		r, _ := headers.Get("echo-from")
		return f.NewLocalResponse(r, false)
	}
	return api.Continue
}

func (f *localReplyFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.NeedBuffer {
		return api.WaitAllData
	}
	f.reqHdr = headers
	f.runFilters = headers.Values("run")
	if f.config.Decode && f.config.Headers {
		return f.NewLocalResponse("reply", true)
	}
	return api.Continue
}

func (f *localReplyFilter) DecodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Decode && f.config.Data {
		return f.NewLocalResponse("reply", true)
	}
	return api.Continue
}

func (f *localReplyFilter) DecodeTrailers(trailers api.RequestTrailerMap) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Decode && f.config.Trailers {
		return f.NewLocalResponse("reply", true)
	}
	return api.Continue
}

func (f *localReplyFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.NeedBuffer {
		return api.WaitAllData
	}
	if f.config.Encode && f.config.Headers {
		r, _ := headers.Get("echo-from")
		return f.NewLocalResponse(r, false)
	}
	return api.Continue
}

func (f *localReplyFilter) EncodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Encode && f.config.Data {
		return f.NewLocalResponse("reply", false)
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
	if c.PanicInInit {
		panic("panic in init")
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

type beforeConsumerAndHasDecodeRequestPlugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *beforeConsumerAndHasDecodeRequestPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAccess,
	}
}

func (p *beforeConsumerAndHasDecodeRequestPlugin) Config() api.PluginConfig {
	return &Config{}
}

func (p *beforeConsumerAndHasDecodeRequestPlugin) Factory() api.FilterFactory {
	return beforeConsumerAndHasDecodeRequestFactory
}

func beforeConsumerAndHasDecodeRequestFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &beforeConsumerAndHasDecodeRequestFilter{
		callbacks: callbacks,
		config:    c.(*Config),
	}
}

type beforeConsumerAndHasDecodeRequestFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *Config
}

func (f *beforeConsumerAndHasDecodeRequestFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	headers.Add("run", "beforeConsumerAndHasDecodeRequest")
	return api.Continue
}

func (f *beforeConsumerAndHasDecodeRequestFilter) DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction {
	headers.Add("run", "beforeConsumerAndHasDecodeRequest:DecodeRequest")
	return api.Continue
}

type onLogPlugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *onLogPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAccess,
	}
}

func (p *onLogPlugin) Config() api.PluginConfig {
	return &Config{}
}

func (p *onLogPlugin) Factory() api.FilterFactory {
	return onLogFactory
}

func onLogFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &onLogFilter{
		callbacks: callbacks,
		config:    c.(*Config),
	}
}

type onLogFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *Config
}

func (f *onLogFilter) OnLog(reqHeaders api.RequestHeaderMap, reqTrailers api.RequestTrailerMap,
	respHeaders api.ResponseHeaderMap, respTrailers api.ResponseTrailerMap) {

	trailers := map[string]string{}
	if reqTrailers != nil {
		reqTrailers.Range(func(k, v string) bool {
			trailers[k] = v
			return true
		})
	}
	api.LogWarnf("receive request trailers: %+v", trailers)
}

type metricsConfig struct {
	Config

	usageCounter capi.CounterMetric
	gauge        capi.GaugeMetric
}

func (m *metricsConfig) MetricsDefinition(c capi.ConfigCallbacks) {
	if c == nil {
		api.LogErrorf("metrics config callback is nil")
		return
	}
	m.usageCounter = c.DefineCounterMetric("metrics-test.usage.counter")
	m.gauge = c.DefineGaugeMetric("metrics-test.usage.gauge")
	api.LogInfo("metrics config loaded for metrics-test")
	// Define more metrics here
}

var _ plugins.MetricsRegister = &metricsConfig{}

type metricsPlugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *metricsPlugin) Config() api.PluginConfig {
	return &metricsConfig{}
}

func (p *metricsPlugin) Factory() api.FilterFactory {
	return metricsFactory
}

func metricsFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &metricsFilter{
		callbacks: callbacks,
		config:    c.(*metricsConfig),
	}
}

type metricsFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *metricsConfig
}

func (f *metricsFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	if f.config.usageCounter != nil {
		f.config.usageCounter.Increment(1)
	} else {
		return &api.LocalResponse{Code: 500, Msg: "metrics config counter is nil"}
	}
	if f.config.gauge != nil {
		f.config.gauge.Record(2)
	} else {
		return &api.LocalResponse{Code: 500, Msg: "metrics config gauge is nil"}
	}
	return &api.LocalResponse{Code: 200, Msg: "metrics works"}
}

func init() {
	plugins.RegisterPlugin("stream", &streamPlugin{})
	plugins.RegisterPlugin("buffer", &bufferPlugin{})
	plugins.RegisterPlugin("localReply", &localReplyPlugin{})
	plugins.RegisterPlugin("bad", &badPlugin{})
	plugins.RegisterPlugin("consumer", &consumerPlugin{})
	plugins.RegisterPlugin("init", &initPlugin{})
	plugins.RegisterPlugin("benchmark", &benchmarkPlugin{})
	plugins.RegisterPlugin("benchmark2", &benchmarkPlugin{})
	plugins.RegisterPlugin("beforeConsumerAndHasOtherMethod", &beforeConsumerAndHasOtherMethodPlugin{})
	plugins.RegisterPlugin("beforeConsumerAndHasDecodeRequest", &beforeConsumerAndHasDecodeRequestPlugin{})
	plugins.RegisterPlugin("onLog", &onLogPlugin{})
	plugins.RegisterPlugin("metrics", &metricsPlugin{})
}
