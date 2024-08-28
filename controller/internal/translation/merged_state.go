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
	"reflect"
	"slices"
	"sort"

	"k8s.io/apimachinery/pkg/types"

	"mosn.io/htnn/api/pkg/filtermanager"
	fmModel "mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/pkg/plugins"
	ctrlcfg "mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/model"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

// mergedState does the following:
// 1. merge policy among the same level policies
// 2. merge policy among different hierarchies
// 3. transform a plugin to different plugins if needed
type mergedState struct {
	Proxies map[Proxy]*mergedProxyConfig
}

type mergedProxyConfig struct {
	Hosts    map[string]*mergedHostPolicy
	Gateways map[string]*mergedGatewayPolicy
}

type mergedHostPolicy struct {
	VirtualHost *model.VirtualHost
	Routes      map[string]*mergedPolicy
}

type mergedGatewayPolicy struct {
	Gateway *model.Gateway
	Policy  *mergedPolicy
}

type mergedPolicy struct {
	Config map[string]interface{}
	Info   *Info
	NsName *types.NamespacedName
}

func toNsName(policy *FilterPolicyWrapper) string {
	return policy.Namespace + "/" + policy.Name
}

// Highest priority policy will be first.
// According to the https://gateway-api.sigs.k8s.io/geps/gep-713/,
// 1. A Policy targeting a more specific scope wins over a policy targeting a lesser specific scope.
// 2. If multiple polices configure the same plugin, the oldest one (based on creation timestamp) wins.
// 3. If there are multiple oldest polices, the one appearing first in alphabetical order by {namespace}/{name} wins.
func sortFilterPolicy(policies []*FilterPolicyWrapper) {
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

type PolicyKind int

const (
	PolicyKindRDS PolicyKind = iota
	PolicyKindLDS
)

func translateFilterManagerConfigToPolicyInRDS(fmc *filtermanager.FilterManagerConfig,
	nsName *types.NamespacedName, virtualHost *model.VirtualHost) map[string]interface{} {

	config := map[string]interface{}{}

	nativeFilters := []*fmModel.FilterConfig{}
	goFilterManager := &filtermanager.FilterManagerConfig{
		Plugins: []*fmModel.FilterConfig{},
	}

	consumerNeeded := false
	for _, plugin := range fmc.Plugins {
		name := plugin.Name
		url := ""
		p := plugins.LoadPlugin(name)
		if p == nil {
			// For Go Plugins, only the type is registered
			p = plugins.LoadPluginType(name)
		}
		// As we don't reject configuration with unknown plugin to keep compatibility...
		if p == nil {
			continue
		}

		var cfg interface{}
		// we validated the filter at the beginning, so theorily err should not happen
		b, ok := plugin.Config.([]byte)
		if !ok {
			panic(fmt.Sprintf("unexpected type: %s", reflect.TypeOf(plugin.Config)))
		}
		_ = json.Unmarshal(b, &cfg)

		nativePlugin, ok := p.(plugins.NativePlugin)
		if !ok {
			plugin.Config = cfg
			goFilterManager.Plugins = append(goFilterManager.Plugins, plugin)
		} else {
			url = nativePlugin.ConfigTypeURL()
			m, ok := cfg.(map[string]interface{})
			if !ok {
				panic(fmt.Sprintf("unexpected type: %s", reflect.TypeOf(cfg)))
			}

			// Extra fields are allowed in cfg, as `--reject-unknown-dynamic-fields` is turned off
			// by default. If users want to break the backward compatibility by turning it on, this
			// is their trouble.

			// We expect user to use camelCase as the field name. If not, the ToRouteConfig may not
			// work as expected.

			if wrapper, ok := p.(plugins.HTTPNativePluginHasRouteConfigWrapper); ok {
				m = wrapper.ToRouteConfig(m)
			}

			m["@type"] = url
			plugin.Config = m
			nativeFilters = append(nativeFilters, plugin)
		}

		_, ok = p.(plugins.ConsumerPlugin)
		if ok {
			consumerNeeded = true
		}
	}
	if consumerNeeded {
		goFilterManager.Namespace = nsName.Namespace
	}

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

		golangFilterName := "htnn.filters.http.golang"
		if ctrlcfg.EnableLDSPluginViaECDS() {
			golangFilterName = virtualHost.ECDSResourceName + "-" + model.CategoryGolangPlugins
		}
		config[golangFilterName] = map[string]interface{}{
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

	return config
}

func translateFilterManagerConfigToPolicyInLDS(fmc *filtermanager.FilterManagerConfig, nsName *types.NamespacedName) map[string]interface{} {
	config := map[string]interface{}{}

	goFilterManager := &filtermanager.FilterManagerConfig{
		Plugins: []*fmModel.FilterConfig{},
	}
	nativeFilters := map[string][]*fmModel.FilterConfig{
		model.CategoryECDSListener: {},
		model.CategoryECDSNetwork:  {},
		model.CategoryListener:     {},
	}

	consumerNeeded := false
	for _, plugin := range fmc.Plugins {
		name := plugin.Name
		p := plugins.LoadPlugin(name)
		if p == nil {
			// For Go Plugins, only the type is registered
			p = plugins.LoadPluginType(name)
		}
		// As we don't reject configuration with unknown plugin to keep compatibility...
		if p == nil {
			continue
		}

		var cfg interface{}
		// we validated the filter at the beginning, so theorily err should not happen
		b, ok := plugin.Config.([]byte)
		if !ok {
			panic(fmt.Sprintf("unexpected type: %s", reflect.TypeOf(plugin.Config)))
		}
		_ = json.Unmarshal(b, &cfg)

		nativePlugin, ok := p.(plugins.NativePlugin)
		if !ok {
			plugin.Config = cfg
			goFilterManager.Plugins = append(goFilterManager.Plugins, plugin)
			_, ok = p.(plugins.ConsumerPlugin)
			if ok {
				consumerNeeded = true
			}
		} else {
			order := nativePlugin.Order()
			if order.Position == plugins.OrderPositionOuter || order.Position == plugins.OrderPositionInner {
				// HTTP Native plugin is not supported
				continue
			}

			m, ok := cfg.(map[string]interface{})
			if !ok {
				panic(fmt.Sprintf("unexpected type: %s", reflect.TypeOf(cfg)))
			}

			url := nativePlugin.ConfigTypeURL()
			if url != "" {
				m["@type"] = url
				plugin.Config = m

				if order.Position == plugins.OrderPositionListener {
					nativeFilters[model.CategoryECDSListener] = append(nativeFilters[model.CategoryECDSListener], plugin)
				} else if order.Position == plugins.OrderPositionNetwork {
					nativeFilters[model.CategoryECDSNetwork] = append(nativeFilters[model.CategoryECDSNetwork], plugin)
				}
			} else {
				plugin.Config = m

				if order.Position == plugins.OrderPositionListener {
					nativeFilters[model.CategoryListener] = append(nativeFilters[model.CategoryListener], plugin)
				}
				// TODO: support network filter
			}
		}
	}

	if consumerNeeded {
		goFilterManager.Namespace = nsName.Namespace
	}

	if len(goFilterManager.Plugins) > 0 {
		cfg := map[string]interface{}{}
		if goFilterManager.Namespace != "" {
			cfg["namespace"] = goFilterManager.Namespace
		}
		plugins := make([]interface{}, len(goFilterManager.Plugins))
		for i, plugin := range goFilterManager.Plugins {
			plugins[i] = map[string]interface{}{
				"name":   plugin.Name,
				"config": plugin.Config,
			}
		}
		cfg["plugins"] = plugins
		config[model.CategoryECDSGolang] = cfg
	}

	for category, filters := range nativeFilters {
		config[category] = filters
	}
	return config
}

func toMergedPolicy(nsName *types.NamespacedName, policies []*FilterPolicyWrapper,
	policyKind PolicyKind, virtualHost *model.VirtualHost) *mergedPolicy {

	sortFilterPolicy(policies)

	p := &mosniov1.FilterPolicy{
		Spec: mosniov1.FilterPolicySpec{
			Filters: make(map[string]mosniov1.Plugin),
		},
	}

	// use map to deduplicate policies, especially for the sub-policies
	usedFP := make(map[string]struct{}, len(policies))
	for _, policy := range policies {
		used := false
		for name, filter := range policy.Spec.Filters {
			if _, ok := p.Spec.Filters[name]; !ok {
				p.Spec.Filters[name] = filter
				used = true
			}
		}

		if used {
			usedFP[toNsName(policy)] = struct{}{}
		}
	}

	info := &Info{
		FilterPolicies: make([]string, 0, len(usedFP)),
	}
	for s := range usedFP {
		info.FilterPolicies = append(info.FilterPolicies, s)
	}
	slices.Sort(info.FilterPolicies) // order is required for later procession

	fmc := translateFilterPolicyToFilterManagerConfig(p)
	var config map[string]interface{}
	if policyKind == PolicyKindRDS {
		config = translateFilterManagerConfigToPolicyInRDS(fmc, nsName, virtualHost)
	} else if policyKind == PolicyKindLDS {
		config = translateFilterManagerConfigToPolicyInLDS(fmc, nsName)
	}

	return &mergedPolicy{
		Config: config,
		Info:   info,
		NsName: nsName,
	}
}

func translateFilterPolicyToFilterManagerConfig(policy *mosniov1.FilterPolicy) *filtermanager.FilterManagerConfig {
	fmc := &filtermanager.FilterManagerConfig{
		Plugins: []*fmModel.FilterConfig{},
	}
	for name, filter := range policy.Spec.Filters {
		fmc.Plugins = append(fmc.Plugins, &fmModel.FilterConfig{
			Name:   name,
			Config: filter.Config.Raw,
		})
	}

	sortPlugins(fmc.Plugins)
	return fmc
}

func sortPlugins(ps []*fmModel.FilterConfig) {
	sort.Slice(ps, func(i, j int) bool {
		return plugins.ComparePluginOrder(ps[i].Name, ps[j].Name)
	})
}

func toMergedState(ctx *Ctx, state *dataPlaneState) (*FinalState, error) {
	s := &mergedState{
		Proxies: make(map[Proxy]*mergedProxyConfig),
	}

	for proxy, cfg := range state.Proxies {
		mergedHosts := make(map[string]*mergedHostPolicy)
		for name, host := range cfg.Hosts {
			mh := &mergedHostPolicy{
				VirtualHost: host.VirtualHost,
				Routes:      make(map[string]*mergedPolicy),
			}

			for routeName, route := range host.Routes {
				mergedPolicy := toMergedPolicy(route.NsName, route.Policies, PolicyKindRDS, mh.VirtualHost)
				mh.Routes[routeName] = mergedPolicy
			}

			mergedHosts[name] = mh
		}

		mergedGateways := make(map[string]*mergedGatewayPolicy)
		for name, gateway := range cfg.Gateways {
			mg := &mergedGatewayPolicy{
				Gateway: gateway.Gateway,
			}
			if len(gateway.Policies) > 0 {
				mg.Policy = toMergedPolicy(&gateway.Gateway.GatewaySection.NsName, gateway.Policies, PolicyKindLDS, nil)
			}

			mergedGateways[name] = mg
		}

		s.Proxies[proxy] = &mergedProxyConfig{
			Hosts:    mergedHosts,
			Gateways: mergedGateways,
		}
	}

	return toFinalState(ctx, s)
}
