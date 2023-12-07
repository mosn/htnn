package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	mosniov1 "mosn.io/moe/controller/api/v1"
)

func mustReadInput(fn string, out interface{}) {
	fn = filepath.Join("testdata", "httpfilterpolicy", fn+".yml")
	input, err := os.ReadFile(fn)
	Expect(err).NotTo(HaveOccurred())
	Expect(yaml.UnmarshalStrict(input, out, yaml.DisallowUnknownFields)).To(Succeed())
}

func mapToObj(in map[string]interface{}) client.Object {
	var out client.Object
	data, _ := json.Marshal(in)
	group := in["apiVersion"].(string)
	if strings.HasPrefix(group, "networking.istio.io") {
		switch in["kind"] {
		case "VirtualService":
			out = &istiov1b1.VirtualService{}
		case "Gateway":
			out = &istiov1b1.Gateway{}
		case "EnvoyFilter":
			out = &istiov1a3.EnvoyFilter{}
		}
	} else if strings.HasPrefix(group, "mosn.io") {
		switch in["kind"] {
		case "HTTPFilterPolicy":
			out = &mosniov1.HTTPFilterPolicy{}
		}
	}
	if out == nil {
		panic("unknown crd")
	}
	json.Unmarshal(data, out)
	return out
}

var _ = Describe("HTTPFilterPolicy controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		DefaultVirtualService *istiov1b1.VirtualService
	)

	Context("When reconciling HTTPFilterPolicy", func() {
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
			mustReadInput("default", &input)

			for _, in := range input {
				obj := mapToObj(in)
				if obj.GetObjectKind().GroupVersionKind().Kind == "VirtualService" {
					DefaultVirtualService = obj.(*istiov1b1.VirtualService)
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
				obj := mapToObj(in)
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
				if ef.Name == "htnn-h-default--vs" {
					Expect(len(ef.Spec.ConfigPatches)).To(Equal(1))
					cp := ef.Spec.ConfigPatches[0]
					Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_VIRTUAL_HOST))
					Expect(cp.Match.GetRouteConfiguration().GetVhost().Name).To(Equal("default.local:8888"))
				}
			}
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default--vs"}))

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

		It("deal with multi policies to one virtualservice", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("multi-policies", &input)

			for _, in := range input {
				obj := mapToObj(in)
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
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default--default"}))

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
			mustReadInput("diff-envoyfilters", &input)

			for _, in := range input {
				obj := mapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				for _, ef := range envoyfilters.Items {
					if ef.Name == "htnn-h-default--vs" {
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
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default--vs", "not-from-htnn"}))
		})

		It("refer virtualservice across namespace", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadInput("refer-virtualservice-across-namespace", &input)

			var virtualService *istiov1b1.VirtualService
			for _, in := range input {
				obj := mapToObj(in)
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
				if ef.Name == "htnn-h-default--vs" {
					Expect(len(ef.Spec.ConfigPatches)).To(Equal(1))
					cp := ef.Spec.ConfigPatches[0]
					Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_VIRTUAL_HOST))
					Expect(cp.Match.GetRouteConfiguration().GetVhost().Name).To(Equal("default.local:8888"))
				}
			}
			Expect(names).To(ConsistOf([]string{"htnn-http-filter", "htnn-h-default--vs"}))

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

})
