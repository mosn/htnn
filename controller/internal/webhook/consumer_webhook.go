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
var consumerlog = logf.Log.WithName("consumer-resource")

type ConsumerWebhook struct {
	mosniov1.Consumer
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *ConsumerWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&r.Consumer).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-htnn.mosn.io-v1-consumer,mutating=true,failurePolicy=fail,sideEffects=None,groups=mosn.io,resources=consumers,verbs=create;update,versions=v1,name=mconsumer.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ConsumerWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ConsumerWebhook) Default() {
}

//+kubebuilder:webhook:path=/validate-htnn.mosn.io-v1-consumer,mutating=false,failurePolicy=fail,sideEffects=None,groups=mosn.io,resources=consumers,verbs=create;update;delete,versions=v1,name=vconsumer.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ConsumerWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ConsumerWebhook) ValidateCreate() (admission.Warnings, error) {
	consumerlog.Info("validate create", "name", r.Name)

	return nil, r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ConsumerWebhook) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	consumerlog.Info("validate update", "name", r.Name)

	return nil, r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ConsumerWebhook) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func (r *ConsumerWebhook) validate() error {
	var allErrs field.ErrorList
	if err := mosniov1.ValidateConsumer(&r.Consumer); err != nil {
		fieldErr := field.Invalid(field.NewPath("spec"), r.Spec, err.Error())
		allErrs = append(allErrs, fieldErr)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "htnn.mosn.io", Kind: "Consumer"},
		r.Name, allErrs)
}
