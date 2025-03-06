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
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	aisystemv1alpha1 "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	recommenddljob "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
	"github.com/raids-lab/crater/pkg/config"
	db "github.com/raids-lab/crater/pkg/db/orm"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/packer"
	"github.com/raids-lab/crater/pkg/profiler"
	"github.com/raids-lab/crater/pkg/reconciler"
	"github.com/raids-lab/crater/pkg/util"
)

func setupCRDManager(cfg *rest.Config, backendConfig *config.Config) (manager.Manager, error) {
	// create manager
	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(schedulerpluginsv1alpha1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				config.GetConfig().Workspace.Namespace:      {},
				config.GetConfig().Workspace.ImageNamespace: {},
			},
		},
		Metrics: metricsserver.Options{
			BindAddress: backendConfig.MetricsAddr,
		},
		HealthProbeBindAddress: backendConfig.ProbeAddr,
		LeaderElection:         backendConfig.EnableLeaderElection,
		LeaderElectionID:       backendConfig.LeaderElectionID,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create manager: %w", err)
	}

	return mgr, nil
}

func setupCustomCRDAddon(
	mgr manager.Manager,
	backendConfig *config.Config,
	registerConfig *handler.RegisterConfig,
	stopCh context.Context,
) error {
	//-------aijob-------
	var taskCtrl aitaskctl.TaskControllerInterface
	if backendConfig.SchedulerFlags.AijobEn {
		utilruntime.Must(aisystemv1alpha1.AddToScheme(mgr.GetScheme()))
		// 1. init task controller
		// taskUpdateChan := make(chan util.TaskUpdateChan)
		jobStatusChan := make(chan util.JobStatusChan)

		taskCtrl = aitaskctl.NewTaskController(
			mgr.GetClient(),
			registerConfig.KubeClient,
			jobStatusChan,
		)
		err := taskCtrl.Init()
		if err != nil {
			return fmt.Errorf("unable to set up task controller: %w", err)
		}

		// 2. init job controller
		jobReconciler := reconciler.NewAIJobReconciler(
			mgr.GetClient(),
			mgr.GetScheme(),
			jobStatusChan,
		)
		err = jobReconciler.SetupWithManager(mgr)
		if err != nil {
			return fmt.Errorf("unable to set up job controller: %w", err)
		}

		// 3. profiler config
		if backendConfig.EnableProfiling {
			aijobProfiler := profiler.NewProfiler(mgr, registerConfig.PrometheusClient, backendConfig.ProfilingTimeout)
			taskCtrl.SetProfiler(aijobProfiler)
			// todo: start profiling
			aijobProfiler.Start(stopCh)
		}

		err = taskCtrl.Start(stopCh)
		if err != nil {
			return fmt.Errorf("unable to start task controller: %w", err)
		}
	}
	registerConfig.AITaskCtrl = taskCtrl
	//-------spjob-------
	if backendConfig.SchedulerFlags.SpjobEn {
		utilruntime.Must(recommenddljob.AddToScheme(mgr.GetScheme()))
	}

	//-------imagepacker-------
	imageRegistry := imageregistry.NewImageRegistry()
	buildkitReconciler := reconciler.NewBuildKitReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		imageRegistry,
	)
	err := buildkitReconciler.SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("unable to set up buildkit controller: %w", err)
	}
	registerConfig.ImagePacker = packer.GetImagePackerMgr(mgr.GetClient())
	registerConfig.ImageRegistry = imageRegistry

	//-------volcano-------
	utilruntime.Must(scheduling.AddToScheme(mgr.GetScheme()))
	utilruntime.Must(batch.AddToScheme(mgr.GetScheme()))
	// Create a new field indexer
	if err = mgr.GetFieldIndexer().IndexField(context.Background(), &batch.Job{}, "spec.queue", func(rawObj client.Object) []string {
		// Extract the `spec.queue` field from the Job object
		job := rawObj.(*batch.Job)
		return []string{job.Spec.Queue}
	}); err != nil {
		return fmt.Errorf("unable to index field spec.queue: %w", err)
	}
	vcjobReconciler := reconciler.NewVcJobReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		registerConfig.PrometheusClient,
	)
	err = vcjobReconciler.SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("unable to set up vcjob controller: %w", err)
	}
	return nil
}

// @title Crater API
// @version 0.3.0
// @description This is the API server for Crater, a Multi-tenant AI Model Training Platform based on Kubernetes.
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description 访问 /login 并获取 TOKEN 后，填入 'Bearer ${TOKEN}' 以访问受保护的接口
func main() {
	//-------------------backend----------------------
	// set global timezone
	time.Local = time.UTC
	// load backend config from file
	backendConfig := config.GetConfig()
	// init gin registerConfig
	registerConfig := handler.RegisterConfig{}
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
	// get k8s config via ./kubeconfig
	cfg := ctrl.GetConfigOrDie()
	registerConfig.Kubeconfig = cfg
	// kube clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	registerConfig.KubeClient = clientset
	if err != nil {
		panic(err.Error())
	}
	// init db
	err = db.InitDB()
	if err != nil {
		logutils.Log.Errorf("unable to init db:%q", err)
		os.Exit(1)
	}

	query.SetDefault(query.GetDB())
	// init promeClient
	prometheusClient := monitor.NewPrometheusClient(backendConfig.PrometheusAPI)
	registerConfig.PrometheusClient = prometheusClient
	//-------------------ctrl manager----------------------
	// init stopCh
	stopCh := ctrl.SetupSignalHandler()
	// set ctrl inner logger
	opts := zap.Options{
		Development:     true,
		StacktraceLevel: zapcore.DPanicLevel,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	// create new ctrl logger with specific name
	setupLog := ctrl.Log.WithName("setup")
	// create manager
	mgr, err := setupCRDManager(cfg, backendConfig)
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}
	registerConfig.Client = mgr.GetClient()
	// add custom CRD addon
	err = setupCustomCRDAddon(mgr, backendConfig, &registerConfig, stopCh)
	if err != nil {
		setupLog.Error(err, "unable to set up custom CRD addon")
		os.Exit(1)
	}

	// start manager
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

	// start server
	setupLog.Info("starting server")
	backend := internal.Register(&registerConfig)
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
