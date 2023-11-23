package ir

import (
	"fmt"

	istiov1b1 "istio.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"

	mosniov1 "mosn.io/moe/controller/api/v1"
)

type dataPlaneState struct {
	Hosts map[string]*hostPolicy
}

type hostPolicy struct {
	Routes map[string]*routePolicy
}

type routePolicy struct {
	Policies []*mosniov1.HTTPFilterPolicy
}

func genRouteId(id *types.NamespacedName, r *istiov1b1.HTTPRoute, order int) string {
	return id.String() + "_" + fmt.Sprintf("%d", order)
}

func toDataPlaneState(ctx Ctx, state *InitState) error {
	s := &dataPlaneState{
		Hosts: make(map[string]*hostPolicy),
	}
	for id, vsp := range state.VirtualServices {
		spec := &vsp.VirtualService.Spec
		routes := make(map[string]*routePolicy)
		for i, r := range spec.Http {
			routes[genRouteId(&id, r, i)] = &routePolicy{
				Policies: vsp.Policies,
			}
		}
		for _, hostName := range spec.Hosts {
			if host, ok := s.Hosts[hostName]; ok {
				for name, route := range routes {
					if _, ok := host.Routes[name]; !ok {
						host.Routes[name] = route
					}
				}
			} else {
				s.Hosts[hostName] = &hostPolicy{
					Routes: routes,
				}
			}
		}
	}

	return toMergedState(ctx, s)
}
