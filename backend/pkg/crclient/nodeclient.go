package crclient

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/server/payload"
	"github.com/raids-lab/crater/pkg/utils"
)

type NodeClient struct {
	client.Client
	KubeClient       kubernetes.Interface
	PrometheusClient monitor.PrometheusInterface
}

// https://stackoverflow.com/questions/67630551/how-to-use-client-go-to-get-the-node-status
/*func isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}*/
func isNodeReady(node *corev1.Node) string {
	if node.Spec.Unschedulable {
		return "Unschedulable"
	}
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return "true"
		}
	}
	return "false"
}

// taintsToString将节点的taint转化成字符串
func taintToString(taint corev1.Taint) string {
	return fmt.Sprintf("%s=%s:%s", taint.Key, taint.Value, taint.Effect)
}

func taintsToString(taints []corev1.Taint) string {
	var taintStrings []string
	for _, taint := range taints {
		taintStrings = append(taintStrings, taintToString(taint))
	}
	return strings.Join(taintStrings, ",")
}

// getNodeRole 获取节点角色
func getNodeRole(node *corev1.Node) string {
	for key := range node.Labels {
		switch key {
		case "node-role.kubernetes.io/master", "node-role.kubernetes.io/control-plane":
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
		if node.Name == "zjlab-gpu1" || node.Name == "zjlab-gpu2" {
			capacity_["nvidia.com/gpu"] = *resource.NewQuantity(int64(2), resource.DecimalSI)
		}
		// 获取节点类型
		nodeType := node.Labels["crater.raids.io/nodetype"]
		nodeInfos[i] = payload.ClusterNodeInfo{
			Type:      nodeType, // 添加节点类型
			Name:      node.Name,
			Taint:     taintsToString(node.Spec.Taints),
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
		Taint:                   taintsToString(node.Spec.Taints),
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
func (nc *NodeClient) UpdateNodeunschedule(ctx context.Context, name string) error {
	node, err := nc.KubeClient.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	node.Spec.Unschedulable = !node.Spec.Unschedulable
	_, err = nc.KubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}
func stringToTaint(taintString string) (corev1.Taint, error) {
	// 拆分字符串
	parts := strings.Split(taintString, "=")
	if len(parts) != 2 {
		return corev1.Taint{}, fmt.Errorf("invalid taint format: %s", taintString)
	}

	key := parts[0]
	valueEffect := strings.Split(parts[1], ":")
	if len(valueEffect) != 2 {
		return corev1.Taint{}, fmt.Errorf("invalid taint format: %s", taintString)
	}

	value := valueEffect[0]
	effect := valueEffect[1]

	// 创建 Taint 结构体
	taint := corev1.Taint{
		Key:    key,
		Value:  value,
		Effect: corev1.TaintEffect(effect),
	}

	return taint, nil
}
func (nc *NodeClient) AddNodetaint(ctx context.Context, name, taint string) error {
	node, err := nc.KubeClient.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	var Taint corev1.Taint
	Taint, err1 := stringToTaint(taint)
	if err != nil {
		return err
	}
	if err1 != nil {
		return err
	}
	// 检查污点是否已经存在
	for _, existingTaint := range node.Spec.Taints {
		if existingTaint.MatchTaint(&Taint) {
			return fmt.Errorf("taint %v already exists on node %s", taint, name)
		}
	}
	// 添加新的污点
	node.Spec.Taints = append(node.Spec.Taints, Taint)

	// 更新节点
	_, err = nc.KubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return err
}
func (nc *NodeClient) DeleteNodetaint(ctx context.Context, name, taint string) error {
	node, err := nc.KubeClient.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	var Taint corev1.Taint
	Taint, err1 := stringToTaint(taint)
	if err != nil {
		return err
	}
	if err1 != nil {
		return err
	}
	// 从现有污点列表中删除指定的污点
	newTaints := []corev1.Taint{}
	for _, existingTaint := range node.Spec.Taints {
		if !existingTaint.MatchTaint(&Taint) {
			newTaints = append(newTaints, existingTaint)
		}
	}
	// 如果污点列表没有变化，则不需要更新
	if len(newTaints) == len(node.Spec.Taints) {
		return fmt.Errorf("taint %v not found on node %s", taint, name)
	}
	// 更新污点列表
	node.Spec.Taints = newTaints
	// 更新节点
	_, err = nc.KubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return err
}
func (nc *NodeClient) GetPodsForNode(ctx context.Context, nodeName string) ([]payload.Pod, error) {
	// Get Pods for the node, which is a costly operation
	// TODO(zhangry): Add a cache for this? Or query from Prometheus?
	podList, err := nc.KubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return nil, err
	}

	// Initialize the return value
	pods := make([]payload.Pod, len(podList.Items))
	for i := range podList.Items {
		pod := &podList.Items[i]
		pods[i] = payload.Pod{
			Name:           pod.Name,
			Namespace:      pod.Namespace,
			IP:             pod.Status.PodIP,
			CreateTime:     pod.CreationTimestamp,
			Status:         pod.Status.Phase,
			OwnerReference: pod.OwnerReferences,
			Resources:      utils.CalculateRequsetsByContainers(pod.Spec.Containers),
		}
	}

	return pods, nil
}

func (nc *NodeClient) GetNodeGPUInfo(name string) (payload.GPUInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return payload.GPUInfo{}, err
	}

	// 初始化返回值
	gpuInfo := payload.GPUInfo{
		Name:        name,
		HaveGPU:     false,
		GPUCount:    0,
		GPUUtil:     make(map[string]float32),
		RelateJobs:  make(map[string][]string),
		GPUMemory:   "",
		GPUArch:     "",
		GPUDriver:   "",
		CudaVersion: "",
		GPUProduct:  "",
	}

	// 首先查询当前节点是否有GPU
	for i := range nodes.Items {
		node := &nodes.Items[i]
		if node.Name != name {
			continue
		}
		gpuCountValue, ok := node.Labels["nvidia.com/gpu.count"]
		if ok {
			gpuCount := 0
			_, err := fmt.Sscanf(gpuCountValue, "%d", &gpuCount)
			if err != nil {
				return payload.GPUInfo{}, err
			}
			gpuInfo.GPUCount = gpuCount
			gpuInfo.HaveGPU = true
			gpuInfo.GPUMemory = node.Labels["nvidia.com/gpu.memory"]
			gpuInfo.GPUArch = node.Labels["nvidia.com/gpu.family"]
			gpuInfo.GPUDriver = node.Labels["nvidia.com/cuda.driver-version.full"]
			gpuInfo.CudaVersion = node.Labels["nvidia.com/cuda.runtime-version.full"]
			gpuInfo.GPUProduct = node.Labels["nvidia.com/gpu.product"]
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
