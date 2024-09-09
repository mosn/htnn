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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"mosn.io/htnn/controller/internal/istio"
	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/internal/metrics"
	"mosn.io/htnn/controller/pkg/component"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

// DynamicConfigReconciler reconciles a DynamicConfig object
type DynamicConfigReconciler struct {
	component.ResourceManager
	Output component.Output
}

//+kubebuilder:rbac:groups=htnn.mosn.io,resources=dynamicconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=dynamicconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=dynamicconfigs/finalizers,verbs=update

func (r *DynamicConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconcilationStart := time.Now()
	defer func() {
		reconcilationDuration := time.Since(reconcilationStart).Seconds()
		metrics.DynamicConfigReconcileDurationDistribution.Record(reconcilationDuration)
	}()

	log.Info("Reconcile DynamicConfig")

	var dynamicConfigs mosniov1.DynamicConfigList
	state, err := r.dynamicconfigsToState(ctx, &dynamicConfigs)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.generateCustomResource(ctx, state)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.updateDynamicConfigs(ctx, &dynamicConfigs)
	return ctrl.Result{}, err
}

type dynamicConfigReconcileState struct {
	namespaceToDynamicConfigs map[string]map[string]*mosniov1.DynamicConfig
}

func (r *DynamicConfigReconciler) dynamicconfigsToState(ctx context.Context,
	dynamicConfigs *mosniov1.DynamicConfigList) (*dynamicConfigReconcileState, error) {

	if err := r.List(ctx, dynamicConfigs); err != nil {
		return nil, fmt.Errorf("failed to list DynamicConfig: %w", err)
	}

	namespaceToDynamicConfigs := make(map[string]map[string]*mosniov1.DynamicConfig)
	for i := range dynamicConfigs.Items {
		dynamicConfig := &dynamicConfigs.Items[i]

		// defensive code in case the webhook doesn't work
		if dynamicConfig.IsSpecChanged() {
			err := mosniov1.ValidateDynamicConfig(dynamicConfig)
			if err != nil {
				log.Errorf("invalid DynamicConfig, err: %v, name: %s, namespace: %s", err, dynamicConfig.Name, dynamicConfig.Namespace)
				dynamicConfig.SetAccepted(mosniov1.ReasonInvalid, err.Error())
				continue
			}
		}
		if !dynamicConfig.IsValid() {
			continue
		}

		namespace := dynamicConfig.Namespace
		if namespaceToDynamicConfigs[namespace] == nil {
			namespaceToDynamicConfigs[namespace] = make(map[string]*mosniov1.DynamicConfig)
		}

		name := dynamicConfig.Spec.Type
		if namespaceToDynamicConfigs[namespace][name] != nil {
			log.Errorf("duplicate DynamicConfig %s/%s, k8s name %s takes effect, k8s name %s ignored", namespace, name,
				namespaceToDynamicConfigs[namespace][name].Name, dynamicConfig.Name)
			dynamicConfig.SetAccepted(mosniov1.ReasonInvalid,
				fmt.Sprintf("duplicate with another DynamicConfig %s/%s, k8s name %s", namespace, name, dynamicConfig.Name))
		} else {
			namespaceToDynamicConfigs[namespace][name] = dynamicConfig
			dynamicConfig.SetAccepted(mosniov1.ReasonAccepted)
		}
	}

	state := &dynamicConfigReconcileState{
		namespaceToDynamicConfigs: namespaceToDynamicConfigs,
	}
	return state, nil
}

func (r *DynamicConfigReconciler) generateCustomResource(ctx context.Context, state *dynamicConfigReconcileState) error {
	efs := istio.GenerateDynamicConfigs(state.namespaceToDynamicConfigs)
	return r.Output.FromDynamicConfig(ctx, efs)
}

func (r *DynamicConfigReconciler) updateDynamicConfigs(ctx context.Context, dynamicConfigs *mosniov1.DynamicConfigList) error {
	for i := range dynamicConfigs.Items {
		dynamicConfig := &dynamicConfigs.Items[i]
		if !dynamicConfig.Status.IsChanged() {
			continue
		}
		dynamicConfig.Status.Reset()
		if err := r.UpdateStatus(ctx, dynamicConfig, &dynamicConfig.Status); err != nil {
			return fmt.Errorf("failed to update DynamicConfig status: %w, namespacedName: %v",
				err,
				types.NamespacedName{Name: dynamicConfig.Name, Namespace: dynamicConfig.Namespace})
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DynamicConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	controller := ctrl.NewControllerManagedBy(mgr).
		Named("dynamicConfig").
		Watches(
			&mosniov1.DynamicConfig{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, _ client.Object) []reconcile.Request {
				return triggerReconciliation()
			}),
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
			),
		)
	return controller.Complete(r)
}
