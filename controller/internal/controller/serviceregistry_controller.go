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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	"mosn.io/htnn/controller/internal/registry"
)

// ServiceRegistryReconciler reconciles a ServiceRegistry object
type ServiceRegistryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=mosn.io,resources=serviceregistries,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mosn.io,resources=serviceregistries/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mosn.io,resources=serviceregistries/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.istio.io,resources=serviceentries,verbs=get;list;watch;update;patch;delete

func (r *ServiceRegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var serviceRegistry mosniov1.ServiceRegistry
	nsName := types.NamespacedName{Name: req.Name, Namespace: req.Namespace}
	err := r.Get(ctx, nsName, &serviceRegistry)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to get ServiceRegistry: %w, namespacedName: %v", err, nsName)
		}

		logger.Info("delete ServiceRegistry")
		err = registry.DeleteRegistry(nsName)

	} else {
		logger.Info("update ServiceRegistry")
		err = registry.UpdateRegistry(&serviceRegistry)
	}

	if err != nil {
		logger.Error(err, "failed to operate registry")
		// don't retry if the err is caused by registry
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceRegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mosniov1.ServiceRegistry{}).
		Complete(r)
}
