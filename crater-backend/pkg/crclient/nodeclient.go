package crclient

import (
	"context"

	"github.com/raids-lab/crater/pkg/server/payload"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NodeClient struct {
	client.Client
	KubeClient *kubernetes.Clientset
}

func calculateAllocatedResources(node *corev1.Node, clientset *kubernetes.Clientset) corev1.ResourceList {
	allocatedResources := make(corev1.ResourceList)
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Spec.NodeName == node.Name {
			for j := range pod.Spec.Containers {
				container := &pod.Spec.Containers[j]
				for resource, quantity := range container.Resources.Requests {
					if allocatedQuantity, found := allocatedResources[resource]; found {
						allocatedQuantity.Add(quantity)
						allocatedResources[resource] = allocatedQuantity
					} else {
						allocatedResources[resource] = quantity.DeepCopy()
					}
				}
			}
		}
	}

	return allocatedResources
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

// GetNodes 获取所有 Node 列表
func (nc *NodeClient) ListNodes() ([]payload.ClusterNodeInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return nil, err
	}

	nodeInfos := make([]payload.ClusterNodeInfo, len(nodes.Items))

	// Loop through each node and print allocated resources
	for i := range nodes.Items {
		node := &nodes.Items[i]
		allocatedResources := calculateAllocatedResources(node, nc.KubeClient)
		nodeInfos[i] = payload.ClusterNodeInfo{
			Name:     node.Name,
			Role:     getNodeRole(node),
			Labels:   node.Labels,
			IsReady:  isNodeReady(node),
			Capacity: node.Status.Capacity,
			Alocated: allocatedResources,
		}
	}

	return nodeInfos, nil
}
