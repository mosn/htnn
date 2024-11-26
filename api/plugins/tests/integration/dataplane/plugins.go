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

package dataplane

import (
	"net/http"
	"runtime/coverage"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/api/plugins/tests/integration/helper"
)

type basePlugin struct {
}

func (p basePlugin) Config() api.PluginConfig {
	return &Config{}
}

type coveragePlugin struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func (p *coveragePlugin) Factory() api.FilterFactory {
	return coverageFactory
}

func coverageFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &coverageFilter{
		callbacks: callbacks,
	}
}

type coverageFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
}

func (f *coverageFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	err := coverage.WriteCountersDir(helper.CoverDir())
	if err != nil {
		api.LogErrorf("failed to write coverage: %v", err)
		return &api.LocalResponse{Code: 500}
	}
	return &api.LocalResponse{Code: 200}
}

type detectorPlugin struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func (p *detectorPlugin) Factory() api.FilterFactory {
	return detectorFactory
}

func detectorFactory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &detectorFilter{
		callbacks: callbacks,
	}
}

type detectorFilter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
}

func (f *detectorFilter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	hdr := http.Header{}
	name := f.callbacks.StreamInfo().GetRouteName()
	hdr.Add("route-version", name)
	return &api.LocalResponse{Code: 200, Header: hdr}
}

func init() {
	plugins.RegisterPlugin("coverage", &coveragePlugin{})
	plugins.RegisterPlugin("detector", &detectorPlugin{})
}
