package monitor

import (
	"fmt"
	"testing"
)

func TestQueryPodUtilMetric(t *testing.T) {
	// 请替换为你的 Prometheus API 地址
	apiURL := ***REMOVED***
	client := NewPrometheusClient(apiURL)

	// 请替换为你的测试 namespace 和 pod 名称
	namespace := "jupyter"
	podname := "img2img-train"

	podUtil, err := client.QueryPodUtilMetric(namespace, podname)
	if err != nil {
		t.Errorf("Error querying PodUtilMetric: %v", err)
		return
	}
	fmt.Printf("PodUtilMetric: %+v\n", podUtil)
}
