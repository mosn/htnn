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

//go:build !so

package envoy

import (
	"bytes"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"mosn.io/htnn/api/internal/cookie"
	"mosn.io/htnn/api/internal/pluginstate"
	"mosn.io/htnn/api/pkg/filtermanager/api"
)

func init() {
	// replace the implementation of methods like api.LogXXX
	capi.SetCommonCAPI(&fakeCapi{})
}

var (
	logLevel = capi.Info
)

func DisableLogInTest() {
	logLevel = capi.Critical
}

func logInGo(level capi.LogType, message string) {
	log.Printf("[%s] %s\n", level, message)
}

type fakeCapi struct{}

func (a *fakeCapi) Log(level capi.LogType, message string) {
	logInGo(level, message)
}

func (a *fakeCapi) LogLevel() capi.LogType {
	return logLevel
}

type HeaderMap struct {
	http.Header

	lock sync.Mutex
}

func (i *HeaderMap) GetRaw(name string) string {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.Header.Get(name)
}

func (i *HeaderMap) Get(key string) (string, bool) {
	i.lock.Lock()
	defer i.lock.Unlock()
	v := i.Header.Get(key)
	if v == "" {
		return v, false
	}
	return v, true
}

func (i *HeaderMap) GetAllHeaders() map[string][]string {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.Header.Clone()
}

func (i *HeaderMap) Values(key string) []string {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.Header.Values(key)
}

func (i *HeaderMap) Set(key, value string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.Header.Set(key, value)
}

func (i *HeaderMap) Add(key, value string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.Header.Add(key, value)
}

func (i *HeaderMap) Del(key string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.Header.Del(key)
}

func (i *HeaderMap) Range(f func(key, value string) bool) {
	i.lock.Lock()
	defer i.lock.Unlock()
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
		HeaderMap: HeaderMap{
			Header: hdr,
			lock:   sync.Mutex{},
		},
	}
}

func (i *RequestHeaderMap) Scheme() string {
	scheme, ok := i.Get(":scheme")
	if !ok {
		return "http"
	}
	return scheme
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

func (i *RequestHeaderMap) SetMethod(method string) {
	i.Set(":method", method)
}

func (i *RequestHeaderMap) SetPath(path string) {
	i.Set(":path", path)
}

func (i *RequestHeaderMap) SetHost(host string) {
	i.Set(":authority", host)
}

func (i *RequestHeaderMap) URL() *url.URL {
	path := i.Path()
	u, _ := url.ParseRequestURI(path)
	return u
}

func (i *RequestHeaderMap) Cookie(name string) *http.Cookie {
	cookieList := cookie.ParseCookies(i)
	for _, c := range cookieList {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (i *RequestHeaderMap) Cookies() []*http.Cookie {
	return cookie.ParseCookies(i)
}

var _ api.RequestHeaderMap = (*RequestHeaderMap)(nil)

type ResponseHeaderMap struct {
	HeaderMap
}

func NewResponseHeaderMap(hdr http.Header) *ResponseHeaderMap {
	return &ResponseHeaderMap{
		HeaderMap: HeaderMap{
			Header: hdr,
			lock:   sync.Mutex{},
		},
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

type dataBuffer struct {
	buffer *bytes.Buffer
}

func (db *dataBuffer) Write(p []byte) (int, error) {
	return db.buffer.Write(p)
}

func (db *dataBuffer) WriteString(s string) (int, error) {
	return db.buffer.WriteString(s)
}

func (db *dataBuffer) WriteByte(b byte) error {
	return db.buffer.WriteByte(b)
}

func (db *dataBuffer) WriteUint16(p uint16) error {
	s := strconv.FormatUint(uint64(p), 10)
	_, err := db.WriteString(s)
	return err
}

func (db *dataBuffer) WriteUint32(p uint32) error {
	s := strconv.FormatUint(uint64(p), 10)
	_, err := db.WriteString(s)
	return err
}

func (db *dataBuffer) WriteUint64(p uint64) error {
	s := strconv.FormatUint(p, 10)
	_, err := db.WriteString(s)
	return err
}

func (db *dataBuffer) Bytes() []byte {
	return db.buffer.Bytes()
}

func (db *dataBuffer) Drain(offset int) {
	db.buffer.Next(offset)
}

func (db *dataBuffer) Len() int {
	return db.buffer.Len()
}

func (db *dataBuffer) Reset() {
	db.buffer.Reset()
}

func (db *dataBuffer) String() string {
	return db.buffer.String()
}

func (db *dataBuffer) Append(data []byte) error {
	_, err := db.buffer.Write(data)
	return err
}

func NewBufferInstance(b []byte) *BufferInstance {
	return &BufferInstance{
		dataBuffer: dataBuffer{
			buffer: bytes.NewBuffer(b),
		},
	}
}

var _ api.DataBufferBase = (*dataBuffer)(nil)

type BufferInstance struct {
	dataBuffer
}

var _ api.BufferInstance = (*BufferInstance)(nil)

func (bi *BufferInstance) Set(data []byte) error {
	bi.buffer = bytes.NewBuffer(data)
	return nil
}

func (bi *BufferInstance) SetString(s string) error {
	bi.buffer = bytes.NewBufferString(s)
	return nil
}

func (bi *BufferInstance) Prepend(data []byte) error {
	bi.buffer = bytes.NewBuffer(append(data, bi.buffer.Bytes()...))
	return nil
}

func (bi *BufferInstance) PrependString(s string) error {
	return bi.Prepend([]byte(s))
}

func (bi *BufferInstance) AppendString(s string) error {
	return bi.Append([]byte(s))
}

func NewRequestTrailerMap(hdr http.Header) api.RequestTrailerMap {
	return &HeaderMap{
		Header: hdr,
	}
}

func NewResponseTrailerMap(hdr http.Header) api.ResponseTrailerMap {
	return &HeaderMap{
		Header: hdr,
	}
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
		dm = map[string]interface{}{}
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

func (i *FilterState) SetString(key, value string, stateType capi.StateType, lifeSpan capi.LifeSpan, streamSharing capi.StreamSharing) {
	i.store[key] = value
}

func (i *FilterState) GetString(key string) string {
	return i.store[key]
}

type StreamInfo struct {
	filterState     api.FilterState
	dynamicMetadata api.DynamicMetadata
}

// use gomonkey to mock the methods below when writing unit test

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
	if i.dynamicMetadata == nil {
		i.dynamicMetadata = NewDynamicMetadata(map[string]map[string]interface{}{})
	}
	return i.dynamicMetadata
}

func (i *StreamInfo) SetDynamicMetadata(data api.DynamicMetadata) {
	i.dynamicMetadata = data
}

func (i *StreamInfo) DownstreamLocalAddress() string {
	return "0.0.0.0:10000"
}

func (i *StreamInfo) DownstreamRemoteAddress() string {
	return "183.128.130.43:54321"
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
	if i.filterState == nil {
		i.filterState = NewFilterState(map[string]string{})
	}
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

func (i *StreamInfo) WorkerID() uint32 {
	return 0
}

func (i *StreamInfo) DownstreamRemoteParsedAddress() *api.IPAddress {
	return &api.IPAddress{
		Address: "183.128.130.43:54321",
		IP:      "183.128.130.43",
		Port:    54321,
	}
}

var _ api.StreamInfo = (*StreamInfo)(nil)

type LocalResponse struct {
	Code    int
	Body    string
	Headers map[string][]string
}

type filterCallbackHandler struct {
	// add lock to the test helper to satisfy -race check
	lock *sync.RWMutex

	streamInfo  api.StreamInfo
	resp        LocalResponse
	consumer    api.Consumer
	pluginState api.PluginState
	ch          chan capi.StatusType
}

func NewFilterCallbackHandler() *filterCallbackHandler {
	return &filterCallbackHandler{
		lock: &sync.RWMutex{},
		// we create channel with buffer so the goroutine won't leak when we don't call WaitContinued
		// manually. When running in Envoy, Envoy won't re-run the filter until Continue is called.
		ch:         make(chan capi.StatusType, 10),
		streamInfo: &StreamInfo{},
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

func (i *filterCallbackHandler) Continue(status capi.StatusType) {
	i.ch <- status
}

func (i *filterCallbackHandler) WaitContinued() capi.StatusType {
	return <-i.ch
}

func (i *filterCallbackHandler) SendLocalReply(responseCode int, bodyText string, headers map[string][]string, grpcStatus int64, details string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.resp = LocalResponse{Code: responseCode, Body: bodyText, Headers: headers}

	i.Continue(capi.LocalReply)
}

func (i *filterCallbackHandler) LocalResponse() LocalResponse {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.resp
}

func (i *filterCallbackHandler) RecoverPanic() {
}

func (i *filterCallbackHandler) Log(level capi.LogType, msg string) {
	logInGo(level, msg)
}

func (i *filterCallbackHandler) LogLevel() capi.LogType {
	return 0
}

func (i *filterCallbackHandler) GetProperty(key string) (string, error) {
	return "", nil
}

func (i *filterCallbackHandler) SecretManager() api.SecretManager {
	// TODO: return a sanity value once we need SecretManager
	return nil
}

func (i *filterCallbackHandler) ClearRouteCache() {
}

func (i *filterCallbackHandler) RefreshRouteCache() {
}

func (i *filterCallbackHandler) AddData([]byte, bool) {
}

func (i *filterCallbackHandler) InjectData([]byte) {
}

func (i *filterCallbackHandler) LookupConsumer(_, _ string) (api.Consumer, bool) {
	return nil, false
}

func (i *filterCallbackHandler) GetConsumer() api.Consumer {
	return i.consumer
}

func (i *filterCallbackHandler) SetConsumer(c api.Consumer) {
	i.consumer = c
}

func (i *filterCallbackHandler) PluginState() api.PluginState {
	if i.pluginState == nil {
		i.pluginState = pluginstate.NewPluginState()
	}
	return i.pluginState
}

func (i *filterCallbackHandler) WithLogArg(key string, value any) api.StreamFilterCallbacks {
	return i
}

func (i *filterCallbackHandler) LogTracef(format string, v ...any) {
}

func (i *filterCallbackHandler) LogTrace(message string) {
}

func (i *filterCallbackHandler) LogDebugf(format string, v ...any) {
}

func (i *filterCallbackHandler) LogDebug(message string) {
}

func (i *filterCallbackHandler) LogInfof(format string, v ...any) {
}

func (i *filterCallbackHandler) LogInfo(message string) {
}

func (i *filterCallbackHandler) LogWarnf(format string, v ...any) {
}

func (i *filterCallbackHandler) LogWarn(message string) {
}

func (i *filterCallbackHandler) LogErrorf(format string, v ...any) {
}

func (i *filterCallbackHandler) LogError(message string) {
}

func (i *filterCallbackHandler) DecoderFilterCallbacks() api.DecoderFilterCallbacks {
	// we don't distinguish between decoder and encoder filter callbacks in the test helper
	return i
}

func (i *filterCallbackHandler) EncoderFilterCallbacks() api.EncoderFilterCallbacks {
	// we don't distinguish between decoder and encoder filter callbacks in the test helper
	return i
}

var _ api.FilterCallbackHandler = (*filterCallbackHandler)(nil)

type capiFilterCallbackHandler struct {
	*filterCallbackHandler
}

func (cb *capiFilterCallbackHandler) StreamInfo() capi.StreamInfo {
	return cb.filterCallbackHandler.StreamInfo()
}

var _ capi.FilterCallbackHandler = (*capiFilterCallbackHandler)(nil)

func NewCAPIFilterCallbackHandler() *capiFilterCallbackHandler {
	return &capiFilterCallbackHandler{
		filterCallbackHandler: NewFilterCallbackHandler(),
	}
}
