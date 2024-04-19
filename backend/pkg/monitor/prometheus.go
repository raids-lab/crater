//nolint:lll // Promtheus Query is too long
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
	PrometheusAPI = ***REMOVED***
	queryTimeout  = 10 * time.Second
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
	podUtil.GPUUtilAvg, err = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace=%q,exported_pod=%q}[60s])/100", namespace, podname))
	if err != nil {
		return podUtil, err
	}
	podUtil.GPUUtilMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace=%q,exported_pod=%q}[60s])/100", namespace, podname))
	podUtil.GPUUtilStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace=%q,exported_pod=%q}[60s])/100", namespace, podname))
	podUtil.SMActiveAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_SM_ACTIVE{exported_namespace=%q,exported_pod=%q}[60s])", namespace, podname))
	podUtil.SMActiveMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_SM_ACTIVE{exported_namespace=%q,exported_pod=%q}[60s])", namespace, podname))
	podUtil.SMActiveStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_PROF_SM_ACTIVE{exported_namespace=%q,exported_pod=%q}[60s])", namespace, podname))
	podUtil.SMOccupancyAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_SM_OCCUPANCY{exported_namespace=%q,exported_pod=%q}[60s])", namespace, podname))
	podUtil.SMOccupancyMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_SM_OCCUPANCY{exported_namespace=%q,exported_pod=%q}[60s])", namespace, podname))
	podUtil.SMOccupancyStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_PROF_SM_OCCUPANCY{exported_namespace=%q,exported_pod=%q}[60s])", namespace, podname))
	podUtil.DramUtilAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_DRAM_ACTIVE{exported_namespace=%q,exported_pod=%q}[60s])", namespace, podname))
	podUtil.DramUtilMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_DRAM_ACTIVE{exported_namespace=%q,exported_pod=%q}[60s])", namespace, podname))
	podUtil.DramUtilStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_PROF_DRAM_ACTIVE{exported_namespace=%q,exported_pod=%q}[60s])", namespace, podname))
	podUtil.MemCopyUtilAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace=%q,exported_pod=%q}[60s])/100", namespace, podname))
	podUtil.MemCopyUtilMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace=%q,exported_pod=%q}[60s])/100", namespace, podname))
	podUtil.MemCopyUtilStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace=%q,exported_pod=%q}[60s])/100", namespace, podname))
	podUtil.PCIETxAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_PCIE_TX_BYTES{exported_namespace=%q,exported_pod=%q}[60s])/1048576", namespace, podname))
	podUtil.PCIETxMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_PCIE_TX_BYTES{exported_namespace=%q,exported_pod=%q}[60s])/1048576", namespace, podname))
	podUtil.PCIERxAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_PCIE_RX_BYTES{exported_namespace=%q,exported_pod=%q}[60s])/1048576", namespace, podname))
	podUtil.PCIERxMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_PCIE_RX_BYTES{exported_namespace=%q,exported_pod=%q}[60s])/1048576", namespace, podname))
	podUtil.CPUUsageAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(irate(container_cpu_usage_seconds_total{namespace=%q,pod=%q,container!=\"POD\",container!=\"\"}[30s])[2m:])", namespace, podname))
	podUtil.GPUMemMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_DEV_FB_USED{exported_namespace=%q,exported_pod=%q}[60s])", namespace, podname))
	podUtil.CPUMemMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(container_memory_usage_bytes{namespace=%q,pod=%q,container!=\"POD\",container!=\"\"}[60s])/1048576", namespace, podname))

	// todo: check stats

	return podUtil, nil
}

func (p *PrometheusClient) queryMetric(expression string) (float32, error) {
	// 构建查询参数

	// 执行查询
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	result, _, err := p.v1api.Query(ctx, expression, time.Now())
	if err != nil {
		return 0.0, err
	}

	// 处理查询结果
	if result.Type() == model.ValVector {
		vector := result.(model.Vector)
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

func (p *PrometheusClient) QueryMetricDemo(expression string) error {
	// 构建查询参数

	// 执行查询
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	result, _, err := p.v1api.Query(ctx, expression, time.Now())
	if err != nil {
		return err
	}

	// 检查结果类型是否为向量
	if result.Type() == model.ValVector {
		vector := result.(model.Vector)
		for _, sample := range vector {
			metric := sample.Metric
			value := sample.Value

			// 提取metric中的标签和value中的值
			uuid := metric["UUID"]
			exportedPod := metric["exported_pod"]
			gpuCount := metric["gpu_count"]
			metricValue := float64(value)

			// 打印提取的信息
			fmt.Printf("UUID: %s, Pod: %s, GPU Count: %s, Value: %f\n", uuid, exportedPod, gpuCount, metricValue)
		}
		return nil
	}

	return fmt.Errorf("expected vector type result but got %s", result.Type())
}

// 公共的查询执行和结果验证逻辑
func (p *PrometheusClient) queryVector(query string) (model.Vector, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	result, _, err := p.v1api.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	if result.Type() != model.ValVector {
		return nil, fmt.Errorf("expected vector type result but got %s", result.Type())
	}

	return result.(model.Vector), nil
}

// Float32MapQuery 使用 queryVector 函数并处理 float32 类型的映射
func (p *PrometheusClient) Float32MapQuery(query, key string) (map[string]float32, error) {
	vector, err := p.queryVector(query)
	if err != nil {
		return nil, err
	}

	cpuUsages := make(map[string]float32)
	for _, sample := range vector {
		instance := string(sample.Metric[model.LabelName(key)])
		value := float32(sample.Value)
		cpuUsages[instance] = value
	}

	return cpuUsages, nil
}

// IntMapQuery 使用 queryVector 函数并处理 int 类型的映射
func (p *PrometheusClient) IntMapQuery(query, key string) (map[string]int, error) {
	vector, err := p.queryVector(query)
	if err != nil {
		return nil, err
	}

	cpuUsages := make(map[string]int)
	for _, sample := range vector {
		instance := string(sample.Metric[model.LabelName(key)])
		value := int(sample.Value)
		cpuUsages[instance] = value
	}

	return cpuUsages, nil
}

func (p *PrometheusClient) PodGPU(expression string) ([]PodGPUAllocate, error) {
	// 执行查询
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	result, _, err := p.v1api.Query(ctx, expression, time.Now())
	if err != nil {
		return nil, err
	}

	// 定义返回的结构体
	var podGPUAllocates []PodGPUAllocate

	// 检查结果类型是否为向量
	if result.Type() == model.ValVector {
		vector := result.(model.Vector)
		for _, sample := range vector {
			metric := sample.Metric
			value := sample.Value

			// 提取metric中的标签和value中的值
			node := metric["node"]
			instance := metric["instance"]
			pod := metric["pod"]
			Value := int(value)

			podGPUAllocateData := PodGPUAllocate{
				Node:     string(node),
				Instance: string(instance),
				Pod:      string(pod),
				GPUCount: Value,
			}
			podGPUAllocates = append(podGPUAllocates, podGPUAllocateData)
		}

		return podGPUAllocates, nil
	}

	return nil, fmt.Errorf("expected vector type result but got %s", result.Type())
}

func (p *PrometheusClient) GetNodeGPUUtil(expression string) ([]NodeGPUUtil, error) {
	// 执行查询
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	result, _, err := p.v1api.Query(ctx, expression, time.Now())
	if err != nil {
		return nil, err
	}

	// 定义返回的结构体
	var nodeGPUUtils []NodeGPUUtil

	// 检查结果类型是否为向量
	if result.Type() == model.ValVector {
		vector := result.(model.Vector)
		for _, sample := range vector {
			metric := sample.Metric

			// 提取metric中的标签和value中的值
			hostname := metric["Hostname"]
			uuid := metric["UUID"]
			container := metric["container"]
			device := metric["device"]
			endpoint := metric["endpoint"]
			gpu := metric["gpu"]
			instance := metric["instance"]
			job := metric["job"]
			modelName := metric["modelName"]
			namespace := metric["namespace"]
			pod := metric["pod"]
			service := metric["service"]
			util := float32(sample.Value)

			nodeGPUUtilData := NodeGPUUtil{
				Hostname:  string(hostname),
				UUID:      string(uuid),
				Container: string(container),
				Device:    string(device),
				Endpoint:  string(endpoint),
				Gpu:       string(gpu),
				Instance:  string(instance),
				Job:       string(job),
				ModelName: string(modelName),
				Namespace: string(namespace),
				Pod:       string(pod),
				Service:   string(service),
				Util:      util,
			}
			nodeGPUUtils = append(nodeGPUUtils, nodeGPUUtilData)
		}

		return nodeGPUUtils, nil
	}

	return nil, fmt.Errorf("expected vector type result but got %s", result.Type())
}
