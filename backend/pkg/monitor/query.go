//nolint:lll // Promtheus Query is too long
package monitor

import (
	"fmt"

	"github.com/raids-lab/crater/pkg/logutils"
)

func (p *PrometheusClient) QueryPodProfileMetric(namespace, podname string) (PodUtil, error) {
	podUtil := PodUtil{}
	var err error
	podUtil.GPUUtilAvg, err = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q}[60s])/100", namespace, podname))
	if err != nil {
		return podUtil, err
	}
	podUtil.GPUUtilMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q}[60s])/100", namespace, podname))
	podUtil.GPUUtilStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q}[60s])/100", namespace, podname))
	podUtil.SMActiveAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_SM_ACTIVE{namespace=%q,pod=%q}[60s])", namespace, podname))
	podUtil.SMActiveMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_SM_ACTIVE{namespace=%q,pod=%q}[60s])", namespace, podname))
	podUtil.SMActiveStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_PROF_SM_ACTIVE{namespace=%q,pod=%q}[60s])", namespace, podname))
	podUtil.SMOccupancyAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_SM_OCCUPANCY{namespace=%q,pod=%q}[60s])", namespace, podname))
	podUtil.SMOccupancyMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_SM_OCCUPANCY{namespace=%q,pod=%q}[60s])", namespace, podname))
	podUtil.SMOccupancyStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_PROF_SM_OCCUPANCY{namespace=%q,pod=%q}[60s])", namespace, podname))
	podUtil.DramUtilAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_DRAM_ACTIVE{namespace=%q,pod=%q}[60s])", namespace, podname))
	podUtil.DramUtilMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_DRAM_ACTIVE{namespace=%q,pod=%q}[60s])", namespace, podname))
	podUtil.DramUtilStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_PROF_DRAM_ACTIVE{namespace=%q,pod=%q}[60s])", namespace, podname))
	podUtil.MemCopyUtilAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{namespace=%q,pod=%q}[60s])/100", namespace, podname))
	podUtil.MemCopyUtilMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{namespace=%q,pod=%q}[60s])/100", namespace, podname))
	podUtil.MemCopyUtilStd, _ = p.queryMetric(fmt.Sprintf("stddev_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{namespace=%q,pod=%q}[60s])/100", namespace, podname))
	podUtil.PCIETxAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_PCIE_TX_BYTES{namespace=%q,pod=%q}[60s])/1048576", namespace, podname))
	podUtil.PCIETxMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_PCIE_TX_BYTES{namespace=%q,pod=%q}[60s])/1048576", namespace, podname))
	podUtil.PCIERxAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(DCGM_FI_PROF_PCIE_RX_BYTES{namespace=%q,pod=%q}[60s])/1048576", namespace, podname))
	podUtil.PCIERxMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_PROF_PCIE_RX_BYTES{namespace=%q,pod=%q}[60s])/1048576", namespace, podname))
	podUtil.CPUUsageAvg, _ = p.queryMetric(fmt.Sprintf("avg_over_time(irate(container_cpu_usage_seconds_total{namespace=%q,pod=%q,container!=\"POD\",container!=\"\"}[30s])[2m:])", namespace, podname))
	podUtil.GPUMemMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(DCGM_FI_DEV_FB_USED{namespace=%q,pod=%q}[60s])", namespace, podname))
	podUtil.CPUMemMax, _ = p.queryMetric(fmt.Sprintf("max_over_time(container_memory_usage_bytes{namespace=%q,pod=%q,container!=\"POD\",container!=\"\"}[60s])/1048576", namespace, podname))

	return podUtil, nil
}

func (p *PrometheusClient) QueryNodeCPUUsageRatio() map[string]float32 {
	query := `100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`
	data, err := p.float32MapQuery(query, "instance")
	if err != nil {
		logutils.Log.Errorf("QueryNodeCPUUsageRatio error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) QueryNodeMemoryUsageRatio() map[string]float32 {
	query := "(1 - (avg by (instance) (node_memory_MemAvailable_bytes) / avg by (instance) (node_memory_MemTotal_bytes))) * 100"
	data, err := p.float32MapQuery(query, "instance")
	if err != nil {
		logutils.Log.Errorf("QueryNodeMemoryUsageRatio error: %v", err)
		return nil
	}
	return data
}

// QueryNodeAllocatedMemory returns the allocated memory of each node
func (p *PrometheusClient) QueryNodeAllocatedMemory() map[string]int {
	query := `sum by (node) (
		kube_pod_container_resource_requests{resource="memory"} * 
		on(pod, namespace) group_left() 
		(kube_pod_status_phase{phase="Running"})
	  )`
	data, err := p.intMapQuery(query, "node")
	if err != nil {
		logutils.Log.Errorf("QueryNodeAllocatedMemory error: %v", err)
		return nil
	}
	return data
}

// QueryNodeAllocatedCPU returns the allocated CPU of each node
func (p *PrometheusClient) QueryNodeAllocatedCPU() map[string]float32 {
	query := `sum by (node) (
		kube_pod_container_resource_requests{resource="cpu"} * 
		on(pod, namespace) group_left() 
		(kube_pod_status_phase{phase="Running"})
	  )`
	data, err := p.float32MapQuery(query, "node")
	if err != nil {
		logutils.Log.Errorf("QueryNodeAllocatedCPU error: %v", err)
		return nil
	}
	return data
}

// QueryNodeAllocatedGPU returns the allocated GPU of each node
func (p *PrometheusClient) QueryNodeAllocatedGPU() map[string]int {
	query := `sum by (node) (
		kube_pod_container_resource_requests{resource=~"nvidia_.*"} * 
		on(pod, namespace) group_left() 
		(kube_pod_status_phase{phase="Running"})
	  )`
	data, err := p.intMapQuery(query, "node")
	if err != nil {
		logutils.Log.Errorf("QueryNodeAllocatedGPU error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) QueryPodCPUUsage(podName string) float32 {
	query := fmt.Sprintf("sum(rate(container_cpu_usage_seconds_total{pod=%q}[5m]))", podName)
	data, err := p.float32MapQuery(query, "")
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

func (p *PrometheusClient) QueryPodCPUAllocate(podName, namespace string) int {
	query := fmt.Sprintf("kube_pod_container_resource_requests{pod=%q, namespace=%q, resource=\"cpu\"}", podName, namespace)
	data, err := p.intMapQuery(query, "")
	if err != nil {
		logutils.Log.Errorf("QueryPodCPUAllocate error: %v", err)
		return -1
	}
	var sum int
	for _, v := range data {
		sum += v
	}
	return sum
}

func (p *PrometheusClient) QueryPodMemoryUsage(podName string) int {
	query := fmt.Sprintf("container_memory_usage_bytes{pod=%q}", podName)
	data, err := p.intMapQuery(query, "")
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

func (p *PrometheusClient) QueryPodMemoryAllocate(podName, namespace string) int {
	query := fmt.Sprintf("kube_pod_container_resource_requests{pod=%q, namespace=%q, resource=\"memory\"}", podName, namespace)
	data, err := p.intMapQuery(query, "")
	if err != nil {
		logutils.Log.Errorf("QueryPodCPUAllocate error: %v", err)
		return -1
	}
	var sum int
	for _, v := range data {
		sum += v
	}
	return sum
}

func (p *PrometheusClient) QueryPodGPUAllocate(podName, namespace string) map[string]int {
	query := fmt.Sprintf("kube_pod_container_resource_requests{pod=%q, namespace=%q, resource!=\"cpu\", resource!=\"memory\"}", podName, namespace)
	data, err := p.intMapQuery(query, "resource")
	if err != nil {
		logutils.Log.Errorf("QueryPodCPUAllocate error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) QueryPodGPU() []PodGPUAllocate {
	expression := `kube_pod_container_resource_requests{namespace="crater-jobs", resource="nvidia_com_gpu"}`
	data, err := p.podGPU(expression)
	if err != nil {
		logutils.Log.Errorf("QueryPodGPU error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) QueryNodeGPUUtil() []NodeGPUUtil {
	expression := "DCGM_FI_DEV_GPU_UTIL"
	data, err := p.getNodeGPUUtil(expression)
	if err != nil {
		logutils.Log.Errorf("QueryNodeGPUUtil error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) GetJobPodsList() map[string][]string {
	query := `kube_pod_info{namespace="crater-workspace",created_by_kind="Job"}`
	data, err := p.getJobPods(query)
	if err != nil {
		logutils.Log.Errorf("GetJobPodsList error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) GetLeastUsedGPUJobList(podName, time, util string) int {
	query := fmt.Sprintf("max_over_time(DCGM_FI_DEV_GPU_UTIL{pod=%q}[%vm]) <= %v", podName, time, util)
	data, err := p.checkGPUUsed(query)
	if err != nil {
		logutils.Log.Errorf("GetLeastUsedGPUJobList error: %v", err)
		return 0
	}
	return data
}
