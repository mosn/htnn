package translation

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slices"

	mosniov1 "mosn.io/moe/controller/api/v1"
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
	PolicyScopeRoute PolicyScope = iota
	PolicyScopeHost
)

type HTTPFilterPolicyWrapper struct {
	*mosniov1.HTTPFilterPolicy

	scope PolicyScope
}
