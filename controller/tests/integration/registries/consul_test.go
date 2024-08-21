package registries

import (
	"context"
	"fmt"
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

var (
	currConsul *mosniov1.ServiceRegistry
)

func enableConsul(consulInstance string) {
	input := []map[string]interface{}{}
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

	params := url.Values{}
	params.Set("ID", name)
	params.Set("Name", name)
	params.Set("Address", ip)
	params.Set("Port", port)

	if metadata != nil {
		for key, value := range metadata {
			params.Set("Meta."+key, fmt.Sprintf("%v", value))
		}
	}

	fullURL := consulServerURL + "/v1/agent/service/register"

	body := fmt.Sprintf(`{
		"ID": "%s",
		"Name": "%s",
		"Address": "%s",
		"Port": %s,
		"Meta": %v
	}`, name, name, ip, port, metadata)

	req, err := http.NewRequest("PUT", fullURL, strings.NewReader(body))
	Expect(err).To(BeNil())
	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(200))
}

func deregisterConsulInstance(consulPort string, name string) {
	consulServerURL := "http://0.0.0.0:" + consulPort

	fullURL := consulServerURL + "/v1/agent/service/deregister/" + name

	req, err := http.NewRequest("PUT", fullURL, nil)
	Expect(err).To(BeNil())
	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(200))
}

func deleteConsulService(consulPort string, name string) {
	deregisterConsulInstance(consulPort, name)
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
			entries := listServiceEntries()
			return len(entries) == 0
		}, timeout, interval).Should(BeTrue())
	})

	It("service life cycle", func() {
		enableConsul("default")

		registerConsulInstance("8500", "test", "1.2.3.4", "8080", nil)

		var entries []*istiov1a3.ServiceEntry
		Eventually(func() bool {
			entries = listServiceEntries()
			return len(entries) == 1
		}, timeout, interval).Should(BeTrue())

		//Expect(entries[0].Name).To(Equal("test.default-group.public.default.consul"))
		//Expect(entries[0].Spec.GetHosts()).To(Equal([]string{"test.default-group.public.default.consul"}))
		Expect(entries[0].Spec.Location).To(Equal(istioapi.ServiceEntry_MESH_INTERNAL))
		Expect(entries[0].Spec.Resolution).To(Equal(istioapi.ServiceEntry_STATIC))
		Expect(len(entries[0].Spec.Endpoints)).To(Equal(1))
		Expect(entries[0].Spec.Endpoints[0].Address).To(Equal("1.2.3.4"))
		Expect(entries[0].Spec.Endpoints[0].Ports).To(Equal(map[string]uint32{
			"HTTP": 8080,
		}))

		registerConsulInstance("8500", "test", "1.2.3.5", "8080", nil)

		Eventually(func() bool {
			entries = listServiceEntries()
			return len(entries[0].Spec.Endpoints) == 2
		}, timeout, interval).Should(BeTrue())

		deregisterConsulInstance("8500", "test")

		Eventually(func() bool {
			entries = listServiceEntries()
			return len(entries[0].Spec.Endpoints) == 1
		}, timeout, interval).Should(BeTrue())

		deleteService("8500", "test")
	})

	It("stop consul should remove service entries", func() {
		registerConsulInstance("8500", "test", "1.2.3.4", "8080", nil)
		enableConsul("default")

		Eventually(func() bool {
			entries := listServiceEntries()
			return len(entries) == 1
		}, timeout, interval).Should(BeTrue())

		disableConsul("default")

		Eventually(func() bool {
			entries := listServiceEntries()
			return len(entries) == 0
		}, timeout, interval).Should(BeTrue())

		deleteConsulService("8500", "test")
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
			entries = listServiceEntries()
			return len(entries) == 3
		}, timeout, interval).Should(BeTrue())
		Expect(entries[0].Spec.Endpoints[0].Address).To(Equal("1.2.3.4"))

		// new
		base := client.MergeFrom(currConsul.DeepCopy())
		currConsul.Spec.Config.Raw = []byte(`{"serviceRefreshInterval":"1s", "serverUrl":"http://127.0.0.1:8501"}`)
		Expect(k8sClient.Patch(ctx, currConsul, base)).Should(Succeed())
		Eventually(func() bool {
			entries = listConsulServiceEntries()
			return len(entries) == 2 && entries[0].Spec.Endpoints[0].Address == "1.2.3.5"
		}, timeout, interval).Should(BeTrue())

		// refresh & unsubscribe
		deleteConsulService("8501", "test3")
		time.Sleep(1 * time.Second)
		entries = listConsulServiceEntries()
		Expect(len(entries)).To(Equal(2))

		// ServiceEntry is removed only when the configuration changed
		base = client.MergeFrom(currConsul.DeepCopy())
		currConsul.Spec.Config.Raw = []byte(`{"serviceRefreshInterval":"2s", "serverUrl":"http://127.0.0.1:8501"}`)
		Expect(k8sClient.Patch(ctx, currConsul, base)).Should(Succeed())
		Eventually(func() bool {
			entries = listServiceEntries()
			return len(entries) == 1
		}, timeout, interval).Should(BeTrue())

		// subscribe change
		registerConsulInstance("8501", "test", "1.2.4.5", "8080", nil)
		deleteConsulService("8500", "test") // should be ignored
		Eventually(func() bool {
			entries = listConsulServiceEntries()
			return len(entries[0].Spec.Endpoints) == 2
		}, timeout, interval).Should(BeTrue())

		// unsubscribe
		disableConsul("default")
		Eventually(func() bool {
			entries := listConsulServiceEntries()
			return len(entries) == 0
		}, timeout, interval).Should(BeTrue())
		deleteConsulService("8848", "test1")
		deleteConsulService("8848", "test2")
		deleteConsulService("8849", "test")
	})
})
