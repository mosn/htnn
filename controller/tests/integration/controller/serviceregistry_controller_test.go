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
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/controller/pkg/registry"
	"mosn.io/htnn/controller/tests/integration/helper"
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
	typesRegistry "mosn.io/htnn/types/pkg/registry"
	typesNacos "mosn.io/htnn/types/registries/nacos"
)

func mustReadServiceRegistry(fn string, out *[]map[string]interface{}) {
	fn = filepath.Join("testdata", "serviceregistry", fn+".yml")
	helper.MustReadInput(fn, out)
}

func registerInstance(nacosPort string, name string, ip string, port string, metadata map[string]any) {
	nacosServerURL := "http://0.0.0.0:" + nacosPort

	params := url.Values{}
	params.Set("serviceName", name)
	params.Set("ip", ip)
	params.Set("port", port)

	if metadata != nil {
		b, err := json.Marshal(metadata)
		Expect(err).To(BeNil())
		params.Set("metadata", string(b))
	}

	fullURL := nacosServerURL + "/nacos/v1/ns/instance?" + params.Encode()

	req, err := http.NewRequest("POST", fullURL, strings.NewReader(""))
	Expect(err).To(BeNil())
	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(200))
}

type TestCounter struct {
	counter int
	lock    sync.Mutex
}

func (t *TestCounter) Config() typesRegistry.RegistryConfig {
	// Just a placeholder
	return &typesNacos.Config{}
}

func (t *TestCounter) Start(config typesRegistry.RegistryConfig) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.counter++
	return nil
}

func (t *TestCounter) Reload(config typesRegistry.RegistryConfig) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.counter++
	return nil
}

func (t *TestCounter) Stop() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.counter++
	return nil
}

func (t *TestCounter) Count() int {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.counter
}

var (
	GlobalTestCounter = &TestCounter{}
)

func init() {
	registry.AddRegistryFactory("test_counter", func(store registry.ServiceEntryStore, om metav1.ObjectMeta) (registry.Registry, error) {
		return GlobalTestCounter, nil
	})
}

var _ = Describe("ServiceRegistry controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	helper.WaitServiceUp(":8848", "Nacos")

	AfterEach(func() {
		var registries mosniov1.ServiceRegistryList
		if err := k8sClient.List(ctx, &registries); err == nil {
			for _, e := range registries.Items {
				pkg.DeleteK8sResource(ctx, k8sClient, &e)
			}
		}

		Eventually(func() bool {
			var entries istiov1a3.ServiceEntryList
			if err := k8sClient.List(ctx, &entries); err != nil {
				return false
			}
			return len(entries.Items) == 0
		}, timeout, interval).Should(BeTrue())
	})

	Context("deal with crd", func() {
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

	Context("deal with multiple registries", func() {
		It("create & delete multiple registries", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadServiceRegistry("multiple_nacos", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			registerInstance("8848", "test", "1.2.3.4", "8080", nil)

			var names []string
			Eventually(func() bool {
				var entries istiov1a3.ServiceEntryList
				if err := k8sClient.List(ctx, &entries); err != nil {
					return false
				}
				names = []string{}
				for _, e := range entries.Items {
					if strings.HasPrefix(e.Name, "test.") {
						names = append(names, e.Name)
					}
				}
				return len(names) == 2
			}, timeout, interval).Should(BeTrue())
			Expect(names).To(ConsistOf([]string{"test.default-group.public.earth.nacos", "test.default-group.public.moon.nacos"}))

			var registryMoon *mosniov1.ServiceRegistry
			var registryEarth *mosniov1.ServiceRegistry
			Eventually(func() bool {
				var registries mosniov1.ServiceRegistryList
				var cs []metav1.Condition
				if err := k8sClient.List(ctx, &registries); err != nil {
					return false
				}
				for _, item := range registries.Items {
					if item.Name == "moon" {
						registryMoon = &item
					} else if item.Name == "earth" {
						registryEarth = &item
					}

					cs = item.Status.Conditions
					if len(cs) != 1 {
						return false
					}

					c := cs[0]
					if c.Reason != string(mosniov1.ReasonAccepted) {
						return false
					}
				}

				return true
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, registryMoon)).Should(Succeed())
			Eventually(func() bool {
				var entries istiov1a3.ServiceEntryList
				if err := k8sClient.List(ctx, &entries); err != nil {
					return false
				}
				for _, e := range entries.Items {
					if e.Name == "test.default-group.public.moon.nacos" {
						return false
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, registryEarth)).Should(Succeed())
			Eventually(func() bool {
				var entries istiov1a3.ServiceEntryList
				if err := k8sClient.List(ctx, &entries); err != nil {
					return false
				}
				return len(entries.Items) == 0
			}, timeout, interval).Should(BeTrue())
		})

		It("don't reinit registry when other registry is changed", func() {
			ctx := context.Background()
			input := []map[string]interface{}{}
			mustReadServiceRegistry("multiple_registries", &input)
			for _, in := range input {
				obj := pkg.MapToObj(in)
				Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
			}

			registerInstance("8848", "test", "1.2.3.4", "8080", nil)

			var registry *mosniov1.ServiceRegistry
			var registryCounter *mosniov1.ServiceRegistry
			var timestamp time.Time
			Eventually(func() bool {
				var registries mosniov1.ServiceRegistryList
				var cs []metav1.Condition
				if err := k8sClient.List(ctx, &registries); err != nil {
					return false
				}
				for _, item := range registries.Items {
					if item.Spec.Type != "test_counter" {
						registry = &item
					} else {
						registryCounter = &item
					}

					cs = item.Status.Conditions
					if len(cs) != 1 {
						return false
					}

					c := cs[0]
					if c.Reason != string(mosniov1.ReasonAccepted) {
						return false
					}

					if item.Spec.Type == "test_counter" {
						timestamp = c.LastTransitionTime.Time
					}
				}

				return len(registries.Items) == 2
			}, timeout, interval).Should(BeTrue())
			Expect(GlobalTestCounter.Count()).To(Equal(1))

			Eventually(func() bool {
				var entries istiov1a3.ServiceEntryList
				if err := k8sClient.List(ctx, &entries); err != nil {
					return false
				}
				// We use the number of ServiceEntries to indicate the reconciliation is done
				return len(entries.Items) > 0
			}, timeout, interval).Should(BeTrue())
			time.Sleep(1 * time.Second)
			// Trigger a reconciliation
			Expect(k8sClient.Delete(ctx, registry)).Should(Succeed())
			Eventually(func() bool {
				var entries istiov1a3.ServiceEntryList
				if err := k8sClient.List(ctx, &entries); err != nil {
					return false
				}
				return len(entries.Items) == 0
			}, timeout, interval).Should(BeTrue())
			Expect(GlobalTestCounter.Count()).To(Equal(1))

			var serviceRegistry mosniov1.ServiceRegistry
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Namespace: registryCounter.Namespace,
				Name:      registryCounter.Name,
			}, &serviceRegistry)).Should(Succeed())
			// Should not write the condition twice
			Expect(serviceRegistry.Status.Conditions[0].LastTransitionTime).To(Equal(metav1.NewTime(timestamp)))

			// Ensure the counter actually works
			Expect(k8sClient.Delete(ctx, registryCounter)).Should(Succeed())
			// We use sleep to indicate the reconciliation is done
			time.Sleep(100 * time.Millisecond)
			Expect(GlobalTestCounter.Count()).To(Equal(2))
		})
	})

})
