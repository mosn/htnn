//go:build !so

package envoy

import (
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

func init() {
	// replace the implementation of methods like api.LogXXX
	api.SetCommonCAPI(&capi{})
}

func logInGo(level api.LogType, message string) {
	log.Printf("[%s] %s\n", level, message)
}

type capi struct{}

func (a *capi) Log(level api.LogType, message string) {
	logInGo(level, message)
}

func (a *capi) LogLevel() api.LogType {
	return 0
}

type HeaderMap struct {
	http.Header
}

func (i *HeaderMap) GetRaw(name string) string {
	return i.Header.Get(name)
}

func (i *HeaderMap) Get(key string) (string, bool) {
	v := i.Header.Get(key)
	if v == "" {
		return v, false
	}
	return v, true
}

func (i *HeaderMap) Values(key string) []string {
	return i.Header.Values(key)
}

func (i *HeaderMap) Set(key, value string) {
	i.Header.Set(key, value)
}

func (i *HeaderMap) Add(key, value string) {
	i.Header.Add(key, value)
}

func (i *HeaderMap) Del(key string) {
	i.Header.Del(key)
}

func (i *HeaderMap) Range(f func(key, value string) bool) {
	for k, v := range map[string][]string(i.Header) {
		for _, vv := range v {
			if !f(k, vv) {
				return
			}
		}
	}
}

func (i *HeaderMap) RangeWithCopy(f func(key, value string) bool) {
	i.Range(f)
}

var _ api.HeaderMap = (*HeaderMap)(nil)

type RequestHeaderMap struct {
	HeaderMap
}

func NewRequestHeaderMap(hdr http.Header) *RequestHeaderMap {
	return &RequestHeaderMap{
		HeaderMap: HeaderMap{hdr},
	}
}

func (i *RequestHeaderMap) Protocol() string {
	return "HTTP/1.1"
}

func (i *RequestHeaderMap) Scheme() string {
	return "http"
}

func (i *RequestHeaderMap) Method() string {
	method, ok := i.Get(":method")
	if !ok {
		return "GET"
	}
	return method
}

func (i *RequestHeaderMap) Host() string {
	host, ok := i.Get(":authority")
	if !ok {
		return "localhost"
	}
	return host
}

func (i *RequestHeaderMap) Path() string {
	path, ok := i.Get(":path")
	if !ok {
		return "/"
	}
	return path
}

var _ api.RequestHeaderMap = (*RequestHeaderMap)(nil)

type ResponseHeaderMap struct {
	HeaderMap
}

func NewResponseHeaderMap(hdr http.Header) *ResponseHeaderMap {
	return &ResponseHeaderMap{
		HeaderMap: HeaderMap{hdr},
	}
}

func (i *ResponseHeaderMap) Status() (int, bool) {
	s, ok := i.Get(":status")
	if !ok {
		// for test
		return 200, true
	}
	code, _ := strconv.Atoi(s)
	return code, true
}

var _ api.ResponseHeaderMap = (*ResponseHeaderMap)(nil)

type DataBuffer struct {
	buffer []byte
}

// TODO: implement methods below

func (db *DataBuffer) Write(p []byte) (int, error) {
	return len(p), nil
}

func (db *DataBuffer) WriteString(s string) (int, error) {
	return len(s), nil
}

func (db *DataBuffer) WriteByte(b byte) error {
	return nil
}

func (db *DataBuffer) WriteUint16(u uint16) error {
	return nil
}

func (db *DataBuffer) WriteUint32(u uint32) error {
	return nil
}

func (db *DataBuffer) WriteUint64(u uint64) error {
	return nil
}

func (db *DataBuffer) Bytes() []byte {
	return db.buffer
}

func (db *DataBuffer) Drain(offset int) {
}

func (db *DataBuffer) Len() int {
	return len(db.buffer)
}

func (db *DataBuffer) Reset() {
	db.buffer = nil
}

func (db *DataBuffer) String() string {
	return string(db.buffer)
}

func (db *DataBuffer) Append(data []byte) error {
	db.buffer = append(db.buffer, data...)
	return nil
}

func NewBufferInstance(b []byte) *BufferInstance {
	return &BufferInstance{
		DataBuffer: DataBuffer{
			buffer: b,
		},
	}
}

var _ api.DataBufferBase = (*DataBuffer)(nil)

type BufferInstance struct {
	DataBuffer
}

var _ api.BufferInstance = (*BufferInstance)(nil)

func (bi *BufferInstance) Set(data []byte) error {
	bi.buffer = data
	return nil
}

func (bi *BufferInstance) SetString(s string) error {
	bi.buffer = []byte(s)
	return nil
}

func (bi *BufferInstance) Prepend(data []byte) error {
	bi.buffer = append(data, bi.buffer...)
	return nil
}

func (bi *BufferInstance) PrependString(s string) error {
	return bi.Prepend([]byte(s))
}

func (bi *BufferInstance) AppendString(s string) error {
	return bi.Append([]byte(s))
}

type DynamicMetadata struct {
	store map[string]map[string]interface{}
}

func NewDynamicMetadata(data map[string]map[string]interface{}) api.DynamicMetadata {
	return &DynamicMetadata{
		store: data,
	}
}

func (i *DynamicMetadata) Get(filterName string) map[string]interface{} {
	return i.store[filterName]
}

func (i *DynamicMetadata) Set(filterName string, key string, value interface{}) {
	dm, ok := i.store[filterName]
	if !ok {
		dm := map[string]interface{}{}
		i.store[filterName] = dm
	}

	dm[key] = value
}

type FilterState struct {
	store map[string]string
}

func NewFilterState(data map[string]string) api.FilterState {
	return &FilterState{
		store: data,
	}
}

func (i *FilterState) SetString(key, value string, stateType api.StateType, lifeSpan api.LifeSpan, streamSharing api.StreamSharing) {
	i.store[key] = value
}

func (i *FilterState) GetString(key string) string {
	return i.store[key]
}

type StreamInfo struct {
	filterState     api.FilterState
	dynamicMetadata api.DynamicMetadata
}

func (i *StreamInfo) GetRouteName() string {
	return ""
}

func (i *StreamInfo) FilterChainName() string {
	return ""
}

func (i *StreamInfo) Protocol() (string, bool) {
	return "", false
}

func (i *StreamInfo) ResponseCode() (uint32, bool) {
	return 0, false
}

func (i *StreamInfo) ResponseCodeDetails() (string, bool) {
	return "", false
}

func (i *StreamInfo) AttemptCount() uint32 {
	return 0
}

func (i *StreamInfo) DynamicMetadata() api.DynamicMetadata {
	return i.dynamicMetadata
}

func (i *StreamInfo) SetDynamicMetadata(data api.DynamicMetadata) {
	i.dynamicMetadata = data
}

func (i *StreamInfo) DownstreamLocalAddress() string {
	return ""
}

func (i *StreamInfo) DownstreamRemoteAddress() string {
	return ""
}

func (i *StreamInfo) UpstreamLocalAddress() (string, bool) {
	return "", false
}

func (i *StreamInfo) UpstreamRemoteAddress() (string, bool) {
	return "", false
}

func (i *StreamInfo) UpstreamClusterName() (string, bool) {
	return "", false
}

func (i *StreamInfo) FilterState() api.FilterState {
	return i.filterState
}

func (i *StreamInfo) SetFilterState(data api.FilterState) {
	i.filterState = data
}

func (i *StreamInfo) VirtualClusterName() (string, bool) {
	return "", false
}

func (i *StreamInfo) GetProperty(key string) (string, bool) {
	return "", false
}

var _ api.StreamInfo = (*StreamInfo)(nil)

type LocalResponse struct {
	Code    int
	Body    string
	Headers map[string]string
}

type filterCallbackHandler struct {
	// add lock to the test helper to satisfy -race check
	lock *sync.RWMutex

	streamInfo api.StreamInfo
	resp       LocalResponse
	ch         chan struct{}
}

func NewFilterCallbackHandler() *filterCallbackHandler {
	return &filterCallbackHandler{
		lock: &sync.RWMutex{},
		// we create channel with buffer so the goroutine won't leak when we don't call WaitContinued
		// manually. When running in Envoy, Envoy won't re-run the filter until Continue is called.
		ch: make(chan struct{}, 10),
	}
}

func (i *filterCallbackHandler) StreamInfo() api.StreamInfo {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.streamInfo
}

func (i *filterCallbackHandler) SetStreamInfo(data api.StreamInfo) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.streamInfo = data
}

func (i *filterCallbackHandler) Continue(status api.StatusType) {
	i.ch <- struct{}{}
}

func (i *filterCallbackHandler) WaitContinued() {
	<-i.ch
}

func (i *filterCallbackHandler) SendLocalReply(responseCode int, bodyText string, headers map[string]string, grpcStatus int64, details string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.resp = LocalResponse{Code: responseCode, Body: bodyText, Headers: headers}

	i.Continue(api.LocalReply)
}

func (i *filterCallbackHandler) LocalResponse() LocalResponse {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.resp
}

func (i *filterCallbackHandler) RecoverPanic() {
}

func (i *filterCallbackHandler) Log(level api.LogType, msg string) {
	logInGo(level, msg)
}

func (i *filterCallbackHandler) LogLevel() api.LogType {
	return 0
}

func (i *filterCallbackHandler) GetProperty(key string) (string, error) {
	return "", nil
}

var _ api.FilterCallbackHandler = (*filterCallbackHandler)(nil)
