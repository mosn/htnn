/*
Copyright The HTNN Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	istioscheme "istio.io/client-go/pkg/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/controller"
	"mosn.io/htnn/controller/internal/controller/component"
	"mosn.io/htnn/controller/internal/gatewayapi"
	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/internal/registry"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc
var clientset *kubernetes.Clientset

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	zlog := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))
	logf.SetLogger(zlog)
	log.InitLogger("console")
	log.SetLogger(log.WrapLogr(zlog))
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "manifests", "charts", "htnn-controller", "templates", "crds"),
			filepath.Join("..", "..", "testdata", "crd"),
		},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s",
			fmt.Sprintf("1.28.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = mosniov1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = istioscheme.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = gatewayapi.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Create root namespace
	clientset, err = kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.RootNamespace(),
		},
	}
	_, err = clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	// use env to set the conf
	config.Init()

	unsafeDisableDeepCopy := true
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		// Don't listen on ports to avoid port conflict & permission issue
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
		Scheme:                 scheme.Scheme,
		Cache: cache.Options{
			DefaultUnsafeDisableDeepCopy: &unsafeDisableDeepCopy,
		},
	})
	Expect(err).ToNot(HaveOccurred())

	output := component.NewK8sOutput(k8sManager.GetClient())
	rm := component.NewK8sResourceManager(k8sManager.GetClient())
	err = controller.NewFilterPolicyReconciler(
		output,
		rm,
	).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controller.ConsumerReconciler{
		ResourceManager: rm,
		Output:          output,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	registry.InitRegistryManager(&registry.RegistryManagerOption{
		Output: output,
	})
	err = controller.NewServiceRegistryReconciler(
		rm,
	).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controller.DynamicConfigReconciler{
		ResourceManager: rm,
		Output:          output,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
