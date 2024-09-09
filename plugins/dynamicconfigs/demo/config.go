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

package demo

import (
	"mosn.io/htnn/api/pkg/dynamicconfig"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/types/dynamicconfigs/demo"
)

var (
	demoKey string
)

func init() {
	// Register the implementation of DynamicConfig demo
	dynamicconfig.RegisterDynamicConfigHandler("demo", &handler{})
}

type handler struct {
	demo.Provider
}

// OnUpdate will be called when the dynamic config is updated
func (d *handler) OnUpdate(config any) error {
	c := config.(*demo.Config)
	api.LogInfof("demo dynamic config: %v", c)

	demoKey = c.Key
	return nil
}

// GetDemoKey can be used in the plugins written in Go. It is for show case, don't call it in
// the production code.
func GetDemoKey() string {
	return demoKey
}
