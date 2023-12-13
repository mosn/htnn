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

	"github.com/go-logr/logr"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"

	mosniov1 "mosn.io/moe/controller/api/v1"
)

type VirtualServicePolicies struct {
	VirtualService *istiov1b1.VirtualService
	RoutePolicies  map[string][]*HTTPFilterPolicyWrapper
}

// InitState is the beginning of our translation.
type InitState struct {
	VirtualServices map[types.NamespacedName]*VirtualServicePolicies
	VsToGateway     map[types.NamespacedName][]*istiov1b1.Gateway

	logger *logr.Logger
}

func NewInitState(logger *logr.Logger) *InitState {
	return &InitState{
		VirtualServices: make(map[types.NamespacedName]*VirtualServicePolicies),
		VsToGateway:     make(map[types.NamespacedName][]*istiov1b1.Gateway),
		logger:          logger,
	}
}

func (s *InitState) AddPolicyForVirtualService(policy *mosniov1.HTTPFilterPolicy, vs *istiov1b1.VirtualService, gw *istiov1b1.Gateway) {
	nn := types.NamespacedName{
		Namespace: vs.ObjectMeta.Namespace,
		Name:      vs.ObjectMeta.Name,
	}

	vsp, ok := s.VirtualServices[nn]
	if !ok {
		vsp = &VirtualServicePolicies{
			VirtualService: vs.DeepCopy(),
			RoutePolicies:  map[string][]*HTTPFilterPolicyWrapper{},
		}
		s.VirtualServices[nn] = vsp
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

func (s *InitState) Process(original_ctx context.Context) (*FinalState, error) {
	// Process chain:
	// InitState -> DataPlaneState -> MergedState -> FinalState
	ctx := &Ctx{
		Context: original_ctx,
		logger:  s.logger,
	}
	return toDataPlaneState(ctx, s)
}
