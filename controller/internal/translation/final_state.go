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
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/idna"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"

	"mosn.io/htnn/controller/internal/istio"
	"mosn.io/htnn/controller/internal/model"
	"mosn.io/htnn/controller/pkg/component"
	"mosn.io/htnn/controller/pkg/constant"
)

const (
	AnnotationInfo = "htnn.mosn.io/info"

	DefaultEnvoyFilterPriority = -10
)

var (
	validEnvoyFilterName = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
)

// We use the domain as the EnvoyFilter's name, so that:
// 1. We can easily find per domain rules.
// 2. Match the EnvoyFilter model which uses domain + routeName as the key.
// 3. Allow merging the same route configuration into virtual host level.
// There are also some drawbacks. For example, a domain shared by hundreds of VirtualServices will
// cause one big EnvoyFilter. Let's see if it's a problem.
func envoyFilterNameFromVirtualHost(vhost *model.VirtualHost) string {
	// Strip the port number. We don't need to create two EnvoyFilters for :80 and :443.
	domain, port, _ := net.SplitHostPort(vhost.Name)
	// We join the host & port in toDataPlaneState so the domain is not nil

	if domain == "*" {
		// specific case for port-only HTTP policies
		domain = port
	} else if strings.HasPrefix(domain, "*.") {
		// '*' is not allowed in EnvoyFilter name. And '.' can only be used after alphanumeric characters.
		// So we replace the '*.' with '-'.
		// The regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'.
		domain = "-" + domain[2:]
	}

	// The `htnn-h` means the HTNN's FilterPolicy.
	prefix := "htnn-h"
	domain, err := idna.ToASCII(domain)
	if err == nil {
		name := fmt.Sprintf("%s-%s", prefix, domain)
		if validEnvoyFilterName.MatchString(name) {
			return name
		}
	}

	// Bad domain specified. Fallback to the source of configuration
	return fmt.Sprintf("%s-%s.%s", prefix, vhost.NsName.Namespace, vhost.NsName.Name)
}

func envoyFilterNameFromLds(ldsName string) string {
	ldsName = strings.ReplaceAll(ldsName, "_", "-")
	ldsName = strings.ReplaceAll(ldsName, ":", "-")
	return fmt.Sprintf("htnn-lds-%s", ldsName)
}

// finalState is the end of the translation. We convert the state to EnvoyFilter and write it to k8s.
type FinalState struct {
	EnvoyFilters map[component.EnvoyFilterKey]*istiov1a3.EnvoyFilter
}

type envoyFilterWrapper struct {
	*istiov1a3.EnvoyFilter
	info *Info
}

func toFinalState(_ *Ctx, state *mergedState) (*FinalState, error) {
	efs := istio.DefaultEnvoyFilters()
	for _, ef := range efs {
		ef.Spec.Priority = DefaultEnvoyFilterPriority
	}
	efList := []*envoyFilterWrapper{}

	for proxy, cfg := range state.Proxies {
		hostRules := cfg.Hosts
		for _, host := range hostRules {
			for routeName, route := range host.Routes {
				ef := istio.GenerateRouteFilter(host.VirtualHost, routeName, route.Config)
				// Set the EnvoyFilter's namespace to the workload's namespace.
				// For k8s Gateway API, the workload's namespace is equal to the Gateway's namespace.
				// For Istio API, we will require env var PILOT_SCOPE_GATEWAY_TO_NAMESPACE to be set.
				// If PILOT_SCOPE_GATEWAY_TO_NAMESPACE is not set, people need to follow the convention
				// that the namespace of workload matches the namespace of gateway.
				ns := proxy.Namespace
				ef.SetNamespace(ns)
				name := envoyFilterNameFromVirtualHost(host.VirtualHost)
				ef.SetName(name)

				efList = append(efList, &envoyFilterWrapper{
					EnvoyFilter: ef,
					info:        route.Info,
				})
			}
		}

		gateways := cfg.Gateways
		for name, gateway := range gateways {
			ns := proxy.Namespace
			key := getECDSResourceName(ns, name)
			var config map[string]interface{}
			var info *Info
			if gateway.Policy != nil {
				config = gateway.Policy.Config
				info = gateway.Policy.Info
			}

			ef := istio.GenerateLDSFilter(key, name, gateway.Gateway.HasHCM, config)
			ef.SetNamespace(ns)
			// Put all LDS level filters of the same LDS into the same EnvoyFilter.
			efName := envoyFilterNameFromLds(name)
			// Each LDS has it own EnvoyFilter, so it's easy to figure out how many filters are inserted into one LDS and their order.
			ef.SetName(efName)

			efList = append(efList, &envoyFilterWrapper{
				EnvoyFilter: ef,
				info:        info,
			})
		}
	}

	// Merge EnvoyFilters with same name. The number of EnvoyFilters is equal to the number of
	// configured domains and lds.
	efws := map[component.EnvoyFilterKey]*envoyFilterWrapper{}
	for _, ef := range efList {
		key := component.EnvoyFilterKey{
			Namespace: ef.GetNamespace(),
			Name:      ef.GetName(),
		}
		if curr, ok := efws[key]; ok {
			curr.Spec.ConfigPatches = append(curr.Spec.ConfigPatches, ef.Spec.ConfigPatches...)
			if ef.info != nil {
				if curr.info == nil {
					curr.info = ef.info
				} else {
					curr.info.Merge(ef.info)
				}
			}
		} else {
			efws[key] = ef
		}
	}

	for key, ef := range efws {
		if ef.info != nil {
			ef.SetAnnotations(map[string]string{
				AnnotationInfo: ef.info.String(),
			})
		}
		if ef.Labels == nil {
			ef.Labels = map[string]string{}
		}
		ef.Labels[constant.LabelCreatedBy] = "FilterPolicy"

		if strings.HasPrefix(ef.Name, "htnn-h-") {
			// Sort here to avoid EnvoyFilter change caused by the order of ConfigPatch.
			sort.Slice(ef.Spec.ConfigPatches, func(i, j int) bool {
				a := ef.Spec.ConfigPatches[i]
				b := ef.Spec.ConfigPatches[j]
				aVhost := a.Match.GetRouteConfiguration().GetVhost()
				bVhost := b.Match.GetRouteConfiguration().GetVhost()
				if aVhost.Name != bVhost.Name {
					return aVhost.Name < bVhost.Name
				}
				return aVhost.GetRoute().Name < bVhost.GetRoute().Name
			})
		}
		// For EnvoyFilter to LDS, we need to keep the original filter order

		efs[key] = ef.EnvoyFilter
	}

	return &FinalState{
		EnvoyFilters: efs,
	}, nil
}
