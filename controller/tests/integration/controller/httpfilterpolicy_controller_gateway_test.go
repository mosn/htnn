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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

var _ = Describe("HTTPFilterPolicy controller, for gateway", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	BeforeEach(func() {
		// use env to set the conf
		os.Setenv("HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS", "true")
		config.Init()
	})

	AfterEach(func() {
		os.Setenv("HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS", "false")
		config.Init()

		var policies mosniov1.HTTPFilterPolicyList
		if err := k8sClient.List(ctx, &policies); err == nil {
			for _, e := range policies.Items {
				pkg.DeleteK8sResource(ctx, k8sClient, &e)
			}
		}

		Eventually(func() bool {
			if err := k8sClient.List(ctx, &policies); err != nil {
				return false
			}
			return len(policies.Items) == 0
		}, timeout, interval).Should(BeTrue())

		var envoyfilters istiov1a3.EnvoyFilterList
		if err := k8sClient.List(ctx, &envoyfilters); err == nil {
			for _, e := range envoyfilters.Items {
				pkg.DeleteK8sResource(ctx, k8sClient, e)
			}
		}

		Eventually(func() bool {
			if err := k8sClient.List(ctx, &envoyfilters); err != nil {
				return false
			}
			return len(envoyfilters.Items) == 0
		}, timeout, interval).Should(BeTrue())

	})

	Context("When generating LDS plugin configuration via ECDS (Istio Gateway)", func() {
		AfterEach(func() {
			var virtualservices istiov1a3.VirtualServiceList
			if err := k8sClient.List(ctx, &virtualservices); err == nil {
				for _, e := range virtualservices.Items {
					pkg.DeleteK8sResource(ctx, k8sClient, e)
				}
			}

			var gateways istiov1a3.GatewayList
			if err := k8sClient.List(ctx, &gateways); err == nil {
				for _, e := range gateways.Items {
					pkg.DeleteK8sResource(ctx, k8sClient, e)
				}
			}
		})

		It("should produce HTTP filter with discovery for ECDS", func() {
			input := []map[string]interface{}{}
			mustReadHTTPFilterPolicy("default_istio", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				efFound := false
				for _, ef := range envoyfilters.Items {
					if ef.Name == "htnn-h-default" {
						efFound = true

						Expect(len(ef.Spec.ConfigPatches)).To(Equal(2))
						cp := ef.Spec.ConfigPatches[0]
						Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_HTTP_FILTER))
						cp = ef.Spec.ConfigPatches[1]
						Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_EXTENSION_CONFIG))
					}
				}
				return efFound
			}, timeout, interval).Should(BeTrue())
		})

		It("deal with Istio gateway", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadHTTPFilterPolicy("default_istio", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			mustReadHTTPFilterPolicy("istio_gateway", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				if len(envoyfilters.Items) != 2 {
					return false
				}
				for _, ef := range envoyfilters.Items {
					// Two from the default gateway, and two from the istio_gateway input
					if ef.Name == "htnn-h-default" && len(ef.Spec.ConfigPatches) == 4 {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			ecdsFound := false
			names := []string{}
			for _, ef := range envoyfilters.Items {
				if ef.Name == "htnn-h-default" {
					for _, cp := range ef.Spec.ConfigPatches {
						if cp.ApplyTo == istioapi.EnvoyFilter_EXTENSION_CONFIG {
							ecdsFound = true
							names = append(names, cp.Patch.Value.AsMap()["name"].(string))
						}
					}
				}
			}
			Expect(ecdsFound).To(BeTrue())
			Expect(names).To(ConsistOf([]string{"htnn-default-0.0.0.0_8989-golang-filter", "htnn-default-0.0.0.0_8888-golang-filter"}))

			var policies mosniov1.HTTPFilterPolicyList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				if len(policies.Items) != 3 {
					return false
				}
				for _, policy := range policies.Items {
					if len(policy.Status.Conditions) == 0 {
						return false
					}
					if policy.Name != "policy" {
						if policy.Status.Conditions[0].Reason != string(gwapiv1a2.PolicyReasonTargetNotFound) {
							return false
						}
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})

		It("deal with Istio gateway via port", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadHTTPFilterPolicy("default_istio", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			mustReadHTTPFilterPolicy("istio_gateway_via_port", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				if len(envoyfilters.Items) != 2 {
					return false
				}
				for _, ef := range envoyfilters.Items {
					if ef.Name == "htnn-h-default" {
						// Two from the default gateway, and four from the input
						if len(ef.Spec.ConfigPatches) != 6 {
							return false
						}

						for _, cp := range ef.Spec.ConfigPatches {
							v := cp.Patch.Value.AsMap()
							name := v["name"].(string)
							if name == "htnn-default-0.0.0.0_80-golang-filter" && cp.ApplyTo == istioapi.EnvoyFilter_EXTENSION_CONFIG {
								pv := v["typed_config"].(map[string]interface{})["plugin_config"].(map[string]interface{})["value"].(map[string]interface{})
								if _, ok := pv["plugins"]; ok {
									// plugins can be nil if the policy targets to the Gateway is not resolved yet
									if len(pv["plugins"].([]interface{})) == 2 {
										return true
									}
								}
							}
						}
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			var policies mosniov1.HTTPFilterPolicyList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				for _, policy := range policies.Items {
					if len(policy.Status.Conditions) == 0 {
						return false
					}
					if policy.Name == "not-found" {
						if policy.Status.Conditions[0].Reason != string(gwapiv1a2.PolicyReasonTargetNotFound) {
							return false
						}
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})

	})

	Context("When generating LDS plugin configuration via ECDS (k8s Gateway)", func() {
		BeforeEach(func() {
			os.Setenv("HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS", "true")
			config.Init()
		})

		AfterEach(func() {
			os.Setenv("HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS", "false")
			config.Init()

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
		})

		It("should produce HTTP filter with discovery for ECDS", func() {
			input := []map[string]interface{}{}
			mustReadHTTPFilterPolicy("default_gwapi", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				efFound := false
				for _, ef := range envoyfilters.Items {
					if ef.Name == "htnn-h-default" {
						efFound = true

						Expect(len(ef.Spec.ConfigPatches)).To(Equal(2))
						cp := ef.Spec.ConfigPatches[0]
						Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_HTTP_FILTER))
						cp = ef.Spec.ConfigPatches[1]
						Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_EXTENSION_CONFIG))
					}
				}
				return efFound
			}, timeout, interval).Should(BeTrue())
		})

		It("deal with k8s gateway", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadHTTPFilterPolicy("default_gwapi", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			mustReadHTTPFilterPolicy("gateway", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				if len(envoyfilters.Items) != 2 {
					return false
				}
				for _, ef := range envoyfilters.Items {
					// Two from the default gateway, and two from the gateway input
					if ef.Name == "htnn-h-default" && len(ef.Spec.ConfigPatches) == 4 {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			ecdsFound := false
			names := []string{}
			for _, ef := range envoyfilters.Items {
				if ef.Name == "htnn-h-default" {
					for _, cp := range ef.Spec.ConfigPatches {
						if cp.ApplyTo == istioapi.EnvoyFilter_EXTENSION_CONFIG {
							ecdsFound = true
							names = append(names, cp.Patch.Value.AsMap()["name"].(string))
						}
					}
				}
			}
			Expect(ecdsFound).To(BeTrue())
			Expect(names).To(ConsistOf([]string{"htnn-default-0.0.0.0_8989-golang-filter", "htnn-default-0.0.0.0_8888-golang-filter"}))

			var policies mosniov1.HTTPFilterPolicyList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				if len(policies.Items) != 2 {
					return false
				}
				for _, policy := range policies.Items {
					if len(policy.Status.Conditions) == 0 {
						return false
					}
					if policy.Name != "policy" {
						if policy.Status.Conditions[0].Reason != string(gwapiv1a2.PolicyReasonTargetNotFound) {
							return false
						}
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})

		It("deal with k8s gateway via port", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadHTTPFilterPolicy("default_gwapi", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			mustReadHTTPFilterPolicy("gateway_via_port", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				if len(envoyfilters.Items) != 2 {
					return false
				}
				for _, ef := range envoyfilters.Items {
					if ef.Name == "htnn-h-default" {
						// Two from the default gateway, and four from the input
						if len(ef.Spec.ConfigPatches) != 6 {
							return false
						}

						for _, cp := range ef.Spec.ConfigPatches {
							v := cp.Patch.Value.AsMap()
							name := v["name"].(string)
							if name == "htnn-default-0.0.0.0_80-golang-filter" && cp.ApplyTo == istioapi.EnvoyFilter_EXTENSION_CONFIG {
								pv := v["typed_config"].(map[string]interface{})["plugin_config"].(map[string]interface{})["value"].(map[string]interface{})
								// One from Gateway level, another from Port level
								if _, ok := pv["plugins"]; ok {
									// plugins can be nil if the policy targets to the Gateway is not resolved yet
									if len(pv["plugins"].([]interface{})) == 2 {
										return true
									}
								}
							}
						}
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			var policies mosniov1.HTTPFilterPolicyList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				for _, policy := range policies.Items {
					if len(policy.Status.Conditions) == 0 {
						return false
					}
					if policy.Name == "not-found" {
						if policy.Status.Conditions[0].Reason != string(gwapiv1a2.PolicyReasonTargetNotFound) {
							return false
						}
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})

	})

})
