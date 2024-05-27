package crclient

import (
	"context"
	"fmt"

	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/server/payload"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NodeClient struct {
	client.Client
	KubeClient       *kubernetes.Clientset
	PrometheusClient *monitor.PrometheusClient
}

// https://stackoverflow.com/questions/67630551/how-to-use-client-go-to-get-the-node-status
func isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// getNodeRole 获取节点角色
func getNodeRole(node *corev1.Node) string {
	for key := range node.Labels {
		if key == "node-role.kubernetes.io/master" {
			return "master"
		}
	}
	return "worker"
}

func calculateCPULoad(ratio float32, cpu int) string {
	// 计算当前CPU使用量
	loadScaleFactor := 100
	load := int(ratio * float32(cpu) / float32(loadScaleFactor))
	// 将计算结果转换为字符串，并在末尾加上'm'
	result := fmt.Sprintf("%dm", load)
	return result
}

func FomatMemoryLoad(mem int) string {
	trans := 1024
	mem_ := mem / trans
	if mem_ < trans {
		result := fmt.Sprintf("%dKi", mem_)
		return result
	} else {
		mem_ /= trans
		result := fmt.Sprintf("%dMi", mem_)
		return result
	}
}

func getNodeGPUCount(podGPUAllocate []monitor.PodGPUAllocate, nodeName string) int {
	count := 0
	for _, podGPU := range podGPUAllocate {
		if podGPU.Node == nodeName {
			count += podGPU.GPUCount
		}
	}
	return count
}

// 节点中GPU的数量
func (nc *NodeClient) getNodeGPUCount(nodeName string) int {
	gpuUtilMap := nc.PrometheusClient.QueryNodeGPUUtil()
	count := 0
	for i := 0; i < len(gpuUtilMap); i++ {
		gpuUtil := &gpuUtilMap[i]
		if gpuUtil.Hostname == nodeName {
			count++
		}
	}
	return count
}

// GetNodes 获取所有 Node 列表
func (nc *NodeClient) ListNodes() ([]payload.ClusterNodeInfo, error) {
	var nodes corev1.NodeList
	PodGPUAllocate := nc.PrometheusClient.QueryPodGPU()

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return nil, err
	}

	nodeInfos := make([]payload.ClusterNodeInfo, len(nodes.Items))
	CPUMap := nc.PrometheusClient.QueryNodeCPUUsageRatio()
	MemMap := nc.PrometheusClient.QueryNodeAllocatedMemory()

	// Loop through each node and print allocated resources
	for i := range nodes.Items {
		node := &nodes.Items[i]
		allocatedInfo := payload.AllocatedInfo{
			CPU: calculateCPULoad(CPUMap[node.Name], int(node.Status.Capacity.Cpu().MilliValue())),
			Mem: FomatMemoryLoad(MemMap[node.Name]),
			GPU: fmt.Sprintf("%d", getNodeGPUCount(PodGPUAllocate, node.Name)),
		}
		gpuCount := nc.getNodeGPUCount(node.Name)
		capacity_ := node.Status.Capacity
		if gpuCount > 0 {
			// 将int类型的gpu_count转换为resource.Quantity类型
			capacity_["nvidia.com/gpu"] = *resource.NewQuantity(int64(gpuCount), resource.DecimalSI)
		}
		nodeInfos[i] = payload.ClusterNodeInfo{
			Name:      node.Name,
			Role:      getNodeRole(node),
			Labels:    node.Labels,
			IsReady:   isNodeReady(node),
			Capacity:  node.Status.Capacity,
			Allocated: allocatedInfo,
		}
	}

	return nodeInfos, nil
}

func (nc *NodeClient) ListNodesPod(name string) (payload.ClusterNodePodInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return payload.ClusterNodePodInfo{}, err
	}

	for i := range nodes.Items {
		node := &nodes.Items[i]
		if node.Name != name {
			continue
		}
		// 初始化节点信息
		nodeInfo := payload.ClusterNodePodInfo{
			Name:                    node.Name,
			Role:                    getNodeRole(node),
			IsReady:                 isNodeReady(node),
			Time:                    node.CreationTimestamp.String(),
			Address:                 node.Status.Addresses[0].Address,
			Os:                      node.Status.NodeInfo.OperatingSystem,
			OsVersion:               node.Status.NodeInfo.OSImage,
			Arch:                    node.Status.NodeInfo.Architecture,
			KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
			ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
			Pods:                    []payload.Pod{},
		}

		// 使用KubeClient获取当前节点上的所有Pods
		podList, err := nc.KubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
			FieldSelector: "spec.nodeName=" + node.Name,
		})
		if err != nil {
			return payload.ClusterNodePodInfo{}, err
		}

		// 遍历当前节点的Pods，收集所需信息
		for i := range podList.Items {
			pod := &podList.Items[i]
			podInfo := payload.Pod{
				Name:       pod.Name,
				IP:         pod.Status.PodIP,
				CreateTime: pod.CreationTimestamp.String(),
				Status:     string(pod.Status.Phase),
				CPU:        nc.PrometheusClient.QueryPodCPURatio(pod.Name),
				Mem:        FomatMemoryLoad(nc.PrometheusClient.QueryPodMemory(pod.Name)),
			}

			nodeInfo.Pods = append(nodeInfo.Pods, podInfo)
		}

		return nodeInfo, nil
	}

	return payload.ClusterNodePodInfo{}, fmt.Errorf("node %s not found", name)
}

func (nc *NodeClient) GetNodeGPUInfo(name string) (payload.GPUInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return payload.GPUInfo{}, err
	}

	// 初始化返回值
	gpuInfo := payload.GPUInfo{
		Name:     name,
		HaveGPU:  false,
		GPUCount: 0,
		GPUUtil:  make(map[string]float32),
	}

	// 首先查询当前节点是否有GPU
	for i := range nodes.Items {
		node := &nodes.Items[i]
		if node.Name != name {
			continue
		}
		// if _, ok := node.Status.Capacity["nvidia.com/gpu"]; ok {
		if nc.getNodeGPUCount(name) > 0 {
			gpuInfo.HaveGPU = true
			gpuInfo.GPUCount = nc.getNodeGPUCount(name)
			GPUCheckTag := 10
			if gpuInfo.GPUCount >= GPUCheckTag {
				gpuInfo.GPUCount /= GPUCheckTag
			}
			for i := 0; i < gpuInfo.GPUCount; i++ {
				gpuInfo.GPUUtil[fmt.Sprintf("%d", i)] = 0
			}
			break
		}
	}

	// 使用PrometheusClient查询当前节点上的GPU使用率
	gpuUtilMap := nc.PrometheusClient.QueryNodeGPUUtil()
	for i := 0; i < len(gpuUtilMap); i++ {
		gpuUtil := &gpuUtilMap[i]
		if gpuUtil.Hostname == name {
			gpuInfo.GPUUtil[gpuUtil.Gpu] = gpuUtil.Util
		}
	}

	return gpuInfo, nil
}

func (nc *NodeClient) ListNodesTest() ([]payload.ClusterNodeInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return nil, err
	}

	nodeInfos := make([]payload.ClusterNodeInfo, len(nodes.Items))
	for i := range nodes.Items {
		node := &nodes.Items[i]
		nodeInfos[i] = payload.ClusterNodeInfo{
			Name:     node.Name,
			Role:     getNodeRole(node),
			Labels:   node.Labels,
			IsReady:  isNodeReady(node),
			Capacity: node.Status.Capacity,
		}
	}
	return nodeInfos, nil
}
