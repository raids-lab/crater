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
	KubeClient       kubernetes.Interface
	PrometheusClient monitor.PrometheusInterface
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
		switch key {
		case "node-role.kubernetes.io/master":
		case "node-role.kubernetes.io/control-plane":
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

func fomatMemoryLoad(mem int) string {
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

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return nil, err
	}

	nodeInfos := make([]payload.ClusterNodeInfo, len(nodes.Items))
	CPUMap := nc.PrometheusClient.QueryNodeAllocatedCPU()
	MemMap := nc.PrometheusClient.QueryNodeAllocatedMemory()
	GPUMap := nc.PrometheusClient.QueryNodeAllocatedGPU()

	// Loop through each node and print allocated resources
	for i := range nodes.Items {
		node := &nodes.Items[i]
		allocatedInfo := payload.AllocatedInfo{
			CPU: calculateCPULoad(CPUMap[node.Name], int(node.Status.Capacity.Cpu().MilliValue())),
			Mem: fomatMemoryLoad(MemMap[node.Name]),
			GPU: fmt.Sprintf("%d", GPUMap[node.Name]),
		}
		gpuCount := nc.getNodeGPUCount(node.Name)
		capacity_ := node.Status.Capacity
		if gpuCount > 0 {
			// 将int类型的gpu_count转换为resource.Quantity类型
			capacity_["nvidia.com/gpu"] = *resource.NewQuantity(int64(gpuCount), resource.DecimalSI)
		}
		// TODO: remove
		if node.Name == "zjlab-5" || node.Name == "zjlab-6" {
			capacity_["nvidia.com/gpu"] = *resource.NewQuantity(int64(2), resource.DecimalSI)
		}
		// 获取节点类型
		nodeType := node.Labels["crater.raids.io/nodetype"]
		nodeInfos[i] = payload.ClusterNodeInfo{
			Type:      nodeType, // 添加节点类型
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

// GetNode 获取指定 Node 的信息
func (nc *NodeClient) GetNode(ctx context.Context, name string) (payload.ClusterNodeDetail, error) {
	node := &corev1.Node{}

	err := nc.Get(ctx, client.ObjectKey{
		Namespace: "",
		Name:      name,
	}, node)
	if err != nil {
		return payload.ClusterNodeDetail{}, err
	}

	nodeInfo := payload.ClusterNodeDetail{
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
	}
	return nodeInfo, nil
}

func (nc *NodeClient) GetPodsForNode(ctx context.Context, name string) (*payload.ClusterNodePodInfo, error) {
	node := &corev1.Node{}

	err := nc.Get(ctx, client.ObjectKey{
		Namespace: "",
		Name:      name,
	}, node)
	if err != nil {
		return nil, err
	}

	// 初始化节点信息
	nodeInfo := payload.ClusterNodePodInfo{
		Pods: []payload.Pod{},
	}

	// 使用KubeClient获取当前节点上的所有Pods
	podList, err := nc.KubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + node.Name,
	})
	if err != nil {
		return nil, err
	}

	// 遍历当前节点的Pods，收集所需信息
	for i := range podList.Items {
		pod := &podList.Items[i]
		ownerKind := ""
		for _, owner := range pod.OwnerReferences {
			ownerKind = owner.Kind
		}
		podInfo := payload.Pod{
			Name:       pod.Name,
			Namespace:  pod.Namespace,
			IP:         pod.Status.PodIP,
			CreateTime: pod.CreationTimestamp.String(),
			Status:     string(pod.Status.Phase),
			CPU:        nc.PrometheusClient.QueryPodCPUUsage(pod.Name),
			Mem:        fomatMemoryLoad(nc.PrometheusClient.QueryPodMemoryUsage(pod.Name)),
			OwnerKind:  ownerKind,
		}

		nodeInfo.Pods = append(nodeInfo.Pods, podInfo)
	}

	return &nodeInfo, nil
}

func (nc *NodeClient) GetNodeGPUInfo(name string) (payload.GPUInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return payload.GPUInfo{}, err
	}

	// 初始化返回值
	gpuInfo := payload.GPUInfo{
		Name:       name,
		HaveGPU:    false,
		GPUCount:   0,
		GPUUtil:    make(map[string]float32),
		RelateJobs: make(map[string][]string),
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
	jobPodsList := nc.PrometheusClient.GetJobPodsList()
	gpuUtilMap := nc.PrometheusClient.QueryNodeGPUUtil()
	for i := 0; i < len(gpuUtilMap); i++ {
		gpuUtil := &gpuUtilMap[i]
		if gpuUtil.Hostname == name {
			gpuInfo.GPUUtil[gpuUtil.Gpu] = gpuUtil.Util
			// 如果gpuUtil.pod在jobPodsList的value中，则将jobPodsList中的job加入gpuInfo.RelateJobs[gpuUtil.Gpu]
			curPod := gpuUtil.Pod
			for job, pods := range jobPodsList {
				// 确认curPod是否在pods中
				for _, pod := range pods {
					if curPod == pod {
						// 将job加入gpuInfo.RelateJobs[gpuUtil.Gpu]
						gpuInfo.RelateJobs[gpuUtil.Gpu] = append(gpuInfo.RelateJobs[gpuUtil.Gpu], job)
						break
					}
				}
			}
		}
	}
	return gpuInfo, nil
}

func (nc *NodeClient) GetLeastUsedGPUJobs(time, util int) []string {
	var gpuJobPodsList map[string]string
	gpuUtilMap := nc.PrometheusClient.QueryNodeGPUUtil()
	jobPodsList := nc.PrometheusClient.GetJobPodsList()
	gpuJobPodsList = make(map[string]string)
	for i := 0; i < len(gpuUtilMap); i++ {
		gpuUtil := &gpuUtilMap[i]
		curPod := gpuUtil.Pod
		for job, pods := range jobPodsList {
			for _, pod := range pods {
				if curPod == pod {
					gpuJobPodsList[curPod] = job
					break
				}
			}
		}
	}

	leastUsedJobs := make([]string, 0)
	for pod, job := range gpuJobPodsList {
		// 将time和util转换为string类型
		_time := fmt.Sprintf("%d", time)
		_util := fmt.Sprintf("%d", util)
		if nc.PrometheusClient.GetLeastUsedGPUJobList(pod, _time, _util) > 0 {
			leastUsedJobs = append(leastUsedJobs, job)
		}
	}
	return leastUsedJobs
}
