package translation

import (
	"errors"
	"fmt"
	"net"
	"strings"

	istioapi "istio.io/api/networking/v1beta1"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"

	mosniov1 "mosn.io/moe/controller/api/v1"
	"mosn.io/moe/controller/internal/model"
)

// dataPlaneState converts the init state to the structure used by the data plane
type dataPlaneState struct {
	Hosts map[string]*hostPolicy
}

type hostPolicy struct {
	VirtualHost *model.VirtualHost
	Routes      map[string]*routePolicy
	Policies    []*mosniov1.HTTPFilterPolicy
}

type routePolicy struct {
	Policies []*mosniov1.HTTPFilterPolicy
}

func genRouteId(id *types.NamespacedName, r *istioapi.HTTPRoute, order int) string {
	return id.String() + "_" + fmt.Sprintf("%d", order)
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

func toDataPlaneState(ctx *Ctx, state *InitState) error {
	s := &dataPlaneState{
		Hosts: make(map[string]*hostPolicy),
	}
	for id, vsp := range state.VirtualServices {
		gws := state.VsToGateway[id]
		spec := &vsp.VirtualService.Spec
		routes := make(map[string]*routePolicy)
		for i, r := range spec.Http {
			routes[genRouteId(&id, r, i)] = &routePolicy{
				Policies: vsp.Policies,
			}
		}
		for _, hostName := range spec.Hosts {
			vh := buildVirtualHost(hostName, gws)
			if vh == nil {
				err := errors.New("can not build virtual host")
				ctx.logger.Error(err, "failed to build virtual host", "hostname", hostName, "virtualservice", id, "gateways", gws)
				return err
			}
			vh.NsName = &id
			policy := &hostPolicy{
				VirtualHost: vh,
				Routes:      routes,
				// It is possible that mutiple VirtualServices share the same host but with different routes.
				// In this case, the host is considered a match condition but not a parent of routes.
				// So it is unreasonable to set host level policy to such VirtualServices. We don't
				// support this case (VirtualServices share same host & Host level policy attached) for now.
				// If people want to add policy to the route under the host, use route level policy instead.
				Policies: vsp.Policies,
			}
			s.Hosts[vh.Name] = policy
		}
	}

	return toMergedState(ctx, s)
}
