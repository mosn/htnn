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
	"fmt"
	"reflect"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"

	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"mosn.io/htnn/api/internal/consumer"
	"mosn.io/htnn/api/internal/reflectx"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	pkgPlugins "mosn.io/htnn/api/pkg/plugins"
)

type filterManager struct {
	filters []*model.FilterWrapper

	decodeRequestNeeded bool
	decodeIdx           int
	reqHdr              api.RequestHeaderMap // don't access it in Encode phases
	contentType         string

	encodeResponseNeeded bool
	encodeIdx            int
	rspHdr               api.ResponseHeaderMap

	runningInGoThread atomic.Int32
	hdrLock           sync.Mutex // FIXME: remove this once we get request headers from the OnLog directly

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

	m.decodeRequestNeeded = false
	m.decodeIdx = -1
	m.reqHdr = nil
	m.contentType = ""

	m.encodeResponseNeeded = false
	m.encodeIdx = -1
	m.rspHdr = nil

	m.runningInGoThread.Store(0) // defence in depth

	m.canSkipDecodeHeaders = false
	m.canSkipDecodeData = false
	m.canSkipEncodeHeaders = false
	m.canSkipEncodeData = false
	m.canSkipOnLog = false

	m.callbacks.Reset()
}

func (m *filterManager) IsRunningInGoThread() bool {
	return m.runningInGoThread.Load() != 0
}

func (m *filterManager) MarkRunningInGoThread(flag bool) {
	if flag {
		m.runningInGoThread.Add(1)
	} else {
		m.runningInGoThread.Add(-1)
	}
}

func (m *filterManager) DebugModeEnabled() bool {
	return m.config.enableDebugMode
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

func needLogExecution() bool {
	return api.GetLogLevel() <= api.LogLevelDebug
}

func FilterManagerFactory(c interface{}, cb capi.FilterCallbackHandler) (streamFilter capi.StreamFilter) {
	// the RecoverPanic requires the underline Go req to be created. However, the Go req is created
	// after the FilterManagerFactory is called. So we implement our own RecoverPanic here to avoid breaking
	// the creation of Go req.
	defer func() {
		if p := recover(); p != nil {
			api.LogErrorf("panic: %v\n%s", p, debug.Stack())
			streamFilter = InternalErrorFactoryForCAPI(c, cb)
		}
	}()

	conf, ok := c.(*filterManagerConfig)
	if !ok {
		panic(fmt.Sprintf("wrong config type: %s", reflect.TypeOf(c)))
	}

	parsedConfig := conf.parsed

	data := conf.pool.Get()
	fm, ok := data.(*filterManager)
	if !ok {
		panic(fmt.Sprintf("unexpected type: %s", reflect.TypeOf(data)))
	}

	fm.callbacks.FilterCallbackHandler = cb

	canSkipMethod := fm.canSkipMethod
	if canSkipMethod == nil {
		canSkipMethod = newSkipMethodsMap()
	}

	filters := make([]*model.FilterWrapper, len(parsedConfig))
	logExecution := needLogExecution()
	for i, fc := range parsedConfig {
		factory := fc.Factory
		config := fc.ParsedConfig
		f := factory(config, fm.callbacks)
		// Technically, the factory might create different f for different calls. We don't support this edge case for now.
		if fm.canSkipMethod == nil {
			definedMethod := make(map[string]bool, len(canSkipMethod))
			for meth := range canSkipMethod {
				definedMethod[meth] = false
			}
			for meth := range canSkipMethod {
				overridden, err := reflectx.IsMethodOverridden(f, meth)
				if err != nil {
					api.LogErrorf("failed to check method %s in plugin %s: %v", meth, fc.Name, err)
					// canSkipMethod[meth] will be false
				}
				canSkipMethod[meth] = canSkipMethod[meth] && !overridden
				definedMethod[meth] = overridden
			}

			if definedMethod["DecodeRequest"] {
				if !definedMethod["DecodeHeaders"] {
					api.LogErrorf("plugin %s has DecodeRequest but not DecodeHeaders. To run DecodeRequest, we need to return api.WaitAllData from DecodeHeaders", fc.Name)
				}

				p := pkgPlugins.LoadPluginType(fc.Name)
				if p != nil {
					order := p.Order()
					if order.Position <= pkgPlugins.OrderPositionAuthn {
						api.LogErrorf("plugin %s has DecodeRequest which is not supported because the order of plugin", fc.Name)
					}
				}
			}
			if definedMethod["EncodeResponse"] && !definedMethod["EncodeHeaders"] {
				api.LogErrorf("plugin %s has EncodeResponse but not EncodeHeaders. To run EncodeResponse, we need to return api.WaitAllData from EncodeHeaders", fc.Name)
			}
		}

		if logExecution {
			filters[i] = model.NewFilterWrapper(fc.Name, NewLogExecutionFilter(fc.Name, f, fm.callbacks))
		} else {
			filters[i] = model.NewFilterWrapper(fc.Name, f)
		}

		if fm.DebugModeEnabled() {
			filters[i] = model.NewFilterWrapper(fc.Name, NewDebugFilter(fc.Name, filters[i].Filter, fm.callbacks))
		}
	}

	if fm.canSkipMethod == nil {
		fm.canSkipMethod = canSkipMethod
	}

	// We can't cache the slice of filters as it may be changed by consumer
	fm.filters = filters

	// The skip check is based on the compiled code. So if the DecodeRequest is defined,
	// even it is not called, DecodeData will not be skipped. Same as EncodeResponse.
	fm.canSkipDecodeHeaders = fm.canSkipMethod["DecodeHeaders"] && fm.canSkipMethod["DecodeRequest"] && fm.config.initOnce == nil
	fm.canSkipDecodeData = fm.canSkipMethod["DecodeData"] && fm.canSkipMethod["DecodeRequest"]
	fm.canSkipEncodeHeaders = fm.canSkipMethod["EncodeHeaders"]
	fm.canSkipEncodeData = fm.canSkipMethod["EncodeData"] && fm.canSkipMethod["EncodeResponse"]
	fm.canSkipOnLog = fm.canSkipMethod["OnLog"]

	return fm
}

func (m *filterManager) recordLocalReplyPluginName(name string) {
	// We can get the plugin name which returns the local response from the dynamic metadata.
	// For example, use %DYNAMIC_METADATA(htnn:local_reply_plugin_name)% in the access log format.
	m.callbacks.StreamInfo().DynamicMetadata().Set("htnn", "local_reply_plugin_name", name)
	// For now, we don't record when the local reply is caused by panic. As we can't always get
	// the name of plugin which is the root of the panic correctly. For example, consider a plugin kicks
	// off a goroutine and the goroutine panics.
}

func (m *filterManager) handleAction(res api.ResultAction, phase phase, filter *model.FilterWrapper) (needReturn bool) {
	if res == api.Continue {
		return false
	}
	if res == api.WaitAllData {
		if phase == phaseDecodeHeaders {
			m.decodeRequestNeeded = true
		} else if phase == phaseEncodeHeaders {
			m.encodeResponseNeeded = true
		} else {
			api.LogErrorf("WaitAllData only allowed when processing headers, phase: %v. "+
				" If you need to buffer the body, please use DecodeRequest or EncodeResponse instead", phase)
		}
		return false
	}

	switch v := res.(type) {
	case *api.LocalResponse:
		m.recordLocalReplyPluginName(filter.Name)
		m.localReply(v, phase < phaseEncodeHeaders)
		return true
	default:
		api.LogErrorf("unknown result action: %+v", v)
		return false
	}
}

type jsonReply struct {
	Msg string `json:"msg"`
}

func (m *filterManager) localReply(v *api.LocalResponse, decoding bool) {
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
		// note that the headers are just Go side cache. There may be Envoy side HTTP filter which
		// changes the content-type headers. But since most of the localReply happens in the early
		// DecodeHeaders phase, the chance to change to the content-type is very low. If this happens,
		// user can specify the content-type in the header by herself.
		if m.rspHdr != nil {
			ct, ok = m.rspHdr.Get("content-type")
		}

		if ok {
			if ct == "application/json" {
				isJSON = true
			}
		} else {
			// use the Content-Type header passed by the client, not the header
			// provided by the gateway if have.
			ct = m.contentType
			if ct == "" || ct == "application/json" {
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

	var cb api.FilterProcessCallbacks
	if decoding {
		cb = m.callbacks.DecoderFilterCallbacks()
	} else {
		cb = m.callbacks.EncoderFilterCallbacks()
	}
	cb.SendLocalReply(v.Code, msg, hdr, 0, "")
}

func (m *filterManager) DecodeHeaders(headers capi.RequestHeaderMap, endStream bool) capi.StatusType {
	m.contentType, _ = headers.Get("content-type")

	// Ensure the headers are cached on the Go side.
	// FIXME: remove this once we support OnLog phase headers in Envoy Go.
	if m.DebugModeEnabled() {
		headers := &filterManagerRequestHeaderMap{
			RequestHeaderMap: headers,
		}
		m.reqHdr = headers
	}

	if m.canSkipDecodeHeaders {
		return capi.Continue
	}

	m.MarkRunningInGoThread(true)

	go func() {
		defer m.MarkRunningInGoThread(false)
		defer m.callbacks.DecoderFilterCallbacks().RecoverPanic()
		var res api.ResultAction

		m.config.InitOnce()
		if m.config.initFailed {
			api.LogErrorf("error in plugin %s: %s", m.config.initFailedPluginName, m.config.initFailure)
			m.recordLocalReplyPluginName(m.config.initFailedPluginName)
			m.localReply(&api.LocalResponse{
				Code: 500,
			}, true)
			return
		}

		m.hdrLock.Lock()
		if m.reqHdr == nil {
			m.reqHdr = &filterManagerRequestHeaderMap{
				RequestHeaderMap: headers,
			}
		}
		m.hdrLock.Unlock()
		if m.config.consumerFiltersEndAt != 0 {
			for i := 0; i < m.config.consumerFiltersEndAt; i++ {
				f := m.filters[i]
				// We don't support DecodeRequest for now
				res = f.DecodeHeaders(m.reqHdr, endStream)
				if m.handleAction(res, phaseDecodeHeaders, f) {
					return
				}
			}

			// we check consumer at the end of authn filters, so we can have multiple authn filters
			// configured and the consumer will be set by any of them
			c, ok := m.callbacks.consumer.(*consumer.Consumer)
			if !ok {
				api.LogInfo("reject for consumer not found")
				m.localReply(&api.LocalResponse{
					Code: 401,
					Msg:  "consumer not found",
				}, true)
				return
			}

			if len(c.FilterConfigs) > 0 {
				api.LogDebugf("merge filters from consumer: %s", c.Name())

				c.InitOnce.Do(func() {
					names := make([]string, 0, len(c.FilterConfigs))
					for name, fc := range c.FilterConfigs {
						names = append(names, name)

						config := fc.ParsedConfig
						if initer, ok := config.(pkgPlugins.Initer); ok {
							// For now, we have nothing to provide as config callbacks
							err := initer.Init(nil)
							if err != nil {
								fc.Factory = NewInternalErrorFactory(fc.Name, err)
							}
						}
					}

					c.FilterNames = names
				})

				filterWrappers := make([]*model.FilterWrapper, len(c.FilterConfigs))
				for i, name := range c.FilterNames {
					fc := c.FilterConfigs[name]
					factory := fc.Factory
					config := fc.ParsedConfig
					f := factory(config, m.callbacks)
					filterWrappers[i] = model.NewFilterWrapper(name, f)
				}

				c.CanSkipMethodOnce.Do(func() {
					canSkipMethod := newSkipMethodsMap()
					for _, fw := range filterWrappers {
						f := fw.Filter
						for meth := range canSkipMethod {
							overridden, err := reflectx.IsMethodOverridden(f, meth)
							if err != nil {
								api.LogErrorf("failed to check method %s in filter: %v", meth, err)
								// canSkipMethod[meth] will be false
							}
							canSkipMethod[meth] = canSkipMethod[meth] && !overridden
						}
					}
					c.CanSkipMethod = canSkipMethod
				})

				if needLogExecution() {
					for _, fw := range filterWrappers {
						f := fw.Filter
						fw.Filter = NewLogExecutionFilter(fw.Name, f, m.callbacks)
					}
				}

				if m.DebugModeEnabled() {
					for _, fw := range filterWrappers {
						f := fw.Filter
						fw.Filter = NewDebugFilter(fw.Name, f, m.callbacks)
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
				m.filters = append(m.filters[:i], filterWrappers...)
				sort.Slice(m.filters, func(i, j int) bool {
					return pkgPlugins.ComparePluginOrder(m.filters[i].Name, m.filters[j].Name)
				})

				if api.GetLogLevel() <= api.LogLevelDebug {
					for _, f := range m.filters {
						fc := c.FilterConfigs[f.Name]
						if fc == nil {
							// the plugin is not from consumer
							for _, cfg := range m.config.parsed {
								if cfg.Name == f.Name {
									fc = cfg
									break
								}
							}
						}
						api.LogDebugf("after merged consumer, plugin: %s, config: %+v", f.Name, fc.ParsedConfig)
					}
				}
			}
		}

		for i := m.config.consumerFiltersEndAt; i < len(m.filters); i++ {
			f := m.filters[i]
			res = f.DecodeHeaders(m.reqHdr, endStream)
			if m.handleAction(res, phaseDecodeHeaders, f) {
				return
			}

			if m.decodeRequestNeeded {
				m.decodeRequestNeeded = false
				if !endStream {
					m.decodeIdx = i
					// some filters, like authorization with request body, need to
					// have a whole body before passing to the next filter
					m.callbacks.Continue(capi.StopAndBuffer, true)
					return
				}

				// no body
				res = f.DecodeRequest(m.reqHdr, nil, nil)
				if m.handleAction(res, phaseDecodeRequest, f) {
					return
				}
			}
		}

		m.callbacks.Continue(capi.Continue, true)
	}()

	return capi.Running
}

func (m *filterManager) DecodeData(buf capi.BufferInstance, endStream bool) capi.StatusType {
	if m.canSkipDecodeData {
		return capi.Continue
	}

	m.MarkRunningInGoThread(true)

	go func() {
		defer m.MarkRunningInGoThread(false)
		defer m.callbacks.DecoderFilterCallbacks().RecoverPanic()
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
				if m.handleAction(res, phaseDecodeData, f) {
					return
				}
			}
			m.callbacks.Continue(capi.Continue, true)

		} else {
			for i := 0; i < m.decodeIdx; i++ {
				f := m.filters[i]
				res = f.DecodeData(buf, endStream)
				if m.handleAction(res, phaseDecodeData, f) {
					return
				}
			}

			f := m.filters[m.decodeIdx]
			res = f.DecodeRequest(m.reqHdr, buf, nil)
			if m.handleAction(res, phaseDecodeRequest, f) {
				return
			}

			i := m.decodeIdx + 1
			for i < n {
				for ; i < n; i++ {
					f := m.filters[i]
					// The endStream in DecodeHeaders indicates whether there is a body.
					// The body always exists when we hit this path.
					res = f.DecodeHeaders(m.reqHdr, false)
					if m.handleAction(res, phaseDecodeHeaders, f) {
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
					if m.handleAction(res, phaseDecodeData, f) {
						return
					}
				}

				if m.decodeRequestNeeded {
					m.decodeRequestNeeded = false
					m.decodeIdx = i
					f := m.filters[m.decodeIdx]
					res = f.DecodeRequest(m.reqHdr, buf, nil)
					if m.handleAction(res, phaseDecodeRequest, f) {
						return
					}
					i++
				}
			}

			m.callbacks.Continue(capi.Continue, true)
		}
	}()

	return capi.Running
}

func (m *filterManager) EncodeHeaders(headers capi.ResponseHeaderMap, endStream bool) capi.StatusType {
	// Ensure the headers are cached on the Go side.
	// FIXME: remove this once we support OnLog phase headers in Envoy Go.
	if m.DebugModeEnabled() {
		headers.Get("test")
		m.rspHdr = headers
	}

	if m.canSkipEncodeHeaders {
		return capi.Continue
	}

	m.MarkRunningInGoThread(true)

	go func() {
		defer m.MarkRunningInGoThread(false)
		defer m.callbacks.EncoderFilterCallbacks().RecoverPanic()
		var res api.ResultAction

		m.hdrLock.Lock()
		m.rspHdr = headers
		m.hdrLock.Unlock()
		n := len(m.filters)
		for i := n - 1; i >= 0; i-- {
			f := m.filters[i]
			res = f.EncodeHeaders(headers, endStream)
			if m.handleAction(res, phaseEncodeHeaders, f) {
				return
			}

			if m.encodeResponseNeeded {
				m.encodeResponseNeeded = false
				if !endStream {
					m.encodeIdx = i
					m.callbacks.Continue(capi.StopAndBuffer, false)
					return
				}

				// no body
				res = f.EncodeResponse(headers, nil, nil)
				if m.handleAction(res, phaseEncodeResponse, f) {
					return
				}
			}
		}

		m.callbacks.Continue(capi.Continue, false)
	}()

	return capi.Running
}

func (m *filterManager) EncodeData(buf capi.BufferInstance, endStream bool) capi.StatusType {
	if m.canSkipEncodeData {
		return capi.Continue
	}

	m.MarkRunningInGoThread(true)

	go func() {
		defer m.MarkRunningInGoThread(false)
		defer m.callbacks.EncoderFilterCallbacks().RecoverPanic()
		var res api.ResultAction

		n := len(m.filters)
		if m.encodeIdx == -1 {
			// every filter doesn't need buffered body
			for i := n - 1; i >= 0; i-- {
				f := m.filters[i]
				res = f.EncodeData(buf, endStream)
				if m.handleAction(res, phaseEncodeData, f) {
					return
				}
			}
			m.callbacks.Continue(capi.Continue, false)

		} else {
			for i := n - 1; i > m.encodeIdx; i-- {
				f := m.filters[i]
				res = f.EncodeData(buf, endStream)
				if m.handleAction(res, phaseEncodeData, f) {
					return
				}
			}

			f := m.filters[m.encodeIdx]
			res = f.EncodeResponse(m.rspHdr, buf, nil)
			if m.handleAction(res, phaseEncodeResponse, f) {
				return
			}

			i := m.encodeIdx - 1
			for i >= 0 {
				for ; i >= 0; i-- {
					f := m.filters[i]
					res = f.EncodeHeaders(m.rspHdr, false)
					if m.handleAction(res, phaseEncodeHeaders, f) {
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
					if m.handleAction(res, phaseEncodeData, f) {
						return
					}
				}

				if m.encodeResponseNeeded {
					m.encodeResponseNeeded = false
					m.encodeIdx = i
					f := m.filters[m.encodeIdx]
					res = f.EncodeResponse(m.rspHdr, buf, nil)
					if m.handleAction(res, phaseEncodeResponse, f) {
						return
					}
					i--
				}
			}

			m.callbacks.Continue(capi.Continue, false)
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
	var reqHdr api.RequestHeaderMap
	m.hdrLock.Lock()
	reqHdr = m.reqHdr
	m.hdrLock.Unlock()
	var rspHdr api.ResponseHeaderMap
	m.hdrLock.Lock()
	rspHdr = m.rspHdr
	m.hdrLock.Unlock()

	for _, f := range m.filters {
		// TODO: the cached headers passed here is not precise. We need to get the real one via
		// Envoy Go API. But it is not supported yet.
		f.OnLog(reqHdr, nil, rspHdr, nil)
	}

	if m.IsRunningInGoThread() {
		return
	}

	// Safe to recycle the filterManager
	m.Reset()
	m.config.pool.Put(m)
}
