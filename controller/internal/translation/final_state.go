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
	"sort"
	"strings"

	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"

	"mosn.io/moe/controller/internal/istio"
	"mosn.io/moe/controller/internal/model"
)

const (
	AnnotationInfo = "htnn.mosn.io/info"
)

// We use the domain as the EnvoyFilter's name, so that:
// 1. We can easily find per domain rules.
// 2. Match the EnvoyFilter model which uses domain + routeName as the key.
// 3. Allow merging the same route configuration into virtual host level.
// There are also some drawbacks. For example, a domain shared by hundreds of VirtualServices will
// cause one big EnvoyFilter. Let's see if it's a problem.
func envoyFilterName(vhost *model.VirtualHost) string {
	// Strip the port number. We don't need to create two EnvoyFilters for :80 and :443.
	domain := strings.Split(vhost.Name, ":")[0]
	// We join the host & port in toDataPlaneState so the domain is not nil
	if strings.HasPrefix(domain, "*.") {
		// '*' is not allowed in EnvoyFilter name. And '.' can only be used after alphanumeric characters.
		// So we replace the '*.' with '-'.
		domain = "-" + domain[2:]
	}
	// The `htnn-h` means the HTNN's HTTPFilterPolicy.
	return fmt.Sprintf("htnn-h-%s", domain)
}

// finalState is the end of the translation. We convert the state to EnvoyFilter and write it to k8s.
type FinalState struct {
	EnvoyFilters map[string]*istiov1a3.EnvoyFilter
}

type envoyFilterWrapper struct {
	*istiov1a3.EnvoyFilter
	info *Info
}

func toFinalState(_ *Ctx, state *mergedState) (*FinalState, error) {
	efs := istio.DefaultEnvoyFilters()
	efList := []*envoyFilterWrapper{}

	for _, host := range state.Hosts {
		for routeName, route := range host.Routes {
			ef := istio.GenerateRouteFilter(host.VirtualHost, routeName, route.Config)
			name := envoyFilterName(host.VirtualHost)
			ef.SetName(name)

			efList = append(efList, &envoyFilterWrapper{
				EnvoyFilter: ef,
				info:        route.Info,
			})
		}
	}

	// Merge EnvoyFilters with same name. The number of EnvoyFilters is equal to the number of
	// configured domains.
	efws := map[string]*envoyFilterWrapper{}
	for _, ef := range efList {
		name := ef.GetName()
		if curr, ok := efws[name]; ok {
			curr.Spec.ConfigPatches = append(curr.Spec.ConfigPatches, ef.Spec.ConfigPatches...)
			curr.info.Merge(ef.info)
		} else {
			efws[name] = ef
		}
	}

	for name, ef := range efws {
		if ef.info != nil {
			ef.SetAnnotations(map[string]string{
				AnnotationInfo: ef.info.String(),
			})
		}
		// Sort here to avoid EnvoyFilter change caused by the order of ConfigPatch.
		// Each ConfigPatch should have a unique (vhost, routeName).
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
		efs[name] = ef.EnvoyFilter
	}

	return &FinalState{
		EnvoyFilters: efs,
	}, nil
}
