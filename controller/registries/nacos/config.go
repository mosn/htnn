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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	istioapi "istio.io/api/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"mosn.io/htnn/controller/pkg/registry"
	"mosn.io/htnn/controller/pkg/registry/log"
	"mosn.io/htnn/controller/registries/nacos/client"
	v1 "mosn.io/htnn/controller/registries/nacos/v1"
	v2 "mosn.io/htnn/controller/registries/nacos/v2"
	registrytype "mosn.io/htnn/types/pkg/registry"
	"mosn.io/htnn/types/registries/nacos"
)

var (
	RegistryType = "nacos"
)

func init() {
	registry.AddRegistryFactory(nacos.Name, func(store registry.ServiceEntryStore, om metav1.ObjectMeta) (registry.Registry, error) {
		reg := &Nacos{
			logger: log.NewLogger(&log.RegistryLoggerOptions{
				Name: om.Name,
			}),
			store:               store,
			name:                om.Name,
			softDeletedServices: map[client.NacosService]bool{},
			done:                make(chan struct{}),
		}
		return reg, nil
	})
}

type Nacos struct {
	nacos.RegistryType
	logger log.RegistryLogger

	store   registry.ServiceEntryStore
	name    string
	client  client.Client
	version string

	lock                sync.RWMutex
	watchingServices    map[client.NacosService]bool
	softDeletedServices map[client.NacosService]bool

	done    chan struct{}
	stopped atomic.Bool
}

func (reg *Nacos) createClient(config *nacos.Config) (client.Client, error) {
	var cli client.Client
	var err error

	switch reg.version {
	case "v1":
		cli, err = v1.NewClient(config)
	case "v2":
		cli, err = v2.NewClient(config)
	default:
		err = fmt.Errorf("unsupported version: %s", config.Version)
	}

	return cli, err
}

func (reg *Nacos) getServiceEntryKey(groupName string, serviceName string) string {
	suffix := strings.Join([]string{groupName, reg.client.GetNamespace(), reg.name, RegistryType}, ".")
	suffix = strings.ReplaceAll(suffix, "_", "-")
	host := strings.Join([]string{serviceName, suffix}, ".")
	return strings.ToLower(host)
}

func (reg *Nacos) getSubscribeCallback(groupName string, serviceName string) func(services []client.SubscribeService, err error) {
	host := reg.getServiceEntryKey(groupName, serviceName)
	return func(services []client.SubscribeService, err error) {
		if err != nil {
			if !strings.Contains(err.Error(), "hosts is empty") {
				reg.logger.Errorf("callback failed, err: %v, host: %s", err, host)
			} else {
				reg.logger.Infof("delete service entry because there are no hosts, service: %s", host)
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

func (reg *Nacos) generateServiceEntry(host string, services []client.SubscribeService) *registry.ServiceEntryWrapper {
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
			Address: service.IP,
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

func (reg *Nacos) Start(c registrytype.RegistryConfig) error {
	config := c.(*nacos.Config)

	reg.version = config.Version

	cli, err := reg.createClient(config)
	if err != nil {
		return err
	}
	reg.client = cli

	fetchedServices, err := reg.client.FetchAllServices()
	if err != nil {
		return fmt.Errorf("fetch all services error: %v", err)
	}

	for key := range fetchedServices {
		callback := reg.getSubscribeCallback(key.GroupName, key.ServiceName)
		err = reg.client.Subscribe(key.GroupName, key.ServiceName, callback)
		if err != nil {
			reg.logger.Errorf("failed to subscribe service, err: %v, service: %v", err, key)
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
		reg.logger.Infof("start refreshing services")
		ticker := time.NewTicker(dur)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := reg.refresh()
				if err != nil {
					reg.logger.Errorf("failed to refresh services, err: %v", err)
				}
			case <-reg.done:
				reg.logger.Infof("stop refreshing services")
				return
			}
		}
	}()

	return nil
}

func (reg *Nacos) removeService(key client.NacosService) {
	callback := reg.getSubscribeCallback(key.GroupName, key.ServiceName)
	err := reg.client.Unsubscribe(key.GroupName, key.ServiceName, callback)
	if err != nil {
		reg.logger.Errorf("failed to unsubscribe service, err: %v, service: %v", err, key)
		// the upcoming event will be thrown away
	}
	reg.store.Delete(reg.getServiceEntryKey(key.GroupName, key.ServiceName))
}

func (reg *Nacos) refresh() error {
	reg.lock.Lock()
	defer reg.lock.Unlock()

	fetchedServices, err := reg.client.FetchAllServices()
	if err != nil {
		return fmt.Errorf("fetch all services error: %v", err)
	}

	for key := range fetchedServices {
		if _, ok := reg.watchingServices[key]; !ok {
			callback := reg.getSubscribeCallback(key.GroupName, key.ServiceName)
			err = reg.client.Subscribe(key.GroupName, key.ServiceName, callback)
			if err != nil {
				reg.logger.Errorf("failed to subscribe service, err: %v, service: %v", err, key)
			}
		}
	}
	prevFetchServices := reg.watchingServices
	reg.watchingServices = fetchedServices

	for key := range prevFetchServices {
		if _, ok := fetchedServices[key]; !ok {
			callback := reg.getSubscribeCallback(key.GroupName, key.ServiceName)
			err := reg.client.Unsubscribe(key.GroupName, key.ServiceName, callback)
			if err != nil {
				reg.logger.Errorf("failed to unsubscribe service, err: %v, service: %v", err, key)
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

	reg.version = config.Version

	reg.lock.Lock()
	defer reg.lock.Unlock()

	cli, err := reg.createClient(config)
	if err != nil {
		return err
	}
	reg.client = cli

	fetchedServices, err := reg.client.FetchAllServices()
	if err != nil {
		return fmt.Errorf("fetch all services error: %v", err)
	}

	for key := range reg.softDeletedServices {
		if _, ok := fetchedServices[key]; !ok {
			reg.store.Delete(reg.getServiceEntryKey(key.GroupName, key.ServiceName))
		}
	}
	reg.softDeletedServices = map[client.NacosService]bool{}

	for key := range reg.watchingServices {
		// unsubscribe with the previous client
		if _, ok := fetchedServices[key]; !ok {
			reg.removeService(key)
		} else {
			callback := reg.getSubscribeCallback(key.GroupName, key.ServiceName)
			err = reg.client.Unsubscribe(key.GroupName, key.ServiceName, callback)
			if err != nil {
				reg.logger.Errorf("failed to unsubscribe service, err: %v, service: %v", err, key)
			}
		}
	}

	for key := range fetchedServices {
		callback := reg.getSubscribeCallback(key.GroupName, key.ServiceName)
		err = reg.client.Subscribe(key.GroupName, key.ServiceName, callback)
		if err != nil {
			reg.logger.Errorf("failed to subscribe service, err: %v, service: %v", err, key)
		}
	}
	reg.watchingServices = fetchedServices

	return nil
}
