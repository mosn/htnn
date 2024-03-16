/*
Copyright The HTNN Authors.

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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/k8s"
	"mosn.io/htnn/controller/internal/metrics"
	"mosn.io/htnn/controller/internal/model"
	"mosn.io/htnn/controller/internal/translation"
)

// HTTPFilterPolicyReconciler reconciles a HTTPFilterPolicy object
type HTTPFilterPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	istioGatewayIndexer *IstioGatewayIndexer
	k8sGatewayIndexer   *K8sGatewayIndexer
}

//+kubebuilder:rbac:groups=mosn.io,resources=httpfilterpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mosn.io,resources=httpfilterpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mosn.io,resources=httpfilterpolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.istio.io,resources=gateways,verbs=get;list;watch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch
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
	logger.Info("reconcile")

	var policies mosniov1.HTTPFilterPolicyList
	initState, err := r.policyToTranslationState(ctx, &logger, &policies)
	if err != nil {
		return ctrl.Result{}, err
	}
	if initState == nil {
		return ctrl.Result{}, nil
	}

	start := time.Now()
	finalState, err := initState.Process(ctx)
	processDurationInSecs := time.Since(start).Seconds()
	metrics.HFPTranslateDurationObserver.Observe(processDurationInSecs)
	if err != nil {
		logger.Error(err, "failed to process state")
		// there is no retryable err during processing
		return ctrl.Result{}, nil
	}

	// In my experience, writing to K8S API server is probably the slowest part.
	// We can add a configured concurrency to write to API server in parallel, if
	// the performance is not good. Note that the API server probably has rate limit.

	err = r.translationStateToCustomResource(ctx, &logger, finalState)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.updatePolicies(ctx, &policies)
	return ctrl.Result{}, err
}

func (r *HTTPFilterPolicyReconciler) resolveVirtualService(ctx context.Context, logger *logr.Logger,
	policy *mosniov1.HTTPFilterPolicy, initState *translation.InitState, gwIdx map[string][]*mosniov1.HTTPFilterPolicy) error {

	ref := policy.Spec.TargetRef
	nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}
	var virtualService istiov1b1.VirtualService
	err := r.Get(ctx, nsName, &virtualService)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get VirtualService: %w, NamespacedName: %v", err, nsName)
		}

		policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound)
		return nil
	}

	err = mosniov1.ValidateVirtualService(&virtualService)
	if err != nil {
		logger.Info("unsupported VirtualService", "name", virtualService.Name, "namespace", virtualService.Namespace, "reason", err.Error())
		// treat invalid target resource as not found
		policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound, err.Error())
		return nil
	}

	if ref.SectionName != nil {
		found := false
		name := string(*ref.SectionName)
		for _, section := range virtualService.Spec.Http {
			if section.Name == name {
				found = true
				break
			}
		}

		if !found {
			policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound, fmt.Sprintf("sectionName %s not found", name))
			return nil
		}
	}

	accepted := false
	gws := initState.GetGatewaysWithVirtualService(&virtualService)
	if len(gws) > 0 {
		for _, gateway := range gws {
			key := k8s.GetObjectKey(&gateway.ObjectMeta)
			gwIdx[key] = append(gwIdx[key], policy)
		}

		accepted = true

	} else {
		gws = make([]*istiov1b1.Gateway, 0, len(virtualService.Spec.Gateways))
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
					return err
				}
				logger.Info("gateway not found", "gateway", gw,
					"name", virtualService.Name, "namespace", virtualService.Namespace)
				continue
			}

			err = mosniov1.ValidateGateway(&gateway)
			if err != nil {
				logger.Info("unsupported Gateway", "name", gateway.Name, "namespace", gateway.Namespace, "reason", err.Error())
				continue
			}

			gws = append(gws, &gateway)

			key := k8s.GetObjectKey(&gateway.ObjectMeta)
			gwIdx[key] = append(gwIdx[key], policy)

			accepted = true
		}
	}

	if accepted {
		initState.AddPolicyForVirtualService(policy, &virtualService, gws)
		policy.SetAccepted(gwapiv1a2.PolicyReasonAccepted)
		// For reducing the write to K8S API server and reconciliation,
		// we don't add `gateway.networking.k8s.io/PolicyAffected` to the affected resource.
		// If people want to check whether the VirtualService/HTTPRoute is affected, they can
		// check whether there is an EnvoyFilter named `httn-h-$host` (the `$host` is one of the resources' hosts).
		// For wildcard host, the `*.` is converted to `-`. For example, `*.example.com` results in
		// EnvoyFilter name `htnn-h--example.com`, and `www.example.com` results in `httn-h-www.example.com`.
	} else {
		policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound, "invalid target resource")
	}

	return nil
}

func (r *HTTPFilterPolicyReconciler) resolveHTTPRoute(ctx context.Context, logger *logr.Logger,
	policy *mosniov1.HTTPFilterPolicy, initState *translation.InitState, gwIdx map[string][]*mosniov1.HTTPFilterPolicy) error {

	ref := policy.Spec.TargetRef
	nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}
	var route gwapiv1.HTTPRoute
	err := r.Get(ctx, nsName, &route)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get HTTPRoute: %w, NamespacedName: %v", err, nsName)
		}

		policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound)
		return nil
	}

	accepted := false
	gws := initState.GetGatewaysWithHTTPRoute(&route)
	if len(gws) > 0 {
		for _, gateway := range gws {
			key := k8s.GetObjectKey(&gateway.ObjectMeta)
			gwIdx[key] = append(gwIdx[key], policy)
		}

		accepted = true

	} else {
		gws = make([]*gwapiv1.Gateway, 0, len(route.Status.Parents))
		for _, pr := range route.Status.Parents {
			// Consider it is valid once the listener is attached
			ref := pr.ParentRef
			if ref.Group != nil && *ref.Group != gwapiv1.GroupName {
				continue
			}
			if ref.Kind != nil && *ref.Kind != gwapiv1.Kind("Gateway") {
				continue
			}

			ns := route.Namespace
			if ref.Namespace != nil {
				ns = string(*ref.Namespace)
			}
			var gateway gwapiv1.Gateway
			err = r.Get(ctx, types.NamespacedName{Name: string(ref.Name), Namespace: ns}, &gateway)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return err
				}
				logger.Info("gateway not found", "gateway", ref,
					"name", route.Name, "namespace", route.Namespace)
				continue
			}

			gws = append(gws, &gateway)

			key := k8s.GetObjectKey(&gateway.ObjectMeta)
			gwIdx[key] = append(gwIdx[key], policy)

			accepted = true
		}
	}

	if accepted {
		// The CRD status only tells us which gateways are referenced & how many attached routes per listener.
		// But we don't know which listerner is referenced by a specific HTTPRoute.
		// So we have to keep the whole gateway and filter the listener by ourselves.
		initState.AddPolicyForHTTPRoute(policy, &route, gws)
		policy.SetAccepted(gwapiv1a2.PolicyReasonAccepted)
	} else {
		policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound, "invalid target resource")
	}

	return nil
}

func (r *HTTPFilterPolicyReconciler) policyToTranslationState(ctx context.Context, logger *logr.Logger,
	policies *mosniov1.HTTPFilterPolicyList) (*translation.InitState, error) {

	// For current implementation, let's rebuild the state each time to avoid complexity.
	// The controller will use local cache when doing read operation.
	if err := r.List(ctx, policies); err != nil {
		return nil, fmt.Errorf("failed to list HTTPFilterPolicy: %w", err)
	}

	initState := translation.NewInitState(logger)
	istioGwIdx := map[string][]*mosniov1.HTTPFilterPolicy{}
	k8sGwIdx := map[string][]*mosniov1.HTTPFilterPolicy{}

	for i := range policies.Items {
		policy := &policies.Items[i]
		ref := policy.Spec.TargetRef
		nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}

		// defensive code in case the webhook doesn't work
		if policy.IsSpecChanged() {
			err := mosniov1.ValidateHTTPFilterPolicy(policy)
			if err != nil {
				logger.Error(err, "invalid HTTPFilterPolicy", "name", policy.Name, "namespace", policy.Namespace)
				// mark the policy as invalid
				policy.SetAccepted(gwapiv1a2.PolicyReasonInvalid, err.Error())
				continue
			}
			if ref.Namespace != nil {
				nsName.Namespace = string(*ref.Namespace)
				if nsName.Namespace != policy.Namespace {
					err := errors.New("namespace in TargetRef doesn't match HTTPFilterPolicy's namespace")
					logger.Error(err, "invalid HTTPFilterPolicy", "name", policy.Name, "namespace", policy.Namespace)
					policy.SetAccepted(gwapiv1a2.PolicyReasonInvalid, err.Error())
					continue
				}
			}
		}
		if !policy.IsValid() {
			continue
		}

		var err error
		if ref.Group == "networking.istio.io" && ref.Kind == "VirtualService" {
			err = r.resolveVirtualService(ctx, logger, policy, initState, istioGwIdx)
		} else if ref.Group == "gateway.networking.k8s.io" && ref.Kind == "HTTPRoute" {
			err = r.resolveHTTPRoute(ctx, logger, policy, initState, k8sGwIdx)
		}
		if err != nil {
			return nil, err
		}
	}

	// only update index when the processing is successful
	r.istioGatewayIndexer.UpdateIndex(istioGwIdx)
	r.k8sGatewayIndexer.UpdateIndex(k8sGwIdx)
	return initState, nil
}

func fillEnvoyFilterMeta(ef *istiov1a3.EnvoyFilter) {
	ef.Namespace = config.RootNamespace()
	if ef.Labels == nil {
		ef.Labels = map[string]string{}
	}
	ef.Labels[model.LabelCreatedBy] = "HTTPFilterPolicy"
}

func (r *HTTPFilterPolicyReconciler) translationStateToCustomResource(ctx context.Context, logger *logr.Logger,
	finalState *translation.FinalState) error {

	var envoyfilters istiov1a3.EnvoyFilterList
	if err := r.List(ctx, &envoyfilters,
		client.MatchingLabels{model.LabelCreatedBy: "HTTPFilterPolicy"},
	); err != nil {
		return fmt.Errorf("failed to list EnvoyFilter: %w", err)
	}

	preEnvoyFilterMap := make(map[string]*istiov1a3.EnvoyFilter, len(envoyfilters.Items))
	for _, e := range envoyfilters.Items {
		if _, ok := finalState.EnvoyFilters[e.Name]; !ok || e.Namespace != config.RootNamespace() {
			logger.Info("delete EnvoyFilter", "name", e.Name, "namespace", e.Namespace)
			if err := r.Delete(ctx, e); err != nil {
				return fmt.Errorf("failed to delete EnvoyFilter: %w, namespacedName: %v",
					err, types.NamespacedName{Name: e.Name, Namespace: e.Namespace})
			}
		} else {
			preEnvoyFilterMap[e.Name] = e
		}
	}

	for _, ef := range finalState.EnvoyFilters {
		envoyfilter, ok := preEnvoyFilterMap[ef.Name]
		if !ok {
			logger.Info("create EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)
			fillEnvoyFilterMeta(ef)

			if err := r.Create(ctx, ef); err != nil {
				nsName := types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace}
				return fmt.Errorf("failed to create EnvoyFilter: %w, namespacedName: %v", err, nsName)
			}

		} else {
			if proto.Equal(&envoyfilter.Spec, &ef.Spec) {
				continue
			}

			logger.Info("update EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)
			fillEnvoyFilterMeta(ef)
			// Address metadata.resourceVersion: Invalid value: 0x0 error
			ef.SetResourceVersion(envoyfilter.ResourceVersion)
			if err := r.Update(ctx, ef); err != nil {
				nsName := types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace}
				return fmt.Errorf("failed to update EnvoyFilter: %w, namespacedName: %v", err, nsName)
			}
		}
	}

	return nil
}

func (r *HTTPFilterPolicyReconciler) updatePolicies(ctx context.Context,
	policies *mosniov1.HTTPFilterPolicyList) error {

	for i := range policies.Items {
		policy := &policies.Items[i]
		// track changed status will be a little faster than iterating policies
		// but make code much complex
		if !policy.Status.IsChanged() {
			continue
		}
		policy.Status.Reset()
		// Update operation will change the original object in cache, so we need to deepcopy it.
		if err := r.Status().Update(ctx, policy.DeepCopy()); err != nil {
			return fmt.Errorf("failed to update HTTPFilterPolicy status: %w, namespacedName: %v",
				err, types.NamespacedName{Name: policy.Name, Namespace: policy.Namespace})
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
	Predicate() predicate.Predicate
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
	ns := po.Namespace
	if po.Spec.TargetRef.Namespace != nil {
		ns = string(*po.Spec.TargetRef.Namespace)
	}
	return []string{fmt.Sprintf("%s/%s", ns, po.Spec.TargetRef.Name)}
}

func (v *VirtualServiceIndexer) RegisterIndexer(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &mosniov1.HTTPFilterPolicy{}, v.IndexName(), v.Index)
}

func (v *VirtualServiceIndexer) FindAffectedObjects(ctx context.Context, obj client.Object) []reconcile.Request {
	return findAffectedObjects(ctx, v.r, obj, "VirtualService", v.IndexName())
}

func (v *VirtualServiceIndexer) Predicate() predicate.Predicate {
	return predicate.GenerationChangedPredicate{}
}

type HTTPRouteIndexer struct {
	r client.Reader
}

func (v *HTTPRouteIndexer) CustomerResource() client.Object {
	return &gwapiv1.HTTPRoute{}
}

func (v *HTTPRouteIndexer) IndexName() string {
	return "spec.targetRef.kind.HTTPRoute"
}

func (v *HTTPRouteIndexer) Index(rawObj client.Object) []string {
	po := rawObj.(*mosniov1.HTTPFilterPolicy)
	if po.Spec.TargetRef.Group != gwapiv1.GroupName || po.Spec.TargetRef.Kind != "HTTPRoute" {
		return []string{}
	}
	ns := po.Namespace
	if po.Spec.TargetRef.Namespace != nil {
		ns = string(*po.Spec.TargetRef.Namespace)
	}
	return []string{fmt.Sprintf("%s/%s", ns, po.Spec.TargetRef.Name)}
}

func (v *HTTPRouteIndexer) RegisterIndexer(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &mosniov1.HTTPFilterPolicy{}, v.IndexName(), v.Index)
}

func (v *HTTPRouteIndexer) FindAffectedObjects(ctx context.Context, obj client.Object) []reconcile.Request {
	return findAffectedObjects(ctx, v.r, obj, "HTTPRoute", v.IndexName())
}

func (v *HTTPRouteIndexer) Predicate() predicate.Predicate {
	return predicate.ResourceVersionChangedPredicate{}
}

func findAffectedObjects(ctx context.Context, reader client.Reader, obj client.Object, kind string, idx string) []reconcile.Request {
	logger := log.FromContext(ctx)

	policies := &mosniov1.HTTPFilterPolicyList{}
	listOps := &client.ListOptions{
		// Use the built index
		FieldSelector: fields.OneTermEqualSelector(idx, fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())),
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
		logger.Info("Target changed, trigger reconciliation", "kind", kind,
			"namespace", obj.GetNamespace(), "name", obj.GetName(), "requests", requests)
		// As we do full regeneration, we only need to reconcile one HTTPFilterPolicy
		return triggerReconciliation()
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
	logger.Info("Target changed, trigger reconciliation", "kind", "IstioGateway",
		"namespace", obj.GetNamespace(), "name", obj.GetName(), "requests", requests)
	return triggerReconciliation()
}

func (v *IstioGatewayIndexer) Predicate() predicate.Predicate {
	return predicate.GenerationChangedPredicate{}
}

type K8sGatewayIndexer struct {
	r client.Reader

	lock  sync.RWMutex
	index map[string][]*mosniov1.HTTPFilterPolicy
}

func (v *K8sGatewayIndexer) CustomerResource() client.Object {
	return &gwapiv1.Gateway{}
}

func (v *K8sGatewayIndexer) RegisterIndexer(ctx context.Context, mgr ctrl.Manager) error {
	return nil
}

func (v *K8sGatewayIndexer) UpdateIndex(idx map[string][]*mosniov1.HTTPFilterPolicy) {
	v.lock.Lock()
	v.index = idx
	v.lock.Unlock()
}

func (v *K8sGatewayIndexer) FindAffectedObjects(ctx context.Context, obj client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)

	gw := obj.(*gwapiv1.Gateway)
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
	logger.Info("Target changed, trigger reconciliation", "kind", "K8sGateway",
		"namespace", obj.GetNamespace(), "name", obj.GetName(), "requests", requests)
	return triggerReconciliation()
}

func (v *K8sGatewayIndexer) Predicate() predicate.Predicate {
	return predicate.GenerationChangedPredicate{}
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPFilterPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctx := context.Background()
	istioGatewayIndexer := &IstioGatewayIndexer{
		r: r,
	}
	r.istioGatewayIndexer = istioGatewayIndexer
	k8sGatewayIndexer := &K8sGatewayIndexer{
		r: r,
	}
	r.k8sGatewayIndexer = k8sGatewayIndexer
	indexers := []CustomerResourceIndexer{
		&VirtualServiceIndexer{
			r: r,
		},
		istioGatewayIndexer,
		&HTTPRouteIndexer{
			r: r,
		},
		k8sGatewayIndexer,
	}
	// IndexField is called per HTTPFilterPolicy
	for _, idxer := range indexers {
		if err := idxer.RegisterIndexer(ctx, mgr); err != nil {
			return err
		}
	}

	controller := ctrl.NewControllerManagedBy(mgr).
		Named("httpfilterpolicy").
		Watches(
			&mosniov1.HTTPFilterPolicy{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, _ client.Object) []reconcile.Request {
				return triggerReconciliation()
			}),
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
			),
		)
		// We don't reconcile when the generated EnvoyFilter is modified.
		// So that user can manually correct the EnvoyFilter, until something else is changed.

	for _, idxer := range indexers {
		controller.Watches(
			idxer.CustomerResource(),
			handler.EnqueueRequestsFromMapFunc(idxer.FindAffectedObjects),
			builder.WithPredicates(idxer.Predicate()),
		)
	}

	return controller.Complete(r)
}
