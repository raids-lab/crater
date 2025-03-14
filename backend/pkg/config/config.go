package config

import (
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/raids-lab/crater/pkg/logutils"
)

type Config struct {
	// Leader Election Settings
	EnableLeaderElection bool `yaml:"enableLeaderElection"` // "Enable leader election for controller manager.
	// Enabling this will ensure there is only one active controller manager."
	LeaderElectionID string `yaml:"leaderElectionID"` // "The ID for leader election."
	// Profiling Settings
	EnableProfiling  bool   `yaml:"enableProfiling"`
	PrometheusAPI    string `yaml:"prometheusAPI"`
	ProfilingTimeout int    `yaml:"profilingTimeout"`
	// New DB Settings
	Postgres struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		DBName   string `yaml:"dbname"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		SSLMode  string `yaml:"sslmode"`
		TimeZone string `yaml:"TimeZone"`
	} `yaml:"postgres"`
	UserSpacePrefix    string `yaml:"userSpacePrefix"`
	AccountSpacePrefix string `yaml:"accountSpacePrefix"`
	PublicSpacePrefix  string `yaml:"publicSpacePrefix"`
	// Port Settings
	Host           string `yaml:"host"`        // "The domain name of the server."
	ServerAddr     string `yaml:"serverAddr"`  // "The address the server endpoint binds to."
	MetricsAddr    string `yaml:"metricsAddr"` // "The address the metric endpoint binds to."
	ProbeAddr      string `yaml:"probeAddr"`   // "The address the probe endpoint binds to."
	MonitoringPort int    `yaml:"monitoringPort"`
	JYCache        bool   `yaml:"jycache"`
	// Workspace Settings
	Workspace struct {
		Namespace      string `yaml:"namespace"`
		RWXPVCName     string `yaml:"rwxpvcName"`
		ROXPVCName     string `yaml:"roxpvcName"`
		IngressName    string `yaml:"ingressName"`
		ImageNamespace string `yaml:"imageNameSpace"`
	} `yaml:"workspace"`
	ACT struct {
		Image struct {
			RegistryServer    string `yaml:"registryServer"`
			RegistryUser      string `yaml:"registryUser"`
			RegistryPass      string `yaml:"registryPass"`
			RegistryProject   string `yaml:"registryProject"`
			RegistryAdmin     string `yaml:"registryAdmin"`
			RegistryAdminPass string `yaml:"registryAdminPass"`
		} `yaml:"image"`
		Auth struct {
			UserName string `yaml:"userName"`
			Password string `yaml:"password"`
			Address  string `yaml:"address"`
			SearchDN string `yaml:"searchDN"`
		} `yaml:"auth"`
		SMTP struct {
			Host     string `yaml:"host"`
			Port     string `yaml:"port"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
			Notify   string `yaml:"notify"`
		} `yaml:"smtp"`
		OpenAPI            ACTOpenAPI `yaml:"openAPI"`            // The Token configuration for ACT OpenAPI
		StrictRegisterMode bool       `yaml:"strictRegisterMode"` // If true, the user must sign up with token.
		UIDServerURL       string     `yaml:"uidServerURL"`       // The URL of the UID server
	} `yaml:"act"`
	// WPSRobot Settings
	WPSRobot struct {
		WebhookAddress string `yaml:"webhookAddress"`
	} `yaml:"wpsRobot"`
	// scheduler plugin
	SchedulerPlugins struct {
		Aijob struct {
			AijobEn bool `yaml:"aijobEn"`
		} `yaml:"aijob"`
		Spjob struct {
			SpjobEn                  bool   `yaml:"spjobEn"`
			PredictionServiceAddress string `yaml:"predictionServiceAddress"`
		} `yaml:"spjob"`
	} `yaml:"schedulerPlugins"`
	// dind plugin
	DindArgs struct {
		BuildkitImage string `yaml:"buildkitImage"`
		NerdctlImage  string `yaml:"nerdctlImage"`
	} `yaml:"dindArgs"`
}

type ACTOpenAPI struct {
	URL          string `yaml:"url"`
	ChameleonKey string `yaml:"chameleonKey"`
	AccessToken  string `yaml:"accessToken"`
}

var (
	once   sync.Once
	config *Config
)

func GetConfig() *Config {
	once.Do(func() {
		config = initConfig()
	})
	return config
}

// InitConfig initializes the configuration by reading the configuration file.
// If the environment is set to debug, it reads the debug-config.yaml file.
// Otherwise, it reads the config.yaml file from ConfigMap.
// It returns a pointer to the Config struct and an error if any occurred.
func initConfig() *Config {
	// 读取配置文件
	config := &Config{}
	var configPath string
	if gin.Mode() == gin.DebugMode {
		configPath = "./etc/debug-config.yaml"
	} else {
		configPath = "/etc/config/config.yaml"
	}

	err := readConfig(configPath, config)
	if err != nil {
		logutils.Log.Error("init config", err)
		panic(err)
	}
	return config
}

func readConfig(filePath string, config *Config) error {
	// 读取 YAML 配置文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	// 解析 YAML 数据到结构体
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return err
	}
	return nil
}
