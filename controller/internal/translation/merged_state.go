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
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/types"

	"mosn.io/htnn/api/pkg/filtermanager"
	fmModel "mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/pkg/plugins"
	mosniov1 "mosn.io/htnn/controller/apis/v1"
	"mosn.io/htnn/controller/internal/model"
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
	Config map[string]interface{}
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
			Filters: make(map[string]mosniov1.HTTPPlugin),
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
	nativeFilters := []*fmModel.FilterConfig{}
	goFilterManager := &filtermanager.FilterManagerConfig{
		Plugins: []*fmModel.FilterConfig{},
	}

	consumerNeeded := false
	for _, plugin := range fmc.Plugins {
		name := plugin.Name
		url := ""
		p := plugins.LoadHttpPlugin(name)
		nativePlugin, ok := p.(plugins.NativePlugin)
		if ok {
			url = nativePlugin.RouteConfigTypeURL()
		}
		if !ok {
			goFilterManager.Plugins = append(goFilterManager.Plugins, plugin)
		} else {
			if wrapper, ok := p.(plugins.NativePluginHasRouteConfigWrapper); ok {
				plugin.Config = wrapper.ToRouteConfig(plugin.Config.(map[string]interface{}))
			}

			m := plugin.Config.(map[string]interface{})
			m["@type"] = url
			nativeFilters = append(nativeFilters, plugin)
		}

		_, ok = p.(plugins.ConsumerPlugin)
		if ok {
			consumerNeeded = true
		}
	}
	if consumerNeeded {
		goFilterManager.Namespace = rp.NsName.Namespace
	}

	config := map[string]interface{}{}
	if len(goFilterManager.Plugins) > 0 {
		v := map[string]interface{}{}
		if goFilterManager.Namespace != "" {
			v["namespace"] = goFilterManager.Namespace
		}
		plugins := make([]interface{}, len(goFilterManager.Plugins))
		for i, plugin := range goFilterManager.Plugins {
			plugins[i] = map[string]interface{}{
				"name":   plugin.Name,
				"config": plugin.Config,
			}
		}
		v["plugins"] = plugins

		config["htnn.filters.http.golang"] = map[string]interface{}{
			"@type": "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.ConfigsPerRoute",
			"plugins_config": map[string]interface{}{
				"fm": map[string]interface{}{
					"config": map[string]interface{}{
						"@type": "type.googleapis.com/xds.type.v3.TypedStruct",
						"value": v,
					},
				},
			},
		}
	}

	for _, filter := range nativeFilters {
		name := fmt.Sprintf("htnn.filters.http.%s", filter.Name)
		config[name] = filter.Config
	}

	return &mergedPolicy{
		Config: config,
		Info:   info,
		NsName: rp.NsName,
	}
}

func sortPlugins(ps []*fmModel.FilterConfig) {
	sort.Slice(ps, func(i, j int) bool {
		return plugins.ComparePluginOrder(ps[i].Name, ps[j].Name)
	})
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
			mergedPolicy := toMergedPolicy(route)
			mh.Routes[routeName] = mergedPolicy
		}

		s.Hosts[name] = mh
	}

	return toFinalState(ctx, s)
}

func translateHTTPFilterPolicyToFilterManagerConfig(policy *mosniov1.HTTPFilterPolicy) *filtermanager.FilterManagerConfig {
	fmc := &filtermanager.FilterManagerConfig{
		Plugins: []*fmModel.FilterConfig{},
	}
	for name, filter := range policy.Spec.Filters {
		var cfg interface{}
		// we validated the filter at the beginning, so theorily err should not happen
		_ = json.Unmarshal(filter.Config.Raw, &cfg)
		fmc.Plugins = append(fmc.Plugins, &fmModel.FilterConfig{
			Name:   name,
			Config: cfg,
		})
	}

	sortPlugins(fmc.Plugins)
	return fmc
}
