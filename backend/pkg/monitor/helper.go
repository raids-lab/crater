package monitor

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/prometheus/common/model"
)

func (p *PrometheusClient) queryMetric(expression string) (float32, error) {
	return p.queryMetricAt(expression, time.Now())
}

func (p *PrometheusClient) queryMetricAt(expression string, at time.Time) (float32, error) {
	vector, err := p.queryVectorAt(context.Background(), expression, at)
	if err != nil {
		return 0, err
	}
	for _, sample := range vector {
		if value := finiteFloat32(float64(sample.Value)); value != nil {
			return *value, nil
		}
	}
	return 0, fmt.Errorf("no finite data found for expression: %s", expression)
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func finiteFloat32(value float64) *float32 {
	narrowed := float32(value)
	if !isFinite(float64(narrowed)) {
		return nil
	}
	return &narrowed
}

func (p *PrometheusClient) queryVector(query string) (model.Vector, error) {
	return p.queryVectorAt(context.Background(), query, time.Now())
}

func (p *PrometheusClient) queryVectorAt(ctx context.Context, query string, at time.Time) (model.Vector, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, _, err := p.v1api.Query(ctx, query, at)
	if err != nil {
		return nil, err
	}
	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("expected vector result for %q, got %T", query, result)
	}
	return slices.DeleteFunc(vector, func(sample *model.Sample) bool {
		return !isFinite(float64(sample.Value))
	}), nil
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
		if value := finiteFloat32(float64(sample.Value)); value != nil {
			cpuUsages[instance] = *value
		}
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

func (p *PrometheusClient) getNodeGPUUtil(expression string) ([]NodeGPUUtil, error) {
	vector, err := p.queryVector(expression)
	if err != nil {
		return nil, err
	}
	nodeGPUUtils := make([]NodeGPUUtil, 0, len(vector))
	for _, sample := range vector {
		util := finiteFloat32(float64(sample.Value))
		if util == nil {
			continue
		}
		metric := sample.Metric
		nodeGPUUtils = append(nodeGPUUtils, NodeGPUUtil{
			Hostname: string(metric["Hostname"]), UUID: string(metric["UUID"]),
			Container: string(metric["container"]), Device: string(metric["device"]),
			Endpoint: string(metric["endpoint"]), Gpu: string(metric["gpu"]),
			Instance: string(metric["instance"]), Job: string(metric["job"]),
			ModelName: string(metric["modelName"]), Namespace: string(metric["namespace"]),
			Pod: string(metric["pod"]), Service: string(metric["service"]), Util: *util,
		})
	}
	return nodeGPUUtils, nil
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
