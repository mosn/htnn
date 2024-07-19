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

package api

import (
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type DecodeWholeRequestFilter interface {
	// DecodeRequest processes the whole request once when WaitAllData is returned from DecodeHeaders
	// headers: the request header
	// data: the whole request body, nil if the request doesn't have body
	// trailers: TODO, just a placeholder
	DecodeRequest(headers RequestHeaderMap, data BufferInstance, trailers RequestTrailerMap) ResultAction
}

type EncodeWholeResponseFilter interface {
	// EncodeResponse processes the whole response once when WaitAllData is returned from EncodeHeaders
	// headers: the response header
	// data: the whole response body, nil if the response doesn't have body
	// trailers: TODO, just a placeholder
	EncodeResponse(headers ResponseHeaderMap, data BufferInstance, trailers ResponseTrailerMap) ResultAction
}

// Filter represents a collection of callbacks in which Envoy will call your Go code.
// Every filter method (except the OnLog) is run in goroutine so it's non-blocking.
// To know how do we run the Filter during request processing, please refer to
// https://github.com/mosn/htnn/blob/main/site/content/en/docs/developer-guide/plugin_development.md
// TODO: change the link to the official website once we have it.
type Filter interface {
	// Callbacks which are called in request path

	// DecodeHeaders processes request headers. The endStream is true if the request doesn't have body
	DecodeHeaders(headers RequestHeaderMap, endStream bool) ResultAction
	// DecodeData might be called multiple times during handling the request body.
	// The endStream is true when handling the last piece of the body.
	DecodeData(data BufferInstance, endStream bool) ResultAction
	// TODO, just a placeholder. DecodeTrailers is not called yet
	DecodeTrailers(trailers RequestTrailerMap) ResultAction
	DecodeWholeRequestFilter

	// Callbacks which are called in response path

	// EncodeHeaders processes response headers. The endStream is true if the response doesn't have body
	EncodeHeaders(headers ResponseHeaderMap, endStream bool) ResultAction
	// EncodeData might be called multiple times during handling the response body.
	// The endStream is true when handling the last piece of the body.
	EncodeData(data BufferInstance, endStream bool) ResultAction
	// TODO, just a placeholder. EncodeTrailers is not called yet
	EncodeTrailers(trailers ResponseTrailerMap) ResultAction
	EncodeWholeResponseFilter

	// OnLog is called when the HTTP stream is ended on HTTP Connection Manager filter.
	// TODO: the trailers here are just placeholders.
	OnLog(reqHeaders RequestHeaderMap, reqTrailers RequestTrailerMap,
		respHeaders ResponseHeaderMap, respTrailers ResponseTrailerMap)
}

type PassThroughFilter struct{}

func (f *PassThroughFilter) DecodeHeaders(headers RequestHeaderMap, endStream bool) ResultAction {
	return Continue
}

func (f *PassThroughFilter) DecodeData(data BufferInstance, endStream bool) ResultAction {
	return Continue
}

func (f *PassThroughFilter) DecodeTrailers(trailers RequestTrailerMap) ResultAction {
	return Continue
}

func (f *PassThroughFilter) EncodeHeaders(headers ResponseHeaderMap, endStream bool) ResultAction {
	return Continue
}

func (f *PassThroughFilter) EncodeData(data BufferInstance, endStream bool) ResultAction {
	return Continue
}

func (f *PassThroughFilter) EncodeTrailers(trailers ResponseTrailerMap) ResultAction {
	return Continue
}

func (f *PassThroughFilter) OnLog(reqHeaders RequestHeaderMap, reqTrailers RequestTrailerMap,
	respHeaders ResponseHeaderMap, respTrailers ResponseTrailerMap) {
}

func (f *PassThroughFilter) DecodeRequest(headers RequestHeaderMap, data BufferInstance, trailers RequestTrailerMap) ResultAction {
	return Continue
}

func (f *PassThroughFilter) EncodeResponse(headers ResponseHeaderMap, data BufferInstance, trailers ResponseTrailerMap) ResultAction {
	return Continue
}

// The filtermanager will run the Filter one by one. So all the API bound with a request (RequestHeaderMap, StreamInfo, etc.)
// is not designed to be concurrent safe. All the object returns from the API is read-only by default.
// If you want to modify the object, please make a copy of it.

type HeaderMap = api.HeaderMap
type RequestHeaderMap interface {
	api.RequestHeaderMap

	// URL returns the parsed `url.URL`. Changing the field in the returned `url.URL` will not affect the original path.
	// Please use `Set(":path", ...)` to change it.
	URL() *url.URL
	// Cookie returns the HTTP Cookie. If there is no cookie with the given name, nil will be returned.
	// If multiple cookies match the given name, only one cookie will be returned.
	// Changing the field in the returned `http.Cookie` will not affect the cookies sent to the upstream.
	// Please use `Cookies` to get all cookies, change the target cookie,
	// then call `String()` to each cookie and join them as a list with `;`,
	// finally use `Set("cookie", $all-cookies-merged-as-list)` to set the previously fetched cookies back.
	Cookie(name string) *http.Cookie
	// Cookies returns all HTTP cookies. Changing the returned cookies will not affect the cookies sent to the upstream.
	// Please see the comment in `Cookie` for how to change the cookies.
	Cookies() []*http.Cookie
}
type ResponseHeaderMap = api.ResponseHeaderMap
type DataBufferBase = api.DataBufferBase
type BufferInstance = api.BufferInstance
type RequestTrailerMap = api.RequestTrailerMap
type ResponseTrailerMap = api.ResponseTrailerMap

type IPAddress struct {
	Address string
	IP      string
	Port    int
}

type StreamInfo interface {
	GetRouteName() string
	FilterChainName() string
	// Protocol return the request's protocol.
	Protocol() (string, bool)
	// ResponseCode return the response code.
	ResponseCode() (uint32, bool)
	// ResponseCodeDetails return the response code details.
	ResponseCodeDetails() (string, bool)
	// AttemptCount return the number of times the request was attempted upstream.
	AttemptCount() uint32
	// Get the dynamic metadata of the request
	DynamicMetadata() DynamicMetadata
	// DownstreamLocalAddress return the downstream local address.
	DownstreamLocalAddress() string
	// DownstreamRemoteAddress return the downstream remote address.
	DownstreamRemoteAddress() string
	// UpstreamLocalAddress return the upstream local address.
	UpstreamLocalAddress() (string, bool)
	// UpstreamRemoteAddress return the upstream remote address.
	UpstreamRemoteAddress() (string, bool)
	// UpstreamClusterName return the upstream host cluster.
	UpstreamClusterName() (string, bool)
	// FilterState return the filter state interface.
	FilterState() FilterState
	// VirtualClusterName returns the name of the virtual cluster which got matched
	VirtualClusterName() (string, bool)

	// Methods added by HTNN

	// DownstreamRemoteParsedAddress returns the downstream remote address, in the IPAddress struct
	DownstreamRemoteParsedAddress() *IPAddress
}

type PluginConfig interface {
	ProtoReflect() protoreflect.Message
	Validate() error
}

type PluginConsumerConfig interface {
	PluginConfig
	Index() string
}

type Consumer interface {
	Name() string
	PluginConfig(name string) PluginConsumerConfig
}

// FilterCallbackHandler provides API that is used during request processing
type FilterCallbackHandler interface {
	// StreamInfo provides API to get/set current stream's context.
	StreamInfo() StreamInfo
	// RecoverPanic covers panic to 500 response to avoid crashing Envoy. If you create goroutine
	// in your Filter, please add `defer RecoverPanic()` to avoid crash by panic.
	RecoverPanic()
	// GetProperty fetch Envoy attribute and return the value as a string.
	// The list of attributes can be found in https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/advanced/attributes.
	// If the fetch succeeded, a string will be returned.
	// If the value is a timestamp, it is returned as a timestamp string like "2023-07-31T07:21:40.695646+00:00".
	// If the fetch failed (including the value is not found), an error will be returned.
	//
	// The error can be one of:
	// * ErrInternalFailure
	// * ErrSerializationFailure (Currently, fetching attributes in List/Map type are unsupported)
	// * ErrValueNotFound
	GetProperty(key string) (string, error)

	// Methods added by HTNN

	// LookupConsumer is used in the Authn plugins to fetch the corresponding consumer, with
	// the plugin name and plugin specific key. We return a 'fat' Consumer so that additional
	// info like `Name` can be retrieved.
	LookupConsumer(pluginName, key string) (Consumer, bool)
	// SetConsumer is used in the Authn plugins to set the corresponding consumer after authentication.
	SetConsumer(c Consumer)
	GetConsumer() Consumer

	// PluginState returns the PluginState associated to this request.
	PluginState() PluginState
}

// FilterFactory returns a per-request Filter which has configuration bound to it.
// This function should be a pure builder and should not have any side effect.
type FilterFactory func(config interface{}, callbacks FilterCallbackHandler) Filter

// DynamicMetadata operates the Envoy's dynamic metadata
type DynamicMetadata = api.DynamicMetadata

// FilterState operates the Envoy's filter state
type FilterState = api.FilterState

// PluginState stores the plugin level state shared between Go plugins. Unlike DynamicMetadata,
// it doesn't do any serialization/deserialization. So:
// 1. modifying the returned state can affect the internal structure.
// 2. fields can't be marshalled can be kept in the state.
// 3. one can't access the state outside the current Envoy Go filter.
type PluginState interface {
	// Get the value. Returns nil if the value doesn't exist.
	Get(namespace string, key string) any
	// Set the value.
	Set(namespace string, key string, value any)
}

// ConfigCallbackHandler provides API that is used during initializing configuration
type ConfigCallbackHandler interface {
	// The ConfigCallbackHandler from Envoy is only available when the plugin is
	// configured in LDS. But the plugin in HTNN is configured in RDS.
	// So let's provide a placeholder here.
}

type LogType = api.LogType

var (
	LogLevelTrace    = api.Trace
	LogLevelDebug    = api.Debug
	LogLevelInfo     = api.Info
	LogLevelWarn     = api.Warn
	LogLevelError    = api.Error
	LogLevelCritical = api.Critical
)

// Drop our log optimization once https://github.com/envoyproxy/envoy/commit/591fb13817ddf1f54945186e3c6de4e0345508d2
// is used.

var (
	currLogLevel atomic.Int32
)

func GetLogLevel() LogType {
	lv := currLogLevel.Load()
	return LogType(lv)
}

func LogTrace(message string) {
	if GetLogLevel() > LogLevelTrace {
		return
	}
	api.LogTrace(message)
}

func LogDebug(message string) {
	if GetLogLevel() > LogLevelDebug {
		return
	}
	api.LogDebug(message)
}

func LogInfo(message string) {
	if GetLogLevel() > LogLevelInfo {
		return
	}
	api.LogInfo(message)
}

func LogWarn(message string) {
	if GetLogLevel() > LogLevelWarn {
		return
	}
	api.LogWarn(message)
}

func LogError(message string) {
	if GetLogLevel() > LogLevelError {
		return
	}
	api.LogError(message)
}

func LogCritical(message string) {
	if GetLogLevel() > LogLevelCritical {
		return
	}
	api.LogCritical(message)
}

func LogTracef(format string, v ...any) {
	if GetLogLevel() > LogLevelTrace {
		return
	}
	api.LogTracef(format, v...)
}

func LogDebugf(format string, v ...any) {
	if GetLogLevel() > LogLevelDebug {
		return
	}
	api.LogDebugf(format, v...)
}

func LogInfof(format string, v ...any) {
	if GetLogLevel() > LogLevelInfo {
		return
	}
	api.LogInfof(format, v...)
}

func LogWarnf(format string, v ...any) {
	if GetLogLevel() > LogLevelWarn {
		return
	}
	api.LogWarnf(format, v...)
}

func LogErrorf(format string, v ...any) {
	if GetLogLevel() > LogLevelError {
		return
	}
	api.LogErrorf(format, v...)
}

func LogCriticalf(format string, v ...any) {
	if GetLogLevel() > LogLevelCritical {
		return
	}
	api.LogCriticalf(format, v...)
}

var (
	ErrInternalFailure      = api.ErrInternalFailure
	ErrSerializationFailure = api.ErrSerializationFailure
	ErrValueNotFound        = api.ErrValueNotFound
)
