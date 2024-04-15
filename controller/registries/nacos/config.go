// Copyright The HTNN Authors.
// Copyright (c) 2022 Alibaba Group Holding Ltd.
// The code which interacts with Nacos is modified from alibaba/higress,
// which is licensed under the Apache License 2.0.
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

package nacos

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"gopkg.in/natefinch/lumberjack.v2"
	istioapi "istio.io/api/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/pkg/registry"
	registrytype "mosn.io/htnn/types/pkg/registry"
	"mosn.io/htnn/types/registries/nacos"
)

func init() {
	registry.AddRegistryFactory("nacos", func(store registry.ServiceEntryStore, om metav1.ObjectMeta) (registry.Registry, error) {
		reg := &Nacos{
			store:               store,
			name:                om.Name,
			softDeletedServices: map[nacosService]bool{},
			done:                make(chan struct{}),
		}
		return reg, nil
	})
}

const (
	defaultFetchPageSize = 1000
	defaultNacosPort     = 8848
	defaultTimeoutMs     = 5 * 1000
	defaultLogLevel      = "warn"
	defaultNotLoadCache  = true
	defaultLogMaxDays    = 3
	defaultLogMaxSizeMB  = 100

	RegistryType = "nacos"
)

type nacosService struct {
	GroupName   string
	ServiceName string
}

type nacosClient struct {
	Groups    []string
	Namespace string

	namingClient naming_client.INamingClient
}

type Nacos struct {
	nacos.RegistryType

	store  registry.ServiceEntryStore
	name   string
	client *nacosClient

	lock                sync.RWMutex
	watchingServices    map[nacosService]bool
	softDeletedServices map[nacosService]bool

	done    chan struct{}
	stopped atomic.Bool
}

func (reg *Nacos) fetchAllServices(client *nacosClient) (map[nacosService]bool, error) {
	fetchedServices := make(map[nacosService]bool)
	for _, groupName := range client.Groups {
		for page := 1; ; page++ {
			// Nacos v1 doesn't provide a method to return all services in a call.
			// We use a large page size to reduce the race but there is still chance
			// that a service is missed. When the Nacos is starting or down (as discussed
			// in https://github.com/alibaba/higress/discussions/769), there is also a chance
			// that the ServiceEntry mismatches the service.
			//
			// Missing to add a new service will be solved by the next refresh.
			// We also use soft-deletion to avoid the ServiceEntry mismatch. The ServiceEntry
			// will be deleted only when the registry's configuration changes or an empty host list
			// is returned from the subscription.
			ss, err := client.namingClient.GetAllServicesInfo(vo.GetAllServiceInfoParam{
				GroupName: groupName,
				PageNo:    uint32(page),
				PageSize:  defaultFetchPageSize,
				NameSpace: client.Namespace,
			})
			if err != nil {
				return nil, err
			}

			for _, serviceName := range ss.Doms {
				s := nacosService{
					GroupName:   groupName,
					ServiceName: serviceName,
				}
				fetchedServices[s] = true
			}
			if len(ss.Doms) < defaultFetchPageSize {
				break
			}
		}
	}
	return fetchedServices, nil
}

func (reg *Nacos) subscribe(groupName string, serviceName string) error {
	log.Infof("subscribe serviceName: %s, groupName: %s", serviceName, groupName)

	err := reg.client.namingClient.Subscribe(&vo.SubscribeParam{
		ServiceName:       serviceName,
		GroupName:         groupName,
		SubscribeCallback: reg.getSubscribeCallback(groupName, serviceName),
	})

	if err != nil {
		return fmt.Errorf("subscribe service error:%v, groupName:%s, serviceName:%s", err, groupName, serviceName)
	}

	return nil
}

func (reg *Nacos) unsubscribe(groupName string, serviceName string) error {
	log.Infof("unsubscribe serviceName: %s, groupName: %s", serviceName, groupName)

	err := reg.client.namingClient.Unsubscribe(&vo.SubscribeParam{
		ServiceName:       serviceName,
		GroupName:         groupName,
		SubscribeCallback: reg.getSubscribeCallback(groupName, serviceName),
	})

	if err != nil {
		return fmt.Errorf("unsubscribe service error:%v, groupName:%s, serviceName:%s", err, groupName, serviceName)
	}

	return nil
}

func (reg *Nacos) getServiceEntryKey(groupName string, serviceName string) string {
	suffix := strings.Join([]string{groupName, reg.client.Namespace, reg.name, RegistryType}, ".")
	suffix = strings.ReplaceAll(suffix, "_", "-")
	host := strings.Join([]string{serviceName, suffix}, ".")
	return strings.ToLower(host)
}

func (reg *Nacos) getSubscribeCallback(groupName string, serviceName string) func(services []model.SubscribeService, err error) {
	host := reg.getServiceEntryKey(groupName, serviceName)
	return func(services []model.SubscribeService, err error) {
		if err != nil {
			if !strings.Contains(err.Error(), "hosts is empty") {
				log.Errorf("callback failed, err: %v, host: %s", err, host)
			} else {
				log.Infof("delete service entry because there are no hosts, service: %s", host)
				reg.store.Delete(host)
				// When the last instance is deleted, Nacos v1 has a protect mechanism that
				// skips the callback. See:
				// https://github.com/nacos-group/nacos-sdk-go/issues/139
				// As a result, if the last instance of a service is deleted, the service is not
				// deleted until Nacos removes the service.
				// The skipping only happens with the subscribed service, it does not
				// happen with the service that is first subscribed. So this branch is still
				// useful.
			}
			return
		}

		if reg.stopped.Load() {
			return
		}
		reg.store.Update(host, reg.generateServiceEntry(host, services))
	}
}

func (reg *Nacos) generateServiceEntry(host string, services []model.SubscribeService) *registry.ServiceEntryWrapper {
	portList := make([]*istioapi.ServicePort, 0, 1)
	endpoints := make([]*istioapi.WorkloadEntry, 0, len(services))

	for _, service := range services {
		protocol := registry.HTTP
		if service.Metadata == nil {
			service.Metadata = make(map[string]string)
		}

		if service.Metadata["protocol"] != "" {
			protocol = registry.ParseProtocol(service.Metadata["protocol"])
		}

		port := &istioapi.ServicePort{
			Name:     string(protocol),
			Number:   uint32(service.Port),
			Protocol: string(protocol),
		}
		if len(portList) == 0 {
			portList = append(portList, port)
		}

		endpoint := istioapi.WorkloadEntry{
			Address: service.Ip,
			Ports:   map[string]uint32{port.Protocol: port.Number},
			Labels:  service.Metadata,
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

func (reg *Nacos) newClient(config *nacos.Config) (*nacosClient, error) {
	uri, err := url.Parse(config.ServerUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid server url: %s", config.ServerUrl)
	}

	domain := uri.Hostname()
	p := uri.Port()
	port := defaultNacosPort
	if p != "" {
		port, _ = strconv.Atoi(p)
	}

	cc := constant.NewClientConfig(
		constant.WithTimeoutMs(defaultTimeoutMs),
		constant.WithLogLevel(defaultLogLevel),
		constant.WithNotLoadCacheAtStart(defaultNotLoadCache),
		constant.WithLogRollingConfig(&lumberjack.Logger{
			MaxSize: defaultLogMaxSizeMB,
			MaxAge:  defaultLogMaxDays,
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
	return &nacosClient{
		Groups:       config.Groups,
		Namespace:    config.Namespace,
		namingClient: namingClient,
	}, nil
}

func (reg *Nacos) Start(c registrytype.RegistryConfig) error {
	config := c.(*nacos.Config)

	client, err := reg.newClient(config)
	if err != nil {
		return err
	}

	fetchedServices, err := reg.fetchAllServices(client)
	if err != nil {
		return fmt.Errorf("fetch all services error: %v", err)
	}
	reg.client = client

	for key := range fetchedServices {
		err = reg.subscribe(key.GroupName, key.ServiceName)
		if err != nil {
			log.Errorf("failed to subscribe service, err: %v, service: %v", err, key)
			// the service will be resubscribed after refresh interval
			delete(fetchedServices, key)
		}
	}

	reg.watchingServices = fetchedServices

	dur := 30 * time.Second
	refreshInteval := config.GetServiceRefreshInterval()
	if refreshInteval != nil {
		dur = refreshInteval.AsDuration()
	}
	go func() {
		log.Infof("start refreshing services, registry: %s", reg.name)
		ticker := time.NewTicker(dur)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := reg.refresh()
				if err != nil {
					log.Errorf("failed to refresh services, err: %v, registry: %s", err, reg.name)
				}
			case <-reg.done:
				log.Infof("stop refreshing services, registry: %s", reg.name)
				return
			}
		}
	}()

	return nil
}

func (reg *Nacos) removeService(key nacosService) {
	err := reg.unsubscribe(key.GroupName, key.ServiceName)
	if err != nil {
		log.Errorf("failed to unsubscribe service, err: %v, service: %v", err, key)
		// the upcoming event will be thrown away
	}
	reg.store.Delete(reg.getServiceEntryKey(key.GroupName, key.ServiceName))
}

func (reg *Nacos) refresh() error {
	reg.lock.Lock()
	defer reg.lock.Unlock()

	fetchedServices, err := reg.fetchAllServices(reg.client)
	if err != nil {
		return fmt.Errorf("fetch all services error: %v", err)
	}

	for key := range fetchedServices {
		if _, ok := reg.watchingServices[key]; !ok {
			err = reg.subscribe(key.GroupName, key.ServiceName)
			if err != nil {
				log.Errorf("failed to subscribe service, err: %v, service: %v", err, key)
			}
		}
	}
	prevFetchServices := reg.watchingServices
	reg.watchingServices = fetchedServices

	for key := range prevFetchServices {
		if _, ok := fetchedServices[key]; !ok {
			err := reg.unsubscribe(key.GroupName, key.ServiceName)
			if err != nil {
				log.Errorf("failed to unsubscribe service, err: %v, service: %v", err, key)
				// the upcoming event will be thrown away
			}
			reg.softDeletedServices[key] = true
		}
	}

	return nil
}

func (reg *Nacos) Stop() error {
	close(reg.done)
	reg.stopped.Store(true)

	reg.lock.Lock()
	defer reg.lock.Unlock()

	for key := range reg.softDeletedServices {
		if _, ok := reg.watchingServices[key]; !ok {
			reg.store.Delete(reg.getServiceEntryKey(key.GroupName, key.ServiceName))
		}
	}
	for key := range reg.watchingServices {
		reg.removeService(key)
	}
	return nil
}

func (reg *Nacos) Reload(c registrytype.RegistryConfig) error {
	config := c.(*nacos.Config)

	client, err := reg.newClient(config)
	if err != nil {
		return err
	}

	reg.lock.Lock()
	defer reg.lock.Unlock()

	fetchedServices, err := reg.fetchAllServices(client)
	if err != nil {
		return fmt.Errorf("fetch all services error: %v", err)
	}

	for key := range reg.softDeletedServices {
		if _, ok := fetchedServices[key]; !ok {
			reg.store.Delete(reg.getServiceEntryKey(key.GroupName, key.ServiceName))
		}
	}
	reg.softDeletedServices = map[nacosService]bool{}

	for key := range reg.watchingServices {
		// unsubscribe with the previous client
		if _, ok := fetchedServices[key]; !ok {
			reg.removeService(key)
		} else {
			err = reg.unsubscribe(key.GroupName, key.ServiceName)
			if err != nil {
				log.Errorf("failed to unsubscribe service, err: %v, service: %v", err, key)
			}
		}
	}

	reg.client = client

	for key := range fetchedServices {
		err = reg.subscribe(key.GroupName, key.ServiceName)
		if err != nil {
			log.Errorf("failed to subscribe service, err: %v, service: %v", err, key)
		}
	}
	reg.watchingServices = fetchedServices

	return nil
}
