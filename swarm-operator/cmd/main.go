/*
Copyright 2025 The Claude Flow Authors.

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
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
	"github.com/claude-flow/swarm-operator/controllers"
	"github.com/claude-flow/swarm-operator/pkg/metrics"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(swarmv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var watchNamespaces string
	var swarmNamespace string
	var hivemindNamespace string
	
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.StringVar(&watchNamespaces, "watch-namespaces", "claude-flow-swarm,claude-flow-hivemind",
		"Comma-separated list of namespaces to watch. Default: claude-flow-swarm,claude-flow-hivemind")
	flag.StringVar(&swarmNamespace, "swarm-namespace", "claude-flow-swarm",
		"Default namespace for swarm agents")
	flag.StringVar(&hivemindNamespace, "hivemind-namespace", "claude-flow-hivemind",
		"Default namespace for hive-mind components")
	
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Create metrics recorder
	metricsRecorder := metrics.NewMetricsRecorder()

	// Parse watch namespaces
	namespaces := strings.Split(watchNamespaces, ",")
	for i := range namespaces {
		namespaces[i] = strings.TrimSpace(namespaces[i])
	}

	// Configure cache options for multi-namespace watching
	cacheOptions := cache.Options{
		DefaultNamespaces: map[string]cache.Config{},
	}
	for _, ns := range namespaces {
		if ns != "" {
			cacheOptions.DefaultNamespaces[ns] = cache.Config{}
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Cache:  cacheOptions,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       nil,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "d3f7a829.claudeflow.io",
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

	// Setup SwarmCluster controller
	if err = (&controllers.SwarmClusterReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Recorder:          mgr.GetEventRecorderFor("swarmcluster-controller"),
		SwarmNamespace:    swarmNamespace,
		HiveMindNamespace: hivemindNamespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SwarmCluster")
		os.Exit(1)
	}

	// Setup Agent controller
	if err = (&controllers.AgentReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		Recorder:        mgr.GetEventRecorderFor("agent-controller"),
		MetricsRecorder: metricsRecorder,
		SwarmNamespace:  swarmNamespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Agent")
		os.Exit(1)
	}
	
	// Setup SwarmTask controller
	if err = (&controllers.SwarmTaskReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Recorder:          mgr.GetEventRecorderFor("swarmtask-controller"),
		SwarmNamespace:    swarmNamespace,
		HiveMindNamespace: hivemindNamespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SwarmTask")
		os.Exit(1)
	}
	
	// Setup SwarmMemoryStore controller
	if err = (&controllers.SwarmMemoryStoreReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		SwarmNamespace: swarmNamespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SwarmMemoryStore")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager",
		"watchNamespaces", namespaces,
		"swarmNamespace", swarmNamespace,
		"hivemindNamespace", hivemindNamespace)
	
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}