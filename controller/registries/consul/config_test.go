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
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

type MockConsulCatalog struct {
	mock.Mock
}

// Services is a mock method for ConsulCatalog.Services
func (m *MockConsulCatalog) Services(q *api.QueryOptions) (map[string][]string, *api.QueryMeta, error) {
	return nil, nil, nil
}

func TestStart(t *testing.T) {
	mockConsulCatalog := new(MockConsulCatalog)
	client := &Client{
		consulCatalog: mockConsulCatalog,
		DataCenter:    "dc1",
		NameSpace:     "ns1",
		Token:         "token",
	}

	reg := &Consul{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		client: client,
		done:   make(chan struct{}),
	}

	config := &consul.Config{}

	mockConsulCatalog.On("Services", mock.Anything).Return(map[string][]string{"service1": {"dc1"}}, &api.QueryMeta{}, nil)

	err := reg.Start(config)
	assert.NoError(t, err)

	err = reg.subscribe("123")
	assert.Nil(t, err)

	err = reg.unsubscribe("123")
	assert.Nil(t, err)

	err = reg.Stop()
	assert.Nil(t, err)

	reg = &Consul{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		done: make(chan struct{}),
	}

	err = reg.Start(config)
	assert.Error(t, err)

	close(reg.done)
}

func TestReload(t *testing.T) {
	reg := &Consul{}
	config := &consul.Config{
		ServerUrl: "http://127.0.0.1:8500",
	}

	err := reg.Reload(config)
	assert.NoError(t, err)
}

func TestRefresh(t *testing.T) {
	reg := &Consul{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		softDeletedServices: map[consulService]bool{},
		done:                make(chan struct{}),
		watchingServices:    map[consulService]bool{},
	}

	config := &consul.Config{
		ServerUrl: "http://127.0.0.1:8500",
	}
	client, _ := reg.NewClient(config)
	reg.client = client
	services := map[string][]string{
		"service1": {"dc1", "dc2"},
		"service2": {"dc1"},
	}

	reg.refresh(services)

	assert.Len(t, reg.watchingServices, 3)
	assert.Contains(t, reg.watchingServices, consulService{ServiceName: "service1", DataCenter: "dc1"})
	assert.Contains(t, reg.watchingServices, consulService{ServiceName: "service1", DataCenter: "dc2"})
	assert.Contains(t, reg.watchingServices, consulService{ServiceName: "service2", DataCenter: "dc1"})
	assert.Empty(t, reg.softDeletedServices)

	reg = &Consul{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		softDeletedServices: map[consulService]bool{},
		watchingServices: map[consulService]bool{
			{ServiceName: "service1", DataCenter: "dc1"}: true,
		},
	}

	services = map[string][]string{}

	reg.refresh(services)

	assert.Len(t, reg.watchingServices, 0)
	assert.Len(t, reg.softDeletedServices, 1)

}

func TestGenerateServiceEntry(t *testing.T) {
	host := "test.default-group.public.earth.nacos"
	reg := &Consul{}

	type test struct {
		name     string
		services []model.SubscribeService
		port     *istioapi.ServicePort
		endpoint *istioapi.WorkloadEntry
	}
	tests := []test{}
	for input, proto := range registry.ProtocolMap {
		s := string(proto)
		tests = append(tests, test{
			name: input,
			services: []model.SubscribeService{
				{Port: 80, Ip: "1.1.1.1", Metadata: map[string]string{
					"protocol": input,
				}},
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

func TestFetchAllServices(t *testing.T) {
	mockConsulCatalog := new(MockConsulCatalog)
	client := &Client{
		consulCatalog: mockConsulCatalog,
		DataCenter:    "dc1",
		NameSpace:     "ns1",
		Token:         "token",
	}

	reg := &Consul{}
	services, err := reg.fetchAllServices(client)
	if err != nil {
		return
	}

	assert.Equal(t, map[consulService]bool{}, services)
}
