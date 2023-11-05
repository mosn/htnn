package plugins

import (
	"encoding/json"
	"runtime/debug"

	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/plugins"
)

type Config struct {
	Need   bool `json:"need"`
	Decode bool `json:"decode"`
	Encode bool `json:"encode"`
}

type streamPlugin struct {
	plugins.PluginMethodDefaultImpl
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

func (f *streamFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "stream")
}

func (f *streamFilter) DecodeData(data api.BufferInstance, endStream bool) {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Decode {
		data.AppendString("stream\n")
	}
}

func (f *streamFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "stream")
	headers.Del("content-length")
}

func (f *streamFilter) EncodeData(data api.BufferInstance, endStream bool) {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Encode {
		data.AppendString("stream\n")
	}
}

func (p *streamPlugin) ConfigFactory() api.FilterConfigFactory {
	return streamConfigFactory
}

func (p *streamPlugin) ConfigParser() api.FilterConfigParser {
	return plugins.NewPluginConfigParser(&parser{})
}

type bufferPlugin struct {
	plugins.PluginMethodDefaultImpl
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

func (f *bufferFilter) NeedDecodeWholeRequest(headers api.RequestHeaderMap) bool {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	_, ok := headers.Get("stream")
	return !ok && f.config.Need
}

func (f *bufferFilter) DecodeRequest(headers api.RequestHeaderMap, buf api.BufferInstance, trailer api.RequestTrailerMap) {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "buffer")
	if buf != nil && f.config.Decode {
		buf.AppendString("buffer\n")
	}
}

func (f *bufferFilter) NeedEncodeWholeResponse(headers api.ResponseHeaderMap) bool {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	_, ok := headers.Get("stream")
	return !ok && f.config.Need
}

func (f *bufferFilter) EncodeResponse(headers api.ResponseHeaderMap, buf api.BufferInstance, trailers api.ResponseTrailerMap) {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "buffer")
	headers.Del("content-length")
	if buf != nil && f.config.Encode {
		buf.AppendString("buffer\n")
	}
}

func (f *bufferFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Add("run", "no buffer")
}

func (f *bufferFilter) DecodeData(data api.BufferInstance, endStream bool) {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Decode {
		data.AppendString("no buffer\n")
	}
}

func (f *bufferFilter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	headers.Del("content-length")
	headers.Add("run", "no buffer")
}

func (f *bufferFilter) EncodeData(data api.BufferInstance, endStream bool) {
	api.LogInfof("traceback: %s", string(debug.Stack()))
	if f.config.Encode {
		data.AppendString("no buffer\n")
	}
}

func (p *bufferPlugin) ConfigFactory() api.FilterConfigFactory {
	return bufferConfigFactory
}

func (p *bufferPlugin) ConfigParser() api.FilterConfigParser {
	return plugins.NewPluginConfigParser(&parser{})
}

type parser struct {
}

func (p *parser) Validate(data []byte) (interface{}, error) {
	conf := &Config{}
	err := json.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func (p *parser) Handle(c interface{}, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	conf := c.(*Config)
	return conf, nil
}

func (p *parser) Merge(parent interface{}, child interface{}) interface{} {
	return child
}

func init() {
	plugins.RegisterHttpPlugin("stream", &streamPlugin{})
	plugins.RegisterHttpPlugin("buffer", &bufferPlugin{})
}
