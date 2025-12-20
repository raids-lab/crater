package monitor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/klog/v2"
)

const (
	queryTimeout = 30 * time.Second
)

//nolint:gocritic // TODO: remove no linter
func PodUtilToJSON(podUtil PodUtil) string {
	jsonBytes, err := json.Marshal(podUtil)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

func JSONToPodUtil(str string) (PodUtil, error) {
	ret := PodUtil{}
	if str == "" {
		return ret, nil
	}
	err := json.Unmarshal([]byte(str), &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

type PrometheusClient struct {
	client api.Client
	v1api  v1.API
}

func NewPrometheusClient(apiURL string) PrometheusInterface {
	client, err := api.NewClient(api.Config{
		Address: apiURL,
	})
	v1api := v1.NewAPI(client)
	if err != nil {
		klog.Errorf("failed to create Prometheus client: %v", err)
		panic(err)
	}
	return &PrometheusClient{
		client: client,
		v1api:  v1api,
	}
}

func (p *PrometheusClient) QueryGpuAnalysisMetrics(namespace, podname string, duration time.Duration) (*GpuAnalysisMetrics, error) {
	metrics := &GpuAnalysisMetrics{}
	var err error

	// 将 duration 转换为 Prometheus 的 range vector 格式, e.g., "120m"
	promRange := fmt.Sprintf("%dm", int(duration.Minutes()))

	// --- 1. 查询 GPU 利用率 (DCGM_FI_DEV_GPU_UTIL) ---
	// 平均值 (原始值为 0-100)
	metrics.GpuUtilAvg, err = p.queryMetric(
		fmt.Sprintf(`avg_over_time(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q}[%s])`, namespace, podname, promRange))
	if err != nil {
		// 我们不直接返回错误，而是记录日志，以便即使缺少某个指标也能继续分析
		klog.Warningf("Failed to query GpuUtilAvg for pod %s: %v", podname, err)
	}

	//nolint:gocritic // Linter incorrectly flags this Chinese comment as code.
	// 标准差 (越小代表利用率曲线越平稳)
	metrics.GpuUtilStdDev, err = p.queryMetric(
		fmt.Sprintf(`stddev_over_time(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q}[%s])`, namespace, podname, promRange))
	if err != nil {
		klog.Warningf("Failed to query GpuUtilStdDev for pod %s: %v", podname, err)
	}

	// --- 2. 查询 GPU 显存使用 (DCGM_FI_DEV_FB_USED in MB) ---
	// 平均值 (MB)
	metrics.GpuMemUsedAvg, err = p.queryMetric(
		fmt.Sprintf(`avg_over_time(DCGM_FI_DEV_FB_USED{namespace=%q,pod=%q}[%s])`, namespace, podname, promRange))
	if err != nil {
		klog.Warningf("Failed to query GpuMemUsedAvg for pod %s: %v", podname, err)
	}

	// 标准差
	metrics.GpuMemUsedStdDev, err = p.queryMetric(
		fmt.Sprintf(`stddev_over_time(DCGM_FI_DEV_FB_USED{namespace=%q,pod=%q}[%s])`, namespace, podname, promRange))
	if err != nil {
		klog.Warningf("Failed to query GpuMemUsedStdDev for pod %s: %v", podname, err)
	}

	return metrics, nil
}
