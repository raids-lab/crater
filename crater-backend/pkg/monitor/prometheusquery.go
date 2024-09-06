package monitor

import (
	"fmt"

	"github.com/raids-lab/crater/pkg/logutils"
)

func (p *PrometheusClient) QueryNodeCPUUsageRatio() map[string]float32 {
	query := `100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`
	data, err := p.Float32MapQuery(query, "instance")
	if err != nil {
		logutils.Log.Errorf("QueryNodeCPUUsageRatio error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) QueryNodeMemoryUsageRatio() map[string]float32 {
	query := "(1 - (avg by (instance) (node_memory_MemAvailable_bytes) / avg by (instance) (node_memory_MemTotal_bytes))) * 100"
	data, err := p.Float32MapQuery(query, "instance")
	if err != nil {
		logutils.Log.Errorf("QueryNodeMemoryUsageRatio error: %v", err)
		return nil
	}
	return data
}

// QueryNodeAllocatedMemory returns the allocated memory of each node
func (p *PrometheusClient) QueryNodeAllocatedMemory() map[string]int {
	query := "sum by (node) (kube_pod_container_resource_requests{resource=\"memory\"})"
	data, err := p.IntMapQuery(query, "node")
	if err != nil {
		logutils.Log.Errorf("QueryNodeAllocatedMemory error: %v", err)
		return nil
	}
	return data
}

// QueryNodeAllocatedCPU returns the allocated CPU of each node
func (p *PrometheusClient) QueryNodeAllocatedCPU() map[string]float32 {
	query := "sum by (node) (kube_pod_container_resource_requests{resource=\"cpu\"})"
	data, err := p.Float32MapQuery(query, "node")
	if err != nil {
		logutils.Log.Errorf("QueryNodeAllocatedCPU error: %v", err)
		return nil
	}
	return data
}

// QueryNodeAllocatedGPU returns the allocated GPU of each node
func (p *PrometheusClient) QueryNodeAllocatedGPU() map[string]int {
	query := "sum by (node) (kube_pod_container_resource_requests{resource=~\"nvidia_.*\"})"
	data, err := p.IntMapQuery(query, "node")
	if err != nil {
		logutils.Log.Errorf("QueryNodeAllocatedGPU error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) QueryPodCPURatio(podName string) float32 {
	query := fmt.Sprintf("sum(rate(container_cpu_usage_seconds_total{pod=%q}[5m]))", podName)
	data, err := p.Float32MapQuery(query, "")
	if err != nil {
		logutils.Log.Errorf("QueryPodCPURatio error: %v", err)
		return 0.0
	}
	var sum float32
	for _, v := range data {
		sum += v
	}
	return sum
}

func (p *PrometheusClient) QueryPodMemory(podName string) int {
	query := fmt.Sprintf("container_memory_usage_bytes{pod=%q}", podName)
	data, err := p.IntMapQuery(query, "")
	if err != nil {
		logutils.Log.Errorf("QueryPodMemory error: %v", err)
		return 0
	}
	var sum int
	for _, v := range data {
		sum += v
	}
	return sum
}

func (p *PrometheusClient) QueryPodGPU() []PodGPUAllocate {
	expression := "kube_pod_container_resource_requests{namespace=\"crater-jobs\", resource=\"nvidia_com_gpu\"}"
	data, err := p.PodGPU(expression)
	if err != nil {
		logutils.Log.Errorf("QueryPodGPU error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) QueryNodeGPUUtil() []NodeGPUUtil {
	expression := "DCGM_FI_DEV_GPU_UTIL"
	data, err := p.GetNodeGPUUtil(expression)
	if err != nil {
		logutils.Log.Errorf("QueryNodeGPUUtil error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) GetJobPodsList() map[string][]string {
	query := "kube_pod_info{namespace=\"crater-workspace\",created_by_kind=\"Job\"}"
	data, err := p.GetJobPods(query)
	if err != nil {
		logutils.Log.Errorf("GetJobPodsList error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) GetLeastUsedGPUJobList(podName, time, util string) int {
	query := fmt.Sprintf("max_over_time(DCGM_FI_DEV_GPU_UTIL{pod=%q}[%vm]) <= %v", podName, time, util)
	data, err := p.CheckGPUUsed(query)
	if err != nil {
		logutils.Log.Errorf("GetLeastUsedGPUJobList error: %v", err)
		return 0
	}
	return data
}
