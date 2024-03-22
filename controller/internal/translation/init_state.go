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
	"k8s.io/apimachinery/pkg/types"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mosniov1 "mosn.io/htnn/controller/apis/v1"
)

type VirtualServicePolicies struct {
	VirtualService *istiov1b1.VirtualService
	RoutePolicies  map[string][]*HTTPFilterPolicyWrapper
	Gateways       []*istiov1b1.Gateway
}

type HTTPRoutePolicies struct {
	HTTPRoute     *gwapiv1.HTTPRoute
	RoutePolicies map[string][]*HTTPFilterPolicyWrapper
	Gateways      []*gwapiv1.Gateway
}

// InitState is the beginning of our translation.
type InitState struct {
	VirtualServicePolicies map[types.NamespacedName]*VirtualServicePolicies
	HTTPRoutePolicies      map[types.NamespacedName]*HTTPRoutePolicies

	logger *logr.Logger
}

func NewInitState(logger *logr.Logger) *InitState {
	return &InitState{
		VirtualServicePolicies: make(map[types.NamespacedName]*VirtualServicePolicies),
		HTTPRoutePolicies:      make(map[types.NamespacedName]*HTTPRoutePolicies),

		logger: logger,
	}
}

func (s *InitState) GetGatewaysWithVirtualService(vs *istiov1b1.VirtualService) []*istiov1b1.Gateway {
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

func (s *InitState) AddPolicyForVirtualService(policy *mosniov1.HTTPFilterPolicy, vs *istiov1b1.VirtualService, gws []*istiov1b1.Gateway) {
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

	if policy.Spec.TargetRef.SectionName == nil {
		for _, httpRoute := range vs.Spec.Http {
			routeName := httpRoute.Name
			vsp.RoutePolicies[routeName] = append(vsp.RoutePolicies[routeName], &HTTPFilterPolicyWrapper{
				HTTPFilterPolicy: policy,
				scope:            PolicyScopeRoute,
			})
		}
	} else {
		routeName := string(*policy.Spec.TargetRef.SectionName)
		vsp.RoutePolicies[routeName] = append(vsp.RoutePolicies[routeName], &HTTPFilterPolicyWrapper{
			HTTPFilterPolicy: policy,
			scope:            PolicyScopeRule,
		})
	}
}

func (s *InitState) GetGatewaysWithHTTPRoute(route *gwapiv1.HTTPRoute) []*gwapiv1.Gateway {
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

func (s *InitState) AddPolicyForHTTPRoute(policy *mosniov1.HTTPFilterPolicy, route *gwapiv1.HTTPRoute, gws []*gwapiv1.Gateway) {
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

func (s *InitState) Process(original_ctx context.Context) (*FinalState, error) {
	// Process chain:
	// InitState -> DataPlaneState -> MergedState -> FinalState
	ctx := &Ctx{
		Context: original_ctx,
		logger:  s.logger,
	}
	return toDataPlaneState(ctx, s)
}
