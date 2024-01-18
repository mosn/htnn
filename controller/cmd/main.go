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
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/controller"
	"mosn.io/htnn/controller/internal/registry"
	"mosn.io/htnn/pkg/log"
)

// Version is specified by build tag, in VERSION file
var (
	Version string = ""
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(mosniov1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	utilruntime.Must(istioscheme.AddToScheme(scheme))
	// For HTTPRoute & Gateway, we only support v1 version
	utilruntime.Must(gwapiv1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var pprofAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":10080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":10081", "The address the probe endpoint binds to.")
	flag.StringVar(&pprofAddr, "pprof-bind-address", "127.0.0.1:10082", "The address the pprof endpoint binds to. Set it to '0' to disable pprof.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(log.DefaultLogger)

	config.Init()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		PprofBindAddress:       pprofAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "c30d658e.mosn.io",
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

	if err = (&controller.HTTPFilterPolicyReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HTTPFilterPolicy")
		os.Exit(1)
	}
	if err = (&controller.ConsumerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Consumer")
		os.Exit(1)
	}

	registry.InitRegistryManager(&registry.RegistryManagerOption{
		Client: mgr.GetClient(),
	})
	if err = (&controller.ServiceRegistryReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceRegistry")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&mosniov1.HTTPFilterPolicy{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "HTTPFilterPolicy")
			os.Exit(1)
		}

		mosniov1.RegisterVirtualServiceWebhook(mgr)

		if err = (&mosniov1.Consumer{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Consumer")
			os.Exit(1)
		}
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
