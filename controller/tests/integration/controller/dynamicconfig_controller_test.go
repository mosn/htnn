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
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/controller/tests/integration/helper"
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func mustReadDynamicConfig(fn string, out *[]map[string]interface{}) {
	fn = filepath.Join("testdata", "dynamicconfig", fn+".yml")
	helper.MustReadInput(fn, out)
}

var _ = Describe("DynamicConfig controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	AfterEach(func() {
		var configs mosniov1.DynamicConfigList
		if err := k8sClient.List(ctx, &configs); err == nil {
			for _, e := range configs.Items {
				pkg.DeleteK8sResource(ctx, k8sClient, &e)
			}
		}

		var envoyfilters istiov1a3.EnvoyFilterList
		if err := k8sClient.List(ctx, &envoyfilters); err == nil {
			for _, e := range envoyfilters.Items {
				pkg.DeleteK8sResource(ctx, k8sClient, e)
			}
		}
	})

	Context("When reconciling DynamicConfig", func() {
		It("deal with crd", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadDynamicConfig("dynamicconfig", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var configs mosniov1.DynamicConfigList
			var c *mosniov1.DynamicConfig
			var cs []metav1.Condition
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &configs); err != nil {
					return false
				}
				handled := len(configs.Items) == 1
				for _, item := range configs.Items {
					item := item
					if item.Name == "test" {
						c = &item
						cs = c.Status.Conditions
					}
					conds := item.Status.Conditions
					if len(conds) != 1 {
						handled = false
						break
					}
				}

				return handled
			}, timeout, interval).Should(BeTrue())
			Expect(c).ToNot(BeNil())
			Expect(cs[0].Type).To(Equal(string(mosniov1.ConditionAccepted)))
			Expect(cs[0].Reason).To(Equal(string(mosniov1.ReasonAccepted)))

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				for _, item := range envoyfilters.Items {
					if item.Name == "htnn-dynamic-config" {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			// to invalid
			base := client.MergeFrom(c.DeepCopy())
			prevType := c.Spec.Type
			c.Spec.Type = "unknown"
			Expect(k8sClient.Patch(ctx, c, base)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &configs); err != nil {
					return false
				}
				for _, item := range configs.Items {
					item := item
					if item.Name == "test" {
						c = &item
						cs = c.Status.Conditions
						if cs[0].Reason == string(mosniov1.ReasonInvalid) {
							return true
						}
					}
				}

				return false
			}, timeout, interval).Should(BeTrue())

			// EnvoyFilter should be updated too
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				for _, item := range envoyfilters.Items {
					if item.Name == "htnn-dynamic-config" {
						return false
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())

			// back to valid
			base = client.MergeFrom(c.DeepCopy())
			c.Spec.Type = prevType
			Expect(k8sClient.Patch(ctx, c, base)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &configs); err != nil {
					return false
				}
				for _, item := range configs.Items {
					item := item
					if item.Name == "test" {
						c = &item
						cs = c.Status.Conditions
						if cs[0].Reason == string(mosniov1.ReasonAccepted) {
							return true
						}
					}
				}

				return false
			}, timeout, interval).Should(BeTrue())

			// EnvoyFilter should be updated too
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				for _, item := range envoyfilters.Items {
					if item.Name == "htnn-dynamic-config" {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})
	})
})
