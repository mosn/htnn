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

package main

import (
	"flag"
	"os"

	istioscheme "istio.io/client-go/pkg/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	pkgLog "mosn.io/htnn/api/pkg/log"
	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/controller"
	"mosn.io/htnn/controller/internal/controller/component"
	"mosn.io/htnn/controller/internal/gatewayapi"
	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/controller/internal/registry"
	v1 "mosn.io/htnn/types/apis/v1"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var pprofAddr string
	var enc string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":10080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":10081", "The address the probe endpoint binds to.")
	flag.StringVar(&pprofAddr, "pprof-bind-address", "127.0.0.1:10082", "The address the pprof endpoint binds to. Set it to '0' to disable pprof.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.CommandLine.StringVar(&enc, "log-encoder", "console", "Log encoding (one of 'json' or 'console', default to 'console')")
	flag.Parse()

	log.InitLogger(enc)
	logrLogger := log.Logger()
	pkgLog.SetLogger(logrLogger)
	ctrl.SetLogger(logrLogger)

	config.Init()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
	utilruntime.Must(istioscheme.AddToScheme(scheme))

	if config.EnableGatewayAPI() {
		utilruntime.Must(gatewayapi.AddToScheme(scheme))
	}

	unsafeDisableDeepCopy := true
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		PprofBindAddress:       pprofAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "c30d658e.htnn.mosn.io",
		Cache: cache.Options{
			DefaultUnsafeDisableDeepCopy: &unsafeDisableDeepCopy,
		},
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	output := component.NewK8sOutput(mgr.GetClient())
	rm := component.NewK8sResourceManager(mgr.GetClient())
	if err = controller.NewHTTPFilterPolicyReconciler(
		output,
		rm,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HTTPFilterPolicy")
		os.Exit(1)
	}
	if err = (&controller.ConsumerReconciler{
		ResourceManager: rm,
		Output:          output,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Consumer")
		os.Exit(1)
	}

	registry.InitRegistryManager(&registry.RegistryManagerOption{
		Output: output,
	})
	if err = (&controller.ServiceRegistryReconciler{
		ResourceManager: rm,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceRegistry")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
