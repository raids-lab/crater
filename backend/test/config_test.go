package test

import (
	"fmt"
	"testing"

	config2 "github.com/raids-lab/crater/pkg/config"
)

func TestYamlConfig(_ *testing.T) {
	// 请替换为你的 Prometheus API 地址
	config, err := config2.InitConfig()
	fmt.Println(config)
	fmt.Println(err)
}
