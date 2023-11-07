package api

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

type DecodeWholeRequestFilter interface {
	NeedDecodeWholeRequest(headers api.RequestHeaderMap) bool
	DecodeRequest(headers api.RequestHeaderMap, buf api.BufferInstance, trailers api.RequestTrailerMap) ResultAction
}

type EncodeWholeResponseFilter interface {
	NeedEncodeWholeResponse(headers api.ResponseHeaderMap) bool
	EncodeResponse(headers api.ResponseHeaderMap, buf api.BufferInstance, trailers api.ResponseTrailerMap) ResultAction
}

type Filter interface {
	DecodeHeaders(RequestHeaderMap, bool) ResultAction
	DecodeData(BufferInstance, bool) ResultAction
	DecodeTrailers(RequestTrailerMap) ResultAction

	EncodeHeaders(ResponseHeaderMap, bool) ResultAction
	EncodeData(BufferInstance, bool) ResultAction
	EncodeTrailers(ResponseTrailerMap) ResultAction

	OnLog()

	DecodeWholeRequestFilter
	EncodeWholeResponseFilter
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

func (f *PassThroughFilter) NeedDecodeWholeRequest(headers api.RequestHeaderMap) bool { return false }
func (f *PassThroughFilter) DecodeRequest(headers api.RequestHeaderMap, buf api.BufferInstance, trailers api.RequestTrailerMap) ResultAction {
	return Continue
}

func (f *PassThroughFilter) NeedEncodeWholeResponse(headers api.ResponseHeaderMap) bool { return false }
func (f *PassThroughFilter) EncodeResponse(headers api.ResponseHeaderMap, buf api.BufferInstance, trailers api.ResponseTrailerMap) ResultAction {
	return Continue
}

type RequestHeaderMap = api.RequestHeaderMap
type ResponseHeaderMap = api.ResponseHeaderMap
type BufferInstance = api.BufferInstance
type RequestTrailerMap = api.RequestTrailerMap
type ResponseTrailerMap = api.ResponseTrailerMap

type FilterConfigParser interface {
	Parse(input interface{}, callbacks ConfigCallbackHandler) (interface{}, error)
	Merge(parentConfig interface{}, childConfig interface{}) interface{}
}
type FilterConfigFactory func(config interface{}) FilterFactory

type StreamInfo = api.StreamInfo
type FilterCallbackHandler interface {
	StreamInfo() StreamInfo
	RecoverPanic()
	GetProperty(key string) (string, error)
	// TODO: remove it later
	SendLocalReply(responseCode int, bodyText string, headers map[string]string, grpcStatus int64, details string)
}

type FilterFactory func(callbacks FilterCallbackHandler) Filter

type DynamicMetadata = api.DynamicMetadata
type FilterState = api.FilterState

type ConfigCallbackHandler = api.ConfigCallbackHandler

var (
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
)
