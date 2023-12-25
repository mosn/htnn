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

	mosniov1 "mosn.io/moe/controller/api/v1"
)

type VirtualServicePolicies struct {
	VirtualService *istiov1b1.VirtualService
	RoutePolicies  map[string][]*HTTPFilterPolicyWrapper
}

type HTTPRoutePolicies struct {
	HTTPRoute     *gwapiv1.HTTPRoute
	RoutePolicies map[string][]*HTTPFilterPolicyWrapper
}

// InitState is the beginning of our translation.
type InitState struct {
	VirtualServicePolicies map[types.NamespacedName]*VirtualServicePolicies
	VsToGateway            map[types.NamespacedName][]*istiov1b1.Gateway

	HTTPRoutePolicies map[types.NamespacedName]*HTTPRoutePolicies
	HrToGateway       map[types.NamespacedName][]*gwapiv1.Gateway

	logger *logr.Logger
}

func NewInitState(logger *logr.Logger) *InitState {
	return &InitState{
		VirtualServicePolicies: make(map[types.NamespacedName]*VirtualServicePolicies),
		VsToGateway:            make(map[types.NamespacedName][]*istiov1b1.Gateway),

		HTTPRoutePolicies: make(map[types.NamespacedName]*HTTPRoutePolicies),
		HrToGateway:       make(map[types.NamespacedName][]*gwapiv1.Gateway),

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

func (s *InitState) AddPolicyForHTTPRoute(policy *mosniov1.HTTPFilterPolicy, route *gwapiv1.HTTPRoute, gw *gwapiv1.Gateway) {
	nn := types.NamespacedName{
		Namespace: route.Namespace,
		Name:      route.Name,
	}

	hp, ok := s.HTTPRoutePolicies[nn]
	if !ok {
		route := route.DeepCopy()
		if len(route.Spec.Hostnames) == 0 {
			// This is how Istio handles empty Hostnames
			route.Spec.Hostnames = []gwapiv1.Hostname{"*"}
		}

		hp = &HTTPRoutePolicies{
			HTTPRoute:     route,
			RoutePolicies: map[string][]*HTTPFilterPolicyWrapper{},
		}
		s.HTTPRoutePolicies[nn] = hp
	}

	for i := range route.Spec.Rules {
		name := fmt.Sprintf("%s.%s.%d", route.Namespace, route.Name, i)
		hp.RoutePolicies[name] = append(hp.RoutePolicies[name], &HTTPFilterPolicyWrapper{
			HTTPFilterPolicy: policy.DeepCopy(),
			scope:            PolicyScopeHost,
		})
	}

	gws, ok := s.HrToGateway[nn]
	if !ok {
		gws = make([]*gwapiv1.Gateway, 0)
	}
	s.HrToGateway[nn] = append(gws, gw.DeepCopy())
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
