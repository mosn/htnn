package ir

import (
	"context"

	"github.com/go-logr/logr"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"

	mosniov1 "mosn.io/moe/controller/api/v1"
)

type VirtualServicePolicies struct {
	VirtualService *istiov1b1.VirtualService
	Policies       []*mosniov1.HTTPFilterPolicy
}

type InitState struct {
	VirtualServices map[types.NamespacedName]*VirtualServicePolicies

	logger *logr.Logger
}

func NewInitState(logger *logr.Logger) *InitState {
	return &InitState{
		VirtualServices: make(map[types.NamespacedName]*VirtualServicePolicies),
		logger:          logger,
	}
}

func (s *InitState) AddPolicyForVirtualService(policy *mosniov1.HTTPFilterPolicy, vs *istiov1b1.VirtualService) {
	nn := types.NamespacedName{
		Namespace: vs.ObjectMeta.Namespace,
		Name:      vs.ObjectMeta.Name,
	}

	vsp, ok := s.VirtualServices[nn]
	if !ok {
		vsp = &VirtualServicePolicies{
			VirtualService: vs.DeepCopy(),
			Policies:       make([]*mosniov1.HTTPFilterPolicy, 0),
		}
		s.VirtualServices[nn] = vsp
	}

	vsp.Policies = append(vsp.Policies, policy.DeepCopy())
}

func (s *InitState) Process(original_ctx context.Context) error {
	// Process chain:
	// InitState -> DataPlaneState -> MergedState -> FinalState
	ctx := &Ctx{
		Context: original_ctx,
		logger:  s.logger,
	}
	err := toDataPlaneState(ctx, s)
	if err != nil {
		if _, ok := err.(*retryableError); ok {
			return err
		}
		s.logger.Error(err, "failed to process state")
		// TODO: report the status to the original policy
	}

	return nil
}
