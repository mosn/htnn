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

	"github.com/stretchr/testify/assert"

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
		done:                make(chan struct{}),
		watchingServices:    map[consulService]bool{},
	}
	config := &consul.Config{
		ServerUrl: "http://127.0.0.1:8500",
	}

	err := reg.Start(config)
	assert.NoError(t, err)

	err = reg.subscribe("123")
	assert.Nil(t, err)

	err = reg.unsubscribe("123")
	assert.Nil(t, err)

	err = reg.Stop()
	assert.Nil(t, err)
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
