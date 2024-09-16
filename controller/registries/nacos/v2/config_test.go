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

package v2

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/controller/registries/nacos/client"
	"mosn.io/htnn/types/registries/nacos"
)

func TestNewClient(t *testing.T) {
	config := &nacos.Config{
		ServerUrl: "http://127.0.0.1:8848",
		Version:   "v2",
	}
	patches := gomonkey.NewPatches()

	patches.ApplyFunc(constant.NewClientConfig, func(options ...constant.ClientOption) *constant.ClientConfig {
		return &constant.ClientConfig{}
	})
	patches.ApplyFunc(constant.NewServerConfig, func(domain string, port uint64, opts ...constant.ServerOption) *constant.ServerConfig {
		return &constant.ServerConfig{}
	})
	patches.ApplyFunc(clients.NewNamingClient, func(param vo.NacosClientParam) (naming_client.INamingClient, error) {
		return nil, nil
	})
	_, err := NewClient(config)
	assert.Nil(t, err)

	patches.ApplyFunc(clients.NewNamingClient, func(param vo.NacosClientParam) (naming_client.INamingClient, error) {
		return nil, fmt.Errorf("err")
	})

	_, err = NewClient(config)
	assert.Error(t, err)

	config = &nacos.Config{
		ServerUrl: "::::::::::::",
		Version:   "v2",
	}
	_, err = NewClient(config)
	assert.Error(t, err)

	patches.Reset()

}

func TestGetGroups(t *testing.T) {
	client := &NacosClient{
		groups: nil,
	}
	groups := client.GetGroups()
	assert.Nil(t, groups)

	client = &NacosClient{
		groups: make([]string, 0),
	}
	groups = client.GetGroups()
	assert.Equal(t, 0, len(groups))

	client = &NacosClient{
		groups: []string{"group1", "group2"},
	}
	groups = client.GetGroups()
	assert.Equal(t, 2, len(groups))
	assert.Equal(t, "group1", groups[0])
	assert.Equal(t, "group2", groups[1])
}

func TestUnSubscribe(t *testing.T) {
	nacosClient := &NacosClient{}

	mockServices := []model.Instance{
		{
			Ip:       "1.2.3.4",
			Metadata: map[string]string{"key": "value"},
			Port:     8080,
		},
		{
			Ip:       "5.6.7.8",
			Metadata: map[string]string{"key": "value2"},
			Port:     9090,
		},
	}

	patches := gomonkey.ApplyPrivateMethod(reflect.TypeOf(nacosClient), "unsubscribe", func(_ *NacosClient, groupName string, serviceName string, callback func(services []model.Instance, err error)) error {
		callback(mockServices, nil)
		return nil
	})
	defer patches.Reset()

	var capturedServices []client.SubscribeService
	var capturedError error
	testCallback := func(services []client.SubscribeService, err error) {
		capturedServices = services
		capturedError = err
	}

	err := nacosClient.Unsubscribe("group", "service", testCallback)

	assert.Nil(t, err)

	assert.Len(t, capturedServices, 2)
	assert.Equal(t, "1.2.3.4", capturedServices[0].IP)
	assert.Equal(t, "5.6.7.8", capturedServices[1].IP)

	assert.Equal(t, "value", capturedServices[0].Metadata["key"])
	assert.Equal(t, 8080, int(capturedServices[0].Port))
	assert.Equal(t, "value2", capturedServices[1].Metadata["key"])
	assert.Equal(t, 9090, int(capturedServices[1].Port))

	assert.Nil(t, capturedError)
}
