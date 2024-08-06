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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"mosn.io/htnn/controller/pkg/registry/log"
	"mosn.io/htnn/types/registries/consul"
)

func TestNewClient(t *testing.T) {
	reg := &Consul{
		clientFactory: factory,
	}
	config := &consul.Config{
		ServerUrl:  "http://127.0.0.1:8500",
		DataCenter: "test",
	}
	client, err := reg.clientFactory.NewClient(config)

	assert.NoError(t, err)
	assert.NotNil(t, client)

	config = &consul.Config{
		ServerUrl:  "::::::::::::",
		DataCenter: "test",
	}

	client, err = reg.clientFactory.NewClient(config)

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

type MockClientFactory struct {
	mock.Mock
}

func (f *MockClientFactory) NewClient(config *consul.Config) (*Client, error) {
	mockConsulCatalog := new(MockConsulCatalog)
	return &Client{
		consulCatalog: mockConsulCatalog,
		DataCenter:    "dc1",
		NameSpace:     "ns1",
		Token:         "token",
	}, nil
}

func TestStart(t *testing.T) {
	mockConsulCatalog := new(MockConsulCatalog)
	cf := new(MockClientFactory)
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
		done:          make(chan struct{}),
		clientFactory: cf,
	}

	config := &consul.Config{}

	mockConsulCatalog.On("Services", mock.Anything).Return(map[string][]string{"service1": {"dc1"}}, &api.QueryMeta{}, nil)
	reg.client = client
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
		done:          make(chan struct{}),
		clientFactory: factory,
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
		clientFactory:       factory,
	}

	config := &consul.Config{
		ServerUrl: "http://127.0.0.1:8500",
	}
	client, _ := reg.clientFactory.NewClient(config)
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
