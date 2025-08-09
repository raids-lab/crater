package helper

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
func (sr *ServerRunner) StartManager(ctx context.Context, mgr manager.Manager) {
	klog.Info("starting manager")
	go func() {
		startErr := mgr.Start(ctx)
		if startErr != nil {
			klog.Error(startErr, "problem running manager")
			os.Exit(1)
		}
	}()

	mgr.GetCache().WaitForCacheSync(ctx)
	klog.Info("cache sync success")
}

var (
	readHeaderTimeout = 10 * time.Second // 设置读取头部的超时时间
	cancelTimeout     = 10 * time.Second // 设置取消操作的超时时间
)

// StartServer 启动HTTP服务器
func (sr *ServerRunner) StartServer(registerConfig *handler.RegisterConfig) {
	klog.Info("starting server")
	backend := internal.Register(registerConfig)

	// reference: https://gin-gonic.com/en/docs/examples/graceful-restart-or-stop
	srv := &http.Server{
		Addr:              sr.backendConfig.ServerAddr,
		Handler:           backend,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no params) by default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	klog.Info("Shutdown Gin Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), cancelTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		klog.Info("Gin Server Shutdown:", err)
	}
	klog.Info("Gin Server exiting")
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
