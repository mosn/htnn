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

package sentinel

import (
	"fmt"
	"strings"
	"sync/atomic"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	types "mosn.io/htnn/types/plugins/sentinel"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &filter{
		callbacks: callbacks,
		config:    c.(*config),
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *config
	entry     atomic.Pointer[base.SentinelEntry]
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	res := f.getSource(f.config.GetResource(), headers)
	if res == "" {
		return api.Continue
	}
	api.LogDebugf("traffic control by: %s", res)

	var attachments = make(map[interface{}]interface{})
	for _, a := range f.config.attachments {
		if a == nil || a.GetKey() == "" {
			continue
		}

		v := f.getSource(a, headers)
		if v == "" {
			continue
		}

		attachments[a.GetKey()] = v
	}

	e, b := sentinel.Entry(res, f.config.params, sentinel.WithAttachments(attachments))

	if b != nil {
		api.LogDebugf("blocked, resource: %s, type: %s, rule: %+v, snapshot: %+v",
			res, b.BlockType().String(), b.TriggeredRule(), b.TriggeredValue())

		resp := &types.BlockResponse{
			Message:    "blocked by sentinel traffic control",
			StatusCode: 429,
		}

		switch b.BlockType() {
		case base.BlockTypeFlow:
			if r, exist := f.config.m.f[res]; exist && r.GetBlockResponse() != nil {
				resp = r.GetBlockResponse()
			}
		case base.BlockTypeHotSpotParamFlow:
			if r, exist := f.config.m.hs[res]; exist && r.GetBlockResponse() != nil {
				resp = r.GetBlockResponse()
			}
		case base.BlockTypeCircuitBreaking:
			if r, exist := f.config.m.cb[res]; exist && r.GetBlockResponse() != nil {
				resp = r.GetBlockResponse()
			}
		}

		header := make(map[string][]string)
		for k, v := range resp.Headers {
			vals := strings.Split(v, ",")
			for i := range vals {
				vals[i] = strings.TrimSpace(vals[i])
			}
			header[k] = vals
		}

		return &api.LocalResponse{
			Code:   int(resp.StatusCode),
			Msg:    resp.Message,
			Header: header,
		}
	}

	f.entry.Store(e)
	api.LogDebugf("passed, resource: %s", res)

	return api.Continue
}

func (f *filter) OnLog(reqHeaders api.RequestHeaderMap, reqTrailers api.RequestTrailerMap,
	respHeaders api.ResponseHeaderMap, respTrailers api.ResponseTrailerMap) {
	e := f.entry.Load()
	if e == nil {
		return
	}
	// Although only CircuitBreaker rules need to do Exit in response phase,
	// the statistics of metrics of Flow and HotSpot rules are completed in request phase (DecodeHeaders).
	// However, considering the boundary problems such as memory leakage caused by client interrupting the request,
	// we do Exit in OnLog after response phase.
	// See https://github.com/mosn/htnn/blob/main/site/content/en/docs/developer-guide/get_involved.md#filter
	defer e.Exit()

	gotSC, ok := respHeaders.Status()
	if !ok {
		api.LogWarn("failed to get response status code")
		return
	}

	for res, rule := range f.config.m.cb {
		// TODO(WeixinX): TriggeredByStatusCodes slice -> map, improve performance
		for _, triggeredSC := range rule.GetTriggeredByStatusCodes() {
			if gotSC == int(triggeredSC) {
				sentinel.TraceError(e, fmt.Errorf("circuit breaker [%s] triggered by status code: %d", res, triggeredSC))
				break
			}
		}
	}
}

func (f *filter) getSource(s *types.Source, headers api.RequestHeaderMap) string {
	var vs []string
	if s.GetFrom() == types.Source_HEADER {
		vs = headers.Values(s.GetKey())
	} else if s.GetFrom() == types.Source_QUERY {
		vs = headers.URL().Query()[s.GetKey()]
	}

	if len(vs) == 0 {
		return ""
	}

	return vs[0]
}
