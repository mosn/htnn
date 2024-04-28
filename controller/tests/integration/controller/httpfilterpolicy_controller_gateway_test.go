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

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

var _ = Describe("HTTPFilterPolicy controller, for gateway", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	AfterEach(func() {
		var policies mosniov1.HTTPFilterPolicyList
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

	Context("When generating LDS plugin configuration via ECDS", func() {
		BeforeEach(func() {
			// use env to set the conf
			os.Setenv("HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS", "true")
			config.Init()

			input := []map[string]interface{}{}
			mustReadHTTPFilterPolicy("default_istio", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}
		})

		AfterEach(func() {
			os.Setenv("HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS", "false")
			config.Init()

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
			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 2
			}, timeout, interval).Should(BeTrue())

			efFound := false
			for _, ef := range envoyfilters.Items {
				if ef.Name == "htnn-h-default" {
					efFound = true

					Expect(len(ef.Spec.ConfigPatches)).To(Equal(1))
					cp := ef.Spec.ConfigPatches[0]
					Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_HTTP_FILTER))
				}
			}
			Expect(efFound).To(BeTrue())
		})

		It("deal with Istio gateway", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
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
					// One from the default gateway, and two from the istio_gateway input
					if ef.Name == "htnn-h-default" && len(ef.Spec.ConfigPatches) == 3 {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			ecdsFound := false
			for _, ef := range envoyfilters.Items {
				if ef.Name == "htnn-h-default" {
					for _, cp := range ef.Spec.ConfigPatches {
						if cp.ApplyTo == istioapi.EnvoyFilter_EXTENSION_CONFIG {
							ecdsFound = true
							name := cp.Patch.Value.AsMap()["name"].(string)
							Expect(name).To(Equal("htnn-default-0.0.0.0_8989-golang-filter"))
							break
						}
					}
				}
			}
			Expect(ecdsFound).To(BeTrue())

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
				}
				return true
			}, timeout, interval).Should(BeTrue())

			for _, policy := range policies.Items {
				cond := policy.Status.Conditions[0]
				switch policy.Name {
				case "policy":
					Expect(cond.Reason).To(Equal(string(gwapiv1a2.PolicyReasonAccepted)))
				default:
					Expect(cond.Reason).To(Equal(string(gwapiv1a2.PolicyReasonTargetNotFound)))
				}
			}
		})
	})

})
