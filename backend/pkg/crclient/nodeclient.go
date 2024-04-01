package crclient

import (
	"context"
	"fmt"

	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/server/payload"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NodeClient struct {
	client.Client
	KubeClient  *kubernetes.Clientset
	PromeClient *monitor.PrometheusClient
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
	mem_ := mem / 1024 / 1024
	result := fmt.Sprintf("%dMi", mem_)
	return result
}

// GetNodes 获取所有 Node 列表
func (nc *NodeClient) ListNodes() ([]payload.ClusterNodeInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return nil, err
	}

	nodeInfos := make([]payload.ClusterNodeInfo, len(nodes.Items))
	CPUMap := nc.PromeClient.QueryNodeCPUUsageRatio()
	MemMap := nc.PromeClient.QueryNodeAllocatedMemory()

	// Loop through each node and print allocated resources
	for i := range nodes.Items {
		node := &nodes.Items[i]
		allocatedInfo := payload.AllocatedInfo{
			CPU: calculateCPULoad(CPUMap[node.Name], int(node.Status.Capacity.Cpu().MilliValue())),
			Mem: FomatMemoryLoad(MemMap[node.Name]),
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
		fmt.Print(podList.Items)

		// 遍历当前节点的Pods，收集所需信息
		for i := range podList.Items {
			pod := &podList.Items[i]
			podInfo := payload.Pod{
				Name:       pod.Name,
				IP:         pod.Status.PodIP,
				CreateTime: pod.CreationTimestamp.String(),
				Status:     string(pod.Status.Phase),
				CPU:        nc.PromeClient.QueryPodCPURatio(pod.Name),
				Mem:        FomatMemoryLoad(nc.PromeClient.QueryPodMemory(pod.Name)),
			}

			nodeInfo.Pods = append(nodeInfo.Pods, podInfo)
		}

		return nodeInfo, nil
	}

	return payload.ClusterNodePodInfo{}, fmt.Errorf("node %s not found", name)
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
