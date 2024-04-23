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
	"strings"

	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/internal/model"
)

// dataPlaneState converts the init state to the structure used by the data plane
type dataPlaneState struct {
	Hosts map[Proxy]map[string]*hostPolicy
}

type hostPolicy struct {
	VirtualHost *model.VirtualHost
	Routes      map[string]*routePolicy
}

type routePolicy struct {
	NsName   *types.NamespacedName
	Policies []*HTTPFilterPolicyWrapper
}

func isWildCarded(s string) bool {
	return len(s) > 0 && s[0] == '*'
}

func hostMatch(gwHost string, host string) bool {
	gwc := isWildCarded(gwHost)
	hwc := isWildCarded(host)
	if gwc {
		if hwc {
			if len(gwHost) < len(host) {
				return strings.HasSuffix(host[1:], gwHost[1:])
			}
			return strings.HasSuffix(gwHost[1:], host[1:])
		}
		return strings.HasSuffix(host, gwHost[1:])
	}

	if hwc {
		return strings.HasSuffix(gwHost, host[1:])
	}

	return gwHost == host
}

func buildVirtualHostsWithIstioGw(host string, nsName *types.NamespacedName, gws []*istiov1a3.Gateway) []*model.VirtualHost {
	vhs := make([]*model.VirtualHost, 0)
	for _, gw := range gws {
		for _, svr := range gw.Spec.Servers {
			port := svr.Port.Number
			for _, h := range svr.Hosts {
				if hostMatch(h, host) {
					name := net.JoinHostPort(host, fmt.Sprintf("%d", port))
					vhs = append(vhs, &model.VirtualHost{
						Gateway: &model.Gateway{
							NsName: &types.NamespacedName{
								Namespace: gw.Namespace,
								Name:      gw.Name,
							},
							Port: port,
						},
						NsName: nsName,
						Name:   name,
					})
				}
			}
		}
	}
	return vhs
}

func buildVirtualHostsWithK8sGw(host string, ls *gwapiv1.Listener, nsName, gwNsName *types.NamespacedName) []*model.VirtualHost {
	vhs := make([]*model.VirtualHost, 0)
	if ls.Protocol != gwapiv1.HTTPProtocolType && ls.Protocol != gwapiv1.HTTPSProtocolType {
		return vhs
	}
	if ls.Hostname == nil || hostMatch(string(*ls.Hostname), host) {
		if host == "*" && ls.Hostname != nil {
			host = string(*ls.Hostname)
		}
		name := net.JoinHostPort(host, fmt.Sprintf("%d", ls.Port))
		vhs = append(vhs, &model.VirtualHost{
			Gateway: &model.Gateway{
				NsName: gwNsName,
				Port:   uint32(ls.Port),
			},
			NsName: nsName,
			Name:   name,
		})
	}
	return vhs
}

func AllowRoute(cond *gwapiv1.AllowedRoutes, route *gwapiv1b1.HTTPRoute, gwNsName *types.NamespacedName) bool {
	if cond == nil {
		return true
	}

	matched := len(cond.Kinds) == 0
	for _, kind := range cond.Kinds {
		if kind.Group != nil && string(*kind.Group) != route.GroupVersionKind().Group {
			continue
		}
		if string(kind.Kind) != route.GroupVersionKind().Kind {
			continue
		}

		matched = true
		break
	}
	if !matched {
		return false
	}

	if cond.Namespaces != nil {
		nsCond := cond.Namespaces
		from := gwapiv1.NamespacesFromSelector
		if nsCond.From != nil {
			from = *nsCond.From
			if from == gwapiv1.NamespacesFromSame && gwNsName.Namespace != route.Namespace {
				return false
			}
		}
		if from == gwapiv1.NamespacesFromSelector && nsCond.Selector != nil {
			sel, err := metav1.LabelSelectorAsSelector(nsCond.Selector)
			if err != nil {
				log.Errorf("failed to convert selector, err: %v, selector: %v", err, nsCond.Selector)
				return false
			}
			if !sel.Matches(labels.Set(route.Labels)) {
				return false
			}
		}
	}
	return true
}

var (
	wildcardHostnams = []gwapiv1.Hostname{"*"}
)

func addVirtualHostToProxy(vh *model.VirtualHost, hosts map[Proxy]map[string]*hostPolicy, routes map[string]*routePolicy) {
	p := Proxy{
		Namespace: vh.Gateway.NsName.Namespace,
	}

	proxy, ok := hosts[p]
	if !ok {
		hosts[p] = make(map[string]*hostPolicy)
		proxy = hosts[p]
	}

	if host, ok := proxy[vh.Name]; ok {
		// TODO: add route name collision detection
		// Currently, it is the webhook or the user configuration to guarantee the same route
		// name won't be used in different VirtualServices that share the same host.
		// For HTTPRoute, Istio guarantees the default route name is unique
		for routeName, policy := range routes {
			host.Routes[routeName] = policy
		}
	} else {
		policy := &hostPolicy{
			VirtualHost: vh,
			Routes:      routes,
		}
		proxy[vh.Name] = policy
	}
}

func toDataPlaneState(ctx *Ctx, state *InitState) (*FinalState, error) {
	s := &dataPlaneState{
		Hosts: make(map[Proxy]map[string]*hostPolicy),
	}
	for id, vsp := range state.VirtualServicePolicies {
		id := id // the copied id will be referenced by address later
		gws := vsp.Gateways
		spec := &vsp.VirtualService.Spec
		routeNsName := &types.NamespacedName{
			Namespace: vsp.VirtualService.Namespace,
			Name:      vsp.VirtualService.Name,
		}
		routes := make(map[string]*routePolicy)
		for name, policies := range vsp.RoutePolicies {
			routes[name] = &routePolicy{
				Policies: policies,
				NsName:   &id,
			}
		}
		for _, hostName := range spec.Hosts {
			vhs := buildVirtualHostsWithIstioGw(hostName, routeNsName, gws)
			if len(vhs) == 0 {
				// maybe a host from an unsupported gateway which is referenced as one of the Hosts
				log.Infof("virtual host not found, skipped, hostname: %s, VirtualService: %s, gateways: %v", hostName,
					id, gws)
				continue
			}
			for _, vh := range vhs {
				addVirtualHostToProxy(vh, s.Hosts, routes)
			}
		}
	}

	for id, route := range state.HTTPRoutePolicies {
		id := id // the copied id will be referenced by address later
		gws := route.Gateways
		spec := &route.HTTPRoute.Spec
		routeNsName := &types.NamespacedName{
			Namespace: route.HTTPRoute.Namespace,
			Name:      route.HTTPRoute.Name,
		}
		routes := make(map[string]*routePolicy)
		for name, policies := range route.RoutePolicies {
			routes[name] = &routePolicy{
				Policies: policies,
				NsName:   &id,
			}
		}
		for _, gw := range gws {
			gwNsName := &types.NamespacedName{
				Namespace: gw.Namespace,
				Name:      gw.Name,
			}
			for _, ls := range gw.Spec.Listeners {
				ls := ls
				matched := false
				for _, ref := range route.HTTPRoute.Spec.ParentRefs {
					if ref.Port != nil && *ref.Port != ls.Port {
						continue
					}
					if ref.SectionName != nil && *ref.SectionName != ls.Name {
						continue
					}
					matched = true
				}
				if !matched {
					continue
				}

				if !AllowRoute(ls.AllowedRoutes, route.HTTPRoute, gwNsName) {
					continue
				}

				hostnames := spec.Hostnames
				if len(hostnames) == 0 {
					// This is how Istio handles empty Hostnames
					hostnames = wildcardHostnams
				}
				for _, hostName := range hostnames {
					vhs := buildVirtualHostsWithK8sGw(string(hostName), &ls, routeNsName, gwNsName)
					if len(vhs) == 0 {
						// It's acceptable to have an unmatched hostname, which is already
						// reported in the HTTPRoute's status
						continue
					}
					for _, vh := range vhs {
						addVirtualHostToProxy(vh, s.Hosts, routes)
					}
				}
			}
		}
	}

	return toMergedState(ctx, s)
}
