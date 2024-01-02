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
	"reflect"
	"runtime"
	"sync"

	xds "github.com/cncf/xds/go/xds/type/v3"
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"

	"mosn.io/htnn/pkg/filtermanager/api"
)

var (
	httpFilterConfigFactoryAndParser = sync.Map{}
)

// Here we introduce extra struct to avoid cyclic import between pkg/filtermanager and pkg/plugins
type FilterConfigParser interface {
	Parse(input interface{}, callbacks api.ConfigCallbackHandler) (interface{}, error)
	Merge(parentConfig interface{}, childConfig interface{}) interface{}
}

type filterConfigFactoryAndParser struct {
	configParser  FilterConfigParser
	configFactory api.FilterConfigFactory
}

// We can't import package below here that will cause build failure in Mac
// "github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
// Therefore, the FilterManagerConfigParser & FilterManagerConfigFactory need to be exportable.
// The http.RegisterHttpFilterConfigFactoryAndParser will be called in the main.go when building
// the shared library in Linux.

type FilterManagerConfigParser struct {
}

type FilterConfig struct {
	Name   string      `json:"name"`
	Config interface{} `json:"config"`
}

type FilterManagerConfig struct {
	Plugins []*FilterConfig `json:"plugins"`
}

type filterConfig struct {
	Name         string
	parsedConfig interface{}
}

type filterManagerConfig struct {
	current []*filterConfig
}

func (p *FilterManagerConfigParser) Parse(any *anypb.Any, callbacks capi.ConfigCallbackHandler) (interface{}, error) {
	configStruct := &xds.TypedStruct{}

	// No configuration
	if any.GetTypeUrl() == "" {
		conf := &filterManagerConfig{
			current: []*filterConfig{},
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
	conf := &filterManagerConfig{
		current: make([]*filterConfig, 0, len(plugins)),
	}
	for _, proto := range plugins {
		name := proto.Name
		if v, ok := httpFilterConfigFactoryAndParser.Load(name); ok {
			plugin := v.(*filterConfigFactoryAndParser)
			// For now, we have nothing to provide as config callbacks
			config, err := plugin.configParser.Parse(proto.Config, nil)
			if err != nil {
				return nil, fmt.Errorf("%w during parsing plugin %s in filtermanager", err, name)
			}

			conf.current = append(conf.current, &filterConfig{
				Name:         proto.Name,
				parsedConfig: config,
			})
		} else {
			api.LogErrorf("plugin %s not found, ignored", name)
		}
	}

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
	filters []api.Filter
	names   []string

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

	callbacks capi.FilterCallbackHandler

	capi.PassThroughStreamFilter
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

func isMethodFromPassThroughFilter(filter api.Filter, methodName string) (bool, error) {
	v := reflect.ValueOf(filter)
	// method by us must exist
	m, _ := v.Type().MethodByName(methodName)
	// Quoted from the doc:
	// the returned pointer is an underlying code pointer, but not necessarily enough to identify a
	// single function uniquely.
	// But as the filter is created statically and Go doesn't do JIT, it should be enough.
	// Since we have integration test for every plugin, if a plugin is skipped by mistake, we will find it.
	p := uintptr(m.Func.UnsafePointer())
	f := runtime.FuncForPC(p)
	if f == nil {
		return false, errors.New("invalid function")
	}

	fileName, _ := f.FileLine(f.Entry())
	wrapped := fileName == "<autogenerated>"
	return wrapped, nil
}

func FilterManagerConfigFactory(c interface{}) capi.StreamFilterFactory {
	conf := c.(*filterManagerConfig)
	newConfig := conf.current
	factories := make([]api.FilterFactory, len(newConfig))
	names := make([]string, len(factories))
	for i, fc := range newConfig {
		var factory api.FilterConfigFactory
		name := fc.Name
		names[i] = name
		if v, ok := httpFilterConfigFactoryAndParser.Load(name); ok {
			plugin := v.(*filterConfigFactoryAndParser)
			factory = plugin.configFactory
			config := fc.parsedConfig
			factories[i] = factory(config)

		} else {
			api.LogErrorf("plugin %s not found, pass through by default", name)
			factory = PassThroughFactory
			factories[i] = factory(nil)
		}
	}

	return func(callbacks capi.FilterCallbackHandler) capi.StreamFilter {
		filters := make([]api.Filter, len(factories))
		canSkipMethod := map[string]bool{
			"DecodeHeaders":  true,
			"DecodeData":     true,
			"DecodeRequest":  true,
			"EncodeHeaders":  true,
			"EncodeData":     true,
			"EncodeResponse": true,
			"OnLog":          true,
		}

		for i, factory := range factories {
			f := factory(callbacks)
			for meth := range canSkipMethod {
				ok, err := isMethodFromPassThroughFilter(f, meth)
				if err != nil {
					api.LogErrorf("failed to check method %s in filter %s: %v", meth, names[i], err)
				}
				canSkipMethod[meth] = ok
			}
			filters[i] = f
		}

		fm := &filterManager{
			callbacks: callbacks,
			names:     names,
			filters:   filters,

			decodeIdx: -1,
			encodeIdx: -1,
		}

		// The skip check is based on the compiled code. So if the DecodeRequest is defined,
		// even it is not called, DecodeData will not be skipped. Same as EncodeResponse.
		fm.canSkipDecodeHeaders = canSkipMethod["DecodeHeaders"]
		fm.canSkipDecodeData = canSkipMethod["DecodeData"] && canSkipMethod["DecodeRequest"]
		fm.canSkipEncodeHeaders = canSkipMethod["EncodeHeaders"]
		fm.canSkipEncodeData = canSkipMethod["EncodeData"] && canSkipMethod["EncodeResponse"]
		fm.canSkipOnLog = canSkipMethod["OnLog"]

		return fm
	}
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

func (m *filterManager) DecodeHeaders(header api.RequestHeaderMap, endStream bool) capi.StatusType {
	if m.canSkipDecodeHeaders {
		return capi.Continue
	}

	go func() {
		defer m.callbacks.RecoverPanic()
		var res api.ResultAction

		m.reqHdr = header
		for i, f := range m.filters {
			res = f.DecodeHeaders(header, endStream)
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
				res = f.DecodeRequest(header, nil, nil)
				if m.handleAction(res, phaseDecodeRequest) {
					return
				}
			}
		}
		m.callbacks.Continue(capi.Continue)
	}()

	return capi.Running
}

func (m *filterManager) DecodeData(buf api.BufferInstance, endStream bool) capi.StatusType {
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

func (m *filterManager) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) capi.StatusType {
	if m.canSkipEncodeHeaders {
		return capi.Continue
	}

	go func() {
		defer m.callbacks.RecoverPanic()
		var res api.ResultAction

		m.rspHdr = header
		n := len(m.filters)
		for i := n - 1; i >= 0; i-- {
			f := m.filters[i]
			res = f.EncodeHeaders(header, endStream)
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
				res = f.EncodeResponse(header, nil, nil)
				if m.handleAction(res, phaseEncodeResponse) {
					return
				}
			}
		}
		m.callbacks.Continue(capi.Continue)
	}()

	return capi.Running
}

func (m *filterManager) EncodeData(buf api.BufferInstance, endStream bool) capi.StatusType {
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

	for _, f := range m.filters {
		f.OnLog()
	}
}

func RegisterHttpFilterConfigFactoryAndParser(name string, factory api.FilterConfigFactory, parser FilterConfigParser) {
	if factory == nil {
		panic("config factory should not be nil")
	}
	httpFilterConfigFactoryAndParser.Store(name, &filterConfigFactoryAndParser{
		parser,
		factory,
	})
}
