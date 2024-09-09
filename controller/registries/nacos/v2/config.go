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
	"net/url"
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"mosn.io/htnn/controller/registries/nacos/client"
	"mosn.io/htnn/types/registries/nacos"
)

type NacosClient struct {
	Groups    []string
	Namespace string
	client    naming_client.INamingClient
}

func NewClient(config *nacos.Config) (*NacosClient, error) {
	uri, err := url.Parse(config.ServerUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid server url: %s", config.ServerUrl)
	}

	domain := uri.Hostname()
	p := uri.Port()
	port := client.DefaultNacosPort
	if p != "" {
		port, _ = strconv.Atoi(p)
	}

	cc := constant.NewClientConfig(
		constant.WithTimeoutMs(client.DefaultTimeoutMs),
		constant.WithLogLevel(client.DefaultLogLevel),
		constant.WithNotLoadCacheAtStart(client.DefaultNotLoadCache),
		constant.WithLogRollingConfig(&constant.ClientLogRollingConfig{
			MaxSize:    client.DefaultLogMaxSizeMB,
			MaxAge:     client.DefaultLogMaxDays,
			MaxBackups: client.DefaultLogMaxBackups,
		}),
		// To simplify the permissions, use the path under current work dir to store log & cache
	)

	sc := []constant.ServerConfig{
		*constant.NewServerConfig(domain, uint64(port),
			constant.WithScheme(uri.Scheme),
		),
	}

	namingClient, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  cc,
		ServerConfigs: sc,
	})
	if err != nil {
		return nil, fmt.Errorf("can not create naming client, err: %v", err)
	}

	if config.Namespace == "" {
		config.Namespace = "public"
	}
	if len(config.Groups) == 0 {
		config.Groups = []string{"DEFAULT_GROUP"}
	}
	return &NacosClient{
		Groups:    config.Groups,
		Namespace: config.Namespace,
		client:    namingClient,
	}, nil
}

func (c *NacosClient) Subscribe(groupName string, serviceName string, callback func(services []client.SubscribeService, err error)) error {
	return nil
}

func (c *NacosClient) Unsubscribe(groupName string, serviceName string, callback func(services []client.SubscribeService, err error)) error {
	return nil
}

func (c *NacosClient) FetchAllServices() (map[client.NacosService]bool, error) {
	return nil, nil
}

func (c *NacosClient) GetNamespace() string {
	return ""
}

func (c *NacosClient) GetGroups() []string {
	return nil
}
