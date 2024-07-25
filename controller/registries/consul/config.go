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
		}
		return reg, nil
	})
}

const (
	//defaultConsulAddress = "127.0.0.1:8500"
	defaultDatacenter = "dc1"
	defaultToken      = ""
	//defaultScheme        = "http"
	//defaultWaitTime      = 5 * time.Second

	//RegistryType = "consul"
)

type Consul struct {
	consul.RegistryType
	logger log.RegistryLogger
	store  registry.ServiceEntryStore
	name   string
	client *Client

	lock                sync.RWMutex
	watchingServices    map[consulService]bool
	softDeletedServices map[consulService]bool

	done    chan struct{}
	stopped atomic.Bool
}

type Client struct {
	consulClient  *consulapi.Client
	consulCatalog *consulapi.Catalog

	DataCenter string
	NameSpace  string
}

type consulService struct {
	DataCenter  string
	ServiceName string
}

func (reg *Consul) NewClient(config *consul.Config) (*Client, error) {
	uri, err := url.Parse(config.ServerUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid server url: %s", config.ServerUrl)
	}
	clientConfig := consulapi.DefaultConfig()
	clientConfig.Address = uri.Host
	clientConfig.Scheme = uri.Scheme
	clientConfig.Token = defaultToken
	clientConfig.Datacenter = config.DataCenter

	client, err := consulapi.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create Consul client, err: %v", err)
	}

	return &Client{
		consulClient:  client,
		consulCatalog: client.Catalog(),
		DataCenter:    config.DataCenter,
	}, nil
}

func (reg *Consul) Start(c registrytype.RegistryConfig) error {
	config := c.(*consul.Config)

	client, err := reg.NewClient(config)
	if err != nil {
		return err
	}

	services, err := reg.fetchAllServices(client)
	if err != nil {
		return fmt.Errorf("fetch all services error: %v", err)
	}
	reg.client = client

	for key := range services {
		err = reg.subscribe(key.ServiceName)
		if err != nil {
			reg.logger.Errorf("failed to subscribe service, err: %v, service: %v", err, key)
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

func (reg *Consul) Stop() error {
	close(reg.done)
	reg.stopped.Store(true)
	reg.logger.Infof("stopped Consul registry")

	reg.lock.Lock()
	defer reg.lock.Unlock()

	//for key := range reg.softDeletedServices {
	//    if _, ok := reg.watchingServices[key]; !ok {
	//        reg.store.Delete(reg.getServiceEntryKey(key.GroupName, key.ServiceName))
	//    }
	//}
	//for key := range reg.watchingServices {
	//    reg.removeService(key)
	//}
	return nil
}

func (reg *Consul) Reload(c registrytype.RegistryConfig) error {
	fmt.Println(c)
	return nil
}

func (reg *Consul) refresh() error {
	return nil
}

func (reg *Consul) fetchAllServices(client *Client) (map[consulService]bool, error) {
	fmt.Println(client)
	return nil, nil
}

func (reg *Consul) subscribe(serviceName string) error {
	fmt.Println(serviceName)
	return nil
}
