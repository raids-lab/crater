/*
Copyright 2017 The Kubernetes Authors.

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

	"go.uber.org/zap/zapcore"

	"k8s.io/apimachinery/pkg/runtime"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	"github.com/raids-lab/crater/pkg/aitaskctl"
	aisystemv1alpha1 "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	recommenddljob "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
	"github.com/raids-lab/crater/pkg/config"
	db "github.com/raids-lab/crater/pkg/db/orm"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/profiler"
	"github.com/raids-lab/crater/pkg/reconciler"
	"github.com/raids-lab/crater/pkg/server"
	"github.com/raids-lab/crater/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

//nolint:gochecknoinits // todo: refactor
func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(schedulerpluginsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(aisystemv1alpha1.AddToScheme(scheme))
	utilruntime.Must(recommenddljob.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var leaderElectionID string
	var probeAddr string
	var gangSchedulerName string
	var monitoringPort int
	var controllerThreads int
	var serverPort string
	var dbConfigFile string
	var enableProfiling bool
	var configFile string
	var shareDirFile string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&leaderElectionID, "leader-election-id", ***REMOVED***, "The ID for leader election.")
	flag.StringVar(&gangSchedulerName, "gang-scheduler-name", "", "Now Supporting volcano and scheduler-plugins."+
		" Note: If you set another scheduler name, the training-operator assumes it's the scheduler-plugins.")
	//nolint:gomnd // TODO: is this necessary?
	flag.IntVar(&monitoringPort, "monitoring-port", 9443, "Endpoint port for displaying monitoring metrics. "+
		"It can be set to \"0\" to disable the metrics serving.")
	flag.IntVar(&controllerThreads, "controller-threads", 1, "Number of worker threads used by the controller.")
	flag.StringVar(&serverPort, "server-port", ":8088", "The address the server endpoint binds to.")
	flag.StringVar(&dbConfigFile, "db-config-file", "", "The db config file path.")
	flag.StringVar(&configFile, "config-file", "", "server config file")
	flag.StringVar(&shareDirFile, "share-dir-file", "", "share dir config file")
	flag.BoolVar(&enableProfiling, "enable-profiling", false, "Enable profiling.")
	opts := zap.Options{
		Development:     true,
		StacktraceLevel: zapcore.DPanicLevel,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	backendConfig, err := config.NewConfig(configFile)
	if err != nil {
		setupLog.Error(err, "unable to init config")
		os.Exit(1)
	}

	// 0. create manager
	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
		// Namespace:              namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}

	// 1. init db
	err = db.InitDB(backendConfig)
	if err != nil {
		setupLog.Error(err, "unable to init db")
		os.Exit(1)
	}
	err = db.InitMigration()
	if err != nil {
		setupLog.Error(err, "unable to init db migration")
		os.Exit(1)
	}

	// 2. init task controller
	// taskUpdateChan := make(chan util.TaskUpdateChan)
	jobStatusChan := make(chan util.JobStatusChan)

	taskCtrl := aitaskctl.NewTaskController(
		mgr.GetClient(),
		clientset,
		jobStatusChan,
	)
	err = taskCtrl.Init()
	if err != nil {
		setupLog.Error(err, "unable to set up task controller")
		os.Exit(1)
	}
	setupLog.Info("task controller init success")

	// 3. init job controller
	jobReconciler := reconciler.NewAIJobReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		jobStatusChan,
	)

	err = jobReconciler.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to set up job controller")
		os.Exit(1)
	}
	stopCh := ctrl.SetupSignalHandler()

	// 4. start manager
	setupLog.Info("starting manager")
	go func() {
		startErr := mgr.Start(stopCh)
		if startErr != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	}()

	mgr.GetCache().WaitForCacheSync(stopCh)
	setupLog.Info("cache sync success")

	// profiler config
	if backendConfig.EnableProfiling {
		prometheusClient := monitor.NewPrometheusClient(backendConfig.PrometheusAPI)
		aijobProfiler := profiler.NewProfiler(mgr, prometheusClient, backendConfig.ProfilingTimeout)
		taskCtrl.SetProfiler(aijobProfiler)
		// todo: start profiling
		aijobProfiler.Start(stopCh)
		setupLog.Info("enable profiling success")
	}

	err = taskCtrl.Start(stopCh)
	if err != nil {
		setupLog.Error(err, "unable to start task controller")
		os.Exit(1)
	}

	// 5. start server
	setupLog.Info("starting server")
	backend, err := server.Register(taskCtrl, mgr.GetClient(), clientset)
	if err != nil {
		setupLog.Error(err, "unable to set up server")
		os.Exit(1)
	}
	if err := backend.R.Run(serverPort); err != nil {
		setupLog.Error(err, "problem running server")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
}
