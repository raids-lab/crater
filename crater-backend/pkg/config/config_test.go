package config

import (
	"fmt"
	"testing"
)

func TestYamlConfig(_ *testing.T) {
	// 请替换为你的 Prometheus API 地址
	config, err := NewConfig("")
	fmt.Println(config)
	fmt.Println(err)
}
