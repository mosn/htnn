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

package translation

import (
	"encoding/json"
	"sort"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	mosniov1 "mosn.io/moe/controller/api/v1"
	"mosn.io/moe/controller/internal/model"
	"mosn.io/moe/pkg/filtermanager"
	"mosn.io/moe/pkg/plugins"
)

// mergedState does the following:
// 1. merge policy among the same level policies
// 2. merge policy among different hierarchies
// 3. transform a plugin to different plugins if needed
type mergedState struct {
	Hosts map[string]*mergedHostPolicy
}

type mergedHostPolicy struct {
	VirtualHost *model.VirtualHost
	Routes      map[string]*mergedPolicy
}

type mergedPolicy struct {
	Config *filtermanager.FilterManagerConfig
	Info   *Info
	NsName *types.NamespacedName
}

func toNsName(policy *HTTPFilterPolicyWrapper) string {
	return policy.Namespace + "/" + policy.Name
}

// Highest priority policy will be first.
// According to the https://gateway-api.sigs.k8s.io/geps/gep-713/,
// 1. A Policy targeting a more specific scope wins over a policy targeting a lesser specific scope.
// 2. If multiple polices configure the same plugin, the oldest one (based on creation timestamp) wins.
// 3. If there are multiple oldest polices, the one appearing first in alphabetical order by {namespace}/{name} wins.
func sortHttpFilterPolicy(policies []*HTTPFilterPolicyWrapper) {
	// use Slice instead of SliceStable because each policy has unique namespace/name
	sort.Slice(policies, func(i, j int) bool {
		if policies[i].scope != policies[j].scope {
			return policies[i].scope < policies[j].scope
		}
		if policies[i].CreationTimestamp != policies[j].CreationTimestamp {
			return policies[i].CreationTimestamp.Before(&policies[j].CreationTimestamp)
		}
		return toNsName(policies[i]) < toNsName(policies[j])
	})
}

func toMergedPolicy(rp *routePolicy) *mergedPolicy {
	policies := rp.Policies
	sortHttpFilterPolicy(policies)

	info := &Info{
		HTTPFilterPolicies: []string{},
	}
	p := &mosniov1.HTTPFilterPolicy{
		Spec: mosniov1.HTTPFilterPolicySpec{
			Filters: make(map[string]runtime.RawExtension),
		},
	}
	for _, policy := range policies {
		used := false
		for name, filter := range policy.Spec.Filters {
			if _, ok := p.Spec.Filters[name]; !ok {
				p.Spec.Filters[name] = filter
				used = true
			}
		}

		if used {
			info.HTTPFilterPolicies = append(info.HTTPFilterPolicies, toNsName(policy))
		}
	}

	fmc := translateHTTPFilterPolicyToFilterManagerConfig(p)
	return &mergedPolicy{
		Config: fmc,
		Info:   info,
		NsName: rp.NsName,
	}
}

func toMergedState(ctx *Ctx, state *dataPlaneState) (*FinalState, error) {
	s := &mergedState{
		Hosts: make(map[string]*mergedHostPolicy),
	}
	for name, host := range state.Hosts {
		mh := &mergedHostPolicy{
			VirtualHost: host.VirtualHost,
			Routes:      make(map[string]*mergedPolicy),
		}

		for routeName, route := range host.Routes {
			mh.Routes[routeName] = toMergedPolicy(route)
		}

		s.Hosts[name] = mh
	}

	return toFinalState(ctx, s)
}

func translateHTTPFilterPolicyToFilterManagerConfig(policy *mosniov1.HTTPFilterPolicy) *filtermanager.FilterManagerConfig {
	fmc := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{},
	}
	for name, filter := range policy.Spec.Filters {
		cfg := model.GoPluginConfig{}
		// we validated the filter at the beginning, so theorily err should not happen
		_ = json.Unmarshal(filter.Raw, &cfg)
		fmc.Plugins = append(fmc.Plugins, &filtermanager.FilterConfig{
			Name:   name,
			Config: cfg.Config,
		})
	}

	sort.Slice(fmc.Plugins, func(i, j int) bool {
		return plugins.ComparePluginOrder(fmc.Plugins[i].Name, fmc.Plugins[j].Name)
	})
	return fmc
}
