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

package dynamicconfig

import (
	"errors"
	"fmt"

	xds "github.com/cncf/xds/go/xds/type/v3"
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"

	"mosn.io/htnn/api/internal/proto"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("dynamicconfig")

	dynamicConfigProviders = map[string]DynamicConfigProvider{}
	dynamicConfigHandlers  = map[string]DynamicConfigHandler{}
)

type dynamicConfigFilter struct {
	capi.PassThroughStreamFilter

	callbacks capi.FilterCallbackHandler
}

func DynamicConfigFactory(c interface{}) capi.StreamFilterFactory {
	return func(callbacks capi.FilterCallbackHandler) capi.StreamFilter {
		return &dynamicConfigFilter{
			callbacks: callbacks,
		}
	}
}

type DynamicConfigParser struct {
}

func (p *DynamicConfigParser) Parse(any *anypb.Any, callbacks capi.ConfigCallbackHandler) (interface{}, error) {
	configStruct := &xds.TypedStruct{}

	placeholder := &struct{}{}
	// No configuration
	if any.GetTypeUrl() == "" {
		return placeholder, nil
	}

	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	if configStruct.Value == nil {
		return nil, errors.New("bad TypedStruct format")
	}

	fields := configStruct.Value.GetFields()
	name := fields["name"].GetStringValue()
	cfg := fields["config"]
	if name == "" || cfg == nil {
		return nil, fmt.Errorf("invalid dynamic config format: %s", configStruct.Value.String())
	}

	cb, ok := dynamicConfigHandlers[name]
	if !ok {
		// ignore unknown dynamic config as like ignoring unknown plugin
		api.LogInfof("no callback for dynamic config %s", name)
		return placeholder, nil
	}

	conf := cb.Config()
	data, err := cfg.MarshalJSON()
	if err != nil {
		return nil, err
	}

	api.LogInfof("receive dynamic config %s, configuration: %s", name, data)
	err = proto.UnmarshalJSON(data, conf)
	if err != nil {
		return nil, err
	}

	err = conf.Validate()
	if err != nil {
		return nil, err
	}

	err = cb.OnUpdate(conf)
	if err != nil {
		return nil, err
	}

	return placeholder, nil
}

func (p *DynamicConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	return child
}

type DynamicConfig interface {
	ProtoReflect() protoreflect.Message
	Validate() error
}

type DynamicConfigProvider interface {
	Config() DynamicConfig
}

type DynamicConfigHandler interface {
	DynamicConfigProvider

	OnUpdate(config any) error
}

// We extra RegisterDynamicConfigProvider out of RegisterDynamicConfigHandler, so that
// the control plane can register the definition of the DynamicConfigHandler, and only the
// data plane needs to know the implementation. Of course, you can also call
// RegisterDynamicConfigHandler only, which is more convenient for the developer.

func RegisterDynamicConfigProvider(name string, c DynamicConfigProvider) {
	if _, ok := dynamicConfigHandlers[name]; !ok {
		// As RegisterDynamicConfigHandler also calls RegisterDynamicConfigProvider, we only log for the first time.
		// Otherwise, we will log twice for the load in the data plane.
		logger.Info("register dynamic config provider", "name", name)
	}
	dynamicConfigProviders[name] = c
}

func LoadDynamicConfigProvider(name string) DynamicConfigProvider {
	return dynamicConfigProviders[name]
}

func RegisterDynamicConfigHandler(name string, c DynamicConfigHandler) {
	logger.Info("register dynamic config handler", "name", name)

	dynamicConfigHandlers[name] = c
	// We don't force developer to divide their dynamic configs into two parts for better DX.
	RegisterDynamicConfigProvider(name, c)
}
