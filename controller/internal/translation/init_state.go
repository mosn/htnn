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
	"fmt"

	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/types"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/internal/model"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

type VirtualServicePolicies struct {
	VirtualService *istiov1a3.VirtualService
	RoutePolicies  map[string][]*HTTPFilterPolicyWrapper
	Gateways       []*istiov1a3.Gateway
}

type HTTPRoutePolicies struct {
	HTTPRoute     *gwapiv1b1.HTTPRoute
	RoutePolicies map[string][]*HTTPFilterPolicyWrapper
	Gateways      []*gwapiv1b1.Gateway
}

type ServerPort struct {
	Bind     string
	Number   uint32
	Protocol string
}

type GatewayPolicies struct {
	Gateway      *model.Gateway
	Ports        map[ServerPort]struct{}
	PortPolicies map[ServerPort][]*HTTPFilterPolicyWrapper
}

type ServerPortKey struct {
	Namespace string
	ServerPort
}

// InitState is the beginning of our translation.
type InitState struct {
	VirtualServicePolicies map[types.NamespacedName]*VirtualServicePolicies
	HTTPRoutePolicies      map[types.NamespacedName]*HTTPRoutePolicies

	GatewayPolicies     map[types.NamespacedName]*GatewayPolicies
	ServerPortToGateway map[ServerPortKey]*model.Gateway
}

func NewInitState() *InitState {
	return &InitState{
		VirtualServicePolicies: make(map[types.NamespacedName]*VirtualServicePolicies),
		HTTPRoutePolicies:      make(map[types.NamespacedName]*HTTPRoutePolicies),

		GatewayPolicies:     make(map[types.NamespacedName]*GatewayPolicies),
		ServerPortToGateway: make(map[ServerPortKey]*model.Gateway),
	}
}

func (s *InitState) GetGatewaysWithVirtualService(vs *istiov1a3.VirtualService) []*istiov1a3.Gateway {
	nn := types.NamespacedName{
		Namespace: vs.Namespace,
		Name:      vs.Name,
	}

	vsp, ok := s.VirtualServicePolicies[nn]
	if !ok {
		return nil
	}
	return vsp.Gateways
}

func (s *InitState) AddPolicyForVirtualService(policy *mosniov1.HTTPFilterPolicy, vs *istiov1a3.VirtualService, gws []*istiov1a3.Gateway) {
	nn := types.NamespacedName{
		Namespace: vs.Namespace,
		Name:      vs.Name,
	}

	vsp, ok := s.VirtualServicePolicies[nn]
	if !ok {
		vsp = &VirtualServicePolicies{
			VirtualService: vs,
			RoutePolicies:  map[string][]*HTTPFilterPolicyWrapper{},
			Gateways:       gws,
		}
		s.VirtualServicePolicies[nn] = vsp
	}

	targetRef := policy.Spec.TargetRef
	if targetRef == nil || targetRef.SectionName == nil {
		for _, httpRoute := range vs.Spec.Http {
			routeName := httpRoute.Name
			vsp.RoutePolicies[routeName] = append(vsp.RoutePolicies[routeName], &HTTPFilterPolicyWrapper{
				HTTPFilterPolicy: policy,
				scope:            PolicyScopeRoute,
			})
		}

		if len(policy.Spec.SubPolicies) > 0 {
			// Some of our cases have over hundreds of sub-policies, so we need to optimize this.
			subPolicies := make(map[string]*mosniov1.HTTPFilterPolicy, len(policy.Spec.SubPolicies))
			for _, subPolicy := range policy.Spec.SubPolicies {
				p := &mosniov1.HTTPFilterPolicy{}
				*p = *policy
				p.Spec = mosniov1.HTTPFilterPolicySpec{
					Filters: subPolicy.Filters,
				}
				subPolicies[string(subPolicy.SectionName)] = p
			}

			for _, httpRoute := range vs.Spec.Http {
				routeName := httpRoute.Name
				if subPolicy, ok := subPolicies[routeName]; ok {
					vsp.RoutePolicies[routeName] = append(vsp.RoutePolicies[routeName], &HTTPFilterPolicyWrapper{
						HTTPFilterPolicy: subPolicy,
						scope:            PolicyScopeRule,
					})
				}
			}
		}

	} else {
		routeName := string(*policy.Spec.TargetRef.SectionName)
		vsp.RoutePolicies[routeName] = append(vsp.RoutePolicies[routeName], &HTTPFilterPolicyWrapper{
			HTTPFilterPolicy: policy,
			scope:            PolicyScopeRule,
		})
	}
}

func (s *InitState) GetGatewaysWithHTTPRoute(route *gwapiv1b1.HTTPRoute) []*gwapiv1b1.Gateway {
	nn := types.NamespacedName{
		Namespace: route.Namespace,
		Name:      route.Name,
	}

	hp, ok := s.HTTPRoutePolicies[nn]
	if !ok {
		return nil
	}

	return hp.Gateways
}

func (s *InitState) AddPolicyForHTTPRoute(policy *mosniov1.HTTPFilterPolicy, route *gwapiv1b1.HTTPRoute, gws []*gwapiv1b1.Gateway) {
	nn := types.NamespacedName{
		Namespace: route.Namespace,
		Name:      route.Name,
	}

	hp, ok := s.HTTPRoutePolicies[nn]
	if !ok {
		hp = &HTTPRoutePolicies{
			HTTPRoute:     route,
			RoutePolicies: map[string][]*HTTPFilterPolicyWrapper{},
			Gateways:      gws,
		}
		s.HTTPRoutePolicies[nn] = hp
	}

	for i := range route.Spec.Rules {
		name := fmt.Sprintf("%s.%s.%d", route.Namespace, route.Name, i)
		hp.RoutePolicies[name] = append(hp.RoutePolicies[name], &HTTPFilterPolicyWrapper{
			HTTPFilterPolicy: policy,
			scope:            PolicyScopeRoute,
		})
	}
}

func (s *InitState) AddIstioGateway(gw *istiov1a3.Gateway) {
	s.AddPolicyForIstioGateway(nil, gw)
}

func (s *InitState) AddPolicyForIstioGateway(policy *mosniov1.HTTPFilterPolicy, gw *istiov1a3.Gateway) {
	nn := types.NamespacedName{
		Namespace: gw.Namespace,
		Name:      gw.Name,
	}

	gwp, ok := s.GatewayPolicies[nn]
	if !ok {
		gwp = &GatewayPolicies{
			Gateway: &model.Gateway{
				NsName: &types.NamespacedName{
					Namespace: gw.Namespace,
					Name:      gw.Name,
				},
			},
			Ports:        map[ServerPort]struct{}{},
			PortPolicies: map[ServerPort][]*HTTPFilterPolicyWrapper{},
		}
		s.GatewayPolicies[nn] = gwp
	}

	for _, svr := range gw.Spec.Servers {
		proto := mosniov1.NormalizeIstioProtocol(svr.Port.Protocol)
		if proto != "HTTP" && proto != "HTTPS" {
			continue
		}

		port := ServerPort{
			Bind:     svr.Bind,
			Number:   svr.Port.Number,
			Protocol: proto,
		}

		s.addPolicyForGateway(policy, gwp, port)
	}
}

func (s *InitState) AddK8sGateway(gw *gwapiv1b1.Gateway) {
	s.AddPolicyForK8sGateway(nil, gw)
}

func (s *InitState) AddPolicyForK8sGateway(policy *mosniov1.HTTPFilterPolicy, gw *gwapiv1b1.Gateway) {
	nn := types.NamespacedName{
		Namespace: gw.Namespace,
		Name:      gw.Name,
	}

	gwp, ok := s.GatewayPolicies[nn]
	if !ok {
		gwp = &GatewayPolicies{
			Gateway: &model.Gateway{
				NsName: &types.NamespacedName{
					Namespace: gw.Namespace,
					Name:      gw.Name,
				},
			},
			Ports:        map[ServerPort]struct{}{},
			PortPolicies: map[ServerPort][]*HTTPFilterPolicyWrapper{},
		}
		s.GatewayPolicies[nn] = gwp
	}

	for _, ls := range gw.Spec.Listeners {
		proto := mosniov1.NormalizeK8sGatewayProtocol(ls.Protocol)
		if proto != "HTTP" && proto != "HTTPS" {
			continue
		}

		port := ServerPort{
			Number:   uint32(ls.Port),
			Protocol: proto,
		}
		s.addPolicyForGateway(policy, gwp, port)
	}
}

func (s *InitState) addPolicyForGateway(policy *mosniov1.HTTPFilterPolicy, gwp *GatewayPolicies, port ServerPort) {
	// If two Gateways have the same port + protocol, like TLS with different hostnames,
	// skip the second one.
	k := ServerPortKey{Namespace: gwp.Gateway.NsName.Namespace, ServerPort: port}
	prevGw := s.ServerPortToGateway[k]
	// Do we need to support cases that people mix use Istio and K8s Gateway?
	if prevGw != nil && prevGw.NsName != gwp.Gateway.NsName {
		if policy != nil {
			log.Errorf("Gateway %s has the same server port %+v with gateway %s, ignore the policies target it."+
				" Maybe we can support Gateway level policy via RDS in the future?",
				gwp.Gateway.NsName, port, prevGw.NsName)
		}
		return
	}

	s.ServerPortToGateway[k] = gwp.Gateway

	gwp.Ports[port] = struct{}{}
	if policy == nil {
		return
	}

	gwp.PortPolicies[port] = append(gwp.PortPolicies[port], &HTTPFilterPolicyWrapper{
		HTTPFilterPolicy: policy,
		scope:            PolicyScopeGateway,
	})
}

func (s *InitState) Process(original_ctx context.Context) (*FinalState, error) {
	// Process chain:
	// InitState -> DataPlaneState -> MergedState -> FinalState
	ctx := &Ctx{
		Context: original_ctx,
	}

	return toDataPlaneState(ctx, s)
}
