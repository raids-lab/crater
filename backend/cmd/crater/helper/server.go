package helper

import (
	"context"
	"os"

	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/raids-lab/crater/internal"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/pkg/config"
)

// ServerRunner 封装服务器运行逻辑
type ServerRunner struct {
	backendConfig *config.Config
}

// NewServerRunner 创建新的ServerRunner实例
func NewServerRunner(backendConfig *config.Config) *ServerRunner {
	return &ServerRunner{
		backendConfig: backendConfig,
	}
}

// SetupLogger 设置日志记录器
func (sr *ServerRunner) SetupLogger() {
	ctrl.SetLogger(klog.NewKlogr())
}

// StartManager 启动管理器
func (sr *ServerRunner) StartManager(mgr manager.Manager, stopCh context.Context) {
	klog.Info("starting manager")
	go func() {
		startErr := mgr.Start(stopCh)
		if startErr != nil {
			klog.Error(startErr, "problem running manager")
			os.Exit(1)
		}
	}()

	mgr.GetCache().WaitForCacheSync(stopCh)
	klog.Info("cache sync success")
}

// StartServer 启动HTTP服务器
func (sr *ServerRunner) StartServer(registerConfig *handler.RegisterConfig) {
	klog.Info("starting server")
	backend := internal.Register(registerConfig)
	if err := backend.R.Run(sr.backendConfig.ServerAddr); err != nil {
		klog.Error(err, "problem running server")
		os.Exit(1)
	}
}

// SetupHealthChecks 设置健康检查
func (sr *ServerRunner) SetupHealthChecks(mgr manager.Manager) {
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		klog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
}
