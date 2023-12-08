package v1

import (
	"fmt"

	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type VirtualServiceWebhook struct {
	*istiov1b1.VirtualService
}

func (r *VirtualServiceWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-networking-istio-io-v1beta1-virtualservice,mutating=true,failurePolicy=fail,sideEffects=None,groups=networking.istio.io,resources=virtualservices,verbs=create;update,versions=v1beta1,name=mvirtualservice.kb.io,admissionReviewVersions=v1

func newVirtualServiceWebhook() *VirtualServiceWebhook {
	return &VirtualServiceWebhook{
		VirtualService: &istiov1b1.VirtualService{},
	}
}

var _ webhook.Defaulter = newVirtualServiceWebhook()
var _ webhook.Defaulter = newVirtualServiceWebhook().DeepCopyObject().(webhook.Defaulter)

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *VirtualServiceWebhook) Default() {
	log.Info("update VirtualService", "name", r.Name, "namespace", r.Namespace)

	for _, httpRoute := range r.Spec.Http {
		if httpRoute.Name == "" {
			// The generated name is designed not be referred by Policy's SectionName.
			// If you need to refer to it, you need to specify the name by yourself.
			httpRoute.Name = fmt.Sprintf("%s/%s", r.Namespace, r.Name)
		}
		// If the name is specified by user, the same route name should not be used in different VirtualServices
		// which share the same host.
	}
}

func (r *VirtualServiceWebhook) DeepCopyObject() runtime.Object {
	return &VirtualServiceWebhook{r.VirtualService.DeepCopyObject().(*istiov1b1.VirtualService)}
}

func RegisterVirtualServiceWebhook(mgr ctrl.Manager) {
	wh := admission.DefaultingWebhookFor(mgr.GetScheme(), newVirtualServiceWebhook()).WithRecoverPanic(true)
	mgr.GetWebhookServer().Register("/mutate-networking-istio-io-v1beta1-virtualservice", wh)
}
