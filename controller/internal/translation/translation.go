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
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	mosniov1 "mosn.io/htnn/types/apis/v1"
)

type Ctx struct {
	context.Context
}

type Info struct {
	// FilterPolicies indicates what FilterPolicies are used to generated the EnvoyFilter.
	FilterPolicies []string `json:"filterpolicies"`
}

func (info *Info) String() string {
	b, _ := json.Marshal(info)
	return string(b)
}

func (info *Info) Merge(other *Info) {
	for _, policy := range other.FilterPolicies {
		n := len(info.FilterPolicies)
		index := sort.Search(n, func(i int) bool { return info.FilterPolicies[i] >= policy })
		if index < n && info.FilterPolicies[index] == policy {
			continue
		}
		info.FilterPolicies = slices.Insert(info.FilterPolicies, index, policy)
	}
}

type PolicyScope int

const (
	// sort from small to large
	PolicyScopeRule    PolicyScope = iota // a route in a VirtualService or a rule in xRoute
	PolicyScopeRoute                      // a VirtualService or a xRoute
	PolicyScopePort                       // a port in a Gateway
	PolicyScopeGateway                    // a Istio/k8s Gateway
)

type FilterPolicyWrapper struct {
	*mosniov1.FilterPolicy

	scope PolicyScope
}

type Proxy struct {
	Namespace string
}

func getECDSResourceName(workloadNamespace string, ldsName string) string {
	return fmt.Sprintf("htnn-%s-%s", workloadNamespace, ldsName)
}
