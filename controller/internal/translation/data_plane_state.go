package translation

import (
	"fmt"
	"net"
	"strings"

	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"

	"mosn.io/moe/controller/internal/model"
)

// dataPlaneState converts the init state to the structure used by the data plane
type dataPlaneState struct {
	Hosts map[string]*hostPolicy
}

type hostPolicy struct {
	VirtualHost *model.VirtualHost
	Routes      map[string]*routePolicy
}

type routePolicy struct {
	NsName   *types.NamespacedName
	Policies []*HTTPFilterPolicyWrapper
}

func hostMatch(gwHost string, host string) bool {
	if gwHost == host {
		return true
	}
	if strings.HasPrefix(gwHost, "*") {
		return strings.HasSuffix(host, gwHost[1:])
	}
	return false
}

func buildVirtualHost(host string, gws []*istiov1b1.Gateway) *model.VirtualHost {
	for _, gw := range gws {
		for _, svr := range gw.Spec.Servers {
			port := svr.Port.Number
			for _, h := range svr.Hosts {
				if hostMatch(h, host) {
					name := net.JoinHostPort(host, fmt.Sprintf("%d", port))
					return &model.VirtualHost{
						Gateway: &model.Gateway{
							NsName: &types.NamespacedName{
								Namespace: gw.Namespace,
								Name:      gw.Name,
							},
							Port: port,
						},
						Name: name,
					}
				}
			}
		}
	}
	return nil
}

func toDataPlaneState(ctx *Ctx, state *InitState) (*FinalState, error) {
	s := &dataPlaneState{
		Hosts: make(map[string]*hostPolicy),
	}
	for id, vsp := range state.VirtualServices {
		id := id
		gws := state.VsToGateway[id]
		spec := &vsp.VirtualService.Spec
		routes := make(map[string]*routePolicy)
		for name, policies := range vsp.RoutePolicies {
			routes[name] = &routePolicy{
				Policies: policies,
				NsName:   &id,
			}
		}
		for _, hostName := range spec.Hosts {
			vh := buildVirtualHost(hostName, gws)
			if vh == nil {
				// maybe a host from an unsupported gateway which is referenced as one of the Hosts
				ctx.logger.Info("virtual host not found, skipped", "hostname", hostName,
					"virtualservice", id, "gateways", gws)
				continue
			}
			if host, ok := s.Hosts[vh.Name]; ok {
				// TODO: add route name collision detection
				// Currently, it is the webhook or the user configuration to guarantee the same route
				// name won't be used in different VirtualServices that share the same host.
				for routeName, policy := range routes {
					host.Routes[routeName] = policy
				}
			} else {
				policy := &hostPolicy{
					VirtualHost: vh,
					Routes:      routes,
				}
				s.Hosts[vh.Name] = policy
			}
		}
	}

	return toMergedState(ctx, s)
}
