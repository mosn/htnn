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
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
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

func enableNacosV2(nacosInstance string) {
	input := []map[string]interface{}{}
	fn := filepath.Join("testdata", "nacos", nacosInstance+".yml")
	helper.MustReadInput(fn, &input)
	for _, in := range input {
		obj := pkg.MapToObj(in)
		Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: nacosInstance, Namespace: "default"}, obj)
			currNacos = obj.(*mosniov1.ServiceRegistry)
			return err == nil
		}, 10*time.Millisecond, 5*time.Second).Should(BeTrue())
	}
}

func disableNacosV2(nacosInstance string) {
	sr := &mosniov1.ServiceRegistry{}
	sr.SetName(nacosInstance)
	sr.SetNamespace("default")
	Expect(k8sClient.Delete(context.Background(), sr)).Should(Succeed())
}

func listNacosV2ServiceEntries() []*istiov1a3.ServiceEntry {
	var entries istiov1a3.ServiceEntryList
	Expect(k8sClient.List(ctx, &entries, client.MatchingLabels{constant.LabelCreatedBy: "ServiceRegistry"})).Should(Succeed())
	return entries.Items
}

func registerNacosV2Instance(nacosPort string, name string, ip string, port string, metadata map[string]any) {
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

	fullURL := nacosServerURL + "/nacos/v2/ns/instance?" + params.Encode()

	req, err := http.NewRequest("POST", fullURL, strings.NewReader(""))
	Expect(err).To(BeNil())
	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(200))
}

func deregisterNacosV2Instance(nacosPort string, name string, ip string, port string) {
	nacosServerURL := "http://0.0.0.0:" + nacosPort

	params := url.Values{}
	params.Set("serviceName", name)
	params.Set("ip", ip)
	params.Set("port", port)

	fullURL := nacosServerURL + "/nacos/v2/ns/instance?" + params.Encode()

	req, err := http.NewRequest("DELETE", fullURL, nil)
	Expect(err).To(BeNil())
	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(200))
}

func deleteNacosV2Service(nacosPort string, name string) {
	nacosServerURL := "http://0.0.0.0:" + nacosPort

	params := url.Values{}
	params.Set("serviceName", name)

	fullURL := nacosServerURL + "/nacos/v2/ns/service?" + params.Encode()

	req, err := http.NewRequest("DELETE", fullURL, nil)
	Expect(err).To(BeNil())
	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(200))
}

var _ = Describe("NacosV2", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	helper.WaitServiceUp(":8850", "Nacos")

	AfterEach(func() {
		var registries mosniov1.ServiceRegistryList
		if err := k8sClient.List(ctx, &registries); err == nil {
			for _, e := range registries.Items {
				pkg.DeleteK8sResource(ctx, k8sClient, &e)
			}
		}

		Eventually(func() bool {
			entries := listNacosV2ServiceEntries()
			return len(entries) == 0
		}, timeout, interval).Should(BeTrue())
	})

	It("service life cycle", func() {
		enableNacosV2("v2")

		registerNacosV2Instance("8850", "test", "1.2.3.4", "8080", nil)

		var entries []*istiov1a3.ServiceEntry
		Eventually(func() bool {
			entries = listNacosV2ServiceEntries()
			return len(entries) == 1
		}, timeout, interval).Should(BeTrue())

		Expect(entries[0].Name).To(Equal("test.default-group.public.default.nacos"))
		Expect(entries[0].Spec.GetHosts()).To(Equal([]string{"test.default-group.public.default.nacos"}))
		Expect(entries[0].Spec.Location).To(Equal(istioapi.ServiceEntry_MESH_INTERNAL))
		Expect(entries[0].Spec.Resolution).To(Equal(istioapi.ServiceEntry_STATIC))
		Expect(len(entries[0].Spec.Endpoints)).To(Equal(1))
		Expect(entries[0].Spec.Endpoints[0].Address).To(Equal("1.2.3.4"))
		Expect(entries[0].Spec.Endpoints[0].Ports).To(Equal(map[string]uint32{
			"HTTP": 8080,
		}))

		registerNacosV2Instance("8850", "test", "1.2.3.5", "8080", nil)

		Eventually(func() bool {
			entries = listNacosV2ServiceEntries()
			return len(entries[0].Spec.Endpoints) == 2
		}, timeout, interval).Should(BeTrue())

		deregisterNacosV2Instance("8850", "test", "1.2.3.5", "8080")

		Eventually(func() bool {
			entries = listNacosV2ServiceEntries()
			return len(entries[0].Spec.Endpoints) == 1
		}, timeout, interval).Should(BeTrue())

		deleteNacosV2Service("8850", "test")
	})

	It("stop nacos should remove service entries", func() {
		registerNacosV2Instance("8850", "test", "1.2.3.4", "8080", nil)
		enableNacosV2("v2")

		Eventually(func() bool {
			entries := listNacosV2ServiceEntries()
			return len(entries) == 1
		}, timeout, interval).Should(BeTrue())

		disableNacosV2("v2")

		Eventually(func() bool {
			entries := listNacosV2ServiceEntries()
			return len(entries) == 0
		}, timeout, interval).Should(BeTrue())

		deleteNacosV2Service("8850", "test")
	})

	It("reload", func() {
		registerNacosV2Instance("8850", "test", "1.2.3.4", "8080", nil)
		registerNacosV2Instance("8850", "test1", "1.2.3.4", "8080", nil)
		registerNacosV2Instance("8850", "test2", "1.2.3.4", "8080", nil)
		registerNacosV2Instance("8851", "test", "1.2.3.5", "8080", nil)
		registerNacosV2Instance("8851", "test3", "1.2.3.5", "8080", nil)

		// old
		enableNacosV2("v2")
		var entries []*istiov1a3.ServiceEntry
		Eventually(func() bool {
			entries = listNacosV2ServiceEntries()
			return len(entries) == 3
		}, timeout, interval).Should(BeTrue())
		Expect(entries[0].Spec.Endpoints[0].Address).To(Equal("1.2.3.4"))

		// new
		base := client.MergeFrom(currNacos.DeepCopy())
		currNacos.Spec.Config.Raw = []byte(`{"serviceRefreshInterval":"1s", "serverUrl":"http://127.0.0.1:8851", "version":"v2"}`)
		Expect(k8sClient.Patch(ctx, currNacos, base)).Should(Succeed())
		Eventually(func() bool {
			entries = listNacosV2ServiceEntries()
			return len(entries) == 2 && entries[0].Spec.Endpoints[0].Address == "1.2.3.5"
		}, timeout, interval).Should(BeTrue())

		// refresh & unsubscribe
		deleteNacosV2Service("8851", "test3")
		time.Sleep(1 * time.Second)
		entries = listNacosV2ServiceEntries()
		Expect(len(entries)).To(Equal(2))

		// ServiceEntry is removed only when the configuration changed
		base = client.MergeFrom(currNacos.DeepCopy())
		currNacos.Spec.Config.Raw = []byte(`{"serviceRefreshInterval":"2s", "serverUrl":"http://127.0.0.1:8851", "version":"v2"}`)
		Expect(k8sClient.Patch(ctx, currNacos, base)).Should(Succeed())
		Eventually(func() bool {
			entries = listNacosV2ServiceEntries()
			return len(entries) == 1
		}, timeout, interval).Should(BeTrue())

		// subscribe change
		registerNacosV2Instance("8851", "test", "1.2.4.5", "8080", nil)
		deleteNacosV2Service("8850", "test") // should be ignored
		Eventually(func() bool {
			entries = listNacosV2ServiceEntries()
			return len(entries[0].Spec.Endpoints) == 2
		}, timeout, interval).Should(BeTrue())

		// unsubscribe
		disableNacosV2("v2")
		Eventually(func() bool {
			entries := listNacosV2ServiceEntries()
			return len(entries) == 0
		}, timeout, interval).Should(BeTrue())

		deleteNacosV2Service("8850", "test1")
		deleteNacosV2Service("8850", "test2")
		deleteNacosV2Service("8851", "test")
	})

})
