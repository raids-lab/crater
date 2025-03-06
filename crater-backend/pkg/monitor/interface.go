package monitor

import (
	"time"

	"k8s.io/apimachinery/pkg/types"
)

// PrometheusClient is a client for Prometheus
type PrometheusInterface interface {
	///////////// Node Related //////////////

	// QueryNodeCPUUsageRatio queries the CPU usage ratio of a node
	QueryNodeCPUUsageRatio() map[string]float32

	// QueryNodeMemoryUsageRatio queries the memory usage ratio of a node
	QueryNodeMemoryUsageRatio() map[string]float32

	// QueryNodeGPUUtil queries the GPU utilization of a node
	QueryNodeGPUUtil() []NodeGPUUtil

	// QueryNodeAllocatedCPU queries the allocated CPU of each node
	QueryNodeAllocatedCPU() map[string]float32

	// QueryNodeAllocatedMemory queries the allocated memory of each node
	QueryNodeAllocatedMemory() map[string]int

	// QueryNodeRunningCount queries the running pod of each node
	QueryNodeRunningPodCount() map[string]int

	// QueryNodeAllocatedGPU queries the allocated GPU of each node
	QueryNodeAllocatedGPU() map[string]int

	///////////// Pod Related //////////////

	// QueryPodCPURatio queries the CPU ratio of a pod
	QueryPodCPUUsage(podName string) float32

	// QueryPodCPUAllocate queries the CPU allocate of a pod
	QueryPodCPUAllocate(podName string, namespace string) int

	// QueryPodMemoryRatio queries the memory ratio of a pod
	QueryPodMemoryUsage(podName string) int

	// QueryPodMemoryAllocate queries the memory allocate of a pod
	QueryPodMemoryAllocate(podName string, namespace string) int

	// QueryPodGPUAllocate queries the GPU allocate of a pod
	QueryPodGPUAllocate(podName string, namespace string) map[string]int

	// QueryPodProfileMetric queries the profile metric of a pod (used for AiJob)
	QueryPodProfileMetric(namespace, podname string) (PodUtil, error)

	// QueryProfileData queries the profile data of a pod
	QueryProfileData(namespacedName types.NamespacedName, from time.Time) *ProfileData

	GetPodOwner(podName string) string

	///////////// Job Related //////////////

	// GetJobPodsList returns the job pods list
	GetJobPodsList() map[string][]string

	// GetLeastUsedGPUJobList returns the least used GPU job list
	GetLeastUsedGPUJobList(podName, _time, util string) int
}
