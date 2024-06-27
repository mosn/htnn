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

	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/internal/metrics"
	"mosn.io/htnn/controller/internal/translation"
	"mosn.io/htnn/controller/pkg/component"
	"mosn.io/htnn/controller/pkg/constant"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func getK8sKey(ns, name string) string {
	return ns + "/" + name
}

// FilterPolicyReconciler reconciles a FilterPolicy object
type FilterPolicyReconciler struct {
	component.ResourceManager
	output component.Output

	indexers map[string]*customResourceIndexer

	virtualServiceIndexer *customResourceIndexer
	httpRouteIndexer      *customResourceIndexer
	istioGatewayIndexer   *customResourceIndexer
	k8sGatewayIndexer     *customResourceIndexer
}

func NewFilterPolicyReconciler(output component.Output, manager component.ResourceManager) *FilterPolicyReconciler {
	r := &FilterPolicyReconciler{
		ResourceManager: manager,
		output:          output,

		indexers: make(map[string]*customResourceIndexer),
	}

	virtualServiceIndexer := &customResourceIndexer{
		Group:          "networking.istio.io",
		Kind:           "VirtualService",
		CustomResource: &istiov1a3.VirtualService{},
	}
	r.virtualServiceIndexer = virtualServiceIndexer
	istioGatewayIndexer := &customResourceIndexer{
		Group:          "networking.istio.io",
		Kind:           "Gateway",
		CustomResource: &istiov1a3.Gateway{},
	}
	r.istioGatewayIndexer = istioGatewayIndexer
	r.addIndexer(virtualServiceIndexer)
	r.addIndexer(istioGatewayIndexer)

	if config.EnableGatewayAPI() {
		httpRouteIndexer := &customResourceIndexer{
			Group:          "gateway.networking.k8s.io",
			Kind:           "HTTPRoute",
			CustomResource: &gwapiv1b1.HTTPRoute{},
		}
		r.httpRouteIndexer = httpRouteIndexer
		k8sGatewayIndexer := &customResourceIndexer{
			Group:          "gateway.networking.k8s.io",
			Kind:           "Gateway",
			CustomResource: &gwapiv1b1.Gateway{},
		}
		r.k8sGatewayIndexer = k8sGatewayIndexer
		r.addIndexer(httpRouteIndexer)
		r.addIndexer(k8sGatewayIndexer)
	}

	return r
}

func (r *FilterPolicyReconciler) addIndexer(idxer *customResourceIndexer) {
	r.indexers[fmt.Sprintf("%s/%s", idxer.Group, idxer.Kind)] = idxer
}

func (r *FilterPolicyReconciler) NeedReconcile(ctx context.Context, meta component.ResourceMeta) bool {
	var reqs []reconcile.Request
	idxer := r.indexers[fmt.Sprintf("%s/%s", meta.GetGroup(), meta.GetKind())]
	if idxer == nil {
		log.Errorf("no indexer found for %s/%s", meta.GetGroup(), meta.GetKind())
		return false
	}

	reqs = idxer.FindAffectedObjects(ctx, meta)
	return len(reqs) > 0
}

//+kubebuilder:rbac:groups=htnn.mosn.io,resources=filterpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=filterpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=filterpolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=httpfilterpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=httpfilterpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=httpfilterpolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *FilterPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconcilationStart := time.Now()
	defer func() {
		reconcilationDuration := time.Since(reconcilationStart).Seconds()
		metrics.HFPReconcileDurationDistribution.Record(reconcilationDuration)
	}()

	log.Info("Reconcile FilterPolicy")

	var policies mosniov1.FilterPolicyList
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
	metrics.HFPTranslateDurationDistribution.Record(processDurationInSecs)
	if err != nil {
		log.Errorf("failed to process state: %v", err)
		// there is no retryable err during processing
		return ctrl.Result{}, nil
	}

	generatedEnvoyFilters := finalState.EnvoyFilters
	err = r.output.FromFilterPolicy(ctx, generatedEnvoyFilters)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.updatePolicies(ctx, &policies)
	return ctrl.Result{}, err
}

func (r *FilterPolicyReconciler) resolveVirtualService(ctx context.Context,
	policy *mosniov1.FilterPolicy, initState *translation.InitState, gwIdx map[string][]*mosniov1.FilterPolicy) error {

	ref := policy.Spec.TargetRef
	nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}
	var virtualService istiov1a3.VirtualService
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

func (r *FilterPolicyReconciler) resolveWithVirtualService(ctx context.Context,
	virtualService *istiov1a3.VirtualService, policy *mosniov1.FilterPolicy, initState *translation.InitState,
	gwIdx map[string][]*mosniov1.FilterPolicy) error {

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
		gws = make([]*istiov1a3.Gateway, 0, len(virtualService.Spec.Gateways))
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

			var gateway istiov1a3.Gateway
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

func (r *FilterPolicyReconciler) resolveHTTPRoute(ctx context.Context,
	policy *mosniov1.FilterPolicy, initState *translation.InitState, gwIdx map[string][]*mosniov1.FilterPolicy) error {

	ref := policy.Spec.TargetRef
	nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}
	var route gwapiv1b1.HTTPRoute
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
		gws = make([]*gwapiv1b1.Gateway, 0, len(route.Spec.ParentRefs))
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

			var gw gwapiv1b1.Gateway
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

func (r *FilterPolicyReconciler) resolveIstioGateway(ctx context.Context,
	policy *mosniov1.FilterPolicy, initState *translation.InitState) error {

	ref := policy.Spec.TargetRef
	nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}
	var gateway istiov1a3.Gateway
	err := r.Get(ctx, nsName, &gateway)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Istio Gateway: %w, NamespacedName: %v", err, nsName)
		}

		policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound)
		return nil
	}

	err = mosniov1.ValidateGateway(&gateway)
	if err != nil {
		log.Infof("unsupported Istio Gateway, name: %s, namespace: %s, reason: %v", gateway.Name, gateway.Namespace, err.Error())
		policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound, err.Error())
		return nil
	}

	if ref.SectionName != nil {
		found := false
		name := string(*ref.SectionName)
		for _, section := range gateway.Spec.Servers {
			if section.Name == name {
				found = true
				break
			}
		}

		if !found {
			policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound, fmt.Sprintf("There is not Server.Name %s", name))
			return nil
		}
	}

	return r.resolveWithIstioGateway(ctx, &gateway, policy, initState)
}

func (r *FilterPolicyReconciler) resolveWithIstioGateway(_ context.Context,
	gateway *istiov1a3.Gateway, policy *mosniov1.FilterPolicy, initState *translation.InitState) error {

	initState.AddPolicyForIstioGateway(policy, gateway)
	policy.SetAccepted(gwapiv1a2.PolicyReasonAccepted)
	return nil
}

func (r *FilterPolicyReconciler) resolveK8sGateway(ctx context.Context,
	policy *mosniov1.FilterPolicy, initState *translation.InitState) error {

	ref := policy.Spec.TargetRef
	nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}
	var gateway gwapiv1b1.Gateway
	err := r.Get(ctx, nsName, &gateway)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get K8S Gateway: %w, NamespacedName: %v", err, nsName)
		}

		policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound)
		return nil
	}

	if ref.SectionName != nil {
		found := false
		name := *ref.SectionName
		for _, section := range gateway.Spec.Listeners {
			if section.Name == name {
				found = true
				break
			}
		}

		if !found {
			policy.SetAccepted(gwapiv1a2.PolicyReasonTargetNotFound, fmt.Sprintf("There is not Listener.Name %s", name))
			return nil
		}
	}

	initState.AddPolicyForK8sGateway(policy, &gateway)
	policy.SetAccepted(gwapiv1a2.PolicyReasonAccepted)
	return nil
}

// When multiple policies target to the same resource, the oldest one wins. Since
// the embedded policy doesn't have a CreationTimestamp, it will has the highest priority.
// One can change this behavior by given a fake CreationTimestamp.
func getFilterPolicyFromAnnotation(obj client.Object) *mosniov1.FilterPolicy {
	ann := obj.GetAnnotations()
	if ann == nil {
		return nil
	}

	if ann[constant.AnnotationFilterPolicy] != "" {
		var policy mosniov1.FilterPolicy
		err := json.Unmarshal([]byte(ann[constant.AnnotationFilterPolicy]), &policy)
		if err != nil {
			log.Errorf("failed to unmarshal policy out, err: %v, name: %s, namespace: %s", err, obj.GetName(), obj.GetNamespace())
			return nil
		}
		return &policy
	}

	if ann[constant.AnnotationHTTPFilterPolicy] != "" {
		var policy mosniov1.HTTPFilterPolicy
		err := json.Unmarshal([]byte(ann[constant.AnnotationHTTPFilterPolicy]), &policy)
		if err != nil {
			log.Errorf("failed to unmarshal policy out, err: %v, name: %s, namespace: %s", err, obj.GetName(), obj.GetNamespace())
			return nil
		}
		fp := mosniov1.ConvertHTTPFilterPolicyToFilterPolicy(&policy)
		return &fp
	}
	return nil
}

func (r *FilterPolicyReconciler) policyToTranslationState(ctx context.Context,
	policies *mosniov1.FilterPolicyList) (*translation.InitState, error) {

	// For current implementation, let's rebuild the state each time to avoid complexity.
	// The controller will use local cache when doing read operation.
	if err := r.List(ctx, policies); err != nil {
		return nil, fmt.Errorf("failed to list FilterPolicy: %w", err)
	}

	var httpfilterpolicies mosniov1.HTTPFilterPolicyList
	if err := r.List(ctx, &httpfilterpolicies); err != nil {
		return nil, fmt.Errorf("failed to list HTTPFilterPolicy: %w", err)
	}
	for _, p := range httpfilterpolicies.Items {
		p := p // avoid capturing loop variable
		policies.Items = append(policies.Items, mosniov1.ConvertHTTPFilterPolicyToFilterPolicy(&p))
	}

	initState := translation.NewInitState()
	vsIdx := map[string][]*mosniov1.FilterPolicy{}
	hrIdx := map[string][]*mosniov1.FilterPolicy{}
	istioGwIdx := map[string][]*mosniov1.FilterPolicy{}
	k8sGwIdx := map[string][]*mosniov1.FilterPolicy{}

	for i := range policies.Items {
		policy := &policies.Items[i]
		ref := policy.Spec.TargetRef
		if ref == nil {
			continue
		}

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

	supportGatewayPolicy := config.EnableLDSPluginViaECDS()

	for i := range policies.Items {
		policy := &policies.Items[i]
		ref := policy.Spec.TargetRef
		if ref == nil {
			policy.SetAccepted(gwapiv1a2.PolicyReasonInvalid, "targetRef is required when using FilterPolicy outside embedded mode")
			continue
		}

		nsName := types.NamespacedName{Name: string(ref.Name), Namespace: policy.Namespace}

		// defensive code in case the webhook doesn't work
		if policy.IsSpecChanged() {
			err := mosniov1.ValidateFilterPolicy(policy)
			if err != nil {
				log.Errorf("invalid FilterPolicy, err: %v, name: %s, namespace: %s", err, policy.Name, policy.Namespace)
				// mark the policy as invalid
				policy.SetAccepted(gwapiv1a2.PolicyReasonInvalid, err.Error())
				continue
			}
			if ref.Namespace != nil {
				nsName.Namespace = string(*ref.Namespace)
				if nsName.Namespace != policy.Namespace {
					err := errors.New("namespace in TargetRef doesn't match FilterPolicy's namespace")
					log.Errorf("invalid FilterPolicy, err: %v, name: %s, namespace: %s", err, policy.Name, policy.Namespace)
					policy.SetAccepted(gwapiv1a2.PolicyReasonInvalid, err.Error())
					continue
				}
			}
		}
		if !policy.IsValid() {
			continue
		}

		var err error
		if ref.Group == "networking.istio.io" {
			if ref.Kind == "VirtualService" {
				err = r.resolveVirtualService(ctx, policy, initState, istioGwIdx)
			} else if ref.Kind == "Gateway" && supportGatewayPolicy {
				key := getK8sKey(nsName.Namespace, nsName.Name)
				istioGwIdx[key] = append(istioGwIdx[key], policy)
				err = r.resolveIstioGateway(ctx, policy, initState)
			}
		} else if ref.Group == "gateway.networking.k8s.io" {
			if ref.Kind == "HTTPRoute" {
				err = r.resolveHTTPRoute(ctx, policy, initState, k8sGwIdx)
			} else if ref.Kind == "Gateway" && supportGatewayPolicy {
				key := getK8sKey(nsName.Namespace, nsName.Name)
				k8sGwIdx[key] = append(k8sGwIdx[key], policy)
				err = r.resolveK8sGateway(ctx, policy, initState)
			}
		}
		if err != nil {
			return nil, err
		}
	}

	if config.EnableEmbeddedMode() {
		// Some of our users use embedded policy mostly, so it's fine to list all
		var virtualServices istiov1a3.VirtualServiceList
		if err := r.List(ctx, &virtualServices); err != nil {
			return nil, fmt.Errorf("failed to list VirtualService: %w", err)
		}

		for _, vs := range virtualServices.Items {
			policy := getFilterPolicyFromAnnotation(vs)
			if policy == nil {
				continue
			}
			// We require the embedded policy to be valid, otherwise it's costly to validate and hard to report the error.
			policy.Namespace = vs.Namespace
			// Name convention is "embedded-$kind-$name"
			policy.Name = "embedded-virtualservice-" + vs.Name
			err := r.resolveWithVirtualService(ctx, vs, policy, initState, istioGwIdx)
			if err != nil {
				return nil, err
			}

			key := getK8sKey(vs.Namespace, vs.Name)
			vsIdx[key] = append(vsIdx[key], policy)
		}

		r.virtualServiceIndexer.UpdateIndex(vsIdx)
	}

	if config.EnableLDSPluginViaECDS() {
		var gateways istiov1a3.GatewayList
		if err := r.List(ctx, &gateways); err != nil {
			return nil, fmt.Errorf("failed to list Istio Gateway: %w", err)
		}

		if config.EnableEmbeddedMode() {
			for _, gw := range gateways.Items {
				policy := getFilterPolicyFromAnnotation(gw)
				if policy == nil {
					continue
				}

				// We require the embedded policy to be valid, otherwise it's costly to validate and hard to report the error.
				policy.Namespace = gw.Namespace
				// Name convention is "embedded-$kind-$name"
				policy.Name = "embedded-gateway-" + gw.Name
				err := r.resolveWithIstioGateway(ctx, gw, policy, initState)
				if err != nil {
					return nil, err
				}

				key := getK8sKey(gw.Namespace, gw.Name)
				istioGwIdx[key] = append(istioGwIdx[key], policy)
			}

			// istioGatewayIndexer will be updated at the end of this method
		}

		for _, gw := range gateways.Items {
			initState.AddIstioGateway(gw)
		}

		if config.EnableGatewayAPI() {
			var k8sGateways gwapiv1b1.GatewayList
			if err := r.List(ctx, &k8sGateways); err != nil {
				return nil, fmt.Errorf("failed to list k8s Gateway: %w", err)
			}

			for i := range k8sGateways.Items {
				initState.AddK8sGateway(&k8sGateways.Items[i])
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

func (r *FilterPolicyReconciler) updatePolicies(ctx context.Context,
	policies *mosniov1.FilterPolicyList) error {

	for i := range policies.Items {
		policy := &policies.Items[i]
		// track changed status will be a little faster than iterating policies
		// but make code much complex
		if !policy.Status.IsChanged() {
			continue
		}

		if policy.FromHTTPFilterPolicy() {
			// The policy is generated by HTTPFilterPolicy
			p := mosniov1.ConvertFilterPolicyToHTTPFilterPolicy(policy)
			if err := r.UpdateStatus(ctx, p.DeepCopy(), &p.Status); err != nil {
				return fmt.Errorf("failed to update HTTPFilterPolicy status: %w, namespacedName: %v",
					err, types.NamespacedName{Name: p.Name, Namespace: p.Namespace})
			}
			continue
		}

		policy.Status.Reset()
		// DeepCopy is used to avoid data race in FindAffectedObjects
		if err := r.UpdateStatus(ctx, policy.DeepCopy(), &policy.Status); err != nil {
			return fmt.Errorf("failed to update FilterPolicy status: %w, namespacedName: %v",
				err, types.NamespacedName{Name: policy.Name, Namespace: policy.Namespace})
		}
	}
	return nil
}

// customResourceIndexer indexes the additional customer resource
// according to the reconciled customer resource
type customResourceIndexer struct {
	lock  sync.RWMutex
	index map[string][]*mosniov1.FilterPolicy

	Group          string
	Kind           string
	CustomResource client.Object
}

func (v *customResourceIndexer) UpdateIndex(idx map[string][]*mosniov1.FilterPolicy) {
	v.lock.Lock()
	v.index = idx
	v.lock.Unlock()
}

func (v *customResourceIndexer) FindAffectedObjects(ctx context.Context, obj component.ResourceMeta) []reconcile.Request {
	if config.EnableEmbeddedMode() {
		ann := obj.GetAnnotations()
		if ann != nil {
			if ann[constant.AnnotationFilterPolicy] != "" {
				log.Infof("Target with embedded FilterPolicy changed, trigger reconciliation, group: %s, kind: %s, namespace: %s, name: %s",
					obj.GetGroup(), obj.GetKind(), obj.GetNamespace(), obj.GetName())
				return triggerReconciliation()
			}
			if ann[constant.AnnotationHTTPFilterPolicy] != "" {
				log.Infof("Target with embedded HTTPFilterPolicy changed, trigger reconciliation, group: %s, kind: %s, namespace: %s, name: %s",
					obj.GetGroup(), obj.GetKind(), obj.GetNamespace(), obj.GetName())
				return triggerReconciliation()
			}
		}
	}

	if config.EnableLDSPluginViaECDS() {
		if (v.Group == "networking.istio.io" || v.Group == "gateway.networking.k8s.io") && v.Kind == "Gateway" {
			log.Infof("Resource changed, trigger reconciliation, group: %s, kind: %s, namespace: %s, name: %s",
				obj.GetGroup(), obj.GetKind(), obj.GetNamespace(), obj.GetName())
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

	log.Infof("Target changed, trigger reconciliation, group: %s, kind: %s, namespace: %s, name: %s",
		obj.GetGroup(), obj.GetKind(), obj.GetNamespace(), obj.GetName())

	return triggerReconciliation()
}

type resourceMetaWrapper struct {
	client.Object

	group string
	kind  string
}

func (r *resourceMetaWrapper) GetGroup() string {
	return r.group
}

func (r *resourceMetaWrapper) GetKind() string {
	return r.kind
}

func wrapClientObjectToResourceMeta(obj client.Object, group, kind string) component.ResourceMeta {
	return &resourceMetaWrapper{
		Object: obj,
		group:  group,
		kind:   kind,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *FilterPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	controller := ctrl.NewControllerManagedBy(mgr).
		Named("filterpolicy").
		Watches(
			&mosniov1.FilterPolicy{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, _ client.Object) []reconcile.Request {
				return triggerReconciliation()
			}),
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
			),
		)
		// We don't reconcile when the generated EnvoyFilter is modified.
		// So that user can manually correct the EnvoyFilter, until something else is changed.

	pred := predicate.Or(
		predicate.GenerationChangedPredicate{},
		predicate.AnnotationChangedPredicate{},
	)
	for name, idxer := range r.indexers {
		idxer := idxer
		ss := strings.Split(name, "/")
		group := ss[0]
		kind := ss[1]
		controller.Watches(
			idxer.CustomResource,
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				return idxer.FindAffectedObjects(ctx, wrapClientObjectToResourceMeta(obj, group, kind))
			}),
			builder.WithPredicates(pred),
		)
	}

	return controller.Complete(r)
}
