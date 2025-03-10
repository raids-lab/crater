package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/common/model"
	"k8s.io/apimachinery/pkg/types"
)

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

// float32MapQuery 使用 queryVector 函数并处理 float32 类型的映射
func (p *PrometheusClient) float32MapQuery(query, key string) (map[string]float32, error) {
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

// intMapQuery 使用 queryVector 函数并处理 int 类型的映射
func (p *PrometheusClient) intMapQuery(query, key string) (map[string]int, error) {
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

func (p *PrometheusClient) podGPU(expression string) ([]PodGPUAllocate, error) {
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

func (p *PrometheusClient) getNodeGPUUtil(expression string) ([]NodeGPUUtil, error) {
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
			var pod model.LabelValue
			if exportedPod, ok := metric["pod"]; ok {
				pod = exportedPod
			}
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

func (p *PrometheusClient) getJobPods(expression string) (map[string][]string, error) {
	// 执行查询
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	result, _, err := p.v1api.Query(ctx, expression, time.Now())
	if err != nil {
		return nil, err
	}

	// 定义返回的结构体
	jobPods := make(map[string][]string)

	// 检查结果类型是否为向量
	if result.Type() == model.ValVector {
		vector := result.(model.Vector)
		for _, sample := range vector {
			metric := sample.Metric
			job := metric["created_by_name"]
			pod := metric["pod"]

			jobPods[string(job)] = append(jobPods[string(job)], string(pod))
		}

		return jobPods, nil
	}

	return nil, fmt.Errorf("expected vector type result but got %s", result.Type())
}

func (p *PrometheusClient) checkGPUUsed(expression string) (int, error) {
	// 执行查询
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	result, _, err := p.v1api.Query(ctx, expression, time.Now())
	if err != nil {
		return 0, err
	}

	// 检查结果类型是否为向量
	if result.Type() == model.ValVector {
		vector := result.(model.Vector)
		// 返回vector的长度
		return len(vector), nil
	}

	return 0, fmt.Errorf("expected vector type result but got %s", result.Type())
}

// checkIfGPURequested 检查 Pod 是否申请了 GPU
func (p *PrometheusClient) checkIfGPURequested(namespacedName types.NamespacedName) bool {
	// 这里实现检查逻辑，例如通过查询 Pod 的资源请求或 Prometheus 中的相关指标
	// 返回 true 表示 Pod 申请了 GPU，false 表示没有申请
	// 这是一个示例实现，具体逻辑需要根据实际情况调整
	query := fmt.Sprintf("count(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q})", namespacedName.Namespace, namespacedName.Name)
	value, err := p.queryMetric(query)
	return err == nil && value > 0
}
