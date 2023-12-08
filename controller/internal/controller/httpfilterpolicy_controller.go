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
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"google.golang.org/protobuf/proto"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
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
	"mosn.io/moe/controller/internal/config"
	"mosn.io/moe/controller/internal/k8s"
	"mosn.io/moe/controller/internal/translation"
)

const (
	LabelCreatedBy = "htnn.mosn.io/created-by"
)

// HTTPFilterPolicyReconciler reconciles a HTTPFilterPolicy object
type HTTPFilterPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	istioGatewayIndexer *IstioGatewayIndexer
}

//+kubebuilder:rbac:groups=mosn.io,resources=httpfilterpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mosn.io,resources=httpfilterpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mosn.io,resources=httpfilterpolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.istio.io,resources=gateways,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.istio.io,resources=envoyfilters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.istio.io,resources=envoyfilters/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *HTTPFilterPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// the controller is run with MaxConcurrentReconciles == 1, so we don't need to worry about concurrent access.
	logger := log.FromContext(ctx)
	logger.Info("reconcile") // req message is contained in the logger ctx

	initState, err := r.policyToTranslationState(ctx, &logger, req)
	if err != nil {
		return ctrl.Result{}, err
	}
	if initState == nil {
		return ctrl.Result{}, nil
	}

	finalState, err := initState.Process(ctx)
	if err != nil {
		logger.Error(err, "failed to process state")
		// there is no retryable err during processing
		return ctrl.Result{}, nil
	}

	err = r.translationStateToCustomResource(ctx, &logger, finalState)
	return ctrl.Result{}, err
}

func (r *HTTPFilterPolicyReconciler) policyToTranslationState(ctx context.Context, logger *logr.Logger,
	req ctrl.Request) (*translation.InitState, error) {

	// For current implementation, let's rebuild the state each time to avoid complexity.
	// The controller will use local cache when doing read operation.
	var policies mosniov1.HTTPFilterPolicyList
	if err := r.List(ctx, &policies, client.InNamespace(req.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list HTTPFilterPolicy: %w", err)
	}

	initState := translation.NewInitState(logger)
	istioGwIdx := map[string][]*mosniov1.HTTPFilterPolicy{}

	for _, policy := range policies.Items {
		policy := policy
		err := mosniov1.ValidateHTTPFilterPolicy(&policy)
		if err != nil {
			// TODO: mark the policy as invalid, and skip logging
			logger.Error(err, "invalid HTTPFilterPolicy", "name", policy.Name, "namespace", policy.Namespace)
			continue
		}

		ref := policy.Spec.TargetRef
		if ref.Group == "networking.istio.io" && ref.Kind == "VirtualService" {
			var virtualService istiov1b1.VirtualService
			nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}
			if ref.Namespace != nil {
				nsName.Namespace = string(*ref.Namespace)
			}
			err := r.Get(ctx, nsName, &virtualService)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return nil, fmt.Errorf("failed to get VirtualService: %w, NamespacedName: %v", err, nsName)
				}

				logger.Info("VirtualService not found", "name", virtualService.Name, "namespace", virtualService.Namespace)
				continue
			}

			err = mosniov1.ValidateVirtualService(&virtualService)
			if err != nil {
				logger.Info("unsupported VirtualService", "name", virtualService.Name, "namespace", virtualService.Namespace, "reason", err.Error())
				continue
			}

			if policy.Spec.TargetRef.SectionName != nil {
				found := false
				name := string(*policy.Spec.TargetRef.SectionName)
				for _, section := range virtualService.Spec.Http {
					if section.Name == name {
						found = true
						break
					}
				}

				if !found {
					logger.Info("skip HTTPFilterPolicy, route not found", "name", policy.Name, "namespace", policy.Namespace, "route", name)
					// TODO: policy implementation should record a resolvedRefs or similar Condition in the Policy's status
					continue
				}
			}

			for _, gw := range virtualService.Spec.Gateways {
				if gw == "mesh" {
					logger.Info("skip unsupported mesh gateway", "name", virtualService.Name, "namespace", virtualService.Namespace)
					continue
				}
				if strings.Contains(gw, "/") {
					logger.Info("skip gateway from other namespace", "name", virtualService.Name, "namespace", virtualService.Namespace)
					continue
				}

				var gateway istiov1b1.Gateway
				err = r.Get(ctx, types.NamespacedName{Name: gw, Namespace: virtualService.Namespace}, &gateway)
				if err != nil {
					if !apierrors.IsNotFound(err) {
						return nil, err
					}
					continue
				}

				err = mosniov1.ValidateGateway(&gateway)
				if err != nil {
					logger.Info("unsupported Gateway", "name", gateway.Name, "namespace", gateway.Namespace, "reason", err.Error())
					continue
				}

				initState.AddPolicyForVirtualService(&policy, &virtualService, &gateway)

				key := k8s.GetObjectKey(&gateway.ObjectMeta)
				if _, ok := istioGwIdx[key]; !ok {
					istioGwIdx[key] = []*mosniov1.HTTPFilterPolicy{}
				}
				istioGwIdx[key] = append(istioGwIdx[key], &policy)
			}
		}
	}

	// only update index when the processing is successful
	r.istioGatewayIndexer.UpdateIndex(istioGwIdx)
	return initState, nil
}

func (r *HTTPFilterPolicyReconciler) translationStateToCustomResource(ctx context.Context, logger *logr.Logger,
	finalState *translation.FinalState) error {

	var envoyfilters istiov1a3.EnvoyFilterList
	if err := r.List(ctx, &envoyfilters, client.MatchingLabels{LabelCreatedBy: "HTTPFilterPolicy"}); err != nil {
		return fmt.Errorf("failed to list EnvoyFilter: %w", err)
	}

	for _, ef := range envoyfilters.Items {
		if _, ok := finalState.EnvoyFilters[ef.Name]; !ok {
			logger.Info("delete EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)
			if err := r.Delete(ctx, ef); err != nil {
				return fmt.Errorf("failed to delete EnvoyFilter: %w, namespacedName: %v",
					err, types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace})
			}
		}
	}

	for _, ef := range finalState.EnvoyFilters {
		ef.Namespace = config.RootNamespace()
		if ef.Labels == nil {
			ef.Labels = map[string]string{}
		}
		ef.Labels[LabelCreatedBy] = "HTTPFilterPolicy"

		var envoyfilter istiov1a3.EnvoyFilter
		nsName := types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace}
		err := r.Get(ctx, nsName, &envoyfilter)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				// If part of EnvoyFilters is already written, retry later is OK as we generate all EnvoyFilters in one reconcile.
				return fmt.Errorf("failed to get EnvoyFilter: %w, namespacedName: %v", err, nsName)
			}

			logger.Info("create EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)

			if err = r.Create(ctx, ef); err != nil {
				return fmt.Errorf("failed to create EnvoyFilter: %w, namespacedName: %v", err, nsName)
			}

		} else {
			if proto.Equal(&envoyfilter.Spec, &ef.Spec) {
				continue
			}

			logger.Info("update EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)
			// Address metadata.resourceVersion: Invalid value: 0x0 error
			ef.SetResourceVersion(envoyfilter.ResourceVersion)
			if err = r.Update(ctx, ef); err != nil {
				return fmt.Errorf("failed to update EnvoyFilter: %w, namespacedName: %v", err, nsName)
			}
		}
	}

	return nil
}

// CustomerResourceIndexer indexes the additional customer resource
// according to the reconciled customer resource
type CustomerResourceIndexer interface {
	CustomerResource() client.Object
	RegisterIndexer(ctx context.Context, mgr ctrl.Manager) error
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

func (v *VirtualServiceIndexer) RegisterIndexer(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &mosniov1.HTTPFilterPolicy{}, v.IndexName(), v.Index)
}

func (v *VirtualServiceIndexer) FindAffectedObjects(ctx context.Context, obj client.Object) []reconcile.Request {
	return findAffectedObjects(ctx, v.r, obj, "VirtualService", v.IndexName())
}

func findAffectedObjects(ctx context.Context, reader client.Reader, obj client.Object, kind string, idx string) []reconcile.Request {
	logger := log.FromContext(ctx)
	logger.Info("Target changed", "kind", kind, "namespace", obj.GetNamespace(), "name", obj.GetName())

	policies := &mosniov1.HTTPFilterPolicyList{}
	listOps := &client.ListOptions{
		// Use the built index
		FieldSelector: fields.OneTermEqualSelector(idx, obj.GetName()),
	}
	err := reader.List(ctx, policies, listOps)
	if err != nil {
		logger.Error(err, "failed to list HTTPFilterPolicy")
		return nil
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

	if len(requests) > 0 {
		logger.Info("Target changed, trigger reconciliation", "kind", kind, "requests", requests)
		// As we do full regeneration, we only need to reconcile one HTTPFilterPolicy
		return []reconcile.Request{requests[0]}
	}
	return requests
}

type IstioGatewayIndexer struct {
	r client.Reader

	lock  sync.RWMutex
	index map[string][]*mosniov1.HTTPFilterPolicy
}

func (v *IstioGatewayIndexer) CustomerResource() client.Object {
	return &istiov1b1.Gateway{}
}

func (v *IstioGatewayIndexer) RegisterIndexer(ctx context.Context, mgr ctrl.Manager) error {
	return nil
}

func (v *IstioGatewayIndexer) UpdateIndex(idx map[string][]*mosniov1.HTTPFilterPolicy) {
	v.lock.Lock()
	v.index = idx
	v.lock.Unlock()
}

func (v *IstioGatewayIndexer) FindAffectedObjects(ctx context.Context, obj client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	logger.Info("Target changed", "kind", "IstioGateway", "namespace", obj.GetNamespace(), "name", obj.GetName())

	gw := obj.(*istiov1b1.Gateway)
	v.lock.RLock()
	policies, ok := v.index[k8s.GetObjectKey(&gw.ObjectMeta)]
	v.lock.RUnlock()
	if !ok {
		return nil
	}

	requests := make([]reconcile.Request, len(policies))
	for i, policy := range policies {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      policy.GetName(),
				Namespace: policy.GetNamespace(),
			},
		}
		requests[i] = request
	}
	logger.Info("Target changed, trigger reconciliation", "kind", "IstioGateway", "requests", requests)
	return []reconcile.Request{requests[0]}
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPFilterPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctx := context.Background()
	istioGatewayIndexer := &IstioGatewayIndexer{
		r: r,
	}
	r.istioGatewayIndexer = istioGatewayIndexer
	indexers := []CustomerResourceIndexer{
		&VirtualServiceIndexer{
			r: r,
		},
		istioGatewayIndexer,
	}
	// IndexField is called per HTTPFilterPolicy
	for _, idxer := range indexers {
		if err := idxer.RegisterIndexer(ctx, mgr); err != nil {
			return err
		}
	}

	controller := ctrl.NewControllerManagedBy(mgr).
		For(&mosniov1.HTTPFilterPolicy{})
		// We don't reconcile when the generated EnvoyFilter is modified.
		// So that user can manually correct the EnvoyFilter, until something else is changed.

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
