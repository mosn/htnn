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

func (m *filterManager) handleAction(res api.ResultAction) (needReturn bool) {
	if res == api.Continue {
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

func (m *filterManager) localReply(v *api.LocalResponse) {
	// TODO: support multiple same name header in Envoy
	var hdr map[string]string
	if v.Header != nil {
		hdr = map[string]string{}
		for k, vv := range map[string][]string(v.Header) {
			hdr[k] = vv[0]
		}
	}
	// TODO: provide JSON / gRPC reply according to the request info
	if v.Code == 0 {
		v.Code = 200
	}
	m.callbacks.SendLocalReply(v.Code, v.Msg, hdr, 0, "")
}

func (m *filterManager) DecodeHeaders(header api.RequestHeaderMap, endStream bool) capi.StatusType {
	go func() {
		defer m.callbacks.RecoverPanic()
		var res api.ResultAction

		for i, f := range m.filters {
			needed := f.NeedDecodeWholeRequest(header)
			if needed {
				if !endStream {
					m.decodeIdx = i
					m.reqHdr = header
					// some filters, like authorization with request body, need to
					// have a whole body before passing to the next filter
					m.callbacks.Continue(capi.StopAndBuffer)
					return
				}

				// no body
				res = f.DecodeRequest(header, nil, nil)
			} else {
				res = f.DecodeHeaders(header, endStream)
			}

			if m.handleAction(res) {
				return
			}
		}
		m.callbacks.Continue(capi.Continue)
	}()

	return capi.Running
}

func (m *filterManager) DecodeData(buf api.BufferInstance, endStream bool) capi.StatusType {
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
				if m.handleAction(res) {
					return
				}
			}
			m.callbacks.Continue(capi.Continue)

		} else {
			for i := 0; i < m.decodeIdx; i++ {
				f := m.filters[i]
				res = f.DecodeData(buf, endStream)
				if m.handleAction(res) {
					return
				}
			}

			f := m.filters[m.decodeIdx]
			res = f.DecodeRequest(m.reqHdr, buf, nil)
			if m.handleAction(res) {
				return
			}

			for i := m.decodeIdx + 1; i < n; i++ {
				f := m.filters[i]
				needed := f.NeedDecodeWholeRequest(m.reqHdr)
				if needed {
					// When there are multiple filters want to decode the whole req,
					// run part of the DecodeData which is before them
					for j := m.decodeIdx + 1; j < i; j++ {
						prevF := m.filters[j]
						res = prevF.DecodeData(buf, endStream)
						if m.handleAction(res) {
							return
						}
					}

					res = f.DecodeRequest(m.reqHdr, buf, nil)
					m.decodeIdx = i
				} else {
					res = f.DecodeHeaders(m.reqHdr, endStream)
				}

				if m.handleAction(res) {
					return
				}
			}

			for j := m.decodeIdx + 1; j < n; j++ {
				f := m.filters[j]
				res = f.DecodeData(buf, endStream)
				if m.handleAction(res) {
					return
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
		var res api.ResultAction

		n := len(m.filters)
		for i := n - 1; i >= 0; i-- {
			f := m.filters[i]
			needed := f.NeedEncodeWholeResponse(header)
			if needed {
				if !endStream {
					m.encodeIdx = i
					m.rspHdr = header
					m.callbacks.Continue(capi.StopAndBuffer)
					return
				}

				// no body
				res = f.EncodeResponse(header, nil, nil)
			} else {
				res = f.EncodeHeaders(header, endStream)
			}

			if m.handleAction(res) {
				return
			}
		}
		m.callbacks.Continue(capi.Continue)
	}()

	return capi.Running
}

func (m *filterManager) EncodeData(buf api.BufferInstance, endStream bool) capi.StatusType {
	go func() {
		defer m.callbacks.RecoverPanic()
		var res api.ResultAction

		n := len(m.filters)
		if m.encodeIdx == -1 {
			// every filter doesn't need buffered body
			for i := n - 1; i >= 0; i-- {
				f := m.filters[i]
				res = f.EncodeData(buf, endStream)
				if m.handleAction(res) {
					return
				}
			}
			m.callbacks.Continue(capi.Continue)

		} else {
			for i := n - 1; i >= 0; i-- {
				f := m.filters[i]

				if i > m.encodeIdx {
					res = f.EncodeData(buf, endStream)
				} else if i == m.encodeIdx {
					res = f.EncodeResponse(m.rspHdr, buf, nil)
				} else {
					needed := f.NeedEncodeWholeResponse(m.rspHdr)
					if needed {
						for j := m.encodeIdx - 1; j > i; j-- {
							prevF := m.filters[j]
							res = prevF.EncodeData(buf, endStream)
							if m.handleAction(res) {
								return
							}
						}

						res = f.EncodeResponse(m.rspHdr, buf, nil)
						m.encodeIdx = i
					} else {
						res = f.EncodeHeaders(m.rspHdr, endStream)
					}
				}

				if m.handleAction(res) {
					return
				}
			}

			for j := m.encodeIdx - 1; j >= 0; j-- {
				f := m.filters[j]
				res = f.EncodeData(buf, endStream)
				if m.handleAction(res) {
					return
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
