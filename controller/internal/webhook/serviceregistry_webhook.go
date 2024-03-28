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

package webhook

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	mosniov1 "mosn.io/htnn/types/apis/v1"
)

// log is for logging in this package.
var serviceregistrylog = logf.Log.WithName("serviceregistry-resource")

type ServiceRegistryWebhook struct {
	mosniov1.ServiceRegistry
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *ServiceRegistryWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&r.ServiceRegistry).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-htnn.mosn.io-v1-serviceregistry,mutating=true,failurePolicy=fail,sideEffects=None,groups=mosn.io,resources=serviceregistries,verbs=create;update,versions=v1,name=mserviceregistry.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ServiceRegistryWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ServiceRegistryWebhook) Default() {

}

//+kubebuilder:webhook:path=/validate-htnn.mosn.io-v1-serviceregistry,mutating=false,failurePolicy=fail,sideEffects=None,groups=mosn.io,resources=serviceregistries,verbs=create;update;delete,versions=v1,name=vserviceregistry.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ServiceRegistryWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceRegistryWebhook) ValidateCreate() (admission.Warnings, error) {
	serviceregistrylog.Info("validate create", "name", r.Name)

	return nil, r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceRegistryWebhook) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	serviceregistrylog.Info("validate update", "name", r.Name)

	return nil, r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceRegistryWebhook) ValidateDelete() (admission.Warnings, error) {
	serviceregistrylog.Info("validate delete", "name", r.Name)

	return nil, nil
}

func (r *ServiceRegistryWebhook) validate() error {
	var allErrs field.ErrorList
	if err := mosniov1.ValidateServiceRegistry(&r.ServiceRegistry); err != nil {
		fieldErr := field.Invalid(field.NewPath("spec"), r.Spec, err.Error())
		allErrs = append(allErrs, fieldErr)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "htnn.mosn.io", Kind: "ServiceRegistry"},
		r.Name, allErrs)
}
