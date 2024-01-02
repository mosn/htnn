// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
