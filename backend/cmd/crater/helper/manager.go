package helper

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	aisystemv1alpha1 "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	recommenddljob "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/indexer"
	"github.com/raids-lab/crater/pkg/packer"
	"github.com/raids-lab/crater/pkg/reconciler"
	"github.com/raids-lab/crater/pkg/util"
)

// ManagerSetup 封装manager相关的设置逻辑
type ManagerSetup struct {
	cfg           *rest.Config
	backendConfig *config.Config
}

// NewManagerSetup 创建新的ManagerSetup实例
func NewManagerSetup(cfg *rest.Config, backendConfig *config.Config) *ManagerSetup {
	return &ManagerSetup{
		cfg:           cfg,
		backendConfig: backendConfig,
	}
}

// CreateCRDManager 创建CRD管理器
func (ms *ManagerSetup) CreateCRDManager() (manager.Manager, error) {
	// create manager
	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(schedulerpluginsv1alpha1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ms.cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: ms.backendConfig.MetricsAddr,
		},
		HealthProbeBindAddress: ms.backendConfig.ProbeAddr,
		LeaderElection:         ms.backendConfig.EnableLeaderElection,
		LeaderElectionID:       ms.backendConfig.LeaderElectionID,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create manager: %w", err)
	}

	return mgr, nil
}

// SetupCustomCRDAddon 设置自定义CRD附加组件
func (ms *ManagerSetup) SetupCustomCRDAddon(
	mgr manager.Manager,
	registerConfig *handler.RegisterConfig,
	stopCh context.Context,
) error {
	// Setup AIJob
	if err := ms.setupEMIASJob(mgr, registerConfig, stopCh); err != nil {
		return err
	}

	// Setup SPJob
	if err := ms.setupSEACSJob(mgr); err != nil {
		return err
	}

	// Setup ImagePacker
	if err := ms.setupImagePacker(mgr, registerConfig); err != nil {
		return err
	}

	// Setup Volcano
	if err := ms.setupVolcano(mgr, registerConfig); err != nil {
		return err
	}

	// Setup Indexeres
	if err := indexer.SetupIndexers(mgr); err != nil {
		return err
	}

	return nil
}

// setupEMIASJob 设置AIJob相关组件
func (ms *ManagerSetup) setupEMIASJob(mgr manager.Manager, registerConfig *handler.RegisterConfig, stopCh context.Context) error {
	var taskCtrl aitaskctl.TaskControllerInterface
	if ms.backendConfig.SchedulerPlugins.EMIAS.Enable {
		utilruntime.Must(aisystemv1alpha1.AddToScheme(mgr.GetScheme()))

		// 1. init task controller
		jobStatusChan := make(chan util.JobStatusChan)

		taskCtrl = aitaskctl.NewTaskController(
			registerConfig.ServiceManager,
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
		if ms.backendConfig.SchedulerPlugins.EMIAS.EnableProfiling {
			aijobProfiler := aitaskctl.NewProfiler(mgr, registerConfig.PrometheusClient, ms.backendConfig.SchedulerPlugins.EMIAS.ProfilingTimeout)
			taskCtrl.SetProfiler(aijobProfiler)
			aijobProfiler.Start(stopCh)
		}

		err = taskCtrl.Start(stopCh)
		if err != nil {
			return fmt.Errorf("unable to start task controller: %w", err)
		}
	}
	registerConfig.AITaskCtrl = taskCtrl
	return nil
}

// setupSEACSJob 设置SPJob相关组件
func (ms *ManagerSetup) setupSEACSJob(mgr manager.Manager) error {
	if ms.backendConfig.SchedulerPlugins.SEACS.Enable {
		utilruntime.Must(recommenddljob.AddToScheme(mgr.GetScheme()))
	}
	return nil
}

// setupImagePacker 设置ImagePacker相关组件
func (ms *ManagerSetup) setupImagePacker(mgr manager.Manager, registerConfig *handler.RegisterConfig) error {
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
	return nil
}

// setupVolcano 设置Volcano相关组件
func (ms *ManagerSetup) setupVolcano(mgr manager.Manager, registerConfig *handler.RegisterConfig) error {
	utilruntime.Must(scheduling.AddToScheme(mgr.GetScheme()))
	utilruntime.Must(batch.AddToScheme(mgr.GetScheme()))

	vcjobReconciler := reconciler.NewVcJobReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		registerConfig.PrometheusClient,
		registerConfig.KubeClient,
	)
	err := vcjobReconciler.SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("unable to set up vcjob controller: %w", err)
	}
	return nil
}
