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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/controller/tests/integration/helper"
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func mustReadConsumer(fn string, out *[]map[string]interface{}) {
	fn = filepath.Join("testdata", "consumer", fn+".yml")
	helper.MustReadInput(fn, out)
}

var _ = Describe("Consumer controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	AfterEach(func() {
		var consumers mosniov1.ConsumerList
		if err := k8sClient.List(ctx, &consumers); err == nil {
			for _, e := range consumers.Items {
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

	Context("When reconciling Consumer", func() {
		It("deal with crd", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadConsumer("consumer", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var consumers mosniov1.ConsumerList
			var c *mosniov1.Consumer
			var cs []metav1.Condition
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &consumers); err != nil {
					return false
				}
				handled := len(consumers.Items) == 2
				for _, item := range consumers.Items {
					item := item
					if item.Name == "spacewander" {
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
			var ef *istiov1a3.EnvoyFilter
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				for _, item := range envoyfilters.Items {
					if item.Name == "htnn-consumer" {
						ef = item
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			Expect(ef.Namespace).To(Equal("istio-system"))
			Expect(len(ef.Spec.ConfigPatches)).To(Equal(2))
			cp := ef.Spec.ConfigPatches[0]
			Expect(cp.ApplyTo).To(Equal(istioapi.EnvoyFilter_EXTENSION_CONFIG))
			value := cp.Patch.Value.AsMap()
			Expect(value["name"]).To(Equal("htnn-consumer"))
			Expect(value["disabled"]).To(Equal(true))
			typedCfg := value["typed_config"].(map[string]interface{})
			pluginCfg := typedCfg["plugin_config"].(map[string]interface{})

			marshaledCfg := map[string]map[string]map[string]interface{}{}
			b, _ := json.Marshal(pluginCfg["value"])
			json.Unmarshal(b, &marshaledCfg)
			// mapping is namespace -> name -> config
			Expect(marshaledCfg["default"]["spacewander"]).ToNot(BeNil())
			Expect(marshaledCfg["default"]["unchanged"]).ToNot(BeNil())
			d := marshaledCfg["default"]["spacewander"]["d"].(string)
			cfg := map[string]interface{}{}
			err := json.Unmarshal([]byte(d), &cfg)
			Expect(err).To(BeNil())
			filter := cfg["auth"].(map[string]interface{})
			Expect(filter["keyAuth"]).ToNot(BeNil())

			v := marshaledCfg["default"]["unchanged"]["v"]

			// to invalid
			base := client.MergeFrom(c.DeepCopy())
			prev := c.Spec.Auth["keyAuth"]
			c.Spec.Auth["keyAuth"] = mosniov1.ConsumerPlugin{
				Config: runtime.RawExtension{
					Raw: []byte(`{}`),
				},
			}
			Expect(k8sClient.Patch(ctx, c, base)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &consumers); err != nil {
					return false
				}
				for _, item := range consumers.Items {
					if item.Name == "spacewander" {
						c = &consumers.Items[0]
						cs = c.Status.Conditions
						return cs[0].Reason == string(mosniov1.ReasonInvalid)
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
					if item.Name == "htnn-consumer" {
						ef = item
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			value = ef.Spec.ConfigPatches[0].Patch.Value.AsMap()
			typedCfg = value["typed_config"].(map[string]interface{})
			pluginCfg = typedCfg["plugin_config"].(map[string]interface{})

			marshaledCfg = map[string]map[string]map[string]interface{}{}
			b, _ = json.Marshal(pluginCfg["value"])
			json.Unmarshal(b, &marshaledCfg)
			Expect(marshaledCfg["default"]["spacewander"]).To(BeNil())
			Expect(marshaledCfg["default"]["unchanged"]).ToNot(BeNil())
			Expect(marshaledCfg["default"]["unchanged"]["v"]).To(Equal(v))

			// back to valid
			base = client.MergeFrom(c.DeepCopy())
			c.Spec.Auth["keyAuth"] = prev
			Expect(k8sClient.Patch(ctx, c, base)).Should(Succeed())
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &consumers); err != nil {
					return false
				}
				for _, item := range consumers.Items {
					if item.Name == "spacewander" {
						c = &consumers.Items[0]
						cs = c.Status.Conditions
						return cs[0].Reason == string(mosniov1.ReasonAccepted)
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
					if item.Name == "htnn-consumer" {
						ef = item
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			value = ef.Spec.ConfigPatches[0].Patch.Value.AsMap()
			typedCfg = value["typed_config"].(map[string]interface{})
			pluginCfg = typedCfg["plugin_config"].(map[string]interface{})

			marshaledCfg = map[string]map[string]map[string]interface{}{}
			b, _ = json.Marshal(pluginCfg["value"])
			json.Unmarshal(b, &marshaledCfg)
			Expect(marshaledCfg["default"]["spacewander"]).ToNot(BeNil())
		})

		It("with filter", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadConsumer("consumer_with_filter", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var envoyfilters istiov1a3.EnvoyFilterList
			marshaledCfg := map[string]map[string]map[string]interface{}{}
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &envoyfilters); err != nil {
					return false
				}
				if len(envoyfilters.Items) != 1 {
					return false
				}
				ef := envoyfilters.Items[0]
				if ef.Namespace != "istio-system" || ef.Name != "htnn-consumer" {
					return false
				}
				if len(ef.Spec.ConfigPatches) != 2 {
					return false
				}
				cp := ef.Spec.ConfigPatches[0]
				if cp.ApplyTo != istioapi.EnvoyFilter_EXTENSION_CONFIG {
					return false
				}
				value := cp.Patch.Value.AsMap()
				if value["name"] != "htnn-consumer" {
					return false
				}
				typedCfg := value["typed_config"].(map[string]interface{})
				pluginCfg := typedCfg["plugin_config"].(map[string]interface{})

				b, _ := json.Marshal(pluginCfg["value"])
				json.Unmarshal(b, &marshaledCfg)
				return marshaledCfg["default"]["spacewander"] != nil
			}, timeout, interval).Should(BeTrue())

			d := marshaledCfg["default"]["spacewander"]["d"].(string)
			cfg := map[string]interface{}{}
			err := json.Unmarshal([]byte(d), &cfg)
			Expect(err).To(BeNil())
			filter := cfg["filters"].(map[string]interface{})
			Expect(filter["demo"]).ToNot(BeNil())
		})

		It("deal with name conflict", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadConsumer("consumer_name_conflict", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			var consumers mosniov1.ConsumerList
			Eventually(func() bool {
				if err := k8sClient.List(ctx, &consumers); err != nil {
					return false
				}
				handled := len(consumers.Items) == 2
				for _, item := range consumers.Items {
					conds := item.Status.Conditions
					if len(conds) != 1 {
						handled = false
						break
					}
				}

				return handled
			}, timeout, interval).Should(BeTrue())

			duplicatedFound := false
			for _, item := range consumers.Items {
				cs := item.Status.Conditions
				if cs[0].Reason != string(mosniov1.ReasonAccepted) {
					duplicatedFound = true
					Expect(strings.Contains(cs[0].Message, "duplicate")).To(BeTrue())
					break
				}
			}
			Expect(duplicatedFound).To(BeTrue())
		})
	})
})
