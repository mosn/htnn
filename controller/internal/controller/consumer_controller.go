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

	"github.com/go-logr/logr"
	"google.golang.org/protobuf/proto"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/istio"
)

// ConsumerReconciler reconciles a Consumer object
type ConsumerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	ConsumerEnvoyFilterName = "htnn-consumer"
)

//+kubebuilder:rbac:groups=mosn.io,resources=consumers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mosn.io,resources=consumers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mosn.io,resources=consumers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ConsumerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile")

	var consumers mosniov1.ConsumerList
	state, err := r.consumersToState(ctx, &logger, &consumers)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.generateCustomResource(ctx, &logger, state)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.updateConsumers(ctx, &consumers)
	return ctrl.Result{}, err
}

type consumerReconcileState struct {
	namespaceToConsumers map[string]map[string]*mosniov1.Consumer
}

func (r *ConsumerReconciler) consumersToState(ctx context.Context, logger *logr.Logger,
	consumers *mosniov1.ConsumerList) (*consumerReconcileState, error) {

	if err := r.List(ctx, consumers); err != nil {
		return nil, fmt.Errorf("failed to list Consumer: %w", err)
	}

	namespaceToConsumers := make(map[string]map[string]*mosniov1.Consumer)
	for i := range consumers.Items {
		consumer := &consumers.Items[i]

		// defensive code in case the webhook doesn't work
		if consumer.IsSpecChanged() {
			err := mosniov1.ValidateConsumer(consumer)
			if err != nil {
				logger.Error(err, "invalid Consumer", "name", consumer.Name, "namespace", consumer.Namespace)
				consumer.SetAccepted(mosniov1.ReasonInvalid, err.Error())
				continue
			}
		}
		if !consumer.IsValid() {
			continue
		}

		namespace := consumer.Namespace
		if namespaceToConsumers[namespace] == nil {
			namespaceToConsumers[namespace] = make(map[string]*mosniov1.Consumer)
		}
		namespaceToConsumers[namespace][consumer.Name] = consumer

		consumer.SetAccepted(mosniov1.ReasonAccepted)
	}

	state := &consumerReconcileState{
		namespaceToConsumers: namespaceToConsumers,
	}
	return state, nil
}

func (r *ConsumerReconciler) generateCustomResource(ctx context.Context, logger *logr.Logger, state *consumerReconcileState) error {
	consumerData := map[string]interface{}{}
	for ns, consumers := range state.namespaceToConsumers {
		data := make(map[string]interface{}, len(consumers))
		for consumerName, consumer := range consumers {
			s := consumer.Marshal()
			data[consumerName] = map[string]interface{}{
				"d": s,
				// only track the change of the Spec, so we use Generation here
				"v": consumer.Generation,
			}
		}
		consumerData[ns] = data
	}

	ef := istio.GenerateConsumers(consumerData)
	ef.Namespace = config.RootNamespace()
	ef.Name = ConsumerEnvoyFilterName
	if ef.Labels == nil {
		ef.Labels = map[string]string{}
	}
	ef.Labels[LabelCreatedBy] = "Consumer"

	var envoyfilter istiov1a3.EnvoyFilter
	nsName := types.NamespacedName{Name: ef.Name, Namespace: ef.Namespace}
	err := r.Get(ctx, nsName, &envoyfilter)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get EnvoyFilter: %w, namespacedName: %v", err, nsName)
		}

		logger.Info("create EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)

		if err = r.Create(ctx, ef); err != nil {
			return fmt.Errorf("failed to create EnvoyFilter: %w, namespacedName: %v", err, nsName)
		}
	} else if !proto.Equal(&envoyfilter.Spec, &ef.Spec) {
		logger.Info("update EnvoyFilter", "name", ef.Name, "namespace", ef.Namespace)

		ef.SetResourceVersion(envoyfilter.ResourceVersion)
		if err = r.Update(ctx, ef); err != nil {
			return fmt.Errorf("failed to update EnvoyFilter: %w, namespacedName: %v", err, nsName)
		}
	}

	return nil
}

func (r *ConsumerReconciler) updateConsumers(ctx context.Context, consumers *mosniov1.ConsumerList) error {
	for i := range consumers.Items {
		consumer := &consumers.Items[i]
		if !consumer.Status.IsChanged() {
			continue
		}
		// Update operation will change the original object in cache, so we need to deepcopy it.
		if err := r.Status().Update(ctx, consumer.DeepCopy()); err != nil {
			return fmt.Errorf("failed to update Consumer status: %w, namespacedName: %v",
				err,
				types.NamespacedName{Name: consumer.Name, Namespace: consumer.Namespace})
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	controller := ctrl.NewControllerManagedBy(mgr).
		Named("consumer").
		Watches(
			&mosniov1.Consumer{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, _ client.Object) []reconcile.Request {
				return triggerReconciliation()
			}),
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
			),
		)
	return controller.Complete(r)
}
