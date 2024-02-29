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

package filtermanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"

	xds "github.com/cncf/xds/go/xds/type/v3"
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"

	pkgConsumer "mosn.io/htnn/pkg/consumer"
	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/filtermanager/model"
	pkgPlugins "mosn.io/htnn/pkg/plugins"
	"mosn.io/htnn/pkg/reflectx"
	"mosn.io/htnn/pkg/request"
)

// We can't import package below here that will cause build failure in Mac
// "github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
// Therefore, the FilterManagerConfigParser & FilterManagerFactory need to be exportable.
// The http.RegisterHttpFilterFactoryAndParser will be called in the main.go when building
// the shared library in Linux.

type FilterManagerConfigParser struct {
}

type FilterManagerConfig struct {
	Namespace string `json:"namespace,omitempty"`

	Plugins []*model.FilterConfig `json:"plugins"`
}

type filterManagerConfig struct {
	consumerFiltersEndAt int

	parsed []*model.ParsedFilterConfig
	pool   *sync.Pool
}

func initFilterManagerConfig(namespace string) *filterManagerConfig {
	config := &filterManagerConfig{}
	config.pool = &sync.Pool{
		New: func() any {
			callbacks := &filterManagerCallbackHandler{
				namespace: namespace,
			}
			fm := &filterManager{
				callbacks: callbacks,
				config:    config,

				decodeIdx: -1,
				encodeIdx: -1,
			}
			return fm
		},
	}
	return config
}

func (p *FilterManagerConfigParser) Parse(any *anypb.Any, callbacks capi.ConfigCallbackHandler) (interface{}, error) {
	configStruct := &xds.TypedStruct{}

	// No configuration
	if any.GetTypeUrl() == "" {
		conf := &filterManagerConfig{
			parsed: []*model.ParsedFilterConfig{},
		}
		return conf, nil
	}

	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	if configStruct.Value == nil {
		return nil, errors.New("bad TypedStruct format")
	}

	data, err := configStruct.Value.MarshalJSON()
	if err != nil {
		return nil, err
	}

	fmConfig := &FilterManagerConfig{}
	if err := json.Unmarshal(data, fmConfig); err != nil {
		return nil, err
	}

	plugins := fmConfig.Plugins
	conf := initFilterManagerConfig(fmConfig.Namespace)
	conf.parsed = make([]*model.ParsedFilterConfig, 0, len(plugins))

	consumerFiltersEndAt := 0
	i := 0

	for _, proto := range plugins {
		name := proto.Name
		if plugin := pkgPlugins.LoadHttpFilterFactoryAndParser(name); plugin != nil {
			// For now, we have nothing to provide as config callbacks
			config, err := plugin.ConfigParser.Parse(proto.Config, nil)
			if err != nil {
				api.LogErrorf("%s during parsing plugin %s in filtermanager", err, name)

				// Return an error from the Parse method will cause assertion failure.
				// See https://github.com/envoyproxy/envoy/blob/f301eebf7acc680e27e03396a1be6be77e1ae3a5/contrib/golang/filters/http/source/golang_filter.cc#L1736-L1737
				// As we can't control what is returned from a plugin, we need to
				// avoid the failure by providing a special factory, which also
				// indicates something is wrong.
				conf.parsed = append(conf.parsed, &model.ParsedFilterConfig{
					Name:    proto.Name,
					Factory: InternalErrorFactory,
				})
			} else {
				conf.parsed = append(conf.parsed, &model.ParsedFilterConfig{
					Name:         proto.Name,
					ParsedConfig: config,
					Factory:      plugin.Factory,
				})

				_, ok := pkgPlugins.LoadHttpPlugin(name).(pkgPlugins.ConsumerPlugin)
				if ok {
					consumerFiltersEndAt = i + 1
				}
			}
			i++

		} else {
			api.LogErrorf("plugin %s not found, ignored", name)
		}
	}
	conf.consumerFiltersEndAt = consumerFiltersEndAt

	return conf, nil
}

func (p *FilterManagerConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	// We have considered to implemented a Merge Policy between the LDS's filter & RDS's per route
	// config. A thought is to reuse the current Merge method. For example, considering we have
	// LDS:
	//	 - name: A
	//	   pet: cat
	// RDS:
	//	 - name: A
	//	   pet: dog
	// we will call plugin A's Merge method, which will produce `pet: [cat, dog]` or `pet: dog`.
	// As there is no real world use case for the Merge feature, I decide to delay its implementation
	// to avoid premature design.
	return child
}

type filterManager struct {
	filters         []*model.FilterWrapper
	consumerFilters []*model.FilterWrapper

	decodeRequestNeeded bool
	decodeIdx           int
	reqHdr              api.RequestHeaderMap

	encodeResponseNeeded bool
	encodeIdx            int
	rspHdr               api.ResponseHeaderMap

	// use a group of bools instead of map to avoid lookup
	canSkipDecodeHeaders bool
	canSkipDecodeData    bool
	canSkipEncodeHeaders bool
	canSkipEncodeData    bool
	canSkipOnLog         bool
	canSkipMethod        map[string]bool

	callbacks *filterManagerCallbackHandler
	config    *filterManagerConfig

	capi.PassThroughStreamFilter
}

func (m *filterManager) Reset() {
	m.filters = nil
	m.consumerFilters = nil

	m.decodeRequestNeeded = false
	m.decodeIdx = -1
	m.reqHdr = nil

	m.encodeResponseNeeded = false
	m.encodeIdx = -1
	m.rspHdr = nil

	m.canSkipDecodeHeaders = false
	m.canSkipDecodeData = false
	m.canSkipEncodeHeaders = false
	m.canSkipEncodeData = false
	m.canSkipOnLog = false

	m.callbacks.Reset()
}

type filterManagerRequestHeaderMap struct {
	capi.RequestHeaderMap

	u       *url.URL
	cookies map[string]*http.Cookie
}

func (headers *filterManagerRequestHeaderMap) expire(key string) {
	switch key {
	case ":path":
		headers.u = nil
	case "cookie":
		headers.cookies = nil
	}
}

func (headers *filterManagerRequestHeaderMap) Set(key, value string) {
	key = strings.ToLower(key)
	headers.expire(key)
	headers.RequestHeaderMap.Set(key, value)
}

func (headers *filterManagerRequestHeaderMap) Add(key, value string) {
	key = strings.ToLower(key)
	headers.expire(key)
	headers.RequestHeaderMap.Add(key, value)
}

func (headers *filterManagerRequestHeaderMap) Del(key string) {
	key = strings.ToLower(key)
	headers.expire(key)
	headers.RequestHeaderMap.Del(key)
}

func (headers *filterManagerRequestHeaderMap) Url() *url.URL {
	if headers.u == nil {
		path := headers.Path()
		u, err := url.ParseRequestURI(path)
		if err != nil {
			panic(fmt.Sprintf("unexpected bad request uri given by envoy: %v", err))
		}
		headers.u = u
	}
	return headers.u
}

// If multiple cookies match the given name, only one cookie will be returned.
func (headers *filterManagerRequestHeaderMap) Cookies() map[string]*http.Cookie {
	if headers.cookies == nil {
		headers.cookies = request.ParseCookies(headers)
	}
	return headers.cookies
}

type filterManagerStreamInfo struct {
	capi.StreamInfo

	ipAddress *api.IPAddress
}

func (s *filterManagerStreamInfo) DownstreamRemoteParsedAddress() *api.IPAddress {
	if s.ipAddress == nil {
		ipport := s.StreamInfo.DownstreamRemoteAddress()
		// the IPPort given by Envoy must be valid
		ip, port, _ := net.SplitHostPort(ipport)
		p, _ := strconv.Atoi(port)
		s.ipAddress = &api.IPAddress{
			Address: ipport,
			IP:      ip,
			Port:    p,
		}
	}
	return s.ipAddress
}

func (s *filterManagerStreamInfo) DownstreamRemoteAddress() string {
	if s.ipAddress != nil {
		return s.ipAddress.Address
	}
	return s.StreamInfo.DownstreamRemoteAddress()
}

type filterManagerCallbackHandler struct {
	capi.FilterCallbackHandler

	namespace string
	consumer  api.Consumer

	streamInfo *filterManagerStreamInfo
}

func (cb *filterManagerCallbackHandler) Reset() {
	cb.FilterCallbackHandler = nil
	// We don't reset namespace, as filterManager will only be reused in the same route,
	// which must have the same namespace.
	cb.consumer = nil
	cb.streamInfo = nil
}

func (cb *filterManagerCallbackHandler) StreamInfo() api.StreamInfo {
	if cb.streamInfo == nil {
		cb.streamInfo = &filterManagerStreamInfo{
			StreamInfo: cb.FilterCallbackHandler.StreamInfo(),
		}
	}
	return cb.streamInfo
}

func (cb *filterManagerCallbackHandler) LookupConsumer(pluginName, key string) (api.Consumer, bool) {
	return pkgConsumer.LookupConsumer(cb.namespace, pluginName, key)
}

func (cb *filterManagerCallbackHandler) GetConsumer() api.Consumer {
	return cb.consumer
}

func (cb *filterManagerCallbackHandler) SetConsumer(c api.Consumer) {
	if c == nil {
		api.LogErrorf("set consumer with nil consumer: %s", debug.Stack())
		return
	}
	api.LogInfof("set consumer, namespace: %s, name: %s", cb.namespace, c.Name())
	cb.consumer = c
}

type phase int

const (
	phaseDecodeHeaders phase = iota
	phaseDecodeData
	phaseDecodeTrailers
	phaseDecodeRequest
	phaseEncodeHeaders
	phaseEncodeData
	phaseEncodeTrailers
	phaseEncodeResponse
)

func newSkipMethodsMap() map[string]bool {
	return map[string]bool{
		"DecodeHeaders":  true,
		"DecodeData":     true,
		"DecodeRequest":  true,
		"EncodeHeaders":  true,
		"EncodeData":     true,
		"EncodeResponse": true,
		"OnLog":          true,
	}
}

func FilterManagerFactory(c interface{}, cb capi.FilterCallbackHandler) capi.StreamFilter {
	conf := c.(*filterManagerConfig)
	parsedConfig := conf.parsed

	fm := conf.pool.Get().(*filterManager)
	fm.callbacks.FilterCallbackHandler = cb

	canSkipMethod := fm.canSkipMethod
	if canSkipMethod == nil {
		canSkipMethod = newSkipMethodsMap()
	}

	filters := make([]*model.FilterWrapper, len(parsedConfig))
	for i, fc := range parsedConfig {
		factory := fc.Factory
		config := fc.ParsedConfig
		f := factory(config, fm.callbacks)
		// Technically, the factory might create different f for different calls. We don't support this edge case for now.
		if fm.canSkipMethod == nil {
			for meth := range canSkipMethod {
				overridden, err := reflectx.IsMethodOverridden(f, meth)
				if err != nil {
					api.LogErrorf("failed to check method %s in filter: %v", meth, err)
					// canSkipMethod[meth] will be false
				}
				canSkipMethod[meth] = canSkipMethod[meth] && !overridden
			}
			// Do we need to check if the correct method is defined? For example, the DecodeRequest
			// requires DecodeHeaders defined. Currently, we just documentate it. Per request check
			// is expensive and not necessary in most of time.
		}
		filters[i] = model.NewFilterWrapper(fc.Name, f)
	}

	if fm.canSkipMethod == nil {
		fm.canSkipMethod = canSkipMethod
	}

	// We can't cache the slice of filters as it may be changed by consumer
	fm.filters = filters

	if conf.consumerFiltersEndAt != 0 {
		consumerFiltersEndAt := conf.consumerFiltersEndAt
		consumerFilters := filters[:consumerFiltersEndAt]
		fm.consumerFilters = consumerFilters
		fm.filters = filters[consumerFiltersEndAt:]
	}

	// The skip check is based on the compiled code. So if the DecodeRequest is defined,
	// even it is not called, DecodeData will not be skipped. Same as EncodeResponse.
	fm.canSkipDecodeHeaders = fm.canSkipMethod["DecodeHeaders"]
	fm.canSkipDecodeData = fm.canSkipMethod["DecodeData"] && fm.canSkipMethod["DecodeRequest"]
	fm.canSkipEncodeHeaders = fm.canSkipMethod["EncodeHeaders"]
	fm.canSkipEncodeData = fm.canSkipMethod["EncodeData"] && fm.canSkipMethod["EncodeResponse"]
	fm.canSkipOnLog = fm.canSkipMethod["OnLog"]

	return fm
}

func (m *filterManager) handleAction(res api.ResultAction, phase phase) (needReturn bool) {
	if res == api.Continue {
		return false
	}
	if res == api.WaitAllData {
		if phase == phaseDecodeHeaders {
			m.decodeRequestNeeded = true
		} else if phase == phaseEncodeHeaders {
			m.encodeResponseNeeded = true
		} else {
			api.LogErrorf("WaitAllData only allowed when processing headers, phase: %v", phase)
		}
		return false
	}

	switch v := res.(type) {
	case *api.LocalResponse:
		m.localReply(v)
		return true
	default:
		api.LogErrorf("unknown result action: %+v", v)
		return false
	}
}

type jsonReply struct {
	Msg string `json:"msg"`
}

func (m *filterManager) localReply(v *api.LocalResponse) {
	var hdr map[string][]string
	if v.Header != nil {
		hdr = map[string][]string(v.Header)
	}
	if v.Code == 0 {
		v.Code = 200
	}

	msg := v.Msg
	// TODO: we can also add custom template response
	if msg != "" && len(hdr["Content-Type"]) == 0 {
		isJSON := false
		var ok bool
		var ct string
		if m.rspHdr != nil {
			ct, ok = m.rspHdr.Get("content-type")
		}

		if ok {
			if ct == "application/json" {
				isJSON = true
			}
		} else {
			ct, ok = m.reqHdr.Get("content-type")
			if !ok || ct == "application/json" {
				isJSON = true
			}
		}

		if isJSON {
			rsp := &jsonReply{Msg: msg}
			data, _ := json.Marshal(rsp)
			msg = string(data)
			if hdr == nil {
				hdr = map[string][]string{}
			}
			hdr["Content-Type"] = []string{"application/json"}
		}
	}
	m.callbacks.SendLocalReply(v.Code, msg, hdr, 0, "")
}

func (m *filterManager) DecodeHeaders(headers capi.RequestHeaderMap, endStream bool) capi.StatusType {
	if m.canSkipDecodeHeaders {
		return capi.Continue
	}

	go func() {
		defer m.callbacks.RecoverPanic()
		var res api.ResultAction

		headers := &filterManagerRequestHeaderMap{
			RequestHeaderMap: headers,
		}
		m.reqHdr = headers
		if len(m.consumerFilters) > 0 {
			for _, f := range m.consumerFilters {
				// Consumer plugins only use DecodeHeaders for now
				res = f.DecodeHeaders(headers, endStream)
				if m.handleAction(res, phaseDecodeHeaders) {
					return
				}
			}

			// we check consumer at the end of authn filters, so we can have multiple authn filters
			// configured and the consumer will be set by any of them
			c, ok := m.callbacks.consumer.(*pkgConsumer.Consumer)
			if !ok {
				api.LogInfo("reject for consumer not found")
				m.localReply(&api.LocalResponse{
					Code: 401,
					Msg:  "consumer not found",
				})
				return
			}

			if len(c.FilterConfigs) > 0 {
				if c.FilterWrappers == nil {
					c.FilterWrappers = make([]*model.FilterWrapper, 0, len(c.FilterConfigs))
					canSkipMethod := newSkipMethodsMap()
					names := make([]string, 0, len(c.FilterConfigs))
					for name, fc := range c.FilterConfigs {
						names = append(names, name)

						factory := fc.Factory
						config := fc.ParsedConfig
						f := factory(config, m.callbacks)
						for meth := range canSkipMethod {
							overridden, err := reflectx.IsMethodOverridden(f, meth)
							if err != nil {
								api.LogErrorf("failed to check method %s in filter: %v", meth, err)
								// canSkipMethod[meth] will be false
							}
							canSkipMethod[meth] = canSkipMethod[meth] && !overridden
						}
						nf := model.NewFilterWrapper(name, f)
						c.FilterWrappers = append(c.FilterWrappers, nf)
					}

					c.FilterNames = names
					c.CanSkipMethod = canSkipMethod
				} else {
					for i, name := range c.FilterNames {
						fc := c.FilterConfigs[name]
						factory := fc.Factory
						config := fc.ParsedConfig
						f := factory(config, m.callbacks)
						c.FilterWrappers[i].Filter = f
					}
				}

				canSkipMethod := c.CanSkipMethod
				m.canSkipDecodeData = m.canSkipDecodeData && canSkipMethod["DecodeData"] && canSkipMethod["DecodeRequest"]
				m.canSkipEncodeHeaders = m.canSkipEncodeData && canSkipMethod["EncodeHeaders"]
				m.canSkipEncodeData = m.canSkipEncodeData && canSkipMethod["EncodeData"] && canSkipMethod["EncodeResponse"]
				m.canSkipOnLog = m.canSkipOnLog && canSkipMethod["OnLog"]

				// TODO: add field to control if merging is allowed
				i := 0
				for _, f := range m.filters {
					if c.FilterConfigs[f.Name] == nil {
						m.filters[i] = f
						i++
					}
				}
				m.filters = append(m.filters[:i], c.FilterWrappers...)
				sort.Slice(m.filters, func(i, j int) bool {
					return pkgPlugins.ComparePluginOrder(m.filters[i].Name, m.filters[j].Name)
				})
			}
		}

		for i, f := range m.filters {
			res = f.DecodeHeaders(headers, endStream)
			if m.handleAction(res, phaseDecodeHeaders) {
				return
			}

			if m.decodeRequestNeeded {
				m.decodeRequestNeeded = false
				if !endStream {
					m.decodeIdx = i
					// some filters, like authorization with request body, need to
					// have a whole body before passing to the next filter
					m.callbacks.Continue(capi.StopAndBuffer)
					return
				}

				// no body
				res = f.DecodeRequest(headers, nil, nil)
				if m.handleAction(res, phaseDecodeRequest) {
					return
				}
			}
		}
		m.callbacks.Continue(capi.Continue)
	}()

	return capi.Running
}

func (m *filterManager) DecodeData(buf capi.BufferInstance, endStream bool) capi.StatusType {
	if m.canSkipDecodeData {
		return capi.Continue
	}

	go func() {
		defer m.callbacks.RecoverPanic()
		var res api.ResultAction

		// We have discussed a lot about how to support processing data both streamingly and
		// as a whole body. Here are some solutions we have considered:
		// 1. let Envoy process data streamingly, and do buffering in Go. This solution is costly
		// and may be broken if the buffered data at Go side is rewritten by later C++ filter.
		// 2. separate the filters which need a whole body in a separate C++ filter. It can't
		// be done without a special control plane.
		// 3. add multiple virtual C++ filters to Envoy when init the Envoy Golang filter. It
		// is complex because we need to share and move the state between multiple Envoy C++
		// filter.
		// 4. when a filter requires a whole body, all the filters will use a whole body.
		// Otherwise, streaming processing is used. It's simple and already satisfies our
		// most demand, so we choose this way for now.

		n := len(m.filters)
		if m.decodeIdx == -1 {
			// every filter doesn't need buffered body
			for i := 0; i < n; i++ {
				f := m.filters[i]
				res = f.DecodeData(buf, endStream)
				if m.handleAction(res, phaseDecodeData) {
					return
				}
			}
			m.callbacks.Continue(capi.Continue)

		} else {
			for i := 0; i < m.decodeIdx; i++ {
				f := m.filters[i]
				res = f.DecodeData(buf, endStream)
				if m.handleAction(res, phaseDecodeData) {
					return
				}
			}

			f := m.filters[m.decodeIdx]
			res = f.DecodeRequest(m.reqHdr, buf, nil)
			if m.handleAction(res, phaseDecodeRequest) {
				return
			}

			i := m.decodeIdx + 1
			for i < n {
				for ; i < n; i++ {
					f := m.filters[i]
					// The endStream in DecodeHeaders indicates whether there is a body.
					// The body always exists when we hit this path.
					res = f.DecodeHeaders(m.reqHdr, false)
					if m.handleAction(res, phaseDecodeHeaders) {
						return
					}
					if m.decodeRequestNeeded {
						// decodeRequestNeeded will be set to false below
						break
					}
				}

				// When there are multiple filters want to decode the whole req,
				// run part of the DecodeData which is before them
				for j := m.decodeIdx + 1; j < i; j++ {
					f := m.filters[j]
					res = f.DecodeData(buf, endStream)
					if m.handleAction(res, phaseDecodeData) {
						return
					}
				}

				if m.decodeRequestNeeded {
					m.decodeRequestNeeded = false
					m.decodeIdx = i
					f := m.filters[m.decodeIdx]
					res = f.DecodeRequest(m.reqHdr, buf, nil)
					if m.handleAction(res, phaseDecodeRequest) {
						return
					}
					i++
				}
			}

			m.callbacks.Continue(capi.Continue)
		}
	}()

	return capi.Running
}

func (m *filterManager) EncodeHeaders(headers capi.ResponseHeaderMap, endStream bool) capi.StatusType {
	if m.canSkipEncodeHeaders {
		return capi.Continue
	}

	go func() {
		defer m.callbacks.RecoverPanic()
		var res api.ResultAction

		m.rspHdr = headers
		n := len(m.filters)
		for i := n - 1; i >= 0; i-- {
			f := m.filters[i]
			res = f.EncodeHeaders(headers, endStream)
			if m.handleAction(res, phaseEncodeHeaders) {
				return
			}

			if m.encodeResponseNeeded {
				m.encodeResponseNeeded = false
				if !endStream {
					m.encodeIdx = i
					m.callbacks.Continue(capi.StopAndBuffer)
					return
				}

				// no body
				res = f.EncodeResponse(headers, nil, nil)
				if m.handleAction(res, phaseEncodeResponse) {
					return
				}
			}
		}
		m.callbacks.Continue(capi.Continue)
	}()

	return capi.Running
}

func (m *filterManager) EncodeData(buf capi.BufferInstance, endStream bool) capi.StatusType {
	if m.canSkipEncodeData {
		return capi.Continue
	}

	go func() {
		defer m.callbacks.RecoverPanic()
		var res api.ResultAction

		n := len(m.filters)
		if m.encodeIdx == -1 {
			// every filter doesn't need buffered body
			for i := n - 1; i >= 0; i-- {
				f := m.filters[i]
				res = f.EncodeData(buf, endStream)
				if m.handleAction(res, phaseEncodeData) {
					return
				}
			}
			m.callbacks.Continue(capi.Continue)

		} else {
			for i := n - 1; i > m.encodeIdx; i-- {
				f := m.filters[i]
				res = f.EncodeData(buf, endStream)
				if m.handleAction(res, phaseEncodeData) {
					return
				}
			}

			f := m.filters[m.encodeIdx]
			res = f.EncodeResponse(m.rspHdr, buf, nil)
			if m.handleAction(res, phaseEncodeResponse) {
				return
			}

			i := m.encodeIdx - 1
			for i >= 0 {
				for ; i >= 0; i-- {
					f := m.filters[i]
					res = f.EncodeHeaders(m.rspHdr, false)
					if m.handleAction(res, phaseEncodeHeaders) {
						return
					}
					if m.encodeResponseNeeded {
						// encodeResponseNeeded will be set to false below
						break
					}
				}

				for j := m.encodeIdx - 1; j > i; j-- {
					f := m.filters[j]
					res = f.EncodeData(buf, endStream)
					if m.handleAction(res, phaseEncodeData) {
						return
					}
				}

				if m.encodeResponseNeeded {
					m.encodeResponseNeeded = false
					m.encodeIdx = i
					f := m.filters[m.encodeIdx]
					res = f.EncodeResponse(m.rspHdr, buf, nil)
					if m.handleAction(res, phaseEncodeResponse) {
						return
					}
					i--
				}
			}

			m.callbacks.Continue(capi.Continue)
		}
	}()

	return capi.Running
}

// TODO: handle trailers

func (m *filterManager) OnLog() {
	if m.canSkipOnLog {
		return
	}

	// It is unsafe to access the f.callbacks in the goroutine, as the underlying request
	// may be destroyed when the goroutine is running. So if people want to do some IO jobs,
	// they need to copy the used data from the request to the Go side before kicking off
	// the goroutine.
	for _, f := range m.filters {
		f.OnLog()
	}

	m.Reset()
	m.config.pool.Put(m)
}
