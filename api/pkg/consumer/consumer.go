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

package consumer

import (
	"errors"
	"fmt"
	"reflect"

	xds "github.com/cncf/xds/go/xds/type/v3"
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"

	internalConsumer "mosn.io/htnn/api/internal/consumer"
)

type consumerManagerConfig struct {
}

type consumerManager struct {
	capi.PassThroughStreamFilter

	callbacks capi.FilterCallbackHandler
	conf      *consumerManagerConfig
}

func ConsumerManagerFactory(c interface{}) capi.StreamFilterFactory {
	conf, ok := c.(*consumerManagerConfig)
	if !ok {
		panic(fmt.Sprintf("wrong config type: %s", reflect.TypeOf(c)))
	}
	return func(callbacks capi.FilterCallbackHandler) capi.StreamFilter {
		return &consumerManager{
			callbacks: callbacks,
			conf:      conf,
		}
	}
}

type ConsumerManagerConfigParser struct {
}

func (p *ConsumerManagerConfigParser) Parse(any *anypb.Any, callbacks capi.ConfigCallbackHandler) (interface{}, error) {
	configStruct := &xds.TypedStruct{}

	conf := &consumerManagerConfig{}
	// No configuration
	if any.GetTypeUrl() == "" {
		return conf, nil
	}

	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	if configStruct.Value == nil {
		return nil, errors.New("bad TypedStruct format")
	}

	internalConsumer.UpdateConsumers(configStruct.Value)

	return &consumerManagerConfig{}, nil
}

func (p *ConsumerManagerConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	return child
}
