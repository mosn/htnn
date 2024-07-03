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
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/tests/integration/helper"
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func mustReadFilterPolicy(fn string, out *[]map[string]interface{}) {
	fn = filepath.Join("testdata", "filterpolicy", fn+".yml")
	helper.MustReadInput(fn, out)
}

var _ = Describe("FilterPolicy controller, for policy", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	AfterEach(func() {
		var policies mosniov1.FilterPolicyList
		if err := k8sClient.List(ctx, &policies); err == nil {
			for _, e := range policies.Items {
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

	Context("When validating FilterPolicy", func() {
		It("deal with invalid crd", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadFilterPolicy("invalid_filterpolicy", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var policies mosniov1.FilterPolicyList
			var p *mosniov1.FilterPolicy
			var cs []metav1.Condition
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				p = &policies.Items[0]
				cs = p.Status.Conditions
				return len(cs) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(cs[0].Type).To(Equal(string(gwapiv1a2.PolicyConditionAccepted)))
			Expect(cs[0].Reason).To(Equal(string(gwapiv1a2.PolicyReasonInvalid)))
			Expect(p.IsValid()).To(BeFalse())

			// to valid
			base := client.MergeFrom(p.DeepCopy())
			p.Spec.Filters["demo"] = mosniov1.Plugin{
				Config: runtime.RawExtension{
					Raw: []byte(`{"hostName":"Mary"}`),
				},
			}
			Expect(k8sClient.Patch(ctx, p, base)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				p = &policies.Items[0]
				cs = p.Status.Conditions
				if len(cs) != 1 {
					return false
				}
				return cs[0].Reason == string(gwapiv1a2.PolicyReasonTargetNotFound)
			}, timeout, interval).Should(BeTrue())
			Expect(p.IsValid()).To(BeTrue())
		})

		It("deal with valid crd", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadFilterPolicy("valid_filterpolicy", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var policies mosniov1.FilterPolicyList
			var p *mosniov1.FilterPolicy
			var cs []metav1.Condition
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				p = &policies.Items[0]
				cs = p.Status.Conditions
				return len(cs) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(cs[0].Type).To(Equal(string(gwapiv1a2.PolicyConditionAccepted)))
			Expect(cs[0].Reason).To(Equal(string(gwapiv1a2.PolicyReasonTargetNotFound)))

			// to invalid
			base := client.MergeFrom(p.DeepCopy())
			p.Spec.Filters["demo"] = mosniov1.Plugin{
				Config: runtime.RawExtension{
					Raw: []byte(`{}`),
				},
			}
			Expect(k8sClient.Patch(ctx, p, base)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				p := policies.Items[0]
				cs = p.Status.Conditions
				return cs[0].Reason == string(gwapiv1a2.PolicyReasonInvalid)
			}, timeout, interval).Should(BeTrue())
		})

		It("deal with policy without targetRef", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadFilterPolicy("filterpolicy_without_targetref", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var policies mosniov1.FilterPolicyList
			var p *mosniov1.FilterPolicy
			var cs []metav1.Condition
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				p = &policies.Items[0]
				cs = p.Status.Conditions
				return len(cs) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(cs[0].Type).To(Equal(string(gwapiv1a2.PolicyConditionAccepted)))
			Expect(cs[0].Reason).To(Equal(string(gwapiv1a2.PolicyReasonInvalid)))
			Expect(p.IsValid()).To(BeFalse())
		})

	})

	Context("When disabling native plugins", func() {
		BeforeEach(func() {
			// config.Init is designed to be called only during startup. As it is only called
			// on the fly by tests, we simply add sleep to avoid race.
			time.Sleep(200 * time.Millisecond)
			// use env to set the conf
			os.Setenv("HTNN_ENABLE_NATIVE_PLUGIN", "false")
			config.Init()

			input := []map[string]interface{}{}
			mustReadFilterPolicy("default_gwapi", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}
		})

		AfterEach(func() {
			var httproutes gwapiv1b1.HTTPRouteList
			if err := k8sClient.List(ctx, &httproutes); err == nil {
				for _, e := range httproutes.Items {
					pkg.DeleteK8sResource(ctx, k8sClient, &e)
				}
			}

			var gateways gwapiv1b1.GatewayList
			if err := k8sClient.List(ctx, &gateways); err == nil {
				for _, e := range gateways.Items {
					pkg.DeleteK8sResource(ctx, k8sClient, &e)
				}
			}

			time.Sleep(200 * time.Millisecond)
			os.Setenv("HTNN_ENABLE_NATIVE_PLUGIN", "true")
			config.Init()
		})

		It("should not produce correndsponding filters", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadFilterPolicy("native_plugin", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 2
			}, timeout, interval).Should(BeTrue())

			for _, ef := range envoyfilters.Items {
				if ef.Name == "htnn-h-default.local" {
					Expect(len(ef.Spec.ConfigPatches)).To(Equal(1))
					cp := ef.Spec.ConfigPatches[0]
					filters := cp.Patch.Value.AsMap()["typed_per_filter_config"].(map[string]interface{})
					Expect(filters["htnn.filters.http.golang"]).NotTo(BeNil())
					Expect(filters["htnn.filters.http.localRatelimit"]).To(BeNil())
				} else {
					Expect(ef.Name).To(Equal("htnn-http-filter"))
					cps := ef.Spec.ConfigPatches
					for _, cp := range cps {
						if cp.ApplyTo == istioapi.EnvoyFilter_HTTP_FILTER {
							Expect(cp.Patch.Value.AsMap()["name"]).To(Equal("htnn.filters.http.golang"))
						}
					}
				}
			}
		})
	})
})
