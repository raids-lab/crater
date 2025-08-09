package config

import (
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type Config struct {
	// Leader Election Settings
	EnableLeaderElection bool `json:"enableLeaderElection"` // "Enable leader election for controller manager.
	// Profiling Settings
	PrometheusAPI string `json:"prometheusAPI"`

	// Port Settings
	Host        string `json:"host"`        // The domain name of the server.
	ServerAddr  string `json:"serverAddr"`  // The address the server endpoint binds to.
	MetricsAddr string `json:"metricsAddr"` // The address the metric endpoint binds to.
	ProbeAddr   string `json:"probeAddr"`   // The address the probe endpoint binds to.

	Auth struct {
		AccessTokenSecret  string `json:"accessTokenSecret"`
		RefreshTokenSecret string `json:"refreshTokenSecret"`
	} `json:"auth"`

	// New DB Settings
	Postgres struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		DBName   string `json:"dbname"`
		User     string `json:"user"`
		Password string `json:"password"`
		SSLMode  string `json:"sslmode"`
		TimeZone string `json:"TimeZone"`
	} `json:"postgres"`

	Storage struct {
		RWXPVCName string `json:"rwxpvcName"`
		ROXPVCName string `json:"roxpvcName"`
		Prefix     struct {
			User    string `json:"user"`    // The prefix for user storage paths.
			Account string `json:"account"` // The prefix for account storage paths.
			Public  string `json:"public"`  // The prefix for public storage paths.
		} `json:"prefix"` // The prefix for storage paths.
	} `json:"storage"`

	// Workspace Settings
	Workspace struct {
		Namespace      string `json:"namespace"`
		ImageNamespace string `json:"imageNameSpace"`
	} `json:"workspace"`

	Secrets struct {
		TLSSecretName        string `json:"tlsSecretName"`        // The name of the TLS secret.
		TLSForwardSecretName string `json:"tlsForwardSecretName"` // The name of the TLS forward secret.
		ImagePullSecretName  string `json:"imagePullSecretName"`  // The name of the image pull secret.
	}

	ImageRegistry struct {
		Server        string `json:"server"`
		User          string `json:"user"`
		Password      string `json:"password"`
		Project       string `json:"project"`
		Admin         string `json:"admin"`
		AdminPassword string `json:"adminPassword"`
	} `json:"imageRegistry"`

	// image build tools
	ImageBuildTools struct {
		BuildxImage  string `json:"buildxImage"` // Image for buildx frontend
		NerdctlImage string `json:"nerdctlImage"`
		EnvdImage    string `json:"envdImage"`
	} `json:"imageBuildTools"`

	SMTP struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Notify   string `json:"notify"`
	} `json:"smtp"`

	RaidsLab struct {
		Enable bool `json:"enable"` // If true, the user must sign up with token.
		LDAP   struct {
			UserName string `json:"userName"`
			Password string `json:"password"`
			Address  string `json:"address"`
			SearchDN string `json:"searchDN"`
		} `json:"ldap"`
		OpenAPI      ACTOpenAPI `json:"openAPI"`      // The Token configuration for ACT OpenAPI
		UIDServerURL string     `json:"uidServerURL"` // The URL of the UID server
	} `json:"raidsLab"`

	// scheduler plugin
	SchedulerPlugins struct {
		EMIAS struct {
			Enable           bool `json:"enable"`
			EnableProfiling  bool `json:"enableProfiling"`
			ProfilingTimeout int  `json:"profilingTimeout"`
		} `json:"aijob"`
		SEACS struct {
			Enable                   bool   `json:"enable"`
			PredictionServiceAddress string `json:"predictionServiceAddress"`
		} `json:"spjob"`
	} `json:"schedulerPlugins"`
}

type ACTOpenAPI struct {
	URL          string `json:"url"`
	ChameleonKey string `json:"chameleonKey"`
	AccessToken  string `json:"accessToken"`
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

func IsDebugMode() bool {
	return gin.Mode() == gin.DebugMode
}

// InitConfig initializes the configuration by reading the configuration file.
// If the environment is set to debug, it reads the debug-config.yaml file.
// Otherwise, it reads the config.yaml file from ConfigMap.
// It returns a pointer to the Config struct and an error if any occurred.
func initConfig() *Config {
	// 读取配置文件
	config := &Config{}
	var configPath string
	if IsDebugMode() {
		if os.Getenv("CRATER_DEBUG_CONFIG_PATH") != "" {
			configPath = os.Getenv("CRATER_DEBUG_CONFIG_PATH")
		} else {
			configPath = "./etc/debug-config.yaml"
		}
	} else {
		configPath = "/etc/config/config.yaml"
	}
	klog.Info("config path: ", configPath)

	err := readConfig(configPath, config)
	if err != nil {
		klog.Error("init config", err)
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
