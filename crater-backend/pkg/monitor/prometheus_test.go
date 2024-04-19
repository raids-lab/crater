package monitor

import (
	"fmt"
	"log"
	"testing"
)

func TestQueryPodUtilMetric(t *testing.T) {
	// 请替换为你的 Prometheus API 地址
	apiURL := ***REMOVED***
	client := NewPrometheusClient(apiURL)
	// print url
	fmt.Println(apiURL)
	podUtil, err := client.queryMetric("kube_gpu_scheduler_pod_bind_gpu")
	if err != nil {
		t.Errorf("Error querying PodUtilMetric: %v", err)
		return
	}
	fmt.Printf("PodUtilMetric: %+v\n", podUtil)
}

func TestQueryPodUtilMetric3(t *testing.T) {
	// 请替换为你的 Prometheus API 地址
	apiURL := ***REMOVED***
	client := NewPrometheusClient(apiURL)

	// print url
	fmt.Println(apiURL)
	expression := "DCGM_FI_DEV_GPU_UTIL"
	data, err := client.GetNodeGPUUtil(expression)
	fmt.Printf("data: %+v\n", data)
	if err != nil {
		log.Fatalf("queryMetric error: %v", err)
		t.Errorf("Error querying PodUtilMetric: %v", err)
	}
}
