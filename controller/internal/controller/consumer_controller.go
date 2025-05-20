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
	"sync"
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

// ConsumerReconciler reconciles a Consumer object
type ConsumerReconciler struct {
	component.ResourceManager
	Output   component.Output
	keyIndex *KeyIndexRegistry // Add a new Key index
}

type KeyIndexRegistry struct {
	mu    sync.RWMutex
	index map[string]map[string]map[string]string // ns -> plugin -> key -> consumerName
}

func NewKeyIndexRegistry() *KeyIndexRegistry {
	return &KeyIndexRegistry{
		index: make(map[string]map[string]map[string]string),
		mu:    sync.RWMutex{},
	}
}

//+kubebuilder:rbac:groups=htnn.mosn.io,resources=consumers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=consumers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=htnn.mosn.io,resources=consumers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ConsumerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconcilationStart := time.Now()
	defer func() {
		reconcilationDuration := time.Since(reconcilationStart).Seconds()
		metrics.ConsumerReconcileDurationDistribution.Record(reconcilationDuration)
	}()

	log.Info("Reconcile Consumer")

	var consumers mosniov1.ConsumerList
	state, err := r.consumersToState(ctx, &consumers)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.generateCustomResource(ctx, state)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.updateConsumers(ctx, &consumers)
	return ctrl.Result{}, err
}

type consumerReconcileState struct {
	namespaceToConsumers map[string]map[string]*mosniov1.Consumer
}

func (r *ConsumerReconciler) consumersToState(ctx context.Context,
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
				log.Errorf("invalid Consumer, err: %v, name: %s, namespace: %s", err, consumer.Name, consumer.Namespace)
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

		name := consumer.Name
		if consumer.Spec.Name != "" {
			name = consumer.Spec.Name
		}

		if namespaceToConsumers[namespace][name] != nil {
			log.Errorf("duplicate Consumer %s/%s, k8s name %s takes effect, k8s name %s ignored", namespace, name,
				namespaceToConsumers[namespace][name].Name, consumer.Name)
			consumer.SetAccepted(mosniov1.ReasonInvalid,
				fmt.Sprintf("duplicate with another consumer %s/%s, k8s name %s", namespace, name, consumer.Name))
		} else {
			namespaceToConsumers[namespace][name] = consumer
			consumer.SetAccepted(mosniov1.ReasonAccepted)
		}
	}

	state := &consumerReconcileState{
		namespaceToConsumers: namespaceToConsumers,
	}
	return state, nil
}

func (r *ConsumerReconciler) generateCustomResource(ctx context.Context, state *consumerReconcileState) error {
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

	return r.Output.FromConsumer(ctx, ef)
}

func (r *ConsumerReconciler) updateConsumers(ctx context.Context, consumers *mosniov1.ConsumerList) error {
	for i := range consumers.Items {
		consumer := &consumers.Items[i]
		if !consumer.Status.IsChanged() {
			continue
		}
		consumer.Status.Reset()
		if err := r.UpdateStatus(ctx, consumer, &consumer.Status); err != nil {
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

// checkConsumerConflicts Perform full conflict detection and rebuild the index
func (r *ConsumerReconciler) checkConsumerConflicts(ctx context.Context) error {
	r.keyIndex.mu.Lock()
	defer r.keyIndex.mu.Unlock()

	// Clear old indexes
	r.keyIndex.index = make(map[string]map[string]map[string]string)

	// Get all Consumer
	var allConsumers mosniov1.ConsumerList
	if err := r.ResourceManager.List(ctx, &allConsumers); err != nil {
		return fmt.Errorf("list consumers failed: %w", err)
	}

	// Perforated detection
	for _, consumer := range allConsumers.Items {
		if !consumer.DeletionTimestamp.IsZero() {
			continue
		}

		if err := r.indexConsumer(consumer); err != nil {
			return fmt.Errorf("consumer %s/%s: %w", consumer.Namespace, consumer.Name, err)
		}
	}
	return nil
}

// indexConsumer Index a single Consumer and detect conflicts
func (r *ConsumerReconciler) indexConsumer(consumer mosniov1.Consumer) error {
	for pluginName, plugin := range consumer.Spec.Auth {
		key, err := extractPluginKey(plugin.Config.Raw)
		if err != nil {
			return fmt.Errorf("invalid plugin %s config: %w", pluginName, err)
		}

		// Inertly initialize the three-level map
		if r.keyIndex.index[consumer.Namespace] == nil {
			r.keyIndex.index[consumer.Namespace] = make(map[string]map[string]string)
		}
		if r.keyIndex.index[consumer.Namespace][pluginName] == nil {
			r.keyIndex.index[consumer.Namespace][pluginName] = make(map[string]string)
		}

		// collision detect
		if existing, exists := r.keyIndex.index[consumer.Namespace][pluginName][key]; exists {
			return fmt.Errorf(
				"key conflict: plugin=%s key=%s (already used by %s)",
				pluginName, key, existing,
			)
		}
		r.keyIndex.index[consumer.Namespace][pluginName][key] = consumer.Name
	}
	return nil
}

func extractPluginKey(raw []byte) (string, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", err
	}

	if key, ok := cfg["key"].(string); ok {
		return key, nil
	}
	if iss, ok := cfg["issuer"].(string); ok {
		return iss, nil
	}
	return "", errors.New("no valid key field found")
}
