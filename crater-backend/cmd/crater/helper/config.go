package helper

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/monitor"
)

// ConfigInitializer 封装配置初始化逻辑
type ConfigInitializer struct {
	backendConfig *config.Config
}

// NewConfigInitializer 创建新的ConfigInitializer实例
func NewConfigInitializer() *ConfigInitializer {
	return &ConfigInitializer{
		backendConfig: config.GetConfig(),
	}
}

// GetBackendConfig 获取后端配置
func (ci *ConfigInitializer) GetBackendConfig() *config.Config {
	return ci.backendConfig
}

// LoadDebugEnvironment 加载调试环境变量
func (ci *ConfigInitializer) LoadDebugEnvironment() error {
	if gin.Mode() != gin.DebugMode {
		return nil
	}

	err := godotenv.Load(".debug.env")
	if err != nil {
		return err
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

	ci.backendConfig.ProbeAddr = ":" + hp
	ci.backendConfig.MetricsAddr = ":" + ms
	ci.backendConfig.ServerAddr = ":" + be

	return nil
}

// InitializeRegisterConfig 初始化注册配置
func (ci *ConfigInitializer) InitializeRegisterConfig() (*handler.RegisterConfig, error) {
	registerConfig := &handler.RegisterConfig{}

	// get k8s config
	cfg := ctrl.GetConfigOrDie()
	registerConfig.KubeConfig = cfg

	// kube clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	registerConfig.KubeClient = clientset

	// init db
	query.SetDefault(query.GetDB())

	// init prometheus client
	prometheusClient := monitor.NewPrometheusClient(ci.backendConfig.PrometheusAPI)
	registerConfig.PrometheusClient = prometheusClient

	return registerConfig, nil
}

// SetupManagerDependencies 设置管理器依赖项
func (ci *ConfigInitializer) SetupManagerDependencies(registerConfig *handler.RegisterConfig, mgr ctrl.Manager) {
	registerConfig.Client = mgr.GetClient()

	// 初始化 ServiceManager
	serviceManager := crclient.NewServiceManager(mgr.GetClient(), registerConfig.KubeClient)
	registerConfig.ServiceManager = serviceManager
}
