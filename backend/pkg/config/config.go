package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	// Profiling Settings
	EnableProfiling  bool   `yaml:"enableProfiling"`
	PrometheusAPI    string `yaml:"prometheusAPI"`
	ProfilingTimeout int    `yaml:"profilingTimeout"`
	// todo: DB Settings
	DBHost              string `yaml:"dbHost"`
	DBPort              string `yaml:"dbPort"`
	DBUser              string `yaml:"dbUser"`
	DBPassword          string `yaml:"dbPassword"`
	DBName              string `yaml:"dbName"`
	DBCharset           string `yaml:"dbCharset"`
	DBConnectionTimeout int    `yaml:"dbConnTimeout"`
	// todo: Port Settings
	ServerAddr     string `yaml:"serverAddr"`
	MetricsAddr    string `yaml:"metricsAddr"`
	ProbeAddr      string `yaml:"probeAddr"`
	MonitoringPort int    `yaml:"monitoringPort"`
}

func NewConfig(configPath string) (*Config, error) {
	// 设置默认值
	config := &Config{
		EnableProfiling: true,
		// PrometheusAPI:       "http://prometheus-k8s.kubesphere-monitoring-system",
		PrometheusAPI:       "http://***REMOVED***:31110/",
		ProfilingTimeout:    120, // todo:
		DBHost:              "mycluster.jupyter",
		DBPort:              "3306",
		DBUser:              "root",
		DBPassword:          "buaak8sportal@2023mysql",
		DBConnectionTimeout: 10,
		DBName:              "ai_portal",
		DBCharset:           "utf8mb4",
		ServerAddr:          ":8088",
		MetricsAddr:         ":8080",
		ProbeAddr:           ":8081",
		MonitoringPort:      9443,
	}
	// data, _ := yaml.Marshal(config)
	// os.WriteFile("config.yaml", data, 0644)
	// 读取配置文件
	if configPath == "" {
		return config, nil
	}

	err := readConfig(configPath, config)
	if err != nil {
		return nil, err
	}
	return config, nil
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
