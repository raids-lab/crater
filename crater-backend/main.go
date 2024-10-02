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
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	aisystemv1alpha1 "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	imagepackv1 "github.com/raids-lab/crater/pkg/apis/imagepack/v1"
	recommenddljob "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
	"github.com/raids-lab/crater/pkg/config"
	db "github.com/raids-lab/crater/pkg/db/orm"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/profiler"
	"github.com/raids-lab/crater/pkg/reconciler"
	"github.com/raids-lab/crater/pkg/util"
)

// @title Crater API
// @version 0.3.0
// @description This is the API server for Crater, a Multi-tenant AI Model Training Platform based on Kubernetes.
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description 访问 /login 并获取 TOKEN 后，填入 'Bearer ${TOKEN}' 以访问受保护的接口
//
//nolint:gocyclo // TODO: refactor main function
func main() {
	// set global timezone
	time.Local = time.UTC
	// set ctrl inner logger
	opts := zap.Options{
		Development:     true,
		StacktraceLevel: zapcore.DPanicLevel,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	// create new ctrl logger with specific name
	setupLog := ctrl.Log.WithName("setup")
	// load backend config from file
	backendConfig := config.GetConfig()
	// variable changes in local development
	if gin.Mode() == gin.DebugMode {
		err := godotenv.Load(".debug.env")
		if err != nil {
			panic(err.Error())
		}
		be := os.Getenv("CRATER_BE_PORT")
		if be == "" {
			panic("CRATER_BE_PORT is not set")
		}
		ms := os.Getenv("CRATER_MS_PORT")
		if ms == "" {
			panic("CRATER_MS_PORT is not set")
		}
		hp := os.Getenv("CRATER_HP_PORT")
		if hp == "" {
			panic("CRATER_HP_PORT is not set")
		}
		backendConfig.ProbeAddr = ":" + hp
		backendConfig.MetricsAddr = ":" + ms
		backendConfig.ServerAddr = ":" + be
	}

	// 0. create manager
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(schedulerpluginsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(aisystemv1alpha1.AddToScheme(scheme))
	utilruntime.Must(recommenddljob.AddToScheme(scheme))
	utilruntime.Must(imagepackv1.AddToScheme(scheme))
	utilruntime.Must(scheduling.AddToScheme(scheme))
	utilruntime.Must(batch.AddToScheme(scheme))

	// get k8s config via ./kubeconfig
	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				config.GetConfig().Workspace.Namespace:      {},
				config.GetConfig().Workspace.ImageNameSpace: {},
			},
		},
		Metrics: metricsserver.Options{
			BindAddress: backendConfig.MetricsAddr,
		},
		HealthProbeBindAddress: backendConfig.ProbeAddr,
		LeaderElection:         backendConfig.EnableLeaderElection,
		LeaderElectionID:       backendConfig.LeaderElectionID,
		// Namespace:              namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create a new field indexer
	if err = mgr.GetFieldIndexer().IndexField(context.Background(), &batch.Job{}, "spec.queue", func(rawObj client.Object) []string {
		// Extract the `spec.queue` field from the Job object
		job := rawObj.(*batch.Job)
		return []string{job.Spec.Queue}
	}); err != nil {
		setupLog.Error(err, "unable to index field spec.queue")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}

	// 1. init db (Used for aijob)
	err = db.InitDB()
	if err != nil {
		setupLog.Error(err, "unable to init db")
		os.Exit(1)
	}

	query.SetDefault(query.GetDB())

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

	vcjobReconciler := reconciler.NewVcJobReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
	)
	err = vcjobReconciler.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to set up vcjob controller")
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
	prometheusClient := monitor.NewPrometheusClient(backendConfig.PrometheusAPI)
	backend := internal.Register(mgr.GetClient(), clientset, prometheusClient, taskCtrl)
	if err := backend.R.Run(backendConfig.ServerAddr); err != nil {
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
