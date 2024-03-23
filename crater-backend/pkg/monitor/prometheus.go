package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	PrometheusAPI = "YOUR_PROMETHEUS_API_URL"
)

type PodUtil struct {
	GPUUtilAvg     float32 `json:"gpu_util_avg"`
	GPUUtilMax     float32 `json:"gpu_util_max"`
	GPUUtilStd     float32 `json:"gpu_util_std"`
	SMActiveAvg    float32 `json:"sm_active_avg"`
	SMActiveMax    float32 `json:"sm_active_max"`
	SMActiveStd    float32 `json:"sm_active_std"`
	SMOccupancyAvg float32 `json:"sm_occupancy_avg"`
	SMOccupancyMax float32 `json:"sm_occupancy_max"`
	SMOccupancyStd float32 `json:"sm_occupancy_std"`
	DramUtilAvg    float32 `json:"dram_util_avg"`
	DramUtilMax    float32 `json:"dram_util_max"`
	DramUtilStd    float32 `json:"dram_util_std"`
	MemCopyUtilAvg float32 `json:"mem_copy_util_avg"`
	MemCopyUtilMax float32 `json:"mem_copy_util_max"`
	MemCopyUtilStd float32 `json:"mem_copy_util_std"`
	PCIETxAvg      float32 `json:"pcie_tx_avg"`
	PCIETxMax      float32 `json:"pcie_tx_max"`
	PCIERxAvg      float32 `json:"pcie_rx_avg"`
	PCIERxMax      float32 `json:"pcie_rx_max"`
	CPUUsageAvg    float32 `json:"cpu_usage_avg"`
	GPUMemMax      float32 `json:"gpu_mem_max"`
	CPUMemMax      float32 `json:"cpu_mem_max"`
}

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

func NewPrometheusClient(apiURL string) *PrometheusClient {
	client, err := api.NewClient(api.Config{
		Address: apiURL,
	})
	v1api := v1.NewAPI(client)
	if err != nil {
		panic(err)
	}
	return &PrometheusClient{
		client: client,
		v1api:  v1api,
	}
}

func (p *PrometheusClient) QueryPodUtilMetric(namespace, podname string) (PodUtil, error) {

	podUtil := PodUtil{}
	var err error
	podUtil.GPUUtilAvg, err = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])/100", namespace, podname))
	if err != nil {
		return podUtil, err
	}
	podUtil.GPUUtilMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])/100", namespace, podname))
	podUtil.GPUUtilStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])/100", namespace, podname))
	podUtil.SMActiveAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_SM_ACTIVE{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])", namespace, podname))
	podUtil.SMActiveMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_SM_ACTIVE{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])", namespace, podname))
	podUtil.SMActiveStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_PROF_SM_ACTIVE{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])", namespace, podname))
	podUtil.SMOccupancyAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_SM_OCCUPANCY{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])", namespace, podname))
	podUtil.SMOccupancyMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_SM_OCCUPANCY{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])", namespace, podname))
	podUtil.SMOccupancyStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_PROF_SM_OCCUPANCY{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])", namespace, podname))
	podUtil.DramUtilAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_DRAM_ACTIVE{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])", namespace, podname))
	podUtil.DramUtilMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_DRAM_ACTIVE{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])", namespace, podname))
	podUtil.DramUtilStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_PROF_DRAM_ACTIVE{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])", namespace, podname))
	podUtil.MemCopyUtilAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])/100", namespace, podname))
	podUtil.MemCopyUtilMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])/100", namespace, podname))
	podUtil.MemCopyUtilStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])/100", namespace, podname))
	podUtil.PCIETxAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_PCIE_TX_BYTES{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])/1048576", namespace, podname))
	podUtil.PCIETxMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_PCIE_TX_BYTES{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])/1048576", namespace, podname))
	podUtil.PCIERxAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_PCIE_RX_BYTES{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])/1048576", namespace, podname))
	podUtil.PCIERxMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_PCIE_RX_BYTES{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])/1048576", namespace, podname))
	podUtil.CPUUsageAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(irate(container_cpu_usage_seconds_total{namespace=\"%s\",pod=\"%s\",container!=\"POD\",container!=\"\"}[30s])[2m:])", namespace, podname))
	podUtil.GPUMemMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_DEV_FB_USED{exported_namespace=\"%s\",exported_pod=\"%s\"}[60s])", namespace, podname))
	podUtil.CPUMemMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(container_memory_usage_bytes{namespace=\"%s\",pod=\"%s\",container!=\"POD\",container!=\"\"}[60s])/1048576", namespace, podname))

	// todo: check stats

	return podUtil, nil

}

func (p *PrometheusClient) queryMetric(expression string) (float32, error) {
	// 构建查询参数

	// 执行查询
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// logrus.Info(expression)
	result, _, err := p.v1api.Query(ctx, expression, time.Now())
	if err != nil {
		// logrus.Error(err)
		return 0.0, err
	}

	// 处理查询结果
	if result.Type() == model.ValVector {
		vector := result.(model.Vector)
		// logrus.Info(len(vector))
		if len(vector) > 0 {
			// 获取最新的样本值
			latestSample := vector[0]
			// 提取最新的值
			latestValue := latestSample.Value
			return float32(latestValue), nil
		}
	}

	return 0.0, fmt.Errorf("no data found for expression: %s", expression)
}
