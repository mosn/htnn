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

package integration

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/yaml"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	"mosn.io/htnn/controller/tests/pkg"
)

func ptrstr(s string) *string {
	return &s
}

func mustReadInput(fn string, out *[]map[string]interface{}) {
	fn = filepath.Join("testdata", "httpfilterpolicy", fn+".yml")
	input, err := os.ReadFile(fn)
	Expect(err).NotTo(HaveOccurred())
	Expect(yaml.UnmarshalStrict(input, out, yaml.DisallowUnknownFields)).To(Succeed())
	// shuffle the input to detect bugs relative to the order
	res := *out
	rand.Shuffle(len(res), func(i, j int) {
		res[i], res[j] = res[j], res[i]
	})
}

func attachGateway(ctx context.Context, httpRoute *gwapiv1.HTTPRoute, gwName string) {
	httpRoute.Status.Parents = []gwapiv1.RouteParentStatus{
		{
			ParentRef: gwapiv1.ParentReference{
				Kind:  (*gwapiv1.Kind)(ptrstr("Gateway")),
				Group: (*gwapiv1.Group)(ptrstr(gwapiv1.GroupName)),
				Name:  (gwapiv1.ObjectName)(gwName),
			},
			ControllerName: "istio.io/gateway-controller",
		},
		{
			ParentRef: gwapiv1.ParentReference{
				Kind:      (*gwapiv1.Kind)(ptrstr("Gateway")),
				Group:     (*gwapiv1.Group)(ptrstr(gwapiv1.GroupName)),
				Name:      (gwapiv1.ObjectName)(gwName),
				Namespace: (*gwapiv1.Namespace)(ptrstr("not-found")),
			},
			ControllerName: "istio.io/gateway-controller",
		},
	}
	Expect(k8sClient.Status().Update(ctx, httpRoute)).Should(Succeed())
}

var _ = Describe("HTTPFilterPolicy controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When validating HTTPFilterPolicy", func() {
		BeforeEach(func() {
			var policies mosniov1.HTTPFilterPolicyList
			if err := k8sClient.List(ctx, &policies); err == nil {
				for _, e := range policies.Items {
					Expect(k8sClient.Delete(ctx, &e)).Should(Succeed())
				}
			}
		})

		It("deal with invalid crd", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("invalid_httpfilterpolicy", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var policies mosniov1.HTTPFilterPolicyList
			var cs []metav1.Condition
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				p := policies.Items[0]
				cs = p.Status.Conditions
				return len(cs) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(cs[0].Type).To(Equal(string(gwapiv1a2.PolicyConditionAccepted)))
			Expect(cs[0].Reason).To(Equal(string(gwapiv1a2.PolicyReasonInvalid)))
			Expect(policies.Items[0].IsValid()).To(BeFalse())
		})

		It("deal with valid crd", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("valid_httpfilterpolicy", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var policies mosniov1.HTTPFilterPolicyList
			var p *mosniov1.HTTPFilterPolicy
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
			p.Spec.Filters["unknown"] = runtime.RawExtension{Raw: []byte(`{"config":"unknown"}`)}
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
	})

	var (
		DefaultVirtualService *istiov1b1.VirtualService
		DefaultIstioGateway   *istiov1b1.Gateway
	)

	Context("When reconciling HTTPFilterPolicy with VirtualService", func() {
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

			var envoyfilters istiov1a3.EnvoyFilterList
			if err := k8sClient.List(ctx, &envoyfilters); err == nil {
				for _, e := range envoyfilters.Items {
					Expect(k8sClient.Delete(ctx, e)).Should(Succeed())
				}
			}

			input := []map[string]interface{}{}
			mustReadInput("default_istio", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				gvk := obj.GetObjectKind().GroupVersionKind()
				if gvk.Kind == "VirtualService" {
					DefaultVirtualService = obj.(*istiov1b1.VirtualService)
				} else if gvk.Group == "networking.istio.io" && gvk.Kind == "Gateway" {
					DefaultIstioGateway = obj.(*istiov1b1.Gateway)
				}
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

		})

		It("deal with virtualservice", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("virtualservice", &input)

			var virtualService *istiov1b1.VirtualService
			for _, in := range input {
				obj := pkg.MapToObj(in)
				if obj.GetName() == "vs" {
					virtualService = obj.(*istiov1b1.VirtualService)
				}
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 2
			}, timeout, interval).Should(BeTrue())

			names := []string{}
			for _, ef := range envoyfilters.Items {
				Expect(ef.Namespace).To(Equal("istio-system"))
				names = append(names, ef.Name)
				if ef.Name == "htnn-h-default.local" {
					Expect(len(ef.Spec.ConfigPatches)).To(Equal(1))
					cp := ef.Spec.ConfigPatches[0]
					Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_HTTP_ROUTE))
					Expect(cp.Match.GetRouteConfiguration().GetVhost().Name).To(Equal("default.local:8888"))
				}
			}
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default.local"}))

			var policies mosniov1.HTTPFilterPolicyList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				return len(policies.Items) > 0
			}, timeout, interval).Should(BeTrue())

			policy := policies.Items[0]
			Expect(len(policy.Status.Conditions) > 0).To(BeTrue())
			cond := policy.Status.Conditions[0]
			Expect(cond.Reason).To(Equal(string(gwapiv1a2.PolicyReasonAccepted)))

			host := virtualService.Spec.Hosts[0]
			virtualService.Spec.Hosts[0] = "no-gateway-match-it.com"
			err := k8sClient.Update(ctx, virtualService)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(envoyfilters.Items[0].Name).To(Equal("htnn-http-filter"))

			virtualService.Spec.Hosts[0] = host
			err = k8sClient.Update(ctx, virtualService)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 2
			}, timeout, interval).Should(BeTrue())

			// delete virtualservice referred by httpfilterpolicy
			Expect(k8sClient.Delete(ctx, virtualService)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(envoyfilters.Items[0].Name).To(Equal("htnn-http-filter"))

			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				if len(policies.Items) == 0 {
					return false
				}
				policy = policies.Items[0]
				cond = policy.Status.Conditions[0]
				return cond.Reason == string(gwapiv1a2.PolicyReasonTargetNotFound)
			}, timeout, interval).Should(BeTrue())
		})

		It("deal with virtualservice when the istio gateway changed", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("virtualservice", &input)
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

			names := []string{}
			for _, ef := range envoyfilters.Items {
				names = append(names, ef.Name)
			}
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default.local"}))

			DefaultIstioGateway.Spec.Servers[0].Port.Number = 8889
			err := k8sClient.Update(ctx, DefaultIstioGateway)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				name := ""
				for _, ef := range envoyfilters.Items {
					if ef.Name == "htnn-h-default.local" {
						name = ef.Spec.ConfigPatches[0].Match.GetRouteConfiguration().GetVhost().GetName()
					}
				}
				// the EnvoyFilter should be updated according to the new gateway
				return name == "default.local:8889"
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, DefaultIstioGateway)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(envoyfilters.Items[0].Name).To(Equal("htnn-http-filter"))
		})

		It("deal with multi policies to one virtualservice", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("multi_policies", &input)

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

			names := []string{}
			for _, ef := range envoyfilters.Items {
				names = append(names, ef.Name)
			}
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default.local"}))

			Expect(k8sClient.Delete(ctx, DefaultVirtualService)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(envoyfilters.Items[0].Name).To(Equal("htnn-http-filter"))
		})

		It("diff envoyfilters", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("diff_envoyfilters", &input)

			// We create EnvoyFilter first, to avoid conflicting with the EnvoyFilter created by VirtualService
			for _, in := range input {
				obj := pkg.MapToObj(in)
				gvk := obj.GetObjectKind().GroupVersionKind()
				if gvk.Kind == "EnvoyFilter" {
					if obj.GetName() == "htnn-http-filter" {
						ef := obj.(*istiov1a3.EnvoyFilter).DeepCopy()
						nsName := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
						err := k8sClient.Get(ctx, nsName, ef)
						if err != nil {
							Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
						} else {
							obj.SetResourceVersion(ef.ResourceVersion)
							// default EnvoyFilter may be created already. Reset it to the one in
							// test case.
							Expect(k8sClient.Update(ctx, obj)).Should(Succeed())
						}
					} else {
						Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
					}
				}
			}
			for _, in := range input {
				obj := pkg.MapToObj(in)
				gvk := obj.GetObjectKind().GroupVersionKind()
				if gvk.Kind != "EnvoyFilter" {
					Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
				}
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				for _, ef := range envoyfilters.Items {
					if ef.Name == "htnn-h-default.local" {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			names := []string{}
			for _, ef := range envoyfilters.Items {
				Expect(ef.Namespace).To(Equal("istio-system"))
				names = append(names, ef.Name)
				if ef.Name == "htnn-http-filter" {
					Expect(len(ef.Spec.ConfigPatches) > 0).Should(BeTrue())
				}
			}
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default.local", "not-from-htnn"}))
		})

		It("refer virtualservice across namespace", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("refer_virtualservice_across_namespace", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var policies mosniov1.HTTPFilterPolicyList
			var cs []metav1.Condition
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				p := policies.Items[0]
				cs = p.Status.Conditions
				return len(cs) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(cs[0].Type).To(Equal(string(gwapiv1a2.PolicyConditionAccepted)))
			Expect(cs[0].Reason).To(Equal(string(gwapiv1a2.PolicyReasonInvalid)))
			Expect(policies.Items[0].IsValid()).To(BeFalse())
		})

		It("route doesn't match", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("virtualservice_match_but_route_not", &input)
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

			names := []string{}
			for _, ef := range envoyfilters.Items {
				names = append(names, ef.Name)
			}
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default.local"}))

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
						continue
					}
					cond := policy.Status.Conditions[0]
					if cond.Reason == string(gwapiv1a2.PolicyReasonTargetNotFound) {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})

		It("deal with virtualservice via route name", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("virtualservice_via_route_name", &input)

			var virtualService *istiov1b1.VirtualService
			for _, in := range input {
				obj := pkg.MapToObj(in)
				if obj.GetName() == "vs" {
					virtualService = obj.(*istiov1b1.VirtualService)
				}
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 2
			}, timeout, interval).Should(BeTrue())

			names := []string{}
			for _, ef := range envoyfilters.Items {
				Expect(ef.Namespace).To(Equal("istio-system"))
				names = append(names, ef.Name)
				if ef.Name == "htnn-h-default.local" {
					Expect(len(ef.Spec.ConfigPatches)).To(Equal(1))
					cp := ef.Spec.ConfigPatches[0]
					Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_HTTP_ROUTE))
					Expect(cp.Match.GetRouteConfiguration().GetVhost().GetRoute().GetName()).To(Equal("route"))
				}
			}
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default.local"}))

			name := virtualService.Spec.Http[1].Name
			virtualService.Spec.Http[1].Name = "not-match"
			err := k8sClient.Update(ctx, virtualService)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(envoyfilters.Items[0].Name).To(Equal("htnn-http-filter"))

			virtualService.Spec.Http[1].Name = name
			err = k8sClient.Update(ctx, virtualService)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 2
			}, timeout, interval).Should(BeTrue())

			// delete virtualservice referred by httpfilterpolicy
			Expect(k8sClient.Delete(ctx, virtualService)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(envoyfilters.Items[0].Name).To(Equal("htnn-http-filter"))
		})

	})

	var (
		DefaultK8sGateway *gwapiv1.Gateway
	)

	Context("When reconciling HTTPFilterPolicy with HTTPRoute", func() {
		BeforeEach(func() {
			var policies mosniov1.HTTPFilterPolicyList
			if err := k8sClient.List(ctx, &policies); err == nil {
				for _, e := range policies.Items {
					Expect(k8sClient.Delete(ctx, &e)).Should(Succeed())
				}
			}

			var httproutes gwapiv1.HTTPRouteList
			if err := k8sClient.List(ctx, &httproutes); err == nil {
				for _, e := range httproutes.Items {
					Expect(k8sClient.Delete(ctx, &e)).Should(Succeed())
				}
			}

			var gateways gwapiv1.GatewayList
			if err := k8sClient.List(ctx, &gateways); err == nil {
				for _, e := range gateways.Items {
					Expect(k8sClient.Delete(ctx, &e)).Should(Succeed())
				}
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			if err := k8sClient.List(ctx, &envoyfilters); err == nil {
				for _, e := range envoyfilters.Items {
					Expect(k8sClient.Delete(ctx, e)).Should(Succeed())
				}
			}

			input := []map[string]interface{}{}
			mustReadInput("default_gwapi", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				gvk := obj.GetObjectKind().GroupVersionKind()
				if gvk.Group == "gateway.networking.k8s.io" && gvk.Kind == "Gateway" {
					DefaultK8sGateway = obj.(*gwapiv1.Gateway)
				}
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

		})

		It("deal with httproute", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("httproute", &input)

			var httpRoute *gwapiv1.HTTPRoute
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())

				if obj.GetName() == "hr" {
					httpRoute = obj.(*gwapiv1.HTTPRoute)
					attachGateway(ctx, httpRoute, DefaultK8sGateway.GetName())
				}
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 2
			}, timeout, interval).Should(BeTrue())

			names := []string{}
			for _, ef := range envoyfilters.Items {
				Expect(ef.Namespace).To(Equal("istio-system"))
				names = append(names, ef.Name)
				if ef.Name == "htnn-h-default.local" {
					Expect(len(ef.Spec.ConfigPatches)).To(Equal(1))
					cp := ef.Spec.ConfigPatches[0]
					Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_HTTP_ROUTE))
					Expect(cp.Match.GetRouteConfiguration().GetVhost().Name).To(Equal("default.local:8888"))
				}
			}
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default.local"}))

			var policies mosniov1.HTTPFilterPolicyList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				return len(policies.Items) > 0
			}, timeout, interval).Should(BeTrue())

			policy := policies.Items[0]
			Expect(len(policy.Status.Conditions) > 0).To(BeTrue())
			cond := policy.Status.Conditions[0]
			Expect(cond.Reason).To(Equal(string(gwapiv1a2.PolicyReasonAccepted)))

			host := httpRoute.Spec.Hostnames[0]
			httpRoute.Spec.Hostnames[0] = "no-gateway-match-it.com"
			err := k8sClient.Update(ctx, httpRoute)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(envoyfilters.Items[0].Name).To(Equal("htnn-http-filter"))

			httpRoute.Spec.Hostnames[0] = host
			err = k8sClient.Update(ctx, httpRoute)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 2
			}, timeout, interval).Should(BeTrue())

			// delete httproute referred by httpfilterpolicy
			Expect(k8sClient.Delete(ctx, httpRoute)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(envoyfilters.Items[0].Name).To(Equal("htnn-http-filter"))

			Eventually(func() bool {
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				if len(policies.Items) == 0 {
					return false
				}
				policy = policies.Items[0]
				cond = policy.Status.Conditions[0]
				return cond.Reason == string(gwapiv1a2.PolicyReasonTargetNotFound)
			}, timeout, interval).Should(BeTrue())
		})

		It("deal with httproute when the k8s gateway changed", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("httproute", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())

				if obj.GetName() == "hr" {
					httpRoute := obj.(*gwapiv1.HTTPRoute)
					attachGateway(ctx, httpRoute, DefaultK8sGateway.GetName())
				}
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 2
			}, timeout, interval).Should(BeTrue())

			names := []string{}
			for _, ef := range envoyfilters.Items {
				names = append(names, ef.Name)
			}
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default.local"}))

			DefaultK8sGateway.Spec.Listeners[0].Port = 8889
			err := k8sClient.Update(ctx, DefaultK8sGateway)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				name := ""
				for _, ef := range envoyfilters.Items {
					if ef.Name == "htnn-h-default.local" {
						name = ef.Spec.ConfigPatches[0].Match.GetRouteConfiguration().GetVhost().GetName()
					}
				}
				// the EnvoyFilter should be updated according to the new gateway
				return name == "default.local:8889"
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, DefaultK8sGateway)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				return len(envoyfilters.Items) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(envoyfilters.Items[0].Name).To(Equal("htnn-http-filter"))
		})

		It("deal with unattached httproute", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("httproute", &input)

			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			Eventually(func() bool {
				var policies mosniov1.HTTPFilterPolicyList
				if err := k8sClient.List(ctx, &policies); err != nil {
					return false
				}
				if len(policies.Items) == 0 {
					return false
				}
				policy := policies.Items[0]
				if len(policy.Status.Conditions) == 0 {
					return false
				}
				cond := policy.Status.Conditions[0]
				return cond.Reason == string(gwapiv1a2.PolicyReasonTargetNotFound)
			}, timeout, interval).Should(BeTrue())
		})
	})

})
