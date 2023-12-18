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
	"encoding/json"

	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/request"
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

var opaResponse struct {
	Result struct {
		Allow bool `json:"allow"`
	} `json:"result"`
}

func (f *filter) buildInput(header api.RequestHeaderMap) map[string]interface{} {
	uri := request.GetUrl(header)
	headers := request.GetHeaders(header)
	req := map[string]interface{}{
		"method":  header.Method(),
		"scheme":  header.Scheme(),
		"host":    header.Host(),
		"path":    uri.Path,
		"headers": headers,
	}
	if uri.RawQuery != "" {
		req["query"] = map[string][]string(uri.Query())
	}

	return map[string]interface{}{
		"input": map[string]interface{}{
			"request": req,
		},
	}
}

func (f *filter) isAllowed(input map[string]interface{}) (bool, error) {
	params, err := json.Marshal(input)
	if err != nil {
		return false, err
	}

	remote := f.config.GetRemote()
	// When parsing the config, we have already validated the remote is not nil
	path := remote.GetUrl() + "/v1/data/" + remote.GetPolicy()
	api.LogInfof("send request to opa: %s, param: %s", path, params)
	resp, err := f.config.client.Post(path, "application/json", bytes.NewReader(params))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&opaResponse); err != nil {
		return false, err
	}

	return opaResponse.Result.Allow, nil
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.ResultAction {
	input := f.buildInput(header)
	allow, err := f.isAllowed(input)
	if err != nil {
		api.LogErrorf("failed to call OPA server: %v", err)
		return &api.LocalResponse{Code: 503}
	}

	if !allow {
		return &api.LocalResponse{Code: 403}
	}
	return api.Continue
}
