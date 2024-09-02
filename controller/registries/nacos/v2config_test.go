package nacos

import (
	"sync"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/controller/pkg/registry/log"
	"mosn.io/htnn/types/registries/nacos"
)

func TestNewClient(t *testing.T) {
	reg := &Nacos{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		softDeletedServices: map[nacosService]bool{},
		done:                make(chan struct{}),
		lock:                sync.RWMutex{},
	}
	config := &nacos.Config{
		ServerUrl: "http://127.0.0.1:8848",
		Version:   "v2",
	}
	patches := gomonkey.NewPatches()

	patches.ApplyFunc(constant.NewClientConfig, func(options ...constant.ClientOption) *constant.ClientConfig {
		return &constant.ClientConfig{}
	})
	patches.ApplyFunc(constant.NewServerConfig, func(domain string, port uint64, opts ...constant.ServerOption) *constant.ServerConfig {
		return &constant.ServerConfig{}
	})
	patches.ApplyFunc(clients.NewNamingClient, func(param vo.NacosClientParam) (naming_client.INamingClient, error) {
		return nil, nil
	})
	_, err := reg.newClient(config)
	assert.Nil(t, err)

	patches.Reset()

	config = &nacos.Config{}
	_, err = reg.newClient(config)
	assert.Error(t, err)

	config = &nacos.Config{
		ServerUrl: "::::::::::::",
	}

	_, err = reg.newClient(config)
	assert.Error(t, err)

}

func TestStart(t *testing.T) {
	reg := &Nacos{
		logger: log.NewLogger(&log.RegistryLoggerOptions{
			Name: "test",
		}),
		softDeletedServices: map[nacosService]bool{},
		done:                make(chan struct{}),
		lock:                sync.RWMutex{},
	}
	config := &nacos.Config{
		Version: "v2",
	}

	patches := gomonkey.NewPatches()
	patches.ApplyFunc(constant.NewClientConfig, func(options ...constant.ClientOption) *constant.ClientConfig {
		return &constant.ClientConfig{}
	})
	patches.ApplyFunc(constant.NewServerConfig, func(domain string, port uint64, opts ...constant.ServerOption) *constant.ServerConfig {
		return &constant.ServerConfig{}
	})
	patches.ApplyFunc(clients.NewNamingClient, func(param vo.NacosClientParam) (naming_client.INamingClient, error) {
		return nil, nil
	})
	err := reg.Start(config)
	assert.Nil(t, err)

	patches.Reset()

	err = reg.Start(config)
	assert.Error(t, err)

	err = reg.Stop()
	assert.Nil(t, err)

	err = reg.subscribeV2("", "")
	assert.Nil(t, err)

	err = reg.Reload(config)
	assert.Nil(t, err)

	err = reg.refreshV2()
	assert.Nil(t, err)
}
