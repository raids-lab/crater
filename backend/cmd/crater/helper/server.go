package helper

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gin-gonic/gin"

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

	if gin.Mode() == gin.DebugMode {
		sr.setupSSProxy(backend.R)
	}
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

// 用于本地开发时转发ss的代理
func NewReverseProxy(targetAddr string) (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(targetAddr)
	if err != nil {
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			originalURL := *req.URL

			// 仅修改路径：/ss/* → /api/ss/*
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = "/api" + originalURL.Path

			req.URL.RawQuery = originalURL.RawQuery

			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Go-ReverseProxy)")
			}
			req.Header.Del("Accept-Encoding") // 防止压缩响应
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "Go-ReverseProxy")
			}
			klog.Infof("Proxying: %s%s → %s",
				req.Host,
				originalURL.Path,
				req.URL.String())
		},

		// 错误处理
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			klog.Errorf("[Proxy] 转发失败: %s → %v", r.URL.String(), err)
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("Bad Gateway"))
		},

		// 响应修改器
		ModifyResponse: func(resp *http.Response) error {
			klog.Infof("[Proxy] 响应: %s %d", resp.Request.URL, resp.StatusCode)

			return nil
		},
	}

	return proxy, nil
}

// setupSSProxy 配置/ss开头的请求代理
func (sr *ServerRunner) setupSSProxy(router *gin.Engine) {
	ssTarget := os.Getenv("CRATER_SS_TARGET")
	if ssTarget == "" {
		klog.Info("CRATER_SS_TARGET not set, skipping SS proxy")
		return
	}

	proxy, err := NewReverseProxy(ssTarget)
	if err != nil {
		klog.Errorf("Failed to create reverse proxy: %v", err)
		return
	}

	proxyHandler := func(c *gin.Context) {
		c.Writer.Header().Del("Access-Control-Allow-Origin")
		c.Writer.Header().Del("Access-Control-Allow-Credentials")
		proxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
	router.Any("/ss/*path", proxyHandler)
	router.Handle("MKCOL", "/ss/*path", proxyHandler)

	klog.Infof("SS proxy enabled: /ss/* → %s/api/ss/*", ssTarget)
}
