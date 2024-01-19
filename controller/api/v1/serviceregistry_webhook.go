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

package v1

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var serviceregistrylog = logf.Log.WithName("serviceregistry-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *ServiceRegistry) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-mosn-io-v1-serviceregistry,mutating=true,failurePolicy=fail,sideEffects=None,groups=mosn.io,resources=serviceregistries,verbs=create;update,versions=v1,name=mserviceregistry.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ServiceRegistry{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ServiceRegistry) Default() {

}

//+kubebuilder:webhook:path=/validate-mosn-io-v1-serviceregistry,mutating=false,failurePolicy=fail,sideEffects=None,groups=mosn.io,resources=serviceregistries,verbs=create;update;delete,versions=v1,name=vserviceregistry.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ServiceRegistry{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceRegistry) ValidateCreate() (admission.Warnings, error) {
	serviceregistrylog.Info("validate create", "name", r.Name)

	return nil, r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceRegistry) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	serviceregistrylog.Info("validate update", "name", r.Name)

	return nil, r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceRegistry) ValidateDelete() (admission.Warnings, error) {
	serviceregistrylog.Info("validate delete", "name", r.Name)

	return nil, nil
}

func (r *ServiceRegistry) validate() error {
	var allErrs field.ErrorList
	if err := ValidateServiceRegistry(r); err != nil {
		fieldErr := field.Invalid(field.NewPath("spec"), r.Spec, err.Error())
		allErrs = append(allErrs, fieldErr)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "mosn.io", Kind: "ServiceRegistry"},
		r.Name, allErrs)
}
