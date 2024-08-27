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

package registries

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/controller/pkg/constant"
	"mosn.io/htnn/controller/tests/integration/helper"
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

var (
	currConsul *mosniov1.ServiceRegistry
)

func enableConsul(consulInstance string) {
	var input []map[string]interface{}
	fn := filepath.Join("testdata", "consul", consulInstance+".yml")
	helper.MustReadInput(fn, &input)
	for _, in := range input {
		obj := pkg.MapToObj(in)
		Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: consulInstance, Namespace: "default"}, obj)
			currConsul = obj.(*mosniov1.ServiceRegistry)
			return err == nil
		}, 10*time.Millisecond, 5*time.Second).Should(BeTrue())
	}
}

func disableConsul(consulInstance string) {
	sr := &mosniov1.ServiceRegistry{}
	sr.SetName(consulInstance)
	sr.SetNamespace("default")
	Expect(k8sClient.Delete(context.Background(), sr)).Should(Succeed())
}

func listConsulServiceEntries() []*istiov1a3.ServiceEntry {
	var entries istiov1a3.ServiceEntryList
	Expect(k8sClient.List(ctx, &entries, client.MatchingLabels{constant.LabelCreatedBy: "ServiceRegistry"})).Should(Succeed())
	return entries.Items
}

func registerConsulInstance(consulPort string, name string, ip string, port string, metadata map[string]any) {
	consulServerURL := "http://0.0.0.0:" + consulPort

	portInt, err := strconv.Atoi(port)
	Expect(err).To(BeNil())

	service := map[string]any{
		"Node":    "node1",
		"Address": ip,
		"Service": map[string]any{
			"ID":      name + ip + port,
			"Service": name,
			"Address": ip,
			"Port":    portInt,
		},
	}

	if metadata != nil {
		service["Service"].(map[string]any)["Meta"] = metadata
	}

	body, err := json.Marshal(service)
	Expect(err).To(BeNil())

	fullURL := consulServerURL + "/v1/catalog/register"

	req, err := http.NewRequest("PUT", fullURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	Expect(err).To(BeNil())

	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).To(BeNil())

	defer resp.Body.Close()

	Expect(resp.StatusCode).To(Equal(200))
}

func deregisterConsulInstance(consulPort string, name string, ip string, port string) {
	consulServerURL := "http://0.0.0.0:" + consulPort

	serviceID := name + ip + port
	body := map[string]any{
		"Node":      "node1",
		"ServiceID": serviceID,
	}

	bodyJSON, err := json.Marshal(body)
	Expect(err).To(BeNil())

	fullURL := consulServerURL + "/v1/catalog/deregister"

	req, err := http.NewRequest("PUT", fullURL, bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	Expect(err).To(BeNil())

	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(200))
}

func deleteConsulService(consulPort string, name string, ip string, port string) {
	deregisterConsulInstance(consulPort, name, ip, port)
}

var _ = Describe("Consul", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	helper.WaitServiceUp(":8500", "Consul")

	AfterEach(func() {
		var registries mosniov1.ServiceRegistryList
		if err := k8sClient.List(ctx, &registries); err == nil {
			for _, e := range registries.Items {
				pkg.DeleteK8sResource(ctx, k8sClient, &e)
			}
		}

		Eventually(func() bool {
			entries := listConsulServiceEntries()
			return len(entries) == 0
		}, timeout, interval).Should(BeTrue())
	})

	It("service life cycle", func() {
		enableConsul("default")

		registerConsulInstance("8500", "test", "1.2.3.4", "8080", nil)

		var entries []*istiov1a3.ServiceEntry
		Eventually(func() bool {
			entries = listConsulServiceEntries()
			return len(entries) == 2
		}, timeout, interval).Should(BeTrue())

		Expect(entries[1].Name).To(Equal("test.default.consul"))
		Expect(entries[1].Spec.GetHosts()).To(Equal([]string{"test.default.consul"}))
		Expect(entries[1].Spec.Location).To(Equal(istioapi.ServiceEntry_MESH_INTERNAL))
		Expect(entries[1].Spec.Resolution).To(Equal(istioapi.ServiceEntry_STATIC))
		Expect(len(entries[1].Spec.Endpoints)).To(Equal(1))
		Expect(entries[1].Spec.Endpoints[0].Address).To(Equal("1.2.3.4"))
		Expect(entries[1].Spec.Endpoints[0].Ports).To(Equal(map[string]uint32{
			"HTTP": 8080,
		}))

		registerConsulInstance("8500", "test", "1.2.3.5", "8080", nil)

		Eventually(func() bool {
			entries = listConsulServiceEntries()
			return len(entries[1].Spec.Endpoints) == 2
		}, timeout, interval).Should(BeTrue())

		deregisterConsulInstance("8500", "test", "1.2.3.4", "8080")

		Eventually(func() bool {
			entries = listConsulServiceEntries()
			return len(entries[1].Spec.Endpoints) == 1
		}, timeout, interval).Should(BeTrue())

		deleteConsulService("8500", "test", "1.2.3.5", "8080")
	})

	It("stop consul should remove service entries", func() {
		registerConsulInstance("8500", "test", "1.2.3.4", "8080", nil)
		enableConsul("default")

		Eventually(func() bool {
			entries := listConsulServiceEntries()
			return len(entries) == 2
		}, timeout, interval).Should(BeTrue())

		disableConsul("default")

		Eventually(func() bool {
			entries := listConsulServiceEntries()
			return len(entries) == 0
		}, timeout, interval).Should(BeTrue())

		deleteConsulService("8500", "test", "1.2.3.4", "8080")
	})

	It("reload", func() {
		registerConsulInstance("8500", "test", "1.2.3.4", "8080", nil)
		registerConsulInstance("8500", "test1", "1.2.3.4", "8080", nil)
		registerConsulInstance("8500", "test2", "1.2.3.4", "8080", nil)
		registerConsulInstance("8501", "test", "1.2.3.5", "8080", nil)
		registerConsulInstance("8501", "test3", "1.2.3.5", "8080", nil)

		// old
		enableConsul("default")
		var entries []*istiov1a3.ServiceEntry
		Eventually(func() bool {
			entries = listConsulServiceEntries()
			return len(entries) == 4
		}, timeout, interval).Should(BeTrue())
		Expect(entries[1].Spec.Endpoints[0].Address).To(Equal("1.2.3.4"))

		// new
		base := client.MergeFrom(currConsul.DeepCopy())
		currConsul.Spec.Config.Raw = []byte(`{"serviceRefreshInterval":"1s", "serverUrl":"http://127.0.0.1:8501"}`)
		Expect(k8sClient.Patch(ctx, currConsul, base)).Should(Succeed())
		Eventually(func() bool {
			entries = listConsulServiceEntries()

			return len(entries) == 3 && entries[1].Spec.Endpoints[0].Address == "1.2.3.5"
		}, timeout, interval).Should(BeTrue())

		// refresh & unsubscribe
		deleteConsulService("8501", "test3", "1.2.3.5", "8080")
		time.Sleep(1 * time.Second)
		entries = listConsulServiceEntries()

		Expect(len(entries)).To(Equal(5))

		// ServiceEntry is removed only when the configuration changed
		base = client.MergeFrom(currConsul.DeepCopy())
		currConsul.Spec.Config.Raw = []byte(`{"serviceRefreshInterval":"2s", "serverUrl":"http://127.0.0.1:8501"}`)
		Expect(k8sClient.Patch(ctx, currConsul, base)).Should(Succeed())
		Eventually(func() bool {
			entries = listConsulServiceEntries()
			return len(entries) == 2
		}, timeout, interval).Should(BeTrue())

		// subscribe change
		registerConsulInstance("8501", "test", "1.2.4.5", "8080", nil)
		deleteConsulService("8500", "test", "1.2.3.4", "8080") // should be ignored
		Eventually(func() bool {
			entries = listConsulServiceEntries()
			return len(entries[1].Spec.Endpoints) == 2
		}, timeout, interval).Should(BeTrue())

		// unsubscribe
		disableConsul("default")
		Eventually(func() bool {
			entries := listConsulServiceEntries()
			return len(entries) == 0
		}, timeout, interval).Should(BeTrue())
		deleteConsulService("8500", "test1", "1.2.3.4", "8080")
		deleteConsulService("8500", "test2", "1.2.3.4", "8080")
		deleteConsulService("8501", "test", "1.2.4.5", "8080")
	})
})
