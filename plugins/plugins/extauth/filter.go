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

package extauth

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"mosn.io/htnn/api/pkg/filtermanager/api"
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
}

func (f *filter) check(headers api.RequestHeaderMap, data api.BufferInstance) api.ResultAction {
	hs := f.config.GetHttpService()
	uri := hs.GetUrl()
	path, err := url.JoinPath(uri, headers.Path())
	if err != nil {
		api.LogWarnf("failed to join path: %v", err)
		return &api.LocalResponse{Code: 503}
	}

	req, err := http.NewRequest(headers.Method(), path, bytes.NewReader([]byte{}))
	if err != nil {
		api.LogWarnf("failed to new request to ext authz server: %v", err)
		return &api.LocalResponse{Code: 503}
	}
	req.Host = headers.Host()
	authz, ok := headers.Get("authorization")
	if ok {
		req.Header.Set("authorization", authz)
	}
	headersToAdd := hs.GetAuthorizationRequest().GetHeadersToAdd()
	for _, h := range headersToAdd {
		// Envoy doesn't support adding multiple same name headers here,
		// so we don't support it until it's needed
		req.Header.Set(h.Key, h.Value)
	}

	if data != nil {
		req.Body = io.NopCloser(bytes.NewReader(data.Bytes()))
	}

	rsp, err := f.config.client.Do(req)
	if err != nil || rsp.StatusCode >= 500 {
		if err != nil {
			api.LogWarnf("failed to call ext authz server: %v", err)
		} else {
			api.LogWarnf("failed to call ext authz server: %s", rsp.Status)
		}
		if f.config.GetFailureModeAllow() {
			if f.config.GetFailureModeAllowHeaderAdd() {
				headers.Set("x-envoy-auth-failure-mode-allowed", "true")
			}
			return api.Continue
		}
		code := int(hs.GetStatusOnError())
		if code == 0 {
			code = 403
		}
		return &api.LocalResponse{Code: code}
	}

	rsp.Body.Close()
	if rsp.StatusCode != 200 {
		rspHdr := rsp.Header
		if f.config.headerToClientMatcher != nil {
			rspHdr = http.Header{}
			for k, v := range rsp.Header {
				if f.config.headerToClientMatcher.Match(k) {
					for _, vv := range v {
						rspHdr.Add(k, vv)
					}
				}
			}
		}
		return &api.LocalResponse{Code: rsp.StatusCode, Header: rspHdr}
	}

	if f.config.headerToUpstreamMatcher != nil {
		for k, v := range rsp.Header {
			if f.config.headerToUpstreamMatcher.Match(k) {
				headers.Set(k, v[len(v)-1])
			}
		}
	}
	return api.Continue
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	if f.config.GetHttpService().GetWithRequestBody() {
		return api.WaitAllData
	}
	return f.check(headers, nil)
}

func (f *filter) DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction {
	return f.check(headers, data)
}
