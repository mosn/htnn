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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

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
	state, err := r.consumersToState(ctx, &consumers)
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
	namespaceToConsumers map[string]map[string]*mosniov1.ConsumerSpec
}

func (r *ConsumerReconciler) consumersToState(ctx context.Context, consumers *mosniov1.ConsumerList) (*consumerReconcileState, error) {
	if err := r.List(ctx, consumers); err != nil {
		return nil, fmt.Errorf("failed to list Consumer: %w", err)
	}

	namespaceToConsumers := make(map[string]map[string]*mosniov1.ConsumerSpec)
	for i := range consumers.Items {
		consumer := &consumers.Items[i]
		namespace := consumer.Namespace
		if namespaceToConsumers[namespace] == nil {
			namespaceToConsumers[namespace] = make(map[string]*mosniov1.ConsumerSpec)
		}
		namespaceToConsumers[namespace][consumer.Name] = &consumer.Spec
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
			cfg := map[string]interface{}{}
			for name, conf := range consumer.Auth {
				cfg[name] = string(conf.Raw)
			}
			data[consumerName] = cfg
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
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mosniov1.Consumer{}).
		Complete(r)
}
