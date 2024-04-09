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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/internal/metrics"
	"mosn.io/htnn/controller/internal/model"
	"mosn.io/htnn/controller/internal/translation"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func getK8sKey(ns, name string) string {
	return ns + "/" + name
}

// HTTPFilterPolicyReconciler reconciles a HTTPFilterPolicy object
type HTTPFilterPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	virtualServiceIndexer *VirtualServiceIndexer
	httpRouteIndexer      *HTTPRouteIndexer
	istioGatewayIndexer   *IstioGatewayIndexer
	k8sGatewayIndexer     *K8sGatewayIndexer
}

//+kubebuilder:rbac:groups=htnn.mosn.io,resources=httpfilterpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=httpfilterpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=httpfilterpolicies/finalizers,verbs=update
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
	var policies mosniov1.HTTPFilterPolicyList
	initState, err := r.policyToTranslationState(ctx, &policies)
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
		log.Errorf("failed to process state: %v", err)
		// there is no retryable err during processing
		return ctrl.Result{}, nil
	}

	// In my experience, writing to K8S API server is probably the slowest part.
	// We can add a configured concurrency to write to API server in parallel, if
	// the performance is not good. Note that the API server probably has rate limit.

	err = r.translationStateToCustomResource(ctx, finalState)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.updatePolicies(ctx, &policies)
	return ctrl.Result{}, err
}

func (r *HTTPFilterPolicyReconciler) resolveVirtualService(ctx context.Context,
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
		log.Infof("unsupported VirtualService, name: %s, namespace: %s, reason: %v", virtualService.Name, virtualService.Namespace, err.Error())
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

	return r.resolveWithVirtualService(ctx, &virtualService, policy, initState, gwIdx)
}

func (r *HTTPFilterPolicyReconciler) resolveWithVirtualService(ctx context.Context,
	virtualService *istiov1b1.VirtualService, policy *mosniov1.HTTPFilterPolicy, initState *translation.InitState,
	gwIdx map[string][]*mosniov1.HTTPFilterPolicy) error {

	var err error
	accepted := false
	gws := initState.GetGatewaysWithVirtualService(virtualService)
	if len(gws) > 0 {
		for _, gateway := range gws {
			key := getK8sKey(gateway.Namespace, gateway.Name)
			gwIdx[key] = append(gwIdx[key], policy)
		}

		accepted = true

	} else {
		gws = make([]*istiov1b1.Gateway, 0, len(virtualService.Spec.Gateways))
		for _, gw := range virtualService.Spec.Gateways {
			if gw == "mesh" {
				log.Infof("skip unsupported mesh gateway, name: %s, namespace: %s", virtualService.Name, virtualService.Namespace)
				continue
			}
			if strings.Contains(gw, "/") {
				log.Infof("skip gateway from other namespace, name: %s, namespace: %s, gateway: %s", virtualService.Name, virtualService.Namespace, gw)
				continue
			}

			key := getK8sKey(virtualService.Namespace, gw)
			// We index the gateway regardless of whether it is valid or not.
			// Otherwise, we don't know whether the gateway is changed from invalid to valid.
			gwIdx[key] = append(gwIdx[key], policy)

			var gateway istiov1b1.Gateway
			err = r.Get(ctx, types.NamespacedName{Name: gw, Namespace: virtualService.Namespace}, &gateway)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return err
				}
				log.Infof("gateway not found, name: %s, namespace: %s, gateway: %s", virtualService.Name, virtualService.Namespace, gw)
				continue
			}

			err = mosniov1.ValidateGateway(&gateway)
			if err != nil {
				log.Infof("unsupported Gateway, name: %s, namespace: %s, gateway name: %s, gateway namespace: %s, reason: %v",
					virtualService.Name, virtualService.Namespace, gateway.Name, gateway.Namespace, err)
				continue
			}

			gws = append(gws, &gateway)

			accepted = true
		}
	}

	if accepted {
		initState.AddPolicyForVirtualService(policy, virtualService, gws)
		policy.SetAccepted(gwapiv1a2.PolicyReasonAccepted)
		// For reducing the write to K8S API server and reconciliation,
		// we don't add `gateway.networking.k8s.io/PolicyAffected` to the affected resource.
		// If people want to check whether the VirtualService/HTTPRoute is affected, they can
		// check whether there is an EnvoyFilter named `httn-h-$host` (the `$host` is one of the resources' hosts).
		// For wildcard host, the `*.` is converted to `-`. For example, `*.example.com` results in
		// EnvoyFilter name `htnn-h--example.com`, and `www.example.com` results in `httn-h-www.example.com`.
	} else {
		policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound, "all gateways are not found or unsupported")
	}

	return nil
}

func (r *HTTPFilterPolicyReconciler) resolveHTTPRoute(ctx context.Context,
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
			key := getK8sKey(gateway.Namespace, gateway.Name)
			gwIdx[key] = append(gwIdx[key], policy)
		}

		accepted = true

	} else {
		gws = make([]*gwapiv1.Gateway, 0, len(route.Spec.ParentRefs))
		ns := route.Namespace

		for _, ref := range route.Spec.ParentRefs {
			if ref.Group != nil && *ref.Group != gwapiv1.GroupName {
				continue
			}
			if ref.Kind != nil && *ref.Kind != gwapiv1.Kind("Gateway") {
				continue
			}
			if ref.Namespace != nil && *ref.Namespace != gwapiv1.Namespace(ns) {
				log.Infof("skip gateway from other namespace, name: %s, namespace: %s, gateway: %v", route.Name, route.Namespace, ref)
				continue
			}

			key := getK8sKey(ns, string(ref.Name))
			// We index the gateway regardless of whether it is valid or not.
			// Otherwise, we don't know whether the gateway is changed from invalid to valid.
			gwIdx[key] = append(gwIdx[key], policy)

			var gw gwapiv1.Gateway
			gwNsName := types.NamespacedName{Name: string(ref.Name), Namespace: ns}
			err = r.Get(ctx, gwNsName, &gw)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return err
				}
				log.Infof("gateway not found, name: %s, namespace: %s, gateway: %v", route.Name, route.Namespace, ref)
				continue
			}

			// This part of code is similar to the code in the translation.
			// The code in the translation filters out which listeners are matched.
			// The code here filters out which gateways have at least one matched listeners.
			atLeastOneListenerMatched := false
			for _, ls := range gw.Spec.Listeners {
				if ref.Port != nil && *ref.Port != ls.Port {
					continue
				}
				if ref.SectionName != nil && *ref.SectionName != ls.Name {
					continue
				}

				if !translation.AllowRoute(ls.AllowedRoutes, &route, &gwNsName) {
					continue
				}

				atLeastOneListenerMatched = true
				break
			}

			if !atLeastOneListenerMatched {
				log.Infof("no matched listeners in gateway %v, name: %s, namespace: %s, listeners: %v", ref,
					route.Name, route.Namespace, gw.Spec.Listeners)
				continue
			}

			gws = append(gws, &gw)

			accepted = true
		}
	}

	if accepted {
		initState.AddPolicyForHTTPRoute(policy, &route, gws)
		policy.SetAccepted(gwapiv1a2.PolicyReasonAccepted)
	} else {
		policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound, "all gateways are not found or unsupported")
	}

	return nil
}

func (r *HTTPFilterPolicyReconciler) policyToTranslationState(ctx context.Context,
	policies *mosniov1.HTTPFilterPolicyList) (*translation.InitState, error) {

	// For current implementation, let's rebuild the state each time to avoid complexity.
	// The controller will use local cache when doing read operation.
	if err := r.List(ctx, policies); err != nil {
		return nil, fmt.Errorf("failed to list HTTPFilterPolicy: %w", err)
	}

	initState := translation.NewInitState()
	vsIdx := map[string][]*mosniov1.HTTPFilterPolicy{}
	hrIdx := map[string][]*mosniov1.HTTPFilterPolicy{}
	istioGwIdx := map[string][]*mosniov1.HTTPFilterPolicy{}
	k8sGwIdx := map[string][]*mosniov1.HTTPFilterPolicy{}

	for i := range policies.Items {
		policy := &policies.Items[i]
		ref := policy.Spec.TargetRef
		nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}

		key := getK8sKey(nsName.Namespace, nsName.Name)
		if ref.Group == "networking.istio.io" && ref.Kind == "VirtualService" {
			vsIdx[key] = append(vsIdx[key], policy)
		} else if ref.Group == "gateway.networking.k8s.io" && ref.Kind == "HTTPRoute" {
			hrIdx[key] = append(hrIdx[key], policy)
		}
	}

	r.virtualServiceIndexer.UpdateIndex(vsIdx)
	if config.EnableGatewayAPI() {
		r.httpRouteIndexer.UpdateIndex(hrIdx)
	}

	for i := range policies.Items {
		policy := &policies.Items[i]
		ref := policy.Spec.TargetRef
		nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}

		// defensive code in case the webhook doesn't work
		if policy.IsSpecChanged() {
			err := mosniov1.ValidateHTTPFilterPolicy(policy)
			if err != nil {
				log.Errorf("invalid HTTPFilterPolicy, err: %v, name: %s, namespace: %s", err, policy.Name, policy.Namespace)
				// mark the policy as invalid
				policy.SetAccepted(gwapiv1a2.PolicyReasonInvalid, err.Error())
				continue
			}
			if ref.Namespace != nil {
				nsName.Namespace = string(*ref.Namespace)
				if nsName.Namespace != policy.Namespace {
					err := errors.New("namespace in TargetRef doesn't match HTTPFilterPolicy's namespace")
					log.Errorf("invalid HTTPFilterPolicy, err: %v, name: %s, namespace: %s", err, policy.Name, policy.Namespace)
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
			err = r.resolveVirtualService(ctx, policy, initState, istioGwIdx)
		} else if ref.Group == "gateway.networking.k8s.io" && ref.Kind == "HTTPRoute" {
			err = r.resolveHTTPRoute(ctx, policy, initState, k8sGwIdx)
		}
		if err != nil {
			return nil, err
		}
	}

	if config.EnableEmbeddedMode() {
		// Some of our users use embedded policy mostly, so it's fine to list all
		var virtualServices istiov1b1.VirtualServiceList
		if err := r.List(ctx, &virtualServices); err != nil {
			return nil, fmt.Errorf("failed to list VirtualService: %w", err)
		}
		for _, vs := range virtualServices.Items {
			ann := vs.GetAnnotations()
			if ann == nil || ann[model.AnnotationHTTPFilterPolicy] == "" {
				continue
			}

			var policy mosniov1.HTTPFilterPolicy
			err := json.Unmarshal([]byte(ann[model.AnnotationHTTPFilterPolicy]), &policy)
			if err != nil {
				log.Errorf("failed to unmarshal policy out from VirtualService, err: %v, name: %s, namespace: %s", err, vs.Name, vs.Namespace)
				continue
			}
			// We require the embedded policy to be valid, otherwise it's costly to validate and hard to report the error.

			policy.Namespace = vs.Namespace
			// Name convention is "embedded-$kind-$name"
			policy.Name = "embedded-virtualservice-" + vs.Name
			err = r.resolveWithVirtualService(ctx, vs, &policy, initState, istioGwIdx)
			if err != nil {
				return nil, err
			}
		}
	}

	// Only update index when the processing is successful. This prevents gateways from being partially indexed.
	r.istioGatewayIndexer.UpdateIndex(istioGwIdx)
	if config.EnableGatewayAPI() {
		r.k8sGatewayIndexer.UpdateIndex(k8sGwIdx)
	}

	return initState, nil
}

func fillEnvoyFilterMeta(ef *istiov1a3.EnvoyFilter) {
	ef.Namespace = config.RootNamespace()
	if ef.Labels == nil {
		ef.Labels = map[string]string{}
	}
	ef.Labels[model.LabelCreatedBy] = "HTTPFilterPolicy"
}

func (r *HTTPFilterPolicyReconciler) translationStateToCustomResource(ctx context.Context,
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
			log.Infof("delete EnvoyFilter, name: %s, namespace: %s", e.Name, e.Namespace)
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
			log.Infof("create EnvoyFilter, name: %s, namespace: %s", ef.Name, ef.Namespace)
			fillEnvoyFilterMeta(ef)

			if err := r.Create(ctx, ef); err != nil {
				nsName := types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace}
				return fmt.Errorf("failed to create EnvoyFilter: %w, namespacedName: %v", err, nsName)
			}

		} else {
			if proto.Equal(&envoyfilter.Spec, &ef.Spec) {
				continue
			}

			log.Infof("update EnvoyFilter, name: %s, namespace: %s", ef.Name, ef.Namespace)
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
	FindAffectedObjects(ctx context.Context, obj client.Object) []reconcile.Request
	Predicate() predicate.Predicate
}

// indexer extracts common logic for indexing the affected resources
type indexer struct {
	lock  sync.RWMutex
	index map[string][]*mosniov1.HTTPFilterPolicy
}

func (v *indexer) UpdateIndex(idx map[string][]*mosniov1.HTTPFilterPolicy) {
	v.lock.Lock()
	v.index = idx
	v.lock.Unlock()
}

func (v *indexer) FindAffectedObjects(ctx context.Context, obj client.Object) []reconcile.Request {
	if config.EnableEmbeddedMode() {
		ann := obj.GetAnnotations()
		if ann != nil && ann[model.AnnotationHTTPFilterPolicy] != "" {
			log.Infof("Target with embedded HTTPFilterPolicy changed, trigger reconciliation, kind: %s, namespace: %s, name: %s",
				obj.GetObjectKind().GroupVersionKind().Kind,
				obj.GetNamespace(), obj.GetName())
			return triggerReconciliation()
		}
	}

	v.lock.RLock()
	policies, ok := v.index[getK8sKey(obj.GetNamespace(), obj.GetName())]
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

	log.Infof("Target changed, trigger reconciliation, kind: %s, namespace: %s, name: %s",
		obj.GetObjectKind().GroupVersionKind().Kind,
		obj.GetNamespace(), obj.GetName())

	return triggerReconciliation()
}

type VirtualServiceIndexer struct {
	indexer
}

func (v *VirtualServiceIndexer) CustomerResource() client.Object {
	return &istiov1b1.VirtualService{}
}

func (v *VirtualServiceIndexer) Predicate() predicate.Predicate {
	return predicate.Or(
		predicate.GenerationChangedPredicate{},
		predicate.AnnotationChangedPredicate{},
	)
}

type HTTPRouteIndexer struct {
	indexer
}

func (v *HTTPRouteIndexer) CustomerResource() client.Object {
	return &gwapiv1.HTTPRoute{}
}

func (v *HTTPRouteIndexer) Predicate() predicate.Predicate {
	return predicate.Or(
		predicate.GenerationChangedPredicate{},
		predicate.AnnotationChangedPredicate{},
	)
}

type IstioGatewayIndexer struct {
	indexer
}

func (v *IstioGatewayIndexer) CustomerResource() client.Object {
	return &istiov1b1.Gateway{}
}

func (v *IstioGatewayIndexer) Predicate() predicate.Predicate {
	return predicate.GenerationChangedPredicate{}
}

type K8sGatewayIndexer struct {
	indexer
}

func (v *K8sGatewayIndexer) CustomerResource() client.Object {
	return &gwapiv1.Gateway{}
}

func (v *K8sGatewayIndexer) Predicate() predicate.Predicate {
	return predicate.GenerationChangedPredicate{}
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPFilterPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	virtualServiceIndexer := &VirtualServiceIndexer{}
	r.virtualServiceIndexer = virtualServiceIndexer
	istioGatewayIndexer := &IstioGatewayIndexer{}
	r.istioGatewayIndexer = istioGatewayIndexer
	indexers := []CustomerResourceIndexer{
		virtualServiceIndexer,
		istioGatewayIndexer,
	}

	if config.EnableGatewayAPI() {
		httpRouteIndexer := &HTTPRouteIndexer{}
		r.httpRouteIndexer = httpRouteIndexer
		k8sGatewayIndexer := &K8sGatewayIndexer{}
		r.k8sGatewayIndexer = k8sGatewayIndexer
		indexers = append(indexers,
			httpRouteIndexer,
			k8sGatewayIndexer,
		)
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
