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

package consul

import (
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"mosn.io/htnn/controller/pkg/registry"
	"mosn.io/htnn/controller/pkg/registry/log"
	registrytype "mosn.io/htnn/types/pkg/registry"
	"mosn.io/htnn/types/registries/consul"
)

func init() {
	registry.AddRegistryFactory(consul.Name, func(store registry.ServiceEntryStore, om metav1.ObjectMeta) (registry.Registry, error) {
		reg := &Consul{
			logger: log.NewLogger(&log.RegistryLoggerOptions{
				Name: om.Name,
			}),
			store:               store,
			name:                om.Name,
			softDeletedServices: map[consulService]bool{},
			done:                make(chan struct{}),
			clientFactory:       factory,
		}
		return reg, nil
	})
}

var (
	//RegistryType               = "consul"
	factory ClientFactory = &DefaultClientFactory{}
)

type consulCatalog interface {
	Services(q *consulapi.QueryOptions) (map[string][]string, *consulapi.QueryMeta, error)
}

type ConsulAPI struct {
	client *consulapi.Client
}

func (c *ConsulAPI) Services(q *consulapi.QueryOptions) (map[string][]string, *consulapi.QueryMeta, error) {
	return c.client.Catalog().Services(q)
}

type Consul struct {
	consul.RegistryType
	logger        log.RegistryLogger
	store         registry.ServiceEntryStore
	name          string
	client        *Client
	clientFactory ClientFactory

	lock                sync.RWMutex
	watchingServices    map[consulService]bool
	softDeletedServices map[consulService]bool

	done    chan struct{}
	stopped atomic.Bool
}

type Client struct {
	consulClient  *consulapi.Client
	consulCatalog consulCatalog

	DataCenter string
	NameSpace  string
	Token      string
}

type ClientFactory interface {
	NewClient(config *consul.Config) (*Client, error)
}

type DefaultClientFactory struct{}

func (f *DefaultClientFactory) NewClient(config *consul.Config) (*Client, error) {
	uri, err := url.Parse(config.ServerUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid server url: %s", config.ServerUrl)
	}
	clientConfig := consulapi.DefaultConfig()
	clientConfig.Address = uri.Host
	clientConfig.Scheme = uri.Scheme
	clientConfig.Token = config.Token
	clientConfig.Datacenter = config.DataCenter

	client, err := consulapi.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create Consul client, err: %v", err)
	}

	return &Client{
		consulClient:  client,
		consulCatalog: client.Catalog(),
		DataCenter:    config.DataCenter,
		NameSpace:     config.Namespace,
		Token:         config.Token,
	}, nil
}

type consulService struct {
	DataCenter  string
	ServiceName string
}

func (reg *Consul) Start(c registrytype.RegistryConfig) error {
	config := c.(*consul.Config)

	client, err := reg.clientFactory.NewClient(config)
	if err != nil {
		return err
	}

	reg.client = client

	services, err := reg.fetchAllServices(reg.client)

	if err != nil {
		return err
	}

	reg.watchingServices = services

	dur := 30 * time.Second
	if config.ServiceRefreshInterval != nil {
		dur = config.ServiceRefreshInterval.AsDuration()
	}
	go func() {
		reg.logger.Infof("start refreshing services")
		q := &consulapi.QueryOptions{
			WaitTime:   dur,
			Namespace:  config.Namespace,
			Datacenter: config.DataCenter,
			Token:      config.Token,
		}
		for {

			select {
			case <-reg.done:
				reg.logger.Infof("stop refreshing services")
				return
			default:
			}
			services, meta, err := reg.client.consulCatalog.Services(q)
			if err != nil {
				reg.logger.Errorf("failed to get services, err: %v", err)
				time.Sleep(dur)
				continue
			}
			reg.refresh(services)

			q.WaitIndex = meta.LastIndex
		}
	}()

	return nil
}

func (reg *Consul) Stop() error {
	close(reg.done)
	reg.stopped.Store(true)
	reg.logger.Infof("stopped Consul registry")

	reg.lock.Lock()
	defer reg.lock.Unlock()

	return nil
}

func (reg *Consul) Reload(c registrytype.RegistryConfig) error {
	fmt.Println(c)
	return nil
}

func (reg *Consul) fetchAllServices(client *Client) (map[consulService]bool, error) {
	q := &consulapi.QueryOptions{}
	q.Datacenter = client.DataCenter
	q.Namespace = client.NameSpace
	q.Token = client.Token
	services, _, err := client.consulCatalog.Services(q)

	if err != nil {
		reg.logger.Errorf("failed to get service, err: %v", err)
		return nil, err
	}
	serviceMap := make(map[consulService]bool)
	for serviceName, dataCenters := range services {
		for _, dc := range dataCenters {
			service := consulService{
				DataCenter:  dc,
				ServiceName: serviceName,
			}
			serviceMap[service] = true
		}
	}
	return serviceMap, nil
}

func (reg *Consul) subscribe(serviceName string) error {
	fmt.Println(serviceName)
	return nil
}

func (reg *Consul) unsubscribe(serviceName string) error {
	fmt.Println(serviceName)
	return nil
}

func (reg *Consul) refresh(services map[string][]string) {
	serviceMap := make(map[consulService]bool)
	for serviceName, dataCenters := range services {
		for _, dc := range dataCenters {
			service := consulService{
				DataCenter:  dc,
				ServiceName: serviceName,
			}
			serviceMap[service] = true
			if _, ok := reg.watchingServices[service]; !ok {
				err := reg.subscribe(serviceName)
				if err != nil {
					reg.logger.Errorf("failed to subscribe service, err: %v, service: %v", err, serviceName)
					delete(serviceMap, service)
				}
			}
		}
	}

	prevFetchServices := reg.watchingServices
	reg.watchingServices = serviceMap

	for key := range prevFetchServices {
		if _, ok := serviceMap[key]; !ok {
			err := reg.unsubscribe(key.ServiceName)
			if err != nil {
				reg.logger.Errorf("failed to unsubscribe service, err: %v, service: %v", err, key)
			}
			reg.softDeletedServices[key] = true
		}
	}

}
