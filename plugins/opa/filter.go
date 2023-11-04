package opa

import (
	"bytes"
	"encoding/json"
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

var opaResponse struct {
	Result struct {
		Allow bool `json:"allow"`
	} `json:"result"`
}

func (f *filter) buildInput(header api.RequestHeaderMap) (map[string]interface{}, error) {
	requestURI := header.Path()
	uri, err := url.ParseRequestURI(requestURI)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"request": map[string]interface{}{
			"method":   header.Method(),
			"scheme":   header.Scheme(),
			"host":     header.Host(),
			"path":     uri.Path,
			"query":    uri.RawQuery,
			"protocol": header.Protocol(),
		},
	}, nil
}

func (f *filter) isAllowed(input map[string]interface{}) (bool, error) {
	params, err := json.Marshal(input)
	if err != nil {
		return false, err
	}

	remote := f.config.GetRemote()
	// When parsing the config, we have already validated the remote is not nil
	path := remote.GetUrl() + "/v1/data/" + remote.GetPolicy()
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

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) {
	input, err := f.buildInput(header)
	if err != nil {
		api.LogErrorf("failed to build input: %v", err)
		f.callbacks.SendLocalReply(503, "", nil, 0, "")
		return
	}

	allow, err := f.isAllowed(input)
	if err != nil {
		api.LogErrorf("failed to call OPA server: %v", err)
		f.callbacks.SendLocalReply(503, "", nil, 0, "")
		return
	}

	if !allow {
		f.callbacks.SendLocalReply(403, "", nil, 0, "")
		return
	}
}
