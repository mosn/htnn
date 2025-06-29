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
	reqBuf              capi.BufferInstance  // don't access it in Encode phases

	encodeResponseNeeded bool
	encodeWaitFirstData  bool
	encodeIdx            int
	rspHdr               api.ResponseHeaderMap
	rspBuf               capi.BufferInstance

	runningInGoThread atomic.Int32
	hdrLock           sync.Mutex

	// use a group of bools instead of map to avoid lookup
	canSkipDecodeHeaders  bool
	canSkipDecodeData     bool
	canSkipDecodeTrailers bool
	canSkipEncodeHeaders  bool
	canSkipEncodeData     bool
	canSkipEncodeTrailers bool
	canSkipOnLog          bool
	canSkipMethods        map[string]bool

	canSyncRunDecodeHeaders  bool
	canSyncRunDecodeData     bool
	canSyncRunDecodeTrailers bool
	canSyncRunEncodeHeaders  bool
	canSyncRunEncodeData     bool
	canSyncRunEncodeTrailers bool
	canSyncRunMethods        map[string]bool

	callbacks *filterManagerCallbackHandler
	config    *filterManagerConfig

	capi.PassThroughStreamFilter
}

func (m *filterManager) Reset() {
	m.filters = nil

	m.decodeRequestNeeded = false
	m.decodeIdx = -1
	m.reqHdr = nil
	m.reqBuf = nil

	m.encodeResponseNeeded = false
	m.encodeWaitFirstData = false
	m.encodeIdx = -1
	m.rspHdr = nil
	m.rspBuf = nil

	m.runningInGoThread.Store(0) // defence in depth

	m.canSkipDecodeHeaders = false
	m.canSkipDecodeData = false
	m.canSkipDecodeTrailers = false
	m.canSkipEncodeHeaders = false
	m.canSkipEncodeData = false
	m.canSkipEncodeTrailers = false
	m.canSkipOnLog = false
	// m.canSkipMethods is reused across filters in the same config

	m.canSyncRunDecodeHeaders = false
	m.canSyncRunDecodeData = false
	m.canSyncRunDecodeTrailers = false
	m.canSyncRunEncodeHeaders = false
	m.canSyncRunEncodeData = false
	m.canSyncRunEncodeTrailers = false
	// m.canSyncRunMethods is reused across filters in the same config

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

	fm.callbacks.metrics = conf.metricsWriters

	canSkipMethods := fm.canSkipMethods
	canSyncRunMethods := fm.canSyncRunMethods
	if canSkipMethods == nil {
		canSkipMethods = api.NewAllMethodsMap()
		canSyncRunMethods = api.NewAllMethodsMap()
	}

	filters := make([]*model.FilterWrapper, len(parsedConfig))
	logExecution := needLogExecution()
	for i, fc := range parsedConfig {
		factory := fc.Factory
		config := fc.ParsedConfig
		f := factory(config, fm.callbacks)
		// Technically, the factory might create different f for different calls. We don't support this edge case for now.
		if fm.canSkipMethods == nil {
			definedMethod := make(map[string]bool, len(canSkipMethods))
			for meth := range canSkipMethods {
				definedMethod[meth] = false
			}
			for meth := range canSkipMethods {
				overridden, err := reflectx.IsMethodOverridden(f, meth)
				if err != nil {
					api.LogErrorf("failed to check method %s in plugin %s: %v", meth, fc.Name, err)
					// canSkipMethods[meth] will be false
				}
				canSkipMethods[meth] = canSkipMethods[meth] && !overridden
				definedMethod[meth] = overridden

				if overridden {
					canSyncRunMethods[meth] = canSyncRunMethods[meth] && fc.SyncRunPhases.Contains(api.MethodToPhase(meth))
				}
			}

			if definedMethod["DecodeRequest"] {
				if !definedMethod["DecodeHeaders"] {
					api.LogErrorf("plugin %s has DecodeRequest but not DecodeHeaders. To run DecodeRequest, we need to return api.WaitAllData from DecodeHeaders", fc.Name)
				}

				if conf.consumerFiltersEndAt != 0 {
					p := pkgPlugins.LoadPluginType(fc.Name)
					if p != nil {
						order := p.Order()
						if order.Position <= pkgPlugins.OrderPositionAuthn {
							api.LogErrorf("plugin %s has DecodeRequest which is not supported because the order of plugin", fc.Name)
						}
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

	if fm.canSkipMethods == nil {
		fm.canSkipMethods = canSkipMethods
		fm.canSyncRunMethods = canSyncRunMethods
	}

	// We can't cache the slice of filters as it may be changed by consumer
	fm.filters = filters

	// The skip check is based on the compiled code. So if the DecodeRequest is defined,
	// even it is not called, DecodeData will not be skipped. Same as EncodeResponse.
	fm.canSkipDecodeHeaders = fm.canSkipMethods["DecodeHeaders"] && fm.canSkipMethods["DecodeRequest"] && fm.config.initOnce == nil
	fm.canSkipDecodeData = fm.canSkipMethods["DecodeData"] && fm.canSkipMethods["DecodeRequest"]
	fm.canSkipDecodeTrailers = fm.canSkipMethods["DecodeTrailers"] && fm.canSkipMethods["DecodeRequest"]
	fm.canSkipEncodeHeaders = fm.canSkipMethods["EncodeHeaders"]
	fm.canSkipEncodeData = fm.canSkipMethods["EncodeData"] && fm.canSkipMethods["EncodeResponse"]
	fm.canSkipEncodeTrailers = fm.canSkipMethods["EncodeTrailers"] && fm.canSkipMethods["EncodeResponse"]
	fm.canSkipOnLog = fm.canSkipMethods["OnLog"]

	// Similar to the skip check, but the canSyncRun check is more granular as
	// it will consider if the request/response is fully buffered.
	fm.canSyncRunDecodeHeaders = fm.canSyncRunMethods["DecodeHeaders"] && fm.canSyncRunMethods["DecodeRequest"] && fm.config.initOnce == nil
	fm.canSyncRunDecodeData = fm.canSyncRunMethods["DecodeData"]
	fm.canSyncRunDecodeTrailers = fm.canSyncRunMethods["DecodeTrailers"]
	fm.canSyncRunEncodeHeaders = fm.canSyncRunMethods["EncodeHeaders"] && fm.canSyncRunMethods["EncodeResponse"]
	fm.canSyncRunEncodeData = fm.canSyncRunMethods["EncodeData"]
	fm.canSyncRunEncodeTrailers = fm.canSyncRunMethods["EncodeTrailers"]

	return wrapFilterManager(fm)
}

func (m *filterManager) recordLocalReplyPluginName(name string, code int) {
	// We can get the plugin name which returns the local response from the dynamic metadata.
	// For example, use %DYNAMIC_METADATA(htnn:local_reply_plugin_name)% in the access log format.
	m.callbacks.StreamInfo().DynamicMetadata().Set("htnn", "local_reply_plugin_name", name)
	// For now, we don't record when the local reply is caused by panic. As we can't always get
	// the name of plugin which is the root of the panic correctly. For example, consider a plugin kicks
	// off a goroutine and the goroutine panics.

	// Also log it in the application log. In some situation, multiple plugins may send local reply.
	// Via the application log, we can know all the calls.
	api.LogInfof("local reply from plugin: %s, code: %d", name, code)
}

func (m *filterManager) handleAction(res api.ResultAction, phase api.Phase, filter *model.FilterWrapper) (needReturn bool) {
	if res == api.Continue {
		return false
	}
	if res == api.WaitAllData {
		if phase == api.PhaseDecodeHeaders {
			m.decodeRequestNeeded = true
		} else if phase == api.PhaseEncodeHeaders {
			m.encodeResponseNeeded = true
		} else {
			api.LogErrorf("WaitAllData only allowed when processing headers, phase: %v. "+
				" In the mean time, use DecodeRequest /  EncodeResponse instead of DecodeData / EncodeData to handle fully buffered body.", phase)
		}
		return false
	}
	if res == api.WaitData {
		if phase == api.PhaseEncodeHeaders {
			m.encodeWaitFirstData = true
		} else {
			api.LogErrorf("WaitAllData only allowed when processing response headers, phase: %v.", phase)
		}
		return false
	}

	switch v := res.(type) {
	case *api.LocalResponse:
		m.recordLocalReplyPluginName(filter.Name, v.Code)
		m.localReply(v, phase < api.PhaseEncodeHeaders)
		return true
	default:
		api.LogErrorf("unknown result action: %+v returned from %s in phase %s", v, filter.Name, phase)
		return false
	}
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
		} else if decoding {
			ct, _ = m.reqHdr.Get("content-type")
			if ct == "" || ct == "application/json" {
				isJSON = true
			}
		}

		if isJSON {
			rsp := &api.DefaultJSONResponse{Msg: msg}
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
	cb.SendLocalReply(v.Code, msg, hdr, 0, v.Details)
}

func (m *filterManager) DecodeHeaders(headers capi.RequestHeaderMap, endStream bool) capi.StatusType {
	if !supportGettingHeadersOnLog {
		// Ensure the headers are cached on the Go side.
		headers.Get("test")
		headers := &filterManagerRequestHeaderMap{
			RequestHeaderMap: headers,
		}
		m.reqHdr = headers
	}

	if m.canSkipDecodeHeaders {
		return capi.Continue
	}

	if m.canSyncRunDecodeHeaders {
		return m.decodeHeaders(headers, endStream)
	}

	// We don't exact the repeated async pattern in a new method as it will require a closure to
	// wrap `m.decodeHeaders`, which makes this method 25% slower.
	m.MarkRunningInGoThread(true)

	go func() {
		defer m.MarkRunningInGoThread(false)
		defer m.callbacks.DecoderFilterCallbacks().RecoverPanic()

		res := m.decodeHeaders(headers, endStream)
		if res != capi.LocalReply {
			m.callbacks.Continue(res, true)
		}
	}()

	return capi.Running
}

func (m *filterManager) decodeHeaders(headers capi.RequestHeaderMap, endStream bool) capi.StatusType {
	var res api.ResultAction

	m.config.InitOnce()
	if m.config.initFailed {
		api.LogErrorf("error in plugin %s: %s", m.config.initFailedPluginName, m.config.initFailure)
		m.recordLocalReplyPluginName(m.config.initFailedPluginName, 500)
		m.localReply(&api.LocalResponse{
			Code: 500,
		}, true)
		return capi.LocalReply
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
			if m.handleAction(res, api.PhaseDecodeHeaders, f) {
				return capi.LocalReply
			}
		}

		// we check consumer at the end of authn filters, so we can have multiple authn filters
		// configured and the consumer will be set by any of them
		c, ok := m.callbacks.consumer.(*consumer.Consumer)
		if ok && len(c.FilterConfigs) > 0 {
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

			c.CanSkipMethodsOnce.Do(func() {
				canSkipMethods := api.NewAllMethodsMap()
				canSyncRunMethods := api.NewAllMethodsMap()
				for _, fw := range filterWrappers {
					f := fw.Filter
					fc := c.FilterConfigs[fw.Name]
					for meth := range canSkipMethods {
						overridden, err := reflectx.IsMethodOverridden(f, meth)
						if err != nil {
							api.LogErrorf("failed to check method %s in filter: %v", meth, err)
							// canSkipMethods[meth] will be false
						}
						canSkipMethods[meth] = canSkipMethods[meth] && !overridden

						if overridden {
							canSyncRunMethods[meth] = canSyncRunMethods[meth] && fc.SyncRunPhases.Contains(api.MethodToPhase(meth))
						}
					}
				}
				c.CanSkipMethods = canSkipMethods
				c.CanSyncRunMethod = canSyncRunMethods
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

			canSkipMethods := c.CanSkipMethods
			m.canSkipDecodeData = m.canSkipDecodeData && canSkipMethods["DecodeData"] && canSkipMethods["DecodeRequest"]
			m.canSkipDecodeTrailers = m.canSkipDecodeTrailers && canSkipMethods["DecodeTrailers"] && canSkipMethods["DecodeRequest"]
			m.canSkipEncodeHeaders = m.canSkipEncodeData && canSkipMethods["EncodeHeaders"]
			m.canSkipEncodeData = m.canSkipEncodeData && canSkipMethods["EncodeData"] && canSkipMethods["EncodeResponse"]
			m.canSkipEncodeTrailers = m.canSkipEncodeTrailers && canSkipMethods["EncodeTrailers"] && canSkipMethods["EncodeResponse"]
			m.canSkipOnLog = m.canSkipOnLog && canSkipMethods["OnLog"]

			canSyncRunMethods := c.CanSyncRunMethod
			m.canSyncRunDecodeHeaders = m.canSyncRunDecodeHeaders && canSyncRunMethods["DecodeHeaders"] && canSyncRunMethods["DecodeRequest"]
			m.canSyncRunDecodeData = m.canSyncRunDecodeData && canSyncRunMethods["DecodeData"]
			m.canSyncRunDecodeTrailers = m.canSyncRunDecodeTrailers && canSyncRunMethods["DecodeTrailers"]
			m.canSyncRunEncodeHeaders = m.canSyncRunEncodeHeaders && canSyncRunMethods["EncodeHeaders"] && canSyncRunMethods["EncodeResponse"]
			m.canSyncRunEncodeData = m.canSyncRunEncodeData && canSyncRunMethods["EncodeData"]
			m.canSyncRunEncodeTrailers = m.canSyncRunEncodeTrailers && canSyncRunMethods["EncodeTrailers"]

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
		if m.handleAction(res, api.PhaseDecodeHeaders, f) {
			return capi.LocalReply
		}

		if m.decodeRequestNeeded {
			m.decodeRequestNeeded = false
			if !endStream {
				m.decodeIdx = i
				// some filters, like authorization with request body, need to
				// have a whole body before passing to the next filter
				return capi.StopAndBuffer
			}

			// no body and no trailers
			res = f.DecodeRequest(m.reqHdr, nil, nil)
			if m.handleAction(res, api.PhaseDecodeRequest, f) {
				return capi.LocalReply
			}
		}
	}

	return capi.Continue
}

func (m *filterManager) DecodeRequest(headers api.RequestHeaderMap, buf capi.BufferInstance, trailers capi.RequestTrailerMap) bool {
	// for readable
	endStreamInBody := trailers == nil
	hasBody := buf != nil
	hasTrailers := trailers != nil

	var res api.ResultAction
	if hasBody {
		for i := 0; i < m.decodeIdx; i++ {
			f := m.filters[i]
			res = f.DecodeData(buf, endStreamInBody)
			if m.handleAction(res, api.PhaseDecodeData, f) {
				return false
			}
		}
	}

	// run DecodeTrailers as well after processing all the data
	if hasTrailers {
		for i := 0; i < m.decodeIdx; i++ {
			f := m.filters[i]
			res = f.DecodeTrailers(trailers)
			if m.handleAction(res, api.PhaseDecodeTrailers, f) {
				return false
			}
		}
	}

	f := m.filters[m.decodeIdx]
	res = f.DecodeRequest(headers, buf, trailers)
	if m.handleAction(res, api.PhaseDecodeRequest, f) {
		return false
	}

	n := len(m.filters)
	i := m.decodeIdx + 1
	for i < n {
		for ; i < n; i++ {
			f := m.filters[i]
			// The endStream in DecodeHeaders indicates whether there is a body.
			// The body always exists when we hit this path.
			res = f.DecodeHeaders(headers, false)
			if m.handleAction(res, api.PhaseDecodeHeaders, f) {
				return false
			}
			if m.decodeRequestNeeded {
				// decodeRequestNeeded will be set to false below
				break
			}
		}

		// When there are multiple filters want to decode the whole req,
		// run part of the DecodeData which is before them
		if hasBody {
			for j := m.decodeIdx + 1; j < i; j++ {
				f := m.filters[j]
				res = f.DecodeData(buf, endStreamInBody)
				if m.handleAction(res, api.PhaseDecodeData, f) {
					return false
				}
			}
		}

		if hasTrailers {
			for j := m.decodeIdx + 1; j < i; j++ {
				f := m.filters[j]
				res = f.DecodeTrailers(trailers)
				if m.handleAction(res, api.PhaseDecodeTrailers, f) {
					return false
				}
			}
		}

		if m.decodeRequestNeeded {
			m.decodeRequestNeeded = false
			m.decodeIdx = i
			f := m.filters[m.decodeIdx]
			res = f.DecodeRequest(headers, buf, trailers)
			if m.handleAction(res, api.PhaseDecodeRequest, f) {
				return false
			}
			i++
		}
	}

	return true
}

func (m *filterManager) DecodeData(buf capi.BufferInstance, endStream bool) capi.StatusType {
	if m.canSkipDecodeData {
		return capi.Continue
	}

	if m.canSyncRunDecodeData && (m.decodeIdx == -1 || (m.canSyncRunDecodeHeaders && m.canSyncRunDecodeTrailers)) {
		return m.decodeData(buf, endStream)
	}

	m.MarkRunningInGoThread(true)

	go func() {
		defer m.MarkRunningInGoThread(false)
		defer m.callbacks.DecoderFilterCallbacks().RecoverPanic()

		res := m.decodeData(buf, endStream)
		if res != capi.LocalReply {
			m.callbacks.Continue(res, true)
		}
	}()

	return capi.Running
}

func (m *filterManager) decodeData(buf capi.BufferInstance, endStream bool) capi.StatusType {
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

	status := capi.Continue
	n := len(m.filters)
	if m.decodeIdx == -1 {
		// every filter doesn't need buffered body
		for i := 0; i < n; i++ {
			f := m.filters[i]
			res = f.DecodeData(buf, endStream)
			if m.handleAction(res, api.PhaseDecodeData, f) {
				return capi.LocalReply
			}
		}
	} else if endStream {
		conti := m.DecodeRequest(m.reqHdr, buf, nil)
		if !conti {
			return capi.LocalReply
		}
	} else {
		m.reqBuf = buf
		status = capi.StopAndBuffer
	}

	return status
}

func (m *filterManager) DecodeTrailers(trailers capi.RequestTrailerMap) capi.StatusType {
	if m.canSkipDecodeTrailers {
		return capi.Continue
	}

	if m.canSyncRunDecodeTrailers && (m.decodeIdx == -1 || (m.canSyncRunDecodeHeaders && m.canSyncRunDecodeTrailers)) {
		return m.decodeTrailers(trailers)
	}

	m.MarkRunningInGoThread(true)

	go func() {
		defer m.MarkRunningInGoThread(false)
		defer m.callbacks.DecoderFilterCallbacks().RecoverPanic()

		res := m.decodeTrailers(trailers)
		if res != capi.LocalReply {
			m.callbacks.Continue(res, true)
		}
	}()

	return capi.Running
}

func (m *filterManager) decodeTrailers(trailers capi.RequestTrailerMap) capi.StatusType {
	var res api.ResultAction

	if m.decodeIdx == -1 {
		for _, f := range m.filters {
			res = f.DecodeTrailers(trailers)
			if m.handleAction(res, api.PhaseDecodeTrailers, f) {
				return capi.LocalReply
			}
		}
	} else {
		conti := m.DecodeRequest(m.reqHdr, m.reqBuf, trailers)
		if !conti {
			return capi.LocalReply
		}
	}

	return capi.Continue
}

func (m *filterManager) EncodeHeaders(headers capi.ResponseHeaderMap, endStream bool) capi.StatusType {
	if !supportGettingHeadersOnLog {
		// Ensure the headers are cached on the Go side.
		headers.Get("test")
		m.rspHdr = headers
	}

	if m.canSkipEncodeHeaders {
		return capi.Continue
	}

	if m.canSyncRunEncodeHeaders {
		return m.encodeHeaders(headers, endStream)
	}

	m.MarkRunningInGoThread(true)

	go func() {
		defer m.MarkRunningInGoThread(false)
		defer m.callbacks.EncoderFilterCallbacks().RecoverPanic()

		res := m.encodeHeaders(headers, endStream)
		if res != capi.LocalReply {
			m.callbacks.Continue(res, false)
		}
	}()

	return capi.Running
}

func (m *filterManager) encodeHeaders(headers capi.ResponseHeaderMap, endStream bool) capi.StatusType {
	var res api.ResultAction

	m.hdrLock.Lock()
	m.rspHdr = headers
	m.hdrLock.Unlock()
	n := len(m.filters)
	for i := n - 1; i >= 0; i-- {
		f := m.filters[i]
		res = f.EncodeHeaders(headers, endStream)
		if m.handleAction(res, api.PhaseEncodeHeaders, f) {
			return capi.LocalReply
		}

		if m.encodeResponseNeeded {
			m.encodeResponseNeeded = false
			if !endStream {
				m.encodeIdx = i
				return capi.StopAndBuffer
			}

			// no body
			res = f.EncodeResponse(headers, nil, nil)
			if m.handleAction(res, api.PhaseEncodeResponse, f) {
				return capi.LocalReply
			}
		}
	}

	if m.encodeWaitFirstData {
		return capi.StopAndBufferWatermark
	}
	return capi.Continue
}

func (m *filterManager) EncodeResponse(headers api.ResponseHeaderMap, buf capi.BufferInstance, trailers capi.ResponseTrailerMap) bool {
	endStreamInBody := trailers == nil
	hasBody := buf != nil
	hasTrailers := trailers != nil

	var res api.ResultAction
	n := len(m.filters)
	if hasBody {
		for i := n - 1; i > m.encodeIdx; i-- {
			f := m.filters[i]
			res = f.EncodeData(buf, endStreamInBody)
			if m.handleAction(res, api.PhaseEncodeData, f) {
				return false
			}
		}
	}

	if hasTrailers {
		for i := n - 1; i > m.encodeIdx; i-- {
			f := m.filters[i]
			res = f.EncodeTrailers(trailers)
			if m.handleAction(res, api.PhaseEncodeTrailers, f) {
				return false
			}
		}
	}

	f := m.filters[m.encodeIdx]
	res = f.EncodeResponse(m.rspHdr, buf, nil)
	if m.handleAction(res, api.PhaseEncodeResponse, f) {
		return false
	}

	i := m.encodeIdx - 1
	for i >= 0 {
		for ; i >= 0; i-- {
			f := m.filters[i]
			res = f.EncodeHeaders(m.rspHdr, false)
			if m.handleAction(res, api.PhaseEncodeHeaders, f) {
				return false
			}
			if m.encodeResponseNeeded {
				// encodeResponseNeeded will be set to false below
				break
			}
		}

		if hasBody {
			for j := m.encodeIdx - 1; j > i; j-- {
				f := m.filters[j]
				res = f.EncodeData(buf, endStreamInBody)
				if m.handleAction(res, api.PhaseEncodeData, f) {
					return false
				}
			}
		}

		if hasTrailers {
			for j := m.encodeIdx - 1; j > i; j-- {
				f := m.filters[j]
				res = f.EncodeTrailers(trailers)
				if m.handleAction(res, api.PhaseEncodeTrailers, f) {
					return false
				}
			}
		}

		if m.encodeResponseNeeded {
			m.encodeResponseNeeded = false
			m.encodeIdx = i
			f := m.filters[m.encodeIdx]
			res = f.EncodeResponse(m.rspHdr, buf, nil)
			if m.handleAction(res, api.PhaseEncodeResponse, f) {
				return false
			}
			i--
		}
	}

	return true
}

func (m *filterManager) EncodeData(buf capi.BufferInstance, endStream bool) capi.StatusType {
	if m.canSkipEncodeData {
		return capi.Continue
	}

	if m.canSyncRunEncodeData && (m.encodeIdx == -1 || (m.canSyncRunEncodeHeaders && m.canSyncRunEncodeTrailers)) {
		return m.encodeData(buf, endStream)
	}

	m.MarkRunningInGoThread(true)

	go func() {
		defer m.MarkRunningInGoThread(false)
		defer m.callbacks.EncoderFilterCallbacks().RecoverPanic()

		res := m.encodeData(buf, endStream)
		if res != capi.LocalReply {
			m.callbacks.Continue(res, false)
		}
	}()

	return capi.Running
}

func (m *filterManager) encodeData(buf capi.BufferInstance, endStream bool) capi.StatusType {
	var res api.ResultAction

	status := capi.Continue
	n := len(m.filters)
	if m.encodeIdx == -1 {
		// every filter doesn't need buffered body
		for i := n - 1; i >= 0; i-- {
			f := m.filters[i]
			res = f.EncodeData(buf, endStream)
			if m.handleAction(res, api.PhaseEncodeData, f) {
				return capi.LocalReply
			}
		}
	} else {
		// FIXME: we should implement like the decode part here, but it will cause server closed the stream without sending trailers.
		// As a result, we don't process the trailers in EncodeResponse for now.
		conti := m.EncodeResponse(m.rspHdr, buf, nil)
		if !conti {
			return capi.LocalReply
		}
	}

	return status
}

func (m *filterManager) EncodeTrailers(trailers capi.ResponseTrailerMap) capi.StatusType {
	if m.canSkipEncodeTrailers {
		return capi.Continue
	}

	if m.canSyncRunEncodeTrailers && (m.encodeIdx == -1 || (m.canSyncRunEncodeHeaders && m.canSyncRunEncodeData)) {
		return m.encodeTrailers(trailers)
	}

	m.MarkRunningInGoThread(true)

	go func() {
		defer m.MarkRunningInGoThread(false)
		defer m.callbacks.EncoderFilterCallbacks().RecoverPanic()

		res := m.encodeTrailers(trailers)
		if res != capi.LocalReply {
			m.callbacks.Continue(res, false)
		}
	}()

	return capi.Running
}

func (m *filterManager) encodeTrailers(trailers capi.ResponseTrailerMap) capi.StatusType {
	var res api.ResultAction

	if m.encodeIdx == -1 {
		for _, f := range m.filters {
			res = f.EncodeTrailers(trailers)
			if m.handleAction(res, api.PhaseEncodeTrailers, f) {
				return capi.LocalReply
			}
		}
	}

	return capi.Continue
}

func (m *filterManager) runOnLogPhase(reqHdr api.RequestHeaderMap, reqTrailer api.RequestTrailerMap,
	rspHdr api.ResponseHeaderMap, rspTrailer api.ResponseTrailerMap) {

	if m.DebugModeEnabled() {
		executionRecords := model.NewExecutionRecords()
		for _, f := range m.filters {
			if df, ok := f.Filter.(*debugFilter); ok {
				name, duration := df.reportExecution()
				executionRecords.Record(name, duration)
			}
		}
		m.callbacks.PluginState().Set("debugMode", "executionRecords", executionRecords)
	}

	// It is unsafe to access the f.callbacks in the goroutine, as the underlying request
	// may be destroyed when the goroutine is running. So if people want to do some IO jobs,
	// they need to copy the used data from the request to the Go side before kicking off
	// the goroutine.
	for _, f := range m.filters {
		f.OnLog(reqHdr, reqTrailer, rspHdr, rspTrailer)
	}

	if m.IsRunningInGoThread() {
		return
	}

	// Safe to recycle the filterManager
	m.Reset()
	m.config.pool.Put(m)
}
