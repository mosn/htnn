package api

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

type DecodeWholeRequestFilter interface {
	NeedDecodeWholeRequest(headers api.RequestHeaderMap) bool
	DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) ResultAction
}

type EncodeWholeResponseFilter interface {
	NeedEncodeWholeResponse(headers api.ResponseHeaderMap) bool
	EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) ResultAction
}

type Filter interface {
	DecodeHeaders(headers RequestHeaderMap, endStream bool) ResultAction
	DecodeData(data BufferInstance, endStream bool) ResultAction
	DecodeTrailers(trailers RequestTrailerMap) ResultAction

	EncodeHeaders(headers ResponseHeaderMap, endStream bool) ResultAction
	EncodeData(data BufferInstance, endStream bool) ResultAction
	EncodeTrailers(trailers ResponseTrailerMap) ResultAction

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
func (f *PassThroughFilter) DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) ResultAction {
	return Continue
}

func (f *PassThroughFilter) NeedEncodeWholeResponse(headers api.ResponseHeaderMap) bool { return false }
func (f *PassThroughFilter) EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) ResultAction {
	return Continue
}

type RequestHeaderMap = api.RequestHeaderMap
type ResponseHeaderMap = api.ResponseHeaderMap
type BufferInstance = api.BufferInstance
type RequestTrailerMap = api.RequestTrailerMap
type ResponseTrailerMap = api.ResponseTrailerMap

type StreamInfo = api.StreamInfo
type FilterCallbackHandler interface {
	StreamInfo() StreamInfo
	RecoverPanic()
	GetProperty(key string) (string, error)
}

type FilterFactory func(callbacks FilterCallbackHandler) Filter
type FilterConfigFactory func(config interface{}) FilterFactory

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
