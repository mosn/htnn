package ext_auth

import (
	"bytes"
	"net/http"
	"net/url"

	"mosn.io/moe/pkg/filtermanager/api"
)

func configFactory(c interface{}) api.FilterFactory {
	conf := c.(*config)
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &filter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *config
}

func (f *filter) check(headers api.RequestHeaderMap) api.ResultAction {
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

	rsp, err := f.config.client.Do(req)
	if err != nil {
		api.LogWarnf("failed to call ext authz server: %v", err)
		code := int(hs.GetStatusOnError())
		if code == 0 {
			code = 403
		}
		return &api.LocalResponse{Code: code}
	}

	rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return &api.LocalResponse{Code: rsp.StatusCode, Header: rsp.Header}
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
	return f.check(headers)
}
