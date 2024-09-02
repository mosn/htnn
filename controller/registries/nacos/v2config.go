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

package nacos

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	registrytype "mosn.io/htnn/types/pkg/registry"
	"mosn.io/htnn/types/registries/nacos"
)

type Client struct {
	Client any
}

func (reg *Nacos) newV2Client(config *nacos.Config) (*nacosClient, error) {
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
	v2Client := &Client{
		Client: namingClient,
	}
	return &nacosClient{
		Groups:       config.Groups,
		Namespace:    config.Namespace,
		namingClient: v2Client,
	}, nil
}

func (reg *Nacos) StartV2(c registrytype.RegistryConfig) error {
	config := c.(*nacos.Config)

	client, err := reg.newV2Client(config)
	if err != nil {
		return err
	}

	fetchedServices, err := reg.fetchAllServicesV2(client)
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
				err := reg.refreshV2()
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

func (reg *Nacos) StopV2() error {
	close(reg.done)
	reg.stopped.Store(true)

	reg.lock.Lock()
	defer reg.lock.Unlock()
	return nil
}

func (reg *Nacos) ReloadV2(c registrytype.RegistryConfig) error {
	return nil
}

func (reg *Nacos) subscribeV2(groupName, serviceName string) error {
	return nil
}

func (reg *Nacos) refreshV2() error {
	return nil
}

func (reg *Nacos) fetchAllServicesV2(client *nacosClient) (map[nacosService]bool, error) {
	return nil, nil
}
