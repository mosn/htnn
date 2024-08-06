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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/hashicorp/consul/api"
	. "github.com/smartystreets/goconvey/convey"
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
		done: make(chan struct{}),
	}

	Convey("Test Start method", t, func() {

		patches := gomonkey.ApplyMethod(reflect.TypeOf(reg), "FetchAllServices", func(_ *Consul, client *Client) (map[consulService]bool, error) {
			return map[consulService]bool{
				{ServiceName: "service1", Tag: "tag1"}: true,
				{ServiceName: "service2", Tag: "tag2"}: true,
			}, nil
		})
		defer patches.Reset()
		config := &consul.Config{}
		err := reg.Start(config)
		So(err, ShouldBeNil)
		err = reg.subscribe("123")
		So(err, ShouldBeNil)

		err = reg.unsubscribe("123")
		So(err, ShouldBeNil)

		err = reg.Stop()
		So(err, ShouldBeNil)

	})

	config := &consul.Config{}

	reg = &Consul{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		done: make(chan struct{}),
	}

	err := reg.Start(config)
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
	assert.Contains(t, reg.watchingServices, consulService{ServiceName: "service1", Tag: "dc1"})
	assert.Contains(t, reg.watchingServices, consulService{ServiceName: "service1", Tag: "dc2"})
	assert.Contains(t, reg.watchingServices, consulService{ServiceName: "service2", Tag: "dc1"})
	assert.Empty(t, reg.softDeletedServices)

	reg = &Consul{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		softDeletedServices: map[consulService]bool{},
		watchingServices: map[consulService]bool{
			{ServiceName: "service1", Tag: "dc1"}: true,
		},
	}

	services = map[string][]string{}

	reg.refresh(services)

	assert.Len(t, reg.watchingServices, 0)
	assert.Len(t, reg.softDeletedServices, 1)

}

func TestFetchAllServices(t *testing.T) {
	Convey("Test FetchAllServices method", t, func() {
		reg := &Consul{
			logger: log.NewLogger(&log.RegistryLoggerOptions{
				Name: "test",
			}),
		}
		client := &Client{
			consulCatalog: &api.Catalog{},
			DataCenter:    "dc1",
			NameSpace:     "ns1",
			Token:         "token",
		}

		patches := gomonkey.ApplyMethod(reflect.TypeOf(client.consulCatalog), "Services", func(_ *api.Catalog, q *api.QueryOptions) (map[string][]string, *api.QueryMeta, error) {
			return map[string][]string{
				"service1": {"tag1", "tag2"},
				"service2": {"tag3"},
			}, nil, nil
		})
		defer patches.Reset()

		services, err := reg.FetchAllServices(client)
		So(err, ShouldBeNil)
		So(services, ShouldNotBeNil)
		So(services[consulService{ServiceName: "service1", Tag: "tag1"}], ShouldBeTrue)
		So(services[consulService{ServiceName: "service1", Tag: "tag2"}], ShouldBeTrue)
		So(services[consulService{ServiceName: "service2", Tag: "tag3"}], ShouldBeTrue)
	})

	Convey("Test FetchAllServices method with error", t, func() {
		reg := &Consul{
			logger: log.NewLogger(&log.RegistryLoggerOptions{
				Name: "test",
			}),
		}
		client := &Client{
			consulCatalog: &api.Catalog{},
			DataCenter:    "dc1",
			NameSpace:     "ns1",
			Token:         "token",
		}

		patches := gomonkey.ApplyMethod(reflect.TypeOf(client.consulCatalog), "Services", func(_ *api.Catalog, q *api.QueryOptions) (map[string][]string, *api.QueryMeta, error) {
			return nil, nil, errors.New("mock error")
		})
		defer patches.Reset()

		services, err := reg.FetchAllServices(client)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "mock error")
		So(services, ShouldBeNil)
	})
}
