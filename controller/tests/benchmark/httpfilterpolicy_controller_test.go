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
	"os"
	"runtime"
	"strconv"
	"sync"
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
	interval = time.Second * 1
)

var (
	timeout time.Duration
	scale   int
)

func init() {
	s := os.Getenv("BENCHMARK_SCALE")
	if s == "" {
		scale = 2500
	} else {
		var err error
		scale, err = strconv.Atoi(s)
		if err != nil {
			panic(err)
		}
	}

	timeout = 5 * time.Second * time.Duration(scale/100)
	if timeout < 10*time.Second {
		timeout = 10 * time.Second
	}
}

func createEventually(ctx context.Context, obj client.Object) {
	Eventually(func() bool {
		if err := k8sClient.Create(ctx, obj); err != nil {
			return false
		}
		return true
	}, timeout, interval).Should(BeTrue())
}

func createResource(ctx context.Context, policy *mosniov1.HTTPFilterPolicy, virtualService *istiov1b1.VirtualService, i int) {
	id := strconv.Itoa(i)
	policy = policy.DeepCopy()

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

			if scale < 1000 {
				wg := sync.WaitGroup{}
				wg.Add(scale)
				for i := 0; i < scale; i++ {
					go func(i int) {
						defer GinkgoRecover()
						defer wg.Done()
						createResource(ctx, policy, virtualService, i)
					}(i)
				}
				wg.Wait()
			} else {
				wg := sync.WaitGroup{}
				size := 50
				if scale%size == 0 {
					wg.Add(scale / size)
				} else {
					wg.Add(scale/size + 1)
				}
				for i := 0; i < scale; i += size {
					go func(i int) {
						defer GinkgoRecover()
						defer wg.Done()
						for j := i; j < i+size && j < scale; j++ {
							createResource(ctx, policy, virtualService, j)
						}
					}(i)
				}
				wg.Wait()
			}

			go func() {
				defer GinkgoRecover()
				Expect(k8sManager.Start(ctx)).ToNot(HaveOccurred(), "failed to run manager")
			}()

			var virtualservices istiov1b1.VirtualServiceList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &virtualservices); err != nil {
					return false
				}
				return len(virtualservices.Items) == scale
			}, timeout, interval).Should(BeTrue())
			var policies mosniov1.HTTPFilterPolicyList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}

				if len(policies.Items) < scale*2 {
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

			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			peakMemAlloc := memStats.Alloc

			stop := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				ticker := time.Tick(1 * time.Second)
				for {
					select {
					case <-stop:
						return
					case <-ticker:
						runtime.ReadMemStats(&memStats)
						if memStats.Alloc > peakMemAlloc {
							peakMemAlloc = memStats.Alloc
						}
					}
				}
			}()

			num := 10
			start := time.Now()
			for i := 0; i < num; i++ {
				httpFilterPolicyReconciler.Reconcile(ctx, controllerruntime.Request{
					NamespacedName: types.NamespacedName{Namespace: "", Name: "httpfilterpolicy"}})
			}
			fmt.Printf("Benchmark with %d VirtualServices (each has two routes), %d HTTPFilterPolicies\n", scale, 2*scale)
			fmt.Printf("Average: %+v\n", time.Since(start)/time.Duration(num))

			close(stop)

			runtime.ReadMemStats(&memStats)
			if memStats.Alloc > peakMemAlloc {
				peakMemAlloc = memStats.Alloc
			}
			fmt.Printf("Allocated memory: %d MB\n", peakMemAlloc/1024/1024)
		})
	})
})
