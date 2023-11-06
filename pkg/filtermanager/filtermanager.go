package filtermanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	xds "github.com/cncf/xds/go/xds/type/v3"
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"

	"mosn.io/moe/pkg/filtermanager/api"
)

var (
	httpFilterConfigFactoryAndParser = sync.Map{}
)

type filterConfigFactoryAndParser struct {
	configParser  api.FilterConfigParser
	configFactory api.FilterConfigFactory
}

// we can't import package below here which will cause the integration test to fail in Mac
// "github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
// therefore, we choice to export these fields

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
	// TODO: We have considered to implemented a Merge Policy between the LDS's filter & RDS's per route
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

	decodeIdx int
	reqHdr    api.RequestHeaderMap

	encodeIdx int
	rspHdr    api.ResponseHeaderMap

	callbacks capi.FilterCallbackHandler

	capi.PassThroughStreamFilter
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
		for i, factory := range factories {
			filters[i] = factory(callbacks)
		}
		return &filterManager{
			callbacks: callbacks,
			names:     names,
			filters:   filters,

			decodeIdx: -1,
			encodeIdx: -1,
		}
	}
}

func (m *filterManager) DecodeHeaders(header api.RequestHeaderMap, endStream bool) capi.StatusType {
	go func() {
		defer m.callbacks.RecoverPanic()

		for i, f := range m.filters {
			runDecodeRequest := false
			if wr, ok := f.(api.DecodeWholeRequestFilter); ok {
				needed := wr.NeedDecodeWholeRequest(header)
				if needed {
					if !endStream {
						m.decodeIdx = i
						m.reqHdr = header
						m.callbacks.Continue(capi.StopAndBuffer)
						return
					}

					// no body
					runDecodeRequest = true
					wr.DecodeRequest(header, nil, nil)
				}
			}

			if !runDecodeRequest {
				f.DecodeHeaders(header, endStream)
			}
		}
		m.callbacks.Continue(capi.Continue)
	}()

	return capi.Running
}

func (m *filterManager) DecodeData(buf api.BufferInstance, endStream bool) capi.StatusType {
	go func() {
		defer m.callbacks.RecoverPanic()

		n := len(m.filters)
		if m.decodeIdx == -1 {
			// every filter doesn't need buffered body
			for i := 0; i < n; i++ {
				f := m.filters[i]
				f.DecodeData(buf, endStream)
			}
			m.callbacks.Continue(capi.Continue)

		} else {
			for i := 0; i < m.decodeIdx; i++ {
				f := m.filters[i]
				f.DecodeData(buf, endStream)
			}
			wr := m.filters[m.decodeIdx].(api.DecodeWholeRequestFilter)
			wr.DecodeRequest(m.reqHdr, buf, nil)
			for i := m.decodeIdx + 1; i < n; i++ {
				f := m.filters[i]
				runDecodeRequest := false
				if wr, ok := f.(api.DecodeWholeRequestFilter); ok {
					needed := wr.NeedDecodeWholeRequest(m.reqHdr)
					if needed {
						runDecodeRequest = true
						wr.DecodeRequest(m.reqHdr, buf, nil)
					}
				}
				if !runDecodeRequest {
					f.DecodeHeaders(m.reqHdr, endStream)
					f.DecodeData(buf, endStream)
				}
			}
			m.callbacks.Continue(capi.Continue)
		}
	}()

	return capi.Running
}

func (m *filterManager) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) capi.StatusType {
	go func() {
		defer m.callbacks.RecoverPanic()

		n := len(m.filters)
		for i := n - 1; i >= 0; i-- {
			f := m.filters[i]
			runEncodeResponse := false
			if wr, ok := f.(api.EncodeWholeResponseFilter); ok {
				needed := wr.NeedEncodeWholeResponse(header)
				if needed {
					if !endStream {
						m.encodeIdx = i
						m.rspHdr = header
						m.callbacks.Continue(capi.StopAndBuffer)
						return
					}

					// no body
					runEncodeResponse = true
					wr.EncodeResponse(header, nil, nil)
				}
			}

			if !runEncodeResponse {
				f.EncodeHeaders(header, endStream)
			}
		}
		m.callbacks.Continue(capi.Continue)
	}()

	return capi.Running
}

func (m *filterManager) EncodeData(buf api.BufferInstance, endStream bool) capi.StatusType {
	go func() {
		defer m.callbacks.RecoverPanic()

		n := len(m.filters)
		if m.encodeIdx == -1 {
			// every filter doesn't need buffered body
			for i := n - 1; i >= 0; i-- {
				f := m.filters[i]
				f.EncodeData(buf, endStream)
			}
			m.callbacks.Continue(capi.Continue)

		} else {
			for i := n - 1; i > m.encodeIdx; i-- {
				f := m.filters[i]
				f.EncodeData(buf, endStream)
			}
			wr := m.filters[m.encodeIdx].(api.EncodeWholeResponseFilter)
			wr.EncodeResponse(m.rspHdr, buf, nil)
			for i := m.encodeIdx - 1; i >= 0; i-- {
				f := m.filters[i]
				runEncodeResponse := false
				if wr, ok := f.(api.EncodeWholeResponseFilter); ok {
					needed := wr.NeedEncodeWholeResponse(m.rspHdr)
					if needed {
						runEncodeResponse = true
						wr.EncodeResponse(m.rspHdr, buf, nil)
					}
				}
				if !runEncodeResponse {
					f.EncodeHeaders(m.rspHdr, endStream)
					f.EncodeData(buf, endStream)
				}
			}
			m.callbacks.Continue(capi.Continue)
		}
	}()

	return capi.Running
}

// TODO: handle trailers

func (m *filterManager) OnLog() {
	for _, f := range m.filters {
		f.OnLog()
	}
}

func RegisterHttpFilterConfigFactoryAndParser(name string, factory api.FilterConfigFactory, parser api.FilterConfigParser) {
	if factory == nil {
		panic("config factory should not be nil")
	}
	httpFilterConfigFactoryAndParser.Store(name, &filterConfigFactoryAndParser{
		parser,
		factory,
	})
}
