/*
Copyright 2023 The HTNN Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mosniov1 "mosn.io/moe/controller/api/v1"
	"mosn.io/moe/controller/internal/ir"
)

// HTTPFilterPolicyReconciler reconciles a HTTPFilterPolicy object
type HTTPFilterPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=mosn.io,resources=httpfilterpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mosn.io,resources=httpfilterpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mosn.io,resources=httpfilterpolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.istio.io,resources=virtualservice,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *HTTPFilterPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// the controller is run with MaxConcurrentReconciles == 1, so we don't need to worry about concurrent access.
	logger := log.FromContext(ctx)
	logger.Info("reconcile") // req message is contained in the logger ctx

	// For current implementation, let's rebuild the state each time to avoid complexity.
	// The controller will use local cache when doing read operation.
	var policies mosniov1.HTTPFilterPolicyList
	if err := r.List(ctx, &policies, client.InNamespace(req.Namespace)); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list HTTPFilterPolicy: %v", err)
	}

	state := ir.NewInitState(&logger)

	for _, policy := range policies.Items {
		err := validateHTTPFilterPolicy(&policy)
		if err != nil {
			// TODO: mark the policy as invalid, and skip logging
			logger.Error(err, "invalid HTTPFilterPolicy", "name", policy.Name, "namespace", policy.Namespace)
			continue
		}

		ref := policy.Spec.TargetRef
		if ref.Group == "networking.istio.io" && ref.Kind == "VirtualService" {
			var virtualService istiov1b1.VirtualService
			err := r.Get(ctx, types.NamespacedName{Name: string(ref.Name), Namespace: req.Namespace}, &virtualService)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
				continue
			}

			err = validateVirtualService(&virtualService)
			if err != nil {
				logger.Error(err, "invalid VirtualService", "name", virtualService.Name, "namespace", virtualService.Namespace)
				continue
			}

			state.AddPolicyForVirtualService(&policy, &virtualService)
		}
	}

	err := state.Process(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// CustomerResourceIndexer indexes the additional customer resource
// according to the reconciled customer resource
type CustomerResourceIndexer interface {
	CustomerResource() client.Object
	IndexName() string
	Index(rawObj client.Object) []string
	FindAffectedObjects(ctx context.Context, obj client.Object) []reconcile.Request
}

type VirtualServiceIndexer struct {
	r client.Reader
}

func (v *VirtualServiceIndexer) CustomerResource() client.Object {
	return &istiov1b1.VirtualService{}
}

func (v *VirtualServiceIndexer) IndexName() string {
	return "spec.targetRef.kind.virtualService"
}

func (v *VirtualServiceIndexer) Index(rawObj client.Object) []string {
	po := rawObj.(*mosniov1.HTTPFilterPolicy)
	if po.Spec.TargetRef.Group != "networking.istio.io" || po.Spec.TargetRef.Kind != "VirtualService" {
		return []string{}
	}
	return []string{string(po.Spec.TargetRef.Name)}
}

func (v *VirtualServiceIndexer) FindAffectedObjects(ctx context.Context, obj client.Object) []reconcile.Request {
	return findAffectedObjects(ctx, v.r, obj, "VirtualService", v.IndexName())
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPFilterPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctx := context.Background()
	indexers := []CustomerResourceIndexer{
		&VirtualServiceIndexer{
			r: r,
		},
	}
	// IndexField is called per HTTPFilterPolicy
	for _, idxer := range indexers {
		if err := mgr.GetFieldIndexer().IndexField(ctx, &mosniov1.HTTPFilterPolicy{}, idxer.IndexName(), idxer.Index); err != nil {
			return err
		}
	}

	controller := ctrl.NewControllerManagedBy(mgr).
		For(&mosniov1.HTTPFilterPolicy{})

	for _, idxer := range indexers {
		controller.Watches(
			idxer.CustomerResource(),
			handler.EnqueueRequestsFromMapFunc(idxer.FindAffectedObjects),
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
			),
		)
	}

	return controller.Complete(r)
}

func findAffectedObjects(ctx context.Context, reader client.Reader, obj client.Object, name string, idx string) []reconcile.Request {
	logger := log.FromContext(ctx)
	logger.Info("Target changed", "name", name, "namespace", obj.GetNamespace(), "name", obj.GetName())

	policies := &mosniov1.HTTPFilterPolicyList{}
	listOps := &client.ListOptions{
		// Use the built index
		FieldSelector: fields.OneTermEqualSelector(idx, obj.GetName()),
		Namespace:     obj.GetNamespace(),
	}
	err := reader.List(ctx, policies, listOps)
	if err != nil {
		logger.Error(err, "failed to list HTTPFilterPolicy")
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(policies.Items))
	for i, item := range policies.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}

	logger.Info("Target changed, trigger reconciliation", "name", name, "requests", requests)
	return requests
}
