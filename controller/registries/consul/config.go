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
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	istioapi "istio.io/api/networking/v1alpha3"
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
			subscriptions:       make(map[string]*watch.Plan),
			done:                make(chan struct{}),
		}
		return reg, nil
	})
}

type Consul struct {
	consul.RegistryType
	logger log.RegistryLogger
	store  registry.ServiceEntryStore
	name   string
	client *Client

	lock                sync.RWMutex
	watchingServices    map[consulService]bool
	subscriptions       map[string]*watch.Plan
	softDeletedServices map[consulService]bool

	done    chan struct{}
	stopped atomic.Bool
}

type Client struct {
	consulClient  *consulapi.Client
	consulCatalog *consulapi.Catalog

	DataCenter string
	NameSpace  string
	Token      string
	Address    string
}

var (
	RegistryType = "consul"
)

func (reg *Consul) NewClient(config *consul.Config) (*Client, error) {
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
		Address:       clientConfig.Address,
	}, nil
}

type consulService struct {
	Tag         string
	ServiceName string
}

func (reg *Consul) Start(c registrytype.RegistryConfig) error {
	config := c.(*consul.Config)

	client, err := reg.NewClient(config)
	if err != nil {
		return err
	}

	reg.client = client

	services, err := reg.fetchAllServices(reg.client)

	if err != nil {
		return err
	}

	for key := range services {
		err = reg.subscribe(key.Tag, key.ServiceName)
		if err != nil {
			reg.logger.Errorf("failed to subscribe service, err: %v, service: %v", err, key)
			// the service will be resubscribed after refresh interval
			delete(services, key)
		}
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
	for key := range reg.softDeletedServices {
		if _, ok := reg.watchingServices[key]; !ok {
			reg.store.Delete(reg.getServiceEntryKey(key.Tag, key.ServiceName))
		}
	}
	for key := range reg.watchingServices {
		reg.removeService(key)
	}

	return nil
}

func (reg *Consul) Reload(c registrytype.RegistryConfig) error {
	config := c.(*consul.Config)

	client, err := reg.NewClient(config)
	if err != nil {
		return err
	}

	fetchedServices, err := reg.fetchAllServices(client)
	if err != nil {
		return fmt.Errorf("fetch all services error: %v", err)
	}

	for key := range reg.softDeletedServices {
		if _, ok := fetchedServices[key]; !ok {
			reg.store.Delete(reg.getServiceEntryKey(key.Tag, key.ServiceName))
		}
	}
	reg.softDeletedServices = map[consulService]bool{}

	for key := range reg.watchingServices {
		// unsubscribe with the previous client
		if _, ok := fetchedServices[key]; !ok {
			reg.removeService(key)
		} else {
			err = reg.unsubscribe(key.ServiceName)
			if err != nil {
				reg.logger.Errorf("failed to unsubscribe service, err: %v, service: %v", err, key)
			}
		}
	}

	reg.client = client

	for key := range fetchedServices {
		err = reg.subscribe(key.Tag, key.ServiceName)
		if err != nil {
			reg.logger.Errorf("failed to subscribe service, err: %v, service: %v", err, key)
			delete(fetchedServices, key)
		}
	}
	reg.watchingServices = fetchedServices

	return nil
}

func (reg *Consul) removeService(key consulService) {
	err := reg.unsubscribe(key.ServiceName)
	if err != nil {
		reg.logger.Errorf("failed to unsubscribe service, err: %v, service: %v", err, key)
	}
	reg.store.Delete(reg.getServiceEntryKey(key.Tag, key.ServiceName))
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
	for serviceName, tags := range services {
		tag := strings.Join(tags, "-")
		service := consulService{
			Tag:         tag,
			ServiceName: serviceName,
		}
		serviceMap[service] = true
	}

	return serviceMap, nil
}

func (reg *Consul) getServiceEntryKey(tag, serviceName string) string {
	host := strings.Join([]string{tag, serviceName, reg.client.NameSpace, reg.client.DataCenter, reg.name, RegistryType}, ".")
	host = strings.ReplaceAll(host, "_", "-")

	re := regexp.MustCompile(`\.+`)
	h := re.ReplaceAllString(host, ".")
	h = strings.Trim(h, ".")
	return strings.ToLower(h)
}

func (reg *Consul) generateServiceEntry(host string, services []*consulapi.ServiceEntry) *registry.ServiceEntryWrapper {
	portList := make([]*istioapi.ServicePort, 0, 1)
	endpoints := make([]*istioapi.WorkloadEntry, 0, len(services))

	for _, service := range services {
		protocol := registry.HTTP
		if service.Service.Meta == nil {
			service.Service.Meta = make(map[string]string)
		}

		if service.Service.Meta["protocol"] != "" {
			protocol = registry.ParseProtocol(service.Service.Meta["protocol"])
		}

		port := &istioapi.ServicePort{
			Name:     string(protocol),
			Number:   uint32(service.Service.Port),
			Protocol: string(protocol),
		}
		if len(portList) == 0 {
			portList = append(portList, port)
		}

		endpoint := istioapi.WorkloadEntry{
			Address: service.Service.Address,
			Ports:   map[string]uint32{port.Protocol: port.Number},
			Labels:  service.Service.Meta,
		}
		endpoints = append(endpoints, &endpoint)
	}

	return &registry.ServiceEntryWrapper{
		ServiceEntry: istioapi.ServiceEntry{
			Hosts:      []string{host},
			Ports:      portList,
			Location:   istioapi.ServiceEntry_MESH_INTERNAL,
			Resolution: istioapi.ServiceEntry_STATIC,
			Endpoints:  endpoints,
		},
		Source: RegistryType,
	}
}

func (reg *Consul) subscribe(tag, serviceName string) error {
	plan, err := watch.Parse(map[string]interface{}{
		"type":    "service",
		"service": serviceName,
	})
	if err != nil {
		return err
	}

	plan.Handler = reg.getSubscribeCallback(tag, serviceName)
	plan.Token = reg.client.Token
	plan.Datacenter = reg.client.DataCenter
	reg.subscriptions[serviceName] = plan

	go func() {
		err := plan.Run(reg.client.Address)
		if err != nil {
			reg.logger.Errorf("failed to subscribe ,err=%v", err)
		}
	}()

	return nil
}

func (reg *Consul) getSubscribeCallback(tag, serviceName string) func(idx uint64, data interface{}) {
	host := reg.getServiceEntryKey(tag, serviceName)
	return func(idx uint64, data interface{}) {
		services, ok := data.([]*consulapi.ServiceEntry)
		if !ok {
			reg.logger.Infof("Unexpected type for data in callback: %t", data)
			return
		}
		if reg.stopped.Load() {
			return
		}
		reg.store.Update(host, reg.generateServiceEntry(host, services))
	}

}

func (reg *Consul) unsubscribe(serviceName string) error {
	plan, exists := reg.subscriptions[serviceName]
	if !exists {
		return fmt.Errorf("no subscription found for service %s", serviceName)
	}

	plan.Stop()
	delete(reg.subscriptions, serviceName)
	return nil
}

func (reg *Consul) refresh(services map[string][]string) {

	serviceMap := make(map[consulService]bool)

	for serviceName, tags := range services {
		tag := strings.Join(tags, "-")
		service := consulService{
			Tag:         tag,
			ServiceName: serviceName,
		}
		serviceMap[service] = true
		if _, ok := reg.watchingServices[service]; !ok {
			err := reg.subscribe("", service.ServiceName)
			if err != nil {
				reg.logger.Errorf("failed to subscribe service, err: %v, service: %v", err, service.ServiceName)
				delete(serviceMap, service)
			}
		}
	}
	reg.lock.Lock()
	defer reg.lock.Unlock()
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
