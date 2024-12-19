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

package filtermanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"

	xds "github.com/cncf/xds/go/xds/type/v3"
	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	pkgPlugins "mosn.io/htnn/api/pkg/plugins"
)

// We can't import package below here that will cause build failure in Mac
// "github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
// Therefore, the FilterManagerConfigParser & FilterManagerFactory need to be exportable.
// The http.RegisterHttpFilterFactoryAndParser will be called in the main.go when building
// the shared library in Linux.

type FilterManagerConfigParser struct {
}

type FilterManagerConfig struct {
	Namespace string `json:"namespace,omitempty"`

	Plugins []*model.FilterConfig `json:"plugins"`
}

type filterManagerConfig struct {
	consumerFiltersEndAt int

	initOnce             *sync.Once
	initFailed           bool
	initFailure          error
	initFailedPluginName string

	parsed []*model.ParsedFilterConfig
	pool   *sync.Pool

	namespace string

	enableDebugMode bool
}

func initFilterManagerConfig(namespace string) *filterManagerConfig {
	config := &filterManagerConfig{
		namespace: namespace,
	}
	config.pool = &sync.Pool{
		New: func() any {
			callbacks := &filterManagerCallbackHandler{
				namespace: namespace,
			}
			fm := &filterManager{
				callbacks: callbacks,
				config:    config,

				decodeIdx: -1,
				encodeIdx: -1,
			}
			return fm
		},
	}
	return config
}

// Merge merges another filterManagerConfig into a copy of current filterManagerConfig, and then returns
// the copy
func (conf *filterManagerConfig) Merge(another *filterManagerConfig) *filterManagerConfig {
	ns := conf.namespace
	if ns == "" {
		ns = another.namespace
	}

	// It's tough to do the data plane merge right. We don't use shallow copy, which may share
	// data structure accidentally. We don't use deep copy all the fields, which may copy unexpected computed data.
	// Let's copy fields manually.
	cp := initFilterManagerConfig(ns)

	if conf.initOnce != nil || another.initOnce != nil {
		cp.initOnce = &sync.Once{}
	}

	cp.enableDebugMode = conf.enableDebugMode
	if another.enableDebugMode {
		cp.enableDebugMode = true
	}

	cp.parsed = make([]*model.ParsedFilterConfig, 0, len(conf.parsed)+len(another.parsed))
	// For now, we don't deepcopy the config. The config may contain connection to the external
	// service, for example, a Redis cluster. Not sure if it is safe to deepcopy them. So far,
	// sharing the config created from route when the previous HTTP filter existed is fine.
	cp.parsed = append(cp.parsed, conf.parsed...)

	// O(n^2) is fine as n is small
	for _, toAdd := range another.parsed {
		needAdd := true
		for _, fc := range conf.parsed {
			if fc.Name == toAdd.Name {
				// The filter is already in the current config, skip it
				needAdd = false
				break
			}
		}

		if needAdd {
			// For now, we don't deepcopy the config from HTTP filter. Consider a case,
			// a HTTP filter, which is shared by 1000 routes, has a hugh ACL. If we deepcopy
			// it, the memory usage is too expensive.
			cp.parsed = append(cp.parsed, toAdd)
		}
	}
	sort.Slice(cp.parsed, func(i, j int) bool {
		return pkgPlugins.ComparePluginOrder(cp.parsed[i].Name, cp.parsed[j].Name)
	})

	// recompute fields which will be different after merging
	cp.consumerFiltersEndAt = len(cp.parsed)
	for i, fc := range cp.parsed {
		_, ok := pkgPlugins.LoadPlugin(fc.Name).(pkgPlugins.ConsumerPlugin)
		if !ok {
			cp.consumerFiltersEndAt = i
			break
		}
	}

	api.LogInfof("after merged http filter, filtermanager config: %+v", cp)
	if api.GetLogLevel() <= api.LogLevelDebug {
		for _, fc := range cp.parsed {
			api.LogDebugf("after merged http filter, plugin: %s, config: %+v", fc.Name, fc.ParsedConfig)
		}
	}
	return cp
}

func (conf *filterManagerConfig) InitOnce() {
	if conf.initOnce == nil {
		return
	}

	conf.initOnce.Do(func() {
		for _, fc := range conf.parsed {
			config := fc.ParsedConfig
			if initer, ok := config.(pkgPlugins.Initer); ok {
				fc.InitOnce.Do(func() {
					// For now, we have nothing to provide as config callbacks
					fc.InitFailure = initer.Init(nil)
				})
				if fc.InitFailure != nil {
					conf.initFailure = fc.InitFailure
					conf.initFailedPluginName = fc.Name
					conf.initFailed = true
				}
			}
		}
	})
}

func (p *FilterManagerConfigParser) Parse(any *anypb.Any, callbacks capi.ConfigCallbackHandler) (interface{}, error) {
	if callbacks == nil {
		api.LogErrorf("no config callback handler provided")
		// the call back handler to be nil only affects plugin metrics, so we can continue
	}
	configStruct := &xds.TypedStruct{}

	// No configuration
	if any.GetTypeUrl() == "" {
		conf := initFilterManagerConfig("")
		return conf, nil
	}

	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	if configStruct.Value == nil {
		return nil, errors.New("bad TypedStruct format")
	}

	data, err := configStruct.Value.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// TODO: figure out a way to identify what the config is belonged to, like using the route name
	api.LogInfof("receive filtermanager config: %s", data)

	fmConfig := &FilterManagerConfig{}
	if err := json.Unmarshal(data, fmConfig); err != nil {
		return nil, err
	}

	plugins := fmConfig.Plugins
	conf := initFilterManagerConfig(fmConfig.Namespace)
	conf.parsed = make([]*model.ParsedFilterConfig, 0, len(plugins))

	consumerFiltersEndAt := 0
	i := 0
	needInit := false

	for _, proto := range plugins {
		name := proto.Name

		if plugin := pkgPlugins.LoadHTTPFilterFactoryAndParser(name); plugin != nil {
			config, err := plugin.ConfigParser.Parse(proto.Config)
			if err != nil {
				api.LogErrorf("%s during parsing plugin %s in filtermanager", err, name)

				// Return an error from the Parse method will cause assertion failure.
				// See https://github.com/envoyproxy/envoy/blob/f301eebf7acc680e27e03396a1be6be77e1ae3a5/contrib/golang/filters/http/source/golang_filter.cc#L1736-L1737
				// As we can't control what is returned from a plugin, we need to
				// avoid the failure by providing a special factory, which also
				// indicates something is wrong.
				conf.parsed = append(conf.parsed, &model.ParsedFilterConfig{
					Name:    proto.Name,
					Factory: NewInternalErrorFactory(proto.Name, err),
				})
			} else {
				conf.parsed = append(conf.parsed, &model.ParsedFilterConfig{
					Name:          proto.Name,
					ParsedConfig:  config,
					Factory:       plugin.Factory,
					SyncRunPhases: plugin.ConfigParser.NonBlockingPhases(),
				})

				_, ok := pkgPlugins.LoadPlugin(name).(pkgPlugins.ConsumerPlugin)
				if ok {
					consumerFiltersEndAt = i + 1
				}

				if _, ok := config.(pkgPlugins.Initer); ok {
					needInit = true
				}
				if register, ok := config.(pkgPlugins.MetricsRegister); ok {
					register.MetricsDefinition(callbacks)
					api.LogInfof("loaded metrics definition for plugin: %s", name)
				}

				if name == "debugMode" {
					// we handle this plugin differently, so we can have debug behavior before
					// executing this plugin.
					conf.enableDebugMode = true
				}
			}
			i++

		} else {
			api.LogErrorf("plugin %s not found, ignored", name)
		}
	}
	conf.consumerFiltersEndAt = consumerFiltersEndAt

	if needInit {
		conf.initOnce = &sync.Once{}
	}

	return conf, nil
}

func (p *FilterManagerConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	httpFilterCfg, ok := parent.(*filterManagerConfig)
	if !ok {
		panic(fmt.Sprintf("wrong config type: %s", reflect.TypeOf(httpFilterCfg)))
	}
	routeCfg, ok := child.(*filterManagerConfig)
	if !ok {
		panic(fmt.Sprintf("wrong config type: %s", reflect.TypeOf(routeCfg)))
	}

	if httpFilterCfg == nil || len(httpFilterCfg.parsed) == 0 {
		return routeCfg
	}

	return routeCfg.Merge(httpFilterCfg)
}
