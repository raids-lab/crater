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

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	aisystemv1alpha1 "github.com/aisystem/ai-protal/pkg/apis/aijob/v1alpha1"
	db "github.com/aisystem/ai-protal/pkg/db/orm"
	"github.com/aisystem/ai-protal/pkg/reconciler"
	"github.com/aisystem/ai-protal/pkg/server"
	"github.com/aisystem/ai-protal/pkg/taskqueue"
	"github.com/aisystem/ai-protal/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(schedulerpluginsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(aisystemv1alpha1.AddToScheme(scheme))
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
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&leaderElectionID, "leader-election-id", ***REMOVED***, "The ID for leader election.")
	flag.StringVar(&gangSchedulerName, "gang-scheduler-name", "", "Now Supporting volcano and scheduler-plugins."+
		" Note: If you set another scheduler name, the training-operator assumes it's the scheduler-plugins.")
	flag.IntVar(&monitoringPort, "monitoring-port", 9443, "Endpoint port for displaying monitoring metrics. "+
		"It can be set to \"0\" to disable the metrics serving.")
	flag.IntVar(&controllerThreads, "controller-threads", 1, "Number of worker threads used by the controller.")
	flag.StringVar(&serverPort, "server-port", ":8088", "The address the server endpoint binds to.")

	opts := zap.Options{
		Development:     true,
		StacktraceLevel: zapcore.DPanicLevel,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// create manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   monitoringPort,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
		// Namespace:              namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// 1. init db
	db.InitDB()
	db.InitMigration()

	// 2. init task controller

	taskUpdateChan := make(chan util.TaskUpdateChan)
	jobStatusChan := make(chan util.JobStatusChan)

	backend, err := server.Register(taskUpdateChan)
	taskCtrl := taskqueue.NewTaskController(
		mgr.GetClient(),
		jobStatusChan,
		taskUpdateChan,
	)
	if err := taskCtrl.Init(); err != nil {
		setupLog.Error(err, "unable to set up task controller")
		os.Exit(1)
	}

	// 3. init job controller

	jobReconciler := reconciler.NewAIJobReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		jobStatusChan,
	)

	jobReconciler.SetupWithManager(mgr)

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// 4. start manager
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

	// 5. start server
	setupLog.Info("starting server")
	if err := backend.R.Run(serverPort); err != nil {
		setupLog.Error(err, "problem running server")
		os.Exit(1)
	}

}
