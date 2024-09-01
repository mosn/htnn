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
	"sync"
	"sync/atomic"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"mosn.io/htnn/controller/pkg/registry"
	"mosn.io/htnn/controller/pkg/registry/log"
	registrytype "mosn.io/htnn/types/pkg/registry"
	"mosn.io/htnn/types/registries/nacos"
)

func init() {
	registry.AddRegistryFactory(nacos.Name, func(store registry.ServiceEntryStore, om metav1.ObjectMeta) (registry.Registry, error) {
		reg := &Nacos{
			logger: log.NewLogger(&log.RegistryLoggerOptions{
				Name: om.Name,
			}),
			store:               store,
			name:                om.Name,
			softDeletedServices: map[nacosService]bool{},
			done:                make(chan struct{}),
		}
		return reg, nil
	})
}

const (
	defaultNacosPort     = 8848
	defaultTimeoutMs     = 5 * 1000
	defaultLogLevel      = "warn"
	defaultNotLoadCache  = true
	defaultLogMaxDays    = 1
	defaultLogMaxBackups = 10
	defaultLogMaxSizeMB  = 1
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
	logger log.RegistryLogger

	store  registry.ServiceEntryStore
	name   string
	client *nacosClient

	lock                sync.RWMutex
	watchingServices    map[nacosService]bool
	softDeletedServices map[nacosService]bool

	done    chan struct{}
	stopped atomic.Bool
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
		constant.WithLogRollingConfig(&constant.ClientLogRollingConfig{
			MaxSize:    defaultLogMaxSizeMB,
			MaxAge:     defaultLogMaxDays,
			MaxBackups: defaultLogMaxBackups,
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

func (reg *Nacos) Stop() error {
	close(reg.done)
	reg.stopped.Store(true)

	reg.lock.Lock()
	defer reg.lock.Unlock()
	return nil
}

func (reg *Nacos) Reload(c registrytype.RegistryConfig) error {
	return nil
}

func (reg *Nacos) subscribe(groupName, serviceName string) error {
	return nil
}

func (reg *Nacos) refresh() error {
	return nil
}

func (reg *Nacos) fetchAllServices(client *nacosClient) (map[nacosService]bool, error) {
	return nil, nil
}
