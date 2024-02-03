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
	"net/http"
	"testing"

	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/filtermanager/model"
	"mosn.io/htnn/plugins/tests/pkg/envoy"
)

// Benchmark code
// go test -v -cpu=1 -run=none -bench=. -benchmem -memprofile memprofile.out -cpuprofile cpuprofile.out ./pkg/filtermanager/

func BenchmarkFilterManagerAllPhase(b *testing.B) {
	cb := envoy.NewFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.current = []*model.ParsedFilterConfig{
		{
			Name:          "allPhase",
			ConfigFactory: PassThroughFactory,
		},
	}
	reqHdr := envoy.NewRequestHeaderMap(http.Header{})
	respHdr := envoy.NewResponseHeaderMap(http.Header{})
	reqBuf := envoy.NewBufferInstance([]byte{})
	respBuf := envoy.NewBufferInstance([]byte{})

	for n := 0; n < b.N; n++ {
		m := FilterManagerConfigFactory(config)(cb)
		m.DecodeHeaders(reqHdr, false)
		cb.WaitContinued()
		m.DecodeData(reqBuf, true)
		cb.WaitContinued()
		m.EncodeHeaders(respHdr, false)
		cb.WaitContinued()
		m.EncodeData(respBuf, true)
		cb.WaitContinued()
		m.OnLog()
	}
}

func regularFactory(c interface{}) api.FilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &regularFilter{}
	}
}

type regularFilter struct {
	api.PassThroughFilter
}

// The majority route which has plugin configuration only has custom logic on DecodeHeaders and OnLog

func (f *regularFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	return api.Continue
}

func (f *regularFilter) OnLog() {
}

func BenchmarkFilterManagerRegular(b *testing.B) {
	cb := envoy.NewFilterCallbackHandler()
	config := initFilterManagerConfig("ns")
	config.current = []*model.ParsedFilterConfig{
		{
			Name:          "regular",
			ConfigFactory: regularFactory,
		},
	}
	reqHdr := envoy.NewRequestHeaderMap(http.Header{})

	for n := 0; n < b.N; n++ {
		m := FilterManagerConfigFactory(config)(cb)
		m.DecodeHeaders(reqHdr, false)
		cb.WaitContinued()
		m.OnLog()
	}
}
