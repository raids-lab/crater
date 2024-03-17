package crclient

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NodeClient struct {
	client.Client
	KubeClient kubernetes.Interface
}

type NodeInfo struct {
	IsReady     bool
	Name        string
	Capacity    corev1.ResourceList
	Allocatable corev1.ResourceList
}

// https://stackoverflow.com/questions/67630551/how-to-use-client-go-to-get-the-node-status
func isNodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// isWorkerNode 检查节点是否为 Worker 节点
func isMasterNode(node corev1.Node) bool {
	// 示例：假设 Worker 节点具有名为 "node-role.kubernetes.io/worker" 的标签
	for key, value := range node.Labels {
		if key == "node-role.kubernetes.io/master" && value == "true" {
			return true
		}
	}
	return false
}

// GetNodes 获取所有 Node 列表 (仅返回是否在线、节点名称、节点资源和已分配资源)
func (nc *NodeClient) ListNodes() ([]NodeInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return nil, err
	}
	var nodeInfoList []NodeInfo
	for _, node := range nodes.Items {
		// 如果节点不是 Worker 节点，则跳过
		if isMasterNode(node) {
			continue
		}
		nodeInfo := NodeInfo{
			IsReady:     isNodeReady(node),
			Name:        node.Name,
			Capacity:    node.Status.Capacity,
			Allocatable: node.Status.Allocatable,
		}
		nodeInfoList = append(nodeInfoList, nodeInfo)
	}

	return nodeInfoList, nil
}
