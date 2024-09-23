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

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	types "mosn.io/htnn/types/plugins/sentinel"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &filter{
		callbacks: callbacks,
		config:    c.(*config),
		ctx:       make(map[string]interface{}),
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *config
	ctx       map[string]interface{}
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
			Message:    "sentinel traffic control",
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

	f.ctx["_entry"] = e
	api.LogDebugf("passed, resource: %s", res)

	return api.Continue
}

func (f *filter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	e, exist := f.ctx["_entry"].(*base.SentinelEntry)
	if !exist {
		return api.Continue
	}
	defer e.Exit()

	gotSC, ok := headers.Status()
	if !ok {
		return api.Continue
	}

	for res, rule := range f.config.m.cb {
		for _, triggeredSC := range rule.GetTriggeredByStatusCodes() {
			if gotSC == int(triggeredSC) {
				sentinel.TraceError(e, fmt.Errorf("circuit breaker [%s] triggered by status code: %d", res, triggeredSC))
				break
			}
		}
	}

	return api.Continue
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
