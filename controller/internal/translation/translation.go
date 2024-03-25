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
	"slices"
	"sort"

	"github.com/go-logr/logr"

	mosniov1 "mosn.io/htnn/controller/apis/v1"
)

type Ctx struct {
	context.Context

	logger *logr.Logger
}

func (c *Ctx) Logger() *logr.Logger {
	return c.logger
}

type Info struct {
	// HTTPFilterPolicies indicates what HTTPFilterPolicies are used to generated the EnvoyFilter.
	HTTPFilterPolicies []string `json:"httpfilterpolicies"`
}

func (info *Info) String() string {
	b, _ := json.Marshal(info)
	return string(b)
}

func (info *Info) Merge(other *Info) {
	for _, policy := range other.HTTPFilterPolicies {
		n := len(info.HTTPFilterPolicies)
		index := sort.Search(n, func(i int) bool { return info.HTTPFilterPolicies[i] >= policy })
		if index < n && info.HTTPFilterPolicies[index] == policy {
			continue
		}
		info.HTTPFilterPolicies = slices.Insert(info.HTTPFilterPolicies, index, policy)
	}
}

type PolicyScope int

const (
	// sort from small to large
	PolicyScopeRule  PolicyScope = iota // a route in a VirtualService or a rule in xRoute
	PolicyScopeRoute                    // a VirtualService or a xRoute
)

type HTTPFilterPolicyWrapper struct {
	*mosniov1.HTTPFilterPolicy

	scope PolicyScope
}
