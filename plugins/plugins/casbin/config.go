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

package casbin

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/casbin/casbin/v2"
	"github.com/fsnotify/fsnotify"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/plugins/pkg/file"
	casbintype "mosn.io/htnn/types/plugins/casbin"
)

func init() {
	plugins.RegisterPlugin(casbintype.Name, &plugin{})
}

type plugin struct {
	casbintype.Plugin
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	casbintype.Config

	lock *sync.RWMutex

	enforcer   *casbin.Enforcer
	modelFile  *file.File
	policyFile *file.File
	updating   atomic.Bool

	watcher *fsnotify.Watcher
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	conf.lock = &sync.RWMutex{}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	conf.watcher = watcher

	f, err := file.Stat(conf.Rule.Model, watcher)
	if err != nil {
		return err
	}
	conf.modelFile = f

	f, err = file.Stat(conf.Rule.Policy, watcher)
	if err != nil {
		return err
	}
	conf.policyFile = f

	e, err := casbin.NewEnforcer(conf.Rule.Model, conf.Rule.Policy)
	if err != nil {
		return err
	}
	conf.enforcer = e

	runtime.SetFinalizer(conf, func(conf *config) {
		err := conf.watcher.Close()
		if err != nil {
			api.LogErrorf("failed to close watcher, err: %v", err)
		}
	})
	return nil
}

func (conf *config) reloadEnforcer() {
	if !conf.updating.Load() {
		conf.updating.Store(true)
		api.LogWarnf("policy %s or model %s changed, reload enforcer", conf.policyFile.Name, conf.modelFile.Name)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					api.LogErrorf("recovered from panic: %v", r)
				}
				conf.updating.Store(false)
			}()
			e, err := casbin.NewEnforcer(conf.Rule.Model, conf.Rule.Policy)
			if err != nil {
				api.LogErrorf("failed to update Enforcer: %v", err)
			} else {
				conf.SetChanged(false)
				conf.lock.Lock()
				conf.enforcer = e
				conf.lock.Unlock()
				api.LogWarnf("policy %s or model %s changed, enforcer reloaded", conf.policyFile.Name, conf.modelFile.Name)
			}
		}()
	}
}

func (conf *config) SetChanged(change bool) {
	conf.lock.Lock()
	Changed = change
	conf.lock.Unlock()
}

func (conf *config) GetChanged() bool {
	conf.lock.RLock()
	changed := Changed
	conf.lock.RUnlock()
	return changed
}
