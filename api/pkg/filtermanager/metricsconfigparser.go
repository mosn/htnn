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
	"encoding/json"
	"errors"

	xds "github.com/cncf/xds/go/xds/type/v3"
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	pkgPlugins "mosn.io/htnn/api/pkg/plugins"
)

type metricsConfigFilter struct {
	capi.PassThroughStreamFilter

	callbacks capi.FilterCallbackHandler
}

func MetricsConfigFactory(_ interface{}, callbacks capi.FilterCallbackHandler) capi.StreamFilter {
	return &metricsConfigFilter{
		callbacks: callbacks,
	}
}

type MetricsConfigParser struct {
}

type MetricsConfig struct {
	Plugins []*model.FilterConfig `json:"plugins"`
}

// MetricsConfigParser is the parser to register metrics only, no real parsing
func (p *MetricsConfigParser) Parse(any *anypb.Any, callbacks capi.ConfigCallbackHandler) (interface{}, error) {
	if callbacks == nil {
		api.LogErrorf("no config callback handler provided")
		// the call back handler to be nil only affects plugin metrics, so we can continue
	}
	configStruct := &xds.TypedStruct{}

	// No configuration
	if any.GetTypeUrl() == "" {
		conf := initFilterManagerConfig("")
		return conf, nil
	}

	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	if configStruct.Value == nil {
		return nil, errors.New("bad TypedStruct format")
	}

	data, err := configStruct.Value.MarshalJSON()
	if err != nil {
		return nil, err
	}

	mConfig := &MetricsConfig{}
	if err := json.Unmarshal(data, mConfig); err != nil {
		return nil, err
	}

	plugins := mConfig.Plugins
	for _, proto := range plugins {
		name := proto.Name

		plugin := pkgPlugins.LoadHTTPFilterFactoryAndParser(name)
		if plugin == nil {
			api.LogErrorf("plugin %s not found, ignored", name)
			continue
		}
		config, err := plugin.ConfigParser.Parse(proto.Config)
		if err != nil {
			api.LogErrorf("%s during parsing plugin %s in metrics manager", err, name)

			continue
		}
		if register, ok := config.(pkgPlugins.MetricsRegister); ok {
			register.MetricsDefinition(callbacks)
			api.LogInfof("loaded metrics definition for plugin: %s", name)
		}

	}

	return any, nil
}

// just to satisfy the interface, no real merge
func (p *MetricsConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	return parent
}
