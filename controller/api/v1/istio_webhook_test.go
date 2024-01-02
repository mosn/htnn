package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	istioapi "istio.io/api/networking/v1beta1"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVirtualServiceWebhookAddName(t *testing.T) {
	r := &VirtualServiceWebhook{
		VirtualService: &istiov1b1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "name",
			},
			Spec: istioapi.VirtualService{
				Hosts: []string{"test"},
				Http: []*istioapi.HTTPRoute{
					{}, {},
				},
			},
		},
	}
	r.Default()
	assert.Equal(t, "ns.name", r.Spec.Http[0].Name)
	assert.Equal(t, "ns.name", r.Spec.Http[1].Name)
}
