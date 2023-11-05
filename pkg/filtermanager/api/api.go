package api

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

type Filter interface {
	DecodeHeaders(RequestHeaderMap, bool)
	DecodeData(BufferInstance, bool)
	DecodeTrailers(RequestTrailerMap)

	EncodeHeaders(ResponseHeaderMap, bool)
	EncodeData(BufferInstance, bool)
	EncodeTrailers(ResponseTrailerMap)

	OnLog()
}

type PassThroughFilter struct{}

func (f *PassThroughFilter) DecodeHeaders(headers RequestHeaderMap, endStream bool) {}

func (f *PassThroughFilter) DecodeData(data BufferInstance, endStream bool) {}

func (f *PassThroughFilter) DecodeTrailers(trailers RequestTrailerMap) {}

func (f *PassThroughFilter) EncodeHeaders(headers ResponseHeaderMap, endStream bool) {}

func (f *PassThroughFilter) EncodeData(data BufferInstance, endStream bool) {}

func (f *PassThroughFilter) EncodeTrailers(trailers ResponseTrailerMap) {}

func (f *PassThroughFilter) OnLog() {}

type DecodeWholeRequestFilter interface {
	NeedDecodeWholeRequest(headers api.RequestHeaderMap) bool
	DecodeRequest(headers api.RequestHeaderMap, buf api.BufferInstance, trailers api.RequestTrailerMap)
}

type EncodeWholeResponseFilter interface {
	NeedEncodeWholeResponse(headers api.ResponseHeaderMap) bool
	EncodeResponse(headers api.ResponseHeaderMap, buf api.BufferInstance, trailers api.ResponseTrailerMap)
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
