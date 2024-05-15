package config

import (
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/logutils"
	"gopkg.in/yaml.v3"
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
	// DB Settings
	DBHost              string `yaml:"dbHost"`
	DBPort              string `yaml:"dbPort"`
	DBUser              string `yaml:"dbUser"`
	DBPassword          string `yaml:"dbPassword"`
	DBName              string `yaml:"dbName"`
	DBCharset           string `yaml:"dbCharset"`
	DBConnectionTimeout int    `yaml:"dbConnTimeout"`
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
	// Port Settings
	ServerAddr     string `yaml:"serverAddr"`  // "The address the server endpoint binds to."
	MetricsAddr    string `yaml:"metricsAddr"` // "The address the metric endpoint binds to."
	ProbeAddr      string `yaml:"probeAddr"`   // "The address the probe endpoint binds to."
	MonitoringPort int    `yaml:"monitoringPort"`
	// Workspace Settings
	Workspace struct {
		Namespace   string `yaml:"namespace"`
		PVCName     string `yaml:"pvcName"`
		IngressName string `yaml:"ingressName"`
	} `yaml:"workspace"`
	ACT struct {
		Image struct {
			RegistryServer  string `yaml:"registryServer"`
			RegistryUser    string `yaml:"registryUser"`
			RegistryPass    string `yaml:"registryPass"`
			RegistryProject string `yaml:"registryProject"`
		} `yaml:"image"`
		Auth struct {
			UserName string `yaml:"userName"`
			Password string `yaml:"password"`
			Address  string `yaml:"address"`
			SearchDN string `yaml:"searchDN"`
		} `yaml:"auth"`
	} `yaml:"act"`
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
