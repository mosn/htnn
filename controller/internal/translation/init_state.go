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

	"github.com/go-logr/logr"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	mosniov1 "mosn.io/moe/controller/api/v1"
)

type VirtualServicePolicies struct {
	VirtualService *istiov1b1.VirtualService
	RoutePolicies  map[string][]*HTTPFilterPolicyWrapper
}

type RoutePolicies struct {
	Route         *Route
	RoutePolicies map[string][]*HTTPFilterPolicyWrapper
}

type RouteKey struct {
	types.NamespacedName
	Kind string
}

// InitState is the beginning of our translation.
type InitState struct {
	VirtualServicePolicies map[types.NamespacedName]*VirtualServicePolicies
	VsToGateway            map[types.NamespacedName][]*istiov1b1.Gateway

	RoutePolicies  map[RouteKey]*RoutePolicies
	RouteToGateway map[RouteKey][]*gwapiv1.Gateway

	logger *logr.Logger
}

func NewInitState(logger *logr.Logger) *InitState {
	return &InitState{
		VirtualServicePolicies: make(map[types.NamespacedName]*VirtualServicePolicies),
		VsToGateway:            make(map[types.NamespacedName][]*istiov1b1.Gateway),

		RoutePolicies:  make(map[RouteKey]*RoutePolicies),
		RouteToGateway: make(map[RouteKey][]*gwapiv1.Gateway),

		logger: logger,
	}
}

func (s *InitState) AddPolicyForVirtualService(policy *mosniov1.HTTPFilterPolicy, vs *istiov1b1.VirtualService, gw *istiov1b1.Gateway) {
	nn := types.NamespacedName{
		Namespace: vs.Namespace,
		Name:      vs.Name,
	}

	vsp, ok := s.VirtualServicePolicies[nn]
	if !ok {
		vsp = &VirtualServicePolicies{
			VirtualService: vs.DeepCopy(),
			RoutePolicies:  map[string][]*HTTPFilterPolicyWrapper{},
		}
		s.VirtualServicePolicies[nn] = vsp
	}

	if policy.Spec.TargetRef.SectionName == nil {
		for _, httpRoute := range vs.Spec.Http {
			routeName := httpRoute.Name
			vsp.RoutePolicies[routeName] = append(vsp.RoutePolicies[routeName], &HTTPFilterPolicyWrapper{
				HTTPFilterPolicy: policy.DeepCopy(),
				scope:            PolicyScopeHost,
			})
		}
	} else {
		routeName := string(*policy.Spec.TargetRef.SectionName)
		vsp.RoutePolicies[routeName] = append(vsp.RoutePolicies[routeName], &HTTPFilterPolicyWrapper{
			HTTPFilterPolicy: policy.DeepCopy(),
			scope:            PolicyScopeRoute,
		})
	}

	gws, ok := s.VsToGateway[nn]
	if !ok {
		gws = make([]*istiov1b1.Gateway, 0)
	}
	s.VsToGateway[nn] = append(gws, gw.DeepCopy())
}

type Route struct {
	Namespace        string
	Name             string
	GroupVersionKind schema.GroupVersionKind
	Labels           map[string]string

	ParentRefs []gwapiv1.ParentReference
	Hostnames  []gwapiv1.Hostname

	SectionNames []string
}

func routeFromHTTPRoute(route *gwapiv1.HTTPRoute, sectionNames []string) *Route {
	hostnames := route.Spec.Hostnames
	if len(hostnames) == 0 {
		// This is how Istio handles empty Hostnames
		hostnames = []gwapiv1.Hostname{"*"}
	}
	// We don't use DeepCopy for the route wrapper. So far the fields here should all be read-only.
	return &Route{
		Namespace:        route.Namespace,
		GroupVersionKind: route.GroupVersionKind(),
		Labels:           route.Labels,
		ParentRefs:       route.Spec.ParentRefs,
		Hostnames:        hostnames,
		SectionNames:     sectionNames,
	}
}

func routeFromGRPCRoute(route *gwapiv1a2.GRPCRoute, sectionNames []string) *Route {
	hostnames := route.Spec.Hostnames
	if len(hostnames) == 0 {
		hostnames = []gwapiv1.Hostname{"*"}
	}
	return &Route{
		Namespace:        route.Namespace,
		GroupVersionKind: route.GroupVersionKind(),
		Labels:           route.Labels,
		ParentRefs:       route.Spec.ParentRefs,
		Hostnames:        hostnames,
		SectionNames:     sectionNames,
	}
}

func (s *InitState) addPolicyForRoute(policy *mosniov1.HTTPFilterPolicy, route *Route, gw *gwapiv1.Gateway) {
	nn := types.NamespacedName{
		Namespace: route.Namespace,
		Name:      route.Name,
	}
	key := RouteKey{
		NamespacedName: nn,
		Kind:           route.GroupVersionKind.Kind,
	}

	hp, ok := s.RoutePolicies[key]
	if !ok {
		hp = &RoutePolicies{
			Route:         route,
			RoutePolicies: map[string][]*HTTPFilterPolicyWrapper{},
		}
		s.RoutePolicies[key] = hp
	}

	for _, name := range route.SectionNames {
		hp.RoutePolicies[name] = append(hp.RoutePolicies[name], &HTTPFilterPolicyWrapper{
			HTTPFilterPolicy: policy.DeepCopy(),
			scope:            PolicyScopeHost,
		})
	}

	gws, ok := s.RouteToGateway[key]
	if !ok {
		gws = make([]*gwapiv1.Gateway, 0)
	}
	s.RouteToGateway[key] = append(gws, gw.DeepCopy())
}

func (s *InitState) AddPolicyForHTTPRoute(policy *mosniov1.HTTPFilterPolicy, route *gwapiv1.HTTPRoute, gw *gwapiv1.Gateway) {
	sectionNames := make([]string, len(route.Spec.Rules))
	for i := range route.Spec.Rules {
		name := fmt.Sprintf("%s.%s.%d", route.Namespace, route.Name, i)
		sectionNames[i] = name
	}
	r := routeFromHTTPRoute(route, sectionNames)
	s.addPolicyForRoute(policy, r, gw)
}

func (s *InitState) AddPolicyForGRPCRoute(policy *mosniov1.HTTPFilterPolicy, route *gwapiv1a2.GRPCRoute, gw *gwapiv1.Gateway) {
	sectionNames := make([]string, len(route.Spec.Rules))
	for i := range route.Spec.Rules {
		name := fmt.Sprintf("%s.%s.%d", route.Namespace, route.Name, i)
		sectionNames[i] = name
	}
	r := routeFromGRPCRoute(route, sectionNames)
	s.addPolicyForRoute(policy, r, gw)
}

func (s *InitState) Process(original_ctx context.Context) (*FinalState, error) {
	// Process chain:
	// InitState -> DataPlaneState -> MergedState -> FinalState
	ctx := &Ctx{
		Context: original_ctx,
		logger:  s.logger,
	}
	return toDataPlaneState(ctx, s)
}
