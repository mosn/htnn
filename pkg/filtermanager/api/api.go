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

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type DecodeWholeRequestFilter interface {
	// DecodeRequest processes the whole request once when WaitAllData is returned
	// headers: the request header
	// data: the whole request body, nil if the request doesn't have body
	// trailers: TODO, just a placeholder
	DecodeRequest(headers RequestHeaderMap, data BufferInstance, trailers RequestTrailerMap) ResultAction
}

type EncodeWholeResponseFilter interface {
	// EncodeResponse processes the whole response once when WaitAllData is returned
	// headers: the response header
	// data: the whole response body, nil if the response doesn't have body
	// trailers: TODO, just a placeholder
	EncodeResponse(headers ResponseHeaderMap, data BufferInstance, trailers ResponseTrailerMap) ResultAction
}

// Filter represents a collection of callbacks in which Envoy will call your Go code.
// Every filter method (except the OnLog) is run in goroutine so it's non-blocking.
// To know how do we run the Filter during request processing, please refer to
// https://github.com/mosn/htnn/blob/main/content/en/docs/developer-guide/plugin_development.md
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
	OnLog()
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

func (f *PassThroughFilter) OnLog() {}

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

	// Url returns the parsed `url.URL`. Changing the field in the returned `url.URL` will not affect the original path.
	// Please use `Set(":path", ...)` to change it.
	Url() *url.URL
	// Cookie returns the HTTP Cookie. If there is no cookie with the given name, nil will be returned.
	// If multiple cookies match the given name, only one cookie will be returned.
	// Changing the field in the returned `http.Cookie` will not affect the Cookies sent to the upstream.
	// Please use `mosn.io/htnn/pkg/cookie.ParseCookies` to get all cookies,
	// change the target cookie,
	// then call `String()` to each cookie and join them as a list with `;`,
	// finally use `Set("cookie", $all-cookies-merged-as-list)` to set the previously fetched cookies back.
	Cookie(name string) *http.Cookie
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

var (
	// Log API family. Note that the Envoy's log level can be changed at runtime.
	LogTrace     = api.LogTrace
	LogDebug     = api.LogDebug
	LogInfo      = api.LogInfo
	LogWarn      = api.LogWarn
	LogError     = api.LogError
	LogCritical  = api.LogCritical
	LogTracef    = api.LogTracef
	LogDebugf    = api.LogDebugf
	LogInfof     = api.LogInfof
	LogWarnf     = api.LogWarnf
	LogErrorf    = api.LogErrorf
	LogCriticalf = api.LogCriticalf
	GetLogLevel  = api.GetLogLevel

	ErrInternalFailure      = api.ErrInternalFailure
	ErrSerializationFailure = api.ErrSerializationFailure
	ErrValueNotFound        = api.ErrValueNotFound
)
