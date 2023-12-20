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

//go:build bench

package benchmark

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	mosniov1 "mosn.io/moe/controller/api/v1"
	"mosn.io/moe/controller/tests/pkg"
)

const (
	timeout  = time.Second * 60
	interval = time.Second * 1
)

func createEventually(ctx context.Context, obj client.Object) {
	Eventually(func() bool {
		if err := k8sClient.Create(ctx, obj); err != nil {
			return false
		}
		return true
	}, timeout, interval).Should(BeTrue())
}

var _ = Describe("HTTPFilterPolicy controller", func() {
	BeforeEach(func() {
		var policies mosniov1.HTTPFilterPolicyList
		if err := k8sClient.List(ctx, &policies); err == nil {
			for _, e := range policies.Items {
				Expect(k8sClient.Delete(ctx, &e)).Should(Succeed())
			}
		}

		var virtualservices istiov1b1.VirtualServiceList
		if err := k8sClient.List(ctx, &virtualservices); err == nil {
			for _, e := range virtualservices.Items {
				Expect(k8sClient.Delete(ctx, e)).Should(Succeed())
			}
		}

		var gateways istiov1b1.GatewayList
		if err := k8sClient.List(ctx, &gateways); err == nil {
			for _, e := range gateways.Items {
				Expect(k8sClient.Delete(ctx, e)).Should(Succeed())
			}
		}
	})

	Context("test reconciliation performance", func() {
		It("standard case", func() {
			// Each VirtualService has two routes, and three HTTPFilterPolicies
			input := []map[string]interface{}{}
			mustReadInput("httpfilterpolicy", &input)

			times := 500
			var virtualService *istiov1b1.VirtualService
			var policy *mosniov1.HTTPFilterPolicy

			for _, in := range input {
				obj := pkg.MapToObj(in)
				gvk := obj.GetObjectKind().GroupVersionKind()
				if gvk.Kind == "VirtualService" {
					virtualService = obj.(*istiov1b1.VirtualService)
				} else if gvk.Group == "mosn.io" && gvk.Kind == "HTTPFilterPolicy" {
					policy = obj.(*mosniov1.HTTPFilterPolicy)
				} else {
					Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
				}
			}

			for i := 0; i < times; i++ {
				go func(i int) {
					defer GinkgoRecover()

					id := strconv.Itoa(i)
					policy := policy.DeepCopy()

					tr := gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
							Name:  gwapiv1a2.ObjectName("vs-" + id),
						},
					}
					name := gwapiv1a2.SectionName("route")
					trRoute := gwapiv1a2.PolicyTargetReferenceWithSectionName{
						PolicyTargetReference: gwapiv1a2.PolicyTargetReference{
							Group: "networking.istio.io",
							Kind:  "VirtualService",
							Name:  gwapiv1a2.ObjectName("vs-" + id),
						},
						SectionName: &name,
					}

					policy.Name = "policy-" + id + "-host"
					policy.Spec.TargetRef = tr
					createEventually(ctx, policy.DeepCopy())
					policy.Name = "policy-" + id + "-same-level"
					policy.Spec.TargetRef = tr
					createEventually(ctx, policy.DeepCopy())
					policy.Name = "policy-" + id + "-route"
					policy.Spec.TargetRef = trRoute
					createEventually(ctx, policy.DeepCopy())

					vs := virtualService.DeepCopy()
					vs.Name = "vs-" + id
					host := vs.Spec.Hosts[0]
					vs.Spec.Hosts[0] = id + "." + host
					route := vs.Spec.Http[0]
					route.Name = "default/vs-" + id
					createEventually(ctx, vs)
				}(i)
			}

			var virtualservices istiov1b1.VirtualServiceList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &virtualservices); err != nil {
					return false
				}
				return len(virtualservices.Items) == times
			}, timeout, interval).Should(BeTrue())
			var policies mosniov1.HTTPFilterPolicyList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}

				if len(policies.Items) < times*3 {
					return false
				}
				for _, policy := range policies.Items {
					if len(policy.Status.Conditions) != 1 {
						return false
					}
					if policy.Status.Conditions[0].Reason != string(gwapiv1a2.PolicyReasonAccepted) {
						return false
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())

			num := 50
			start := time.Now()
			for i := 0; i < num; i++ {
				httpFilterPolicyReconciler.Reconcile(ctx, controllerruntime.Request{
					NamespacedName: types.NamespacedName{Namespace: "", Name: "httpfilterpolicy"}})
			}
			fmt.Println("Benchmark with 500 VirtualServices (each has two routes), 1500 HTTPFilterPolicies")
			fmt.Printf("Average: %+v\n", time.Since(start)/time.Duration(num))

			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			fmt.Printf("Allocated memory: %d MB\n", memStats.Alloc/1024/1024)
		})
	})
})
