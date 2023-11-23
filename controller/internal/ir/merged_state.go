package ir

import (
	mosniov1 "mosn.io/moe/controller/api/v1"
)

type mergedState struct {
	Hosts map[string]*hostPolicy
}

func toMergedState(ctx Ctx, state *dataPlaneState) error {
	s := &mergedState{
		Hosts: state.Hosts,
	}
	for _, host := range s.Hosts {
		for _, route := range host.Routes {
			// TODO: implement merge policy
			// According to the https://gateway-api.sigs.k8s.io/geps/gep-713/,
			// 1. A Policy targeting a more specific scope wins over a policy targeting a lesser specific scope.
			// 2. If multiple polices configure the same plugin, the oldest one (based on creation timestamp) wins.
			// 3. If there are multiple oldest polices, the one appearing first in alphabetical order by {namespace}/{name} wins.
			route.Policies = []*mosniov1.HTTPFilterPolicy{route.Policies[0]}
		}
	}

	return toFinalState(ctx, s)
}
