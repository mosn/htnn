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
	"github.com/nacos-group/nacos-sdk-go/v2/model"
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

func (c *NacosClient) fetchAllServices() (map[client.NacosService]bool, error) {
	fetchedServices := make(map[client.NacosService]bool)
	for _, groupName := range c.Groups {
		for page := 1; ; page++ {
			// Nacos v2 doesn't provide a method to return all services in a call.
			// We use a large page size to reduce the race but there is still chance
			// that a service is missed. When the Nacos is starting or down (as discussed
			// in https://github.com/alibaba/higress/discussions/769), there is also a chance
			// that the ServiceEntry mismatches the service.
			//
			// Missing to add a new service will be solved by the next refresh.
			// We also use soft-deletion to avoid the ServiceEntry mismatch. The ServiceEntry
			// will be deleted only when the registry's configuration changes or an empty host list
			// is returned from the subscription.
			ss, err := c.client.GetAllServicesInfo(vo.GetAllServiceInfoParam{
				GroupName: groupName,
				PageNo:    uint32(page),
				PageSize:  client.DefaultFetchPageSize,
				NameSpace: c.Namespace,
			})
			if err != nil {
				return nil, err
			}

			for _, serviceName := range ss.Doms {
				s := client.NacosService{
					GroupName:   groupName,
					ServiceName: serviceName,
				}
				fetchedServices[s] = true
			}
			if len(ss.Doms) < client.DefaultFetchPageSize {
				break
			}
		}
	}
	return fetchedServices, nil
}

func (c *NacosClient) GetNamespace() string {
	return c.Namespace
}

func (c *NacosClient) GetGroups() []string {
	return c.Groups
}

func (c *NacosClient) FetchAllServices() (map[client.NacosService]bool, error) {
	return c.fetchAllServices()
}

func (c *NacosClient) Subscribe(groupName string, serviceName string, callback func(services []client.SubscribeService, err error)) error {
	return c.subscribe(groupName, serviceName, func(services []model.Instance, err error) {
		var adaptedServices []client.SubscribeService
		for _, svc := range services {
			adaptedServices = append(adaptedServices, client.SubscribeService{
				IP:       svc.Ip,
				Metadata: svc.Metadata,
				Port:     svc.Port,
			})
		}
		callback(adaptedServices, err)
	})
}

func (c *NacosClient) Unsubscribe(groupName string, serviceName string, callback func(services []client.SubscribeService, err error)) error {
	return c.unsubscribe(groupName, serviceName, func(services []model.Instance, err error) {
		var adaptedServices []client.SubscribeService
		for _, svc := range services {
			adaptedServices = append(adaptedServices, client.SubscribeService{
				IP:       svc.Ip,
				Metadata: svc.Metadata,
				Port:     svc.Port,
			})
		}
		callback(adaptedServices, err)
	})
}

func (c *NacosClient) subscribe(groupName string, serviceName string, callback func(services []model.Instance, err error)) error {
	err := c.client.Subscribe(&vo.SubscribeParam{
		ServiceName:       serviceName,
		GroupName:         groupName,
		SubscribeCallback: callback,
	})

	if err != nil {
		return fmt.Errorf("subscribe service error:%v, groupName:%s, serviceName:%s", err, groupName, serviceName)
	}

	return nil
}

func (c *NacosClient) unsubscribe(groupName string, serviceName string, callback func(services []model.Instance, err error)) error {
	err := c.client.Unsubscribe(&vo.SubscribeParam{
		ServiceName:       serviceName,
		GroupName:         groupName,
		SubscribeCallback: callback,
	})

	if err != nil {
		return fmt.Errorf("subscribe service error:%v, groupName:%s, serviceName:%s", err, groupName, serviceName)
	}

	return nil
}
