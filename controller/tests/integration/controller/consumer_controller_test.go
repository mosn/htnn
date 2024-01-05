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

package controller

import (
	"context"
	"encoding/json"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	"mosn.io/htnn/controller/tests/pkg"
)

func mustReadConsumer(fn string, out *[]map[string]interface{}) {
	fn = filepath.Join("testdata", "consumer", fn+".yml")
	mustReadInput(fn, out)
}

var _ = Describe("Consumer controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When reconciling Consumer", func() {
		BeforeEach(func() {
			var policies mosniov1.HTTPFilterPolicyList
			if err := k8sClient.List(ctx, &policies); err == nil {
				for _, e := range policies.Items {
					Expect(k8sClient.Delete(ctx, &e)).Should(Succeed())
				}
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			if err := k8sClient.List(ctx, &envoyfilters); err == nil {
				for _, e := range envoyfilters.Items {
					Expect(k8sClient.Delete(ctx, e)).Should(Succeed())
				}
			}
		})

		It("deal with crd", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadConsumer("consumer", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 1
			}, timeout, interval).Should(BeTrue())

			ef := envoyfilters.Items[0]
			Expect(ef.Namespace).To(Equal("istio-system"))
			Expect(ef.Name).To(Equal("htnn-consumer"))
			Expect(len(ef.Spec.ConfigPatches)).To(Equal(1))
			cp := ef.Spec.ConfigPatches[0]
			Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_EXTENSION_CONFIG))
			value := cp.Patch.Value.AsMap()
			Expect(value["name"]).To(Equal("htnn-consumer"))
			typedCfg := value["typed_config"].(map[string]interface{})
			pluginCfg := typedCfg["plugin_config"].(map[string]interface{})

			marshaledCfg := map[string]map[string]interface{}{}
			b, _ := json.Marshal(pluginCfg["value"])
			json.Unmarshal(b, &marshaledCfg)
			// mapping is namespace -> name -> config
			Expect(marshaledCfg["default"]["spacewander"]).ToNot(BeNil())
		})
	})
})
