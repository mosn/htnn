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
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
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
// Every filter is run in goroutine so it's non-blocking.
// To know how do we run the Filter during request processing, please refer to
// https://github.com/mosn/moe/blob/main/content/en/docs/developer-guide/plugin_development.md
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

func (f *PassThroughFilter) DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) ResultAction {
	return Continue
}

func (f *PassThroughFilter) EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) ResultAction {
	return Continue
}

type RequestHeaderMap = api.RequestHeaderMap
type ResponseHeaderMap = api.ResponseHeaderMap
type BufferInstance = api.BufferInstance
type RequestTrailerMap = api.RequestTrailerMap
type ResponseTrailerMap = api.ResponseTrailerMap

type StreamInfo = api.StreamInfo

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
}

type FilterFactory func(callbacks FilterCallbackHandler) Filter
type FilterConfigFactory func(config interface{}) FilterFactory

// DynamicMetadata operates the Envoy's dynamic metadata
type DynamicMetadata = api.DynamicMetadata

// FilterState operates the Envoy's filter state
type FilterState = api.FilterState

// ConfigCallbackHandler provides API that is used during initializing configuration
type ConfigCallbackHandler = api.ConfigCallbackHandler

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
