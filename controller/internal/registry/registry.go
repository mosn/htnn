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

package registry

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	pkgRegistry "mosn.io/htnn/controller/pkg/registry"
)

var (
	registries = map[types.NamespacedName]pkgRegistry.Registry{}
)

type RegistryManagerOption struct {
	Client client.Client
}

func InitRegistryManager(opt *RegistryManagerOption) {
}

func UpdateRegistry(registry *mosniov1.ServiceRegistry) error {
	key := types.NamespacedName{Namespace: registry.Namespace, Name: registry.Name}
	if reg, ok := registries[key]; !ok {
		reg, err := pkgRegistry.CreateRegistry(registry.Spec.Type, nil)
		if err != nil {
			return err
		}

		conf, err := pkgRegistry.ParseConfig(reg, registry.Spec.Config.Raw)
		if err != nil {
			return err
		}

		err = reg.Start(conf)
		if err != nil {
			return err
		}

		// only started registry can be put into registries
		registries[key] = reg

	} else {
		conf, err := pkgRegistry.ParseConfig(reg, registry.Spec.Config.Raw)
		if err != nil {
			return err
		}

		err = reg.Reload(conf)
		if err != nil {
			return err
		}
	}

	return nil
}

func DeleteRegistry(key types.NamespacedName) error {
	prev, ok := registries[key]
	if !ok {
		return nil
	}

	delete(registries, key)
	return prev.Stop()
}
