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
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	istioapi "istio.io/api/networking/v1alpha3"

	"mosn.io/htnn/controller/pkg/registry"
	"mosn.io/htnn/controller/pkg/registry/log"
	"mosn.io/htnn/types/registries/consul"
)

func TestNewClient(t *testing.T) {
	reg := &Consul{}
	config := &consul.Config{
		ServerUrl:  "http://127.0.0.1:8500",
		DataCenter: "test",
	}
	client, err := reg.NewClient(config)

	assert.NoError(t, err)
	assert.NotNil(t, client)

	config = &consul.Config{
		ServerUrl:  "::::::::::::",
		DataCenter: "test",
	}

	client, err = reg.NewClient(config)

	assert.Error(t, err)
	assert.Nil(t, client)

}

func TestStart(t *testing.T) {
	reg := &Consul{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		softDeletedServices: map[consulService]bool{},
		subscriptions:       make(map[string]*watch.Plan),
		done:                make(chan struct{}),
		lock:                sync.RWMutex{},
	}

	config := &consul.Config{
		ServerUrl: "::::::::::::",
	}

	err := reg.Start(config)
	assert.Error(t, err)
	err = reg.Stop()
	assert.Nil(t, err)

	reg = &Consul{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		softDeletedServices: map[consulService]bool{},
		subscriptions:       make(map[string]*watch.Plan),
		done:                make(chan struct{}),
		lock:                sync.RWMutex{},
		store:               registry.FakeServiceEntryStore(),
	}

	config = &consul.Config{}

	patches := gomonkey.ApplyPrivateMethod(reg, "fetchAllServices", func(_ *Consul, client *Client) (map[consulService]bool, error) {
		return map[consulService]bool{
			{ServiceName: "service1", Tag: "tag1"}: true,
			{ServiceName: "service2", Tag: "tag2"}: true,
		}, nil
	})
	patches.ApplyPrivateMethod(reg, "subscribe", func(serviceName string) error { return nil })
	defer patches.Reset()

	err = reg.Start(config)
	assert.Nil(t, err)

	err = reg.Stop()
	assert.Nil(t, err)
}

func TestRefresh(t *testing.T) {
	reg := &Consul{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		softDeletedServices: map[consulService]bool{},
		done:                make(chan struct{}),
		watchingServices:    map[consulService]bool{},
		lock:                sync.RWMutex{},
	}

	config := &consul.Config{
		ServerUrl: "::::::::::::::::::",
	}

	client, _ := reg.NewClient(config)
	reg.client = client
	services := map[string][]string{
		"service1": {"tag1", "tag2"},
		"service2": {"tag1"},
	}

	patches := gomonkey.ApplyPrivateMethod(reg, "fetchAllServices", func(_ *Consul, client *Client) (map[consulService]bool, error) {
		return map[consulService]bool{
			{ServiceName: "service1", Tag: "tag1"}: true,
			{ServiceName: "service2", Tag: "tag2"}: true,
		}, nil
	})
	patches.ApplyPrivateMethod(reg, "subscribe", func(serviceName string) error { return nil })
	defer patches.Reset()

	reg.refresh(services)

	assert.Len(t, reg.watchingServices, 2)
	assert.Contains(t, reg.watchingServices, consulService{ServiceName: "service1", Tag: "tag1-tag2"})
	assert.Contains(t, reg.watchingServices, consulService{ServiceName: "service2", Tag: "tag1"})
	assert.Empty(t, reg.softDeletedServices)

	reg = &Consul{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		softDeletedServices: map[consulService]bool{},
		watchingServices: map[consulService]bool{
			{ServiceName: "service1", Tag: "tag1"}: true,
		},
		lock: sync.RWMutex{},
	}

	services = map[string][]string{}

	reg.refresh(services)

	assert.Len(t, reg.watchingServices, 0)
	assert.Len(t, reg.softDeletedServices, 1)

}

func TestFetchAllServices(t *testing.T) {
	t.Run("Test fetchAllServices method", func(t *testing.T) {
		reg := &Consul{
			logger: log.NewLogger(&log.RegistryLoggerOptions{
				Name: "test",
			}),
			lock: sync.RWMutex{},
		}
		client := &Client{
			consulCatalog: &api.Catalog{},
			DataCenter:    "dc1",
			NameSpace:     "ns1",
			Token:         "token",
		}

		patches := gomonkey.ApplyMethod(client.consulCatalog, "Services", func(_ *api.Catalog, q *api.QueryOptions) (map[string][]string, *api.QueryMeta, error) {
			return map[string][]string{
				"service1": {"tag1", "tag2"},
				"service2": {"tag3"},
			}, nil, nil
		})
		defer patches.Reset()

		services, err := reg.fetchAllServices(client)
		assert.NoError(t, err)
		assert.NotNil(t, services)
		assert.True(t, services[consulService{ServiceName: "service1", Tag: "tag1-tag2"}])
		assert.True(t, services[consulService{ServiceName: "service2", Tag: "tag3"}])
	})

	t.Run("Test fetchAllServices method with error", func(t *testing.T) {
		reg := &Consul{
			logger: log.NewLogger(&log.RegistryLoggerOptions{
				Name: "test",
			}),
			lock: sync.RWMutex{},
		}
		client := &Client{
			consulCatalog: &api.Catalog{},
			DataCenter:    "dc1",
			NameSpace:     "ns1",
			Token:         "token",
		}

		patches := gomonkey.ApplyMethod(client.consulCatalog, "Services", func(_ *api.Catalog, q *api.QueryOptions) (map[string][]string, *api.QueryMeta, error) {
			return nil, nil, errors.New("mock error")
		})
		defer patches.Reset()

		services, err := reg.fetchAllServices(client)
		assert.Error(t, err)
		assert.Equal(t, "mock error", err.Error())
		assert.Nil(t, services)
	})
}
func TestGenerateServiceEntry(t *testing.T) {
	host := "test.default.default-dc.earth.consul"
	reg := &Consul{}

	type test struct {
		name     string
		services []*api.ServiceEntry
		port     *istioapi.ServicePort
		endpoint *istioapi.WorkloadEntry
	}
	tests := []test{}
	for input, proto := range registry.ProtocolMap {
		s := string(proto)
		tests = append(tests, test{
			name: input,
			services: []*api.ServiceEntry{
				{
					Service: &api.AgentService{
						Port:    80,
						Address: "1.1.1.1",
						Meta: map[string]string{
							"protocol": input,
						},
						Namespace: "default",
					},
				},
			},
			port: &istioapi.ServicePort{
				Name:     s,
				Protocol: s,
				Number:   80,
			},
			endpoint: &istioapi.WorkloadEntry{
				Address: "1.1.1.1",
				Ports:   map[string]uint32{s: 80},
				Labels: map[string]string{
					"protocol": input,
				},
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := reg.generateServiceEntry(host, tt.services)
			require.True(t, proto.Equal(se.ServiceEntry.Ports[0], tt.port))
			require.True(t, proto.Equal(se.ServiceEntry.Endpoints[0], tt.endpoint))
		})
	}
}

func TestGetServiceEntryKey(t *testing.T) {
	reg := &Consul{
		client: &Client{
			NameSpace:  "default_namespace",
			DataCenter: "dc1",
		},
		name: "test_registry",
	}

	testCases := []struct {
		serviceName string
		expectedKey string
		tag         string
	}{
		{
			serviceName: "service",
			expectedKey: "service.default-namespace.dc1.test-registry.consul",
		},
		{
			serviceName: "service",
			expectedKey: "service.default-namespace.dc1.test-registry.consul",
		},
		{
			tag:         "default",
			serviceName: "service",
			expectedKey: "default.service.default-namespace.dc1.test-registry.consul",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.serviceName, func(t *testing.T) {
			result := reg.getServiceEntryKey(tt.tag, tt.serviceName)
			assert.Equal(t, tt.expectedKey, result)
		})
	}
}

func TestGetSubscribeCallback(t *testing.T) {
	reg := &Consul{
		store:   registry.FakeServiceEntryStore(),
		stopped: atomic.Bool{},
		client: &Client{
			NameSpace:  "default",
			DataCenter: "default-dc",
		},
	}

	callback := reg.getSubscribeCallback("", "test-service")

	var services []*api.ServiceEntry
	callback(0, services)

}

func TestReload(t *testing.T) {
	reg := &Consul{
		watchingServices:    make(map[consulService]bool),
		softDeletedServices: make(map[consulService]bool),
		subscriptions:       make(map[string]*watch.Plan),
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		store: registry.FakeServiceEntryStore(),
		lock:  sync.RWMutex{},
	}

	patches := gomonkey.ApplyFunc(reg.NewClient, func(config *consul.Config) (*Client, error) {
		return &Client{
			Address:    "new-client-address",
			Token:      "new-token",
			DataCenter: "new-datacenter",
		}, nil
	})

	service := consulService{"test-service", "new-datacenter"}
	patches.ApplyPrivateMethod(reg, "fetchAllServices", func(client *Client) (map[consulService]bool, error) {
		return map[consulService]bool{
			service: true,
		}, nil
	})

	patches.ApplyPrivateMethod(reg, "subscribe", func(_ *Consul, serviceName string) error {
		return nil
	})

	patches.ApplyPrivateMethod(reg, "unsubscribe", func(_ *Consul, serviceName string) error {
		return nil
	})
	defer patches.Reset()

	err := reg.Reload(&consul.Config{})

	assert.Nil(t, err)
	assert.Equal(t, reg.client.Address, "127.0.0.1:8500")
	assert.Contains(t, reg.watchingServices, consulService{"test-service", "new-datacenter"})

	reg.removeService(service)
}

func TestSubscribe(t *testing.T) {
	reg := &Consul{
		client: &Client{
			Token:      "test-token",
			DataCenter: "dc1",
			Address:    "127.0.0.1:8500",
		},
		subscriptions: make(map[string]*watch.Plan),
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		store: registry.FakeServiceEntryStore(),
		lock:  sync.RWMutex{},
	}

	plan := &watch.Plan{
		Token:      "test-token",
		Datacenter: "dc1",
		Handler: func(idx uint64, data interface{}) {
		},
	}

	patch := gomonkey.ApplyMethod(reflect.TypeOf(plan), "Run", func(_ *watch.Plan, address string) error {
		return nil
	})

	err := reg.subscribe("", "test-service")

	assert.Nil(t, err)
	assert.NotNil(t, reg.subscriptions["test-service"])
	assert.Equal(t, reg.subscriptions["test-service"].Token, "test-token")
	assert.Equal(t, reg.subscriptions["test-service"].Datacenter, "dc1")

	patch.ApplyMethod(reflect.TypeOf(&watch.Plan{}), "Stop", func(_ *watch.Plan) {})
	err = reg.unsubscribe("test-service")
	patch.Reset()

	assert.Nil(t, err)
	assert.Nil(t, reg.subscriptions["test-service"])
	_, exists := reg.subscriptions["test-service"]
	assert.False(t, exists)
}
