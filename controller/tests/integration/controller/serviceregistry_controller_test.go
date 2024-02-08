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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	"mosn.io/htnn/controller/tests/integration/helper"
	"mosn.io/htnn/controller/tests/pkg"
)

func mustReadServiceRegistry(fn string, out *[]map[string]interface{}) {
	fn = filepath.Join("testdata", "serviceregistry", fn+".yml")
	helper.MustReadInput(fn, out)
}

var _ = Describe("ServiceRegistry controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When reconciling ServiceRegistry", func() {
		BeforeEach(func() {
			var registries mosniov1.ServiceRegistryList
			if err := k8sClient.List(ctx, &registries); err == nil {
				for _, e := range registries.Items {
					Expect(k8sClient.Delete(ctx, &e)).Should(Succeed())
				}
			}

			helper.WaitServiceUp(":8848", "Nacos")
		})

		It("deal with invalid serviceregistry crd", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadServiceRegistry("invalid", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var registries mosniov1.ServiceRegistryList
			var r *mosniov1.ServiceRegistry
			var cs []metav1.Condition
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &registries); err != nil {
					return false
				}
				for _, item := range registries.Items {
					item := item
					if item.Name == "invalid" {
						r = &item
						cs = r.Status.Conditions
					}
				}

				return len(cs) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(cs[0].Type).To(Equal(string(mosniov1.ConditionAccepted)))
			Expect(cs[0].Reason).To(Equal(string(mosniov1.ReasonInvalid)))
		})

		It("deal with serviceregistry crd", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadServiceRegistry("default", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var registries mosniov1.ServiceRegistryList
			var r *mosniov1.ServiceRegistry
			var cs []metav1.Condition
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &registries); err != nil {
					return false
				}
				for _, item := range registries.Items {
					item := item
					if item.Name == "earth" {
						r = &item
						cs = r.Status.Conditions
					}
				}

				return len(cs) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(cs[0].Type).To(Equal(string(mosniov1.ConditionAccepted)))
			Expect(cs[0].Reason).To(Equal(string(mosniov1.ReasonAccepted)))

			// to invalid
			base := client.MergeFrom(r.DeepCopy())
			r.Spec.Config.Raw = []byte(`{}`)
			Expect(k8sClient.Patch(ctx, r, base)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &registries); err != nil {
					return false
				}
				for _, item := range registries.Items {
					if item.Name == "earth" {
						r = &registries.Items[0]
						cs = r.Status.Conditions
						return cs[0].Reason == string(mosniov1.ReasonInvalid)
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})
	})
})
