package monitor

import (
	"testing"
)

func TestQueryPodUtilMetric(t *testing.T) {
	// 请替换为你的 Prometheus API 地址
	apiURL := ***REMOVED***
	_ = NewPrometheusClient(apiURL)
	// print url
	t.Log(apiURL)
	//nolint:gocritic // TODO: remove no linter
	// podUtil, err := client.queryMetric("kube_gpu_scheduler_pod_bind_gpu")
	// if err != nil {
	// 	t.Errorf("Error querying PodUtilMetric: %v", err)
	// 	return
	// }
	// fmt.Printf("PodUtilMetric: %+v\n", podUtil)
}

func TestQueryPodUtilMetric3(t *testing.T) {
	// 请替换为你的 Prometheus API 地址
	apiURL := ***REMOVED***
	client := NewPrometheusClient(apiURL)

	// print url
	t.Log(apiURL)
	data := client.QueryNodeGPUUtil()
	t.Logf("data: %+v\n", data)
}
