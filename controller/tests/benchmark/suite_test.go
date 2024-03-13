//go:build benchmark

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

package benchmark

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

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
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/yaml"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/controller"
	controlleroutput "mosn.io/htnn/controller/internal/controller/output"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc
var clientset *kubernetes.Clientset
var k8sManager manager.Manager
var httpFilterPolicyReconciler *controller.HTTPFilterPolicyReconciler

func mustReadInput(fn string, out interface{}) {
	fn = filepath.Join("testdata", fn+".yml")
	input, err := os.ReadFile(fn)
	Expect(err).NotTo(HaveOccurred())
	Expect(yaml.UnmarshalStrict(input, out, yaml.DisallowUnknownFields)).To(Succeed())
}

const (
	interval = time.Second * 1
)

var (
	timeout time.Duration
	scale   int

	enableProfile = false
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

	if os.Getenv("ENABLE_PROFILE") != "" {
		enableProfile = true
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

func TestBenchmark(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Benchmark Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("..", "testdata", "crd"),
		},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.28.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = mosniov1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	err = istioscheme.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = gwapiv1.AddToScheme(scheme.Scheme)
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

	unsafeDisableDeepCopy := true
	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
		Scheme:                 scheme.Scheme,
		Cache: cache.Options{
			DefaultUnsafeDisableDeepCopy: &unsafeDisableDeepCopy,
		},
	})
	Expect(err).ToNot(HaveOccurred())

	output, err := controlleroutput.NewMcpOutput(ctx)
	Expect(err).ToNot(HaveOccurred())

	httpFilterPolicyReconciler = &controller.HTTPFilterPolicyReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
		Output: output,
	}
	err = httpFilterPolicyReconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
