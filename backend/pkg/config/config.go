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
	// DB Settings
	DBHost              string `yaml:"dbHost"`
	DBPort              string `yaml:"dbPort"`
	DBUser              string `yaml:"dbUser"`
	DBPassword          string `yaml:"dbPassword"`
	DBName              string `yaml:"dbName"`
	DBCharset           string `yaml:"dbCharset"`
	DBConnectionTimeout int    `yaml:"dbConnTimeout"`
	// Port Settings
	ServerAddr     string `yaml:"serverAddr"`
	MetricsAddr    string `yaml:"metricsAddr"`
	ProbeAddr      string `yaml:"probeAddr"`
	MonitoringPort int    `yaml:"monitoringPort"`
}

func NewConfig(configPath string) (*Config, error) {
	config := &Config{}
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
