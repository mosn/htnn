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

package opa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/open-policy-agent/opa/rego"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/plugins/pkg/request"
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

type opaResponse struct {
	Result Result `json:"result"`
}

type Result struct {
	Allow          bool            `json:"allow"`
	CustomResponse *CustomResponse `json:"custom_response"`
}

type CustomResponse struct {
	Body       string              `json:"body"`
	Headers    map[string][]string `json:"headers"`
	StatusCode int                 `json:"status_code"`
}

func mapStrsToMapStr(strs map[string][]string) map[string]string {
	m := make(map[string]string)
	for k, v := range strs {
		n := len(v)
		if n > 0 {
			if n > 1 {
				m[k] = strings.Join(v, ",")
			} else {
				m[k] = v[0]
			}
		}
	}
	return m
}

func (f *filter) buildInput(header api.RequestHeaderMap) map[string]interface{} {
	uri := header.URL()
	headers := request.GetHeaders(header)
	req := map[string]interface{}{
		"method": header.Method(),
		"scheme": header.Scheme(),
		"host":   header.Host(),
		"path":   uri.Path,
		// It's inconvenient and error-proning to use []string in rego.
		// Dapr, APISIX, Kong all use a single string to represent header in their example.
		"headers": mapStrsToMapStr(headers),
	}
	if uri.RawQuery != "" {
		req["query"] = mapStrsToMapStr(uri.Query())
	}

	return map[string]interface{}{
		"input": map[string]interface{}{
			"request": req,
		},
	}
}

func (f *filter) isAllowed(input map[string]interface{}) (Result, error) {
	remote := f.config.GetRemote()
	if remote != nil {
		params, err := json.Marshal(input)
		if err != nil {
			return Result{Allow: false}, err
		}

		path := remote.GetUrl() + "/v1/data/" + remote.GetPolicy()
		api.LogInfof("send request to opa: %s, param: %s", path, params)
		resp, err := f.config.client.Post(path, "application/json", bytes.NewReader(params))
		if err != nil {
			return Result{Allow: false}, err
		}
		defer resp.Body.Close()

		var opaResponse opaResponse
		if err := json.NewDecoder(resp.Body).Decode(&opaResponse); err != nil {
			return Result{Allow: false}, err
		}

		return opaResponse.Result, nil
	}

	ctx := context.TODO()
	results, err := f.config.query.Eval(ctx, rego.EvalInput(input["input"]))
	if err != nil {
		return Result{Allow: false}, err
	}
	if len(results) == 0 {
		return Result{Allow: false}, errors.New("result is missing in the response")
	}

	result, allowOk := results[0].Bindings["allow"].(bool)
	if !allowOk {
		return Result{Allow: false}, errors.New("unexpected type for 'allow' binding in OPA result")
	}

	customResponseData, crExists := results[0].Bindings["custom_response"]
	if crExists {
		var customResp CustomResponse
		if responseMap, ok := customResponseData.(map[string]interface{}); ok {
			if bodyVal, found := responseMap["body"]; found {
				if bodyStr, isStr := bodyVal.(string); isStr {
					customResp.Body = bodyStr
				}
			}

			if statusCodeVal, found := responseMap["status_code"]; found {
				if v, ok := statusCodeVal.(json.Number); ok {
					if statusCodeInt, err := v.Int64(); err == nil {
						customResp.StatusCode = int(statusCodeInt)
					}
				}
			}

			if headersVal, found := responseMap["headers"]; found {
				if headersMapInterface, isMap := headersVal.(map[string]interface{}); isMap {
					parsedHeaders := make(map[string][]string)
					for key, valueInterface := range headersMapInterface {
						if valueSliceInterface, isSlice := valueInterface.([]interface{}); isSlice {
							var headerValues []string
							allStringsInSlice := true
							for _, itemInterface := range valueSliceInterface {
								if itemStr, isStr := itemInterface.(string); isStr {
									headerValues = append(headerValues, itemStr)
								} else {
									allStringsInSlice = false
									break
								}
							}
							if allStringsInSlice && len(headerValues) > 0 {
								parsedHeaders[key] = headerValues
							}
						}
					}

					if len(parsedHeaders) > 0 {
						customResp.Headers = parsedHeaders
					}
				}
			}
		}
		return Result{Allow: result, CustomResponse: &customResp}, nil
	}

	return Result{Allow: result, CustomResponse: nil}, nil
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	input := f.buildInput(headers)
	result, err := f.isAllowed(input)
	if err != nil {
		api.LogErrorf("failed to do OPA auth: %v", err)
		return &api.LocalResponse{Code: 503}
	}

	if !result.Allow {
		customResponse := result.CustomResponse
		if result.CustomResponse != nil {
			if customResponse.StatusCode == 0 {
				customResponse.StatusCode = 403
			}

			canonicalHeaders := make(http.Header)
			for key, values := range customResponse.Headers {
				canonicalKey := textproto.CanonicalMIMEHeaderKey(key)
				canonicalHeaders[canonicalKey] = values
			}

			if canonicalHeaders.Get("Content-Type") == "" {
				canonicalHeaders.Set("Content-Type", "text/plain")
			}

			return &api.LocalResponse{
				Code:   customResponse.StatusCode,
				Msg:    customResponse.Body,
				Header: canonicalHeaders,
			}
		}
		return &api.LocalResponse{Code: 403}
	}
	return api.Continue
}
