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

// log is for logging webhook in this package.
var log = logf.Log.WithName("webhook")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *HTTPFilterPolicy) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-htnn.mosn.io-v1-httpfilterpolicy,mutating=true,failurePolicy=fail,sideEffects=None,groups=mosn.io,resources=httpfilterpolicies,verbs=create;update,versions=v1,name=mhttpfilterpolicy.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &HTTPFilterPolicy{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *HTTPFilterPolicy) Default() {
}

//+kubebuilder:webhook:path=/validate-htnn.mosn.io-v1-httpfilterpolicy,mutating=false,failurePolicy=fail,sideEffects=None,groups=mosn.io,resources=httpfilterpolicies,verbs=create;update;delete,versions=v1,name=vhttpfilterpolicy.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &HTTPFilterPolicy{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *HTTPFilterPolicy) ValidateCreate() (admission.Warnings, error) {
	log.Info("validate create", "name", r.Name)

	// The generated webhook doesn't allow querying k8s, so we can't check if the resource referred by the HTTPFilterPolicy exists or valid.
	// We can use custom validator instead of the one generated by kubebuilder if needed.

	return nil, r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *HTTPFilterPolicy) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	log.Info("validate update", "name", r.Name)

	return nil, r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *HTTPFilterPolicy) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func (r *HTTPFilterPolicy) validate() error {
	var allErrs field.ErrorList
	if err := ValidateHTTPFilterPolicy(r); err != nil {
		fieldErr := field.Invalid(field.NewPath("spec"), r.Spec, err.Error())
		allErrs = append(allErrs, fieldErr)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "htnn.mosn.io", Kind: "HTTPFilterPolicy"},
		r.Name, allErrs)
}
