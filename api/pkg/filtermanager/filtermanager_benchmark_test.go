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
	"fmt"
	"net/http"
	"strconv"
	"testing"

	internalConsumer "mosn.io/htnn/api/internal/consumer"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
)

// Benchmark code
// go test -v -cpu=1 -run=none -bench=. -benchmem -memprofile memprofile.out -cpuprofile cpuprofile.out ./pkg/filtermanager/

func BenchmarkFilterManagerAllPhase(b *testing.B) {
	envoy.DisableLogInTest() // otherwise, there is too much output
	cb := envoy.NewCAPIFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "allPhase",
			Factory: PassThroughFactory,
		},
	}
	reqHdr := envoy.NewRequestHeaderMap(http.Header{})
	respHdr := envoy.NewResponseHeaderMap(http.Header{})
	reqBuf := envoy.NewBufferInstance([]byte{})
	respBuf := envoy.NewBufferInstance([]byte{})

	for n := 0; n < b.N; n++ {
		m := unwrapFilterManager(FilterManagerFactory(config, cb))
		m.DecodeHeaders(reqHdr, false)
		cb.WaitContinued()
		m.DecodeData(reqBuf, true)
		cb.WaitContinued()
		m.EncodeHeaders(respHdr, false)
		cb.WaitContinued()
		m.EncodeData(respBuf, true)
		cb.WaitContinued()
		m.OnLog(reqHdr, nil, respHdr, nil)
	}
}

func regularFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &regularFilter{}
}

type regularFilter struct {
	api.PassThroughFilter
}

// The majority route which has plugin configuration only has custom logic on DecodeHeaders and OnLog

func (f *regularFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	return api.Continue
}

func (f *regularFilter) OnLog(reqHeaders api.RequestHeaderMap, reqTrailers api.RequestTrailerMap,
	respHeaders api.ResponseHeaderMap, respTrailers api.ResponseTrailerMap) {
}

func BenchmarkFilterManagerRegular(b *testing.B) {
	envoy.DisableLogInTest() // otherwise, there is too much output
	cb := envoy.NewCAPIFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "regular",
			Factory: regularFactory,
		},
	}
	reqHdr := envoy.NewRequestHeaderMap(http.Header{})

	for n := 0; n < b.N; n++ {
		m := unwrapFilterManager(FilterManagerFactory(config, cb))
		m.DecodeHeaders(reqHdr, false)
		cb.WaitContinued()
		m.OnLog(reqHdr, nil, nil, nil)
	}
}

func BenchmarkFilterManagerConsumerWithFilter(b *testing.B) {
	envoy.DisableLogInTest() // otherwise, there is too much output
	cb := envoy.NewCAPIFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.consumerFiltersEndAt = 1

	consumers := map[string]*internalConsumer.Consumer{}
	num := 10
	reqHdrs := make([]api.RequestHeaderMap, num)
	for i := 0; i < num; i++ {
		c := internalConsumer.Consumer{
			FilterConfigs: map[string]*model.ParsedFilterConfig{
				"regular": {
					Name:    "regular",
					Factory: regularFactory,
				},
			},
		}
		consumers[strconv.Itoa(i)] = &c
		h := http.Header{}
		h.Add("Consumer", strconv.Itoa(i))
		reqHdrs[i] = envoy.NewRequestHeaderMap(h)
	}
	config.parsed = []*model.ParsedFilterConfig{
		{
			Name:    "set_consumer",
			Factory: setConsumerFactory,
			ParsedConfig: setConsumerConf{
				Consumers: consumers,
			},
		},
	}

	for n := 0; n < b.N; n++ {
		m := unwrapFilterManager(FilterManagerFactory(config, cb))
		m.DecodeHeaders(reqHdrs[n%num], false)
		cb.WaitContinued()
		m.OnLog(reqHdrs[n%num], nil, nil, nil)
	}
}

func BenchmarkFilterManagerDebugEnabled(b *testing.B) {
	envoy.DisableLogInTest() // otherwise, there is too much output
	cb := envoy.NewCAPIFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	pc := []*model.ParsedFilterConfig{}
	for i := 0; i < 5; i++ {
		pc = append(pc, &model.ParsedFilterConfig{
			Name:    fmt.Sprintf("all-%d", i),
			Factory: PassThroughFactory,
		})
	}
	config.parsed = pc
	config.enableDebugMode = true
	reqHdr := envoy.NewRequestHeaderMap(http.Header{})
	respHdr := envoy.NewResponseHeaderMap(http.Header{})
	reqBuf := envoy.NewBufferInstance([]byte{})
	respBuf := envoy.NewBufferInstance([]byte{})

	for n := 0; n < b.N; n++ {
		m := unwrapFilterManager(FilterManagerFactory(config, cb))
		m.DecodeHeaders(reqHdr, false)
		cb.WaitContinued()
		m.DecodeData(reqBuf, true)
		cb.WaitContinued()
		m.EncodeHeaders(respHdr, false)
		cb.WaitContinued()
		m.EncodeData(respBuf, true)
		cb.WaitContinued()
		m.OnLog(reqHdr, nil, respHdr, nil)
	}
}
