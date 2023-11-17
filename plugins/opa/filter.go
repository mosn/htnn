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
	req := map[string]interface{}{
		"method": header.Method(),
		"scheme": header.Scheme(),
		"host":   header.Host(),
		"path":   uri.Path,
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
