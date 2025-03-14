package config

import (
	"os"
	"sync"

	"webdav/logutils"

	"gopkg.in/yaml.v3"
)

type Config struct {
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
	configPath := "./etc/config.yaml"

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
