package crclient

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/indexer"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/utils"
)

const (
	VCJOBAPIVERSION = "batch.volcano.sh/v1alpha1"
	VCJOBKIND       = "Job"
)

// formatOperatorInfo 格式化操作员信息为 "昵称(@用户名)" 形式
func formatOperatorInfo(ctx context.Context, username string) string {
	if username == "" {
		return ""
	}

	// 查询用户昵称 (注意: username来自token,应该查询User表而不是Account表)
	q := query.User
	user, err := q.WithContext(ctx).Where(q.Name.Eq(username)).First()
	if err != nil {
		// 如果查询失败，返回用户名
		return username
	}

	// 返回 "昵称(@用户名)" 格式
	nickname := user.Nickname
	if nickname == "" {
		nickname = username
	}

	return fmt.Sprintf("%s(@%s)", nickname, username)
}

type NodeRole string

const (
	NodeRoleControlPlane NodeRole = "control-plane"
	NodeRoleWorker       NodeRole = "worker"
	NodeRoleVirtual      NodeRole = "virtual"
)

const (
	NodeStatusUnschedulable corev1.NodeConditionType = "Unschedulable"
	NodeStatusOccupied      corev1.NodeConditionType = "Occupied"
	NodeStatusUnknown       corev1.NodeConditionType = "Unknown"
)

type NodeBriefInfo struct {
	Name          string                   `json:"name"`
	Role          NodeRole                 `json:"role"`
	Arch          string                   `json:"arch"`
	Status        corev1.NodeConditionType `json:"status"`
	Vendor        string                   `json:"vendor"`
	Taints        []corev1.Taint           `json:"taints"`
	Capacity      corev1.ResourceList      `json:"capacity"`
	Allocatable   corev1.ResourceList      `json:"allocatable"`
	Used          corev1.ResourceList      `json:"used"`
	Workloads     int                      `json:"workloads"`
	Annotations   map[string]string        `json:"annotations"`
	KernelVersion string                   `json:"kernelVersion"`
	GPUDriver     string                   `json:"gpuDriver"`
}

type Pod struct {
	Name            string                  `json:"name"`
	Namespace       string                  `json:"namespace"`
	OwnerReference  []metav1.OwnerReference `json:"ownerReference"`
	IP              string                  `json:"ip"`
	CreateTime      metav1.Time             `json:"createTime"`
	Status          corev1.PodPhase         `json:"status"`
	Resources       corev1.ResourceList     `json:"resources"`
	Locked          bool                    `json:"locked"`
	PermanentLocked bool                    `json:"permanentLocked"`
	LockedTimestamp metav1.Time             `json:"lockedTimestamp"`
}

type ClusterNodeDetail struct {
	Name                    string                   `json:"name"`
	Role                    string                   `json:"role"`
	Status                  corev1.NodeConditionType `json:"status"`
	Taint                   string                   `json:"taint"`
	Time                    string                   `json:"time"`
	Address                 string                   `json:"address"`
	Os                      string                   `json:"os"`
	OsVersion               string                   `json:"osVersion"`
	Arch                    string                   `json:"arch"`
	KubeletVersion          string                   `json:"kubeletVersion"`
	ContainerRuntimeVersion string                   `json:"containerRuntimeVersion"`
	KernelVersion           string                   `json:"kernelVersion"`
	Capacity                corev1.ResourceList      `json:"capacity"`
	Allocatable             corev1.ResourceList      `json:"allocatable"`
	Used                    corev1.ResourceList      `json:"used"`
	GPUMemory               string                   `json:"gpuMemory"`
	GPUCount                int                      `json:"gpuCount"`
	GPUArch                 string                   `json:"gpuArch"`
	GPUDriver               string                   `json:"gpuDriver"`
}

type GPUInfo struct {
	Name        string              `json:"name"`
	HaveGPU     bool                `json:"haveGPU"`
	GPUCount    int                 `json:"gpuCount"`
	GPUUtil     map[string]float32  `json:"gpuUtil"`
	RelateJobs  map[string][]string `json:"relateJobs"`
	GPUMemory   string              `json:"gpuMemory"`
	GPUArch     string              `json:"gpuArch"`
	GPUDriver   string              `json:"gpuDriver"`
	CudaVersion string              `json:"cudaVersion"`
	GPUProduct  string              `json:"gpuProduct"`
}

type NodeMarkInfo struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Taints      []corev1.Taint    `json:"taints"`
}
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
const (
	StatusFalse = "false"
	StatusTrue  = "true"
)

func getNodeStatus(node *corev1.Node) corev1.NodeConditionType {
	if isNodeOccupied(node) {
		return NodeStatusOccupied
	} else if node.Spec.Unschedulable {
		return NodeStatusUnschedulable
	}

	return getNodeCondition(node) // 节点正常时返回 NodeReady
}

func isNodeOccupied(node *corev1.Node) bool {
	for _, taint := range node.Spec.Taints {
		taintStr := taint.ToString()
		if strings.Contains(taintStr, "crater.raids.io/account=") && strings.HasSuffix(taintStr, ":NoSchedule") {
			return true
		}
	}
	return false
}

func getNodeCondition(node *corev1.Node) corev1.NodeConditionType {
	for _, condition := range node.Status.Conditions {
		if condition.Status == corev1.ConditionTrue {
			switch condition.Type {
			case corev1.NodeReady:
				return corev1.NodeReady
			case corev1.NodeDiskPressure:
				return corev1.NodeDiskPressure
			case corev1.NodeMemoryPressure:
				return corev1.NodeMemoryPressure
			case corev1.NodePIDPressure:
				return corev1.NodePIDPressure
			case corev1.NodeNetworkUnavailable:
				return corev1.NodeNetworkUnavailable
			default:
				// 其他条件不影响就绪状态
				continue
			}
		}
	}

	return NodeStatusUnknown // 如果没有任何条件为 True，则视为就绪
}

func taintsToString(taints []corev1.Taint) string {
	var taintStrings []string
	for _, taint := range taints {
		taintStrings = append(taintStrings, taint.ToString())
	}
	return strings.Join(taintStrings, ",")
}

// getNodeRole 获取节点角色
func getNodeRole(node *corev1.Node) NodeRole {
	for key := range node.Labels {
		switch key {
		case "node-role.kubernetes.io/master", "node-role.kubernetes.io/control-plane":
			return NodeRoleControlPlane
		}
	}
	// 检查是否为虚拟节点
	if nodeType, exists := node.Labels["crater.raids.io/nodetype"]; exists && nodeType == "virtual" {
		return NodeRoleVirtual
	}
	return NodeRoleWorker
}

// ListNodes 获取所有 Node 列表
func (nc *NodeClient) ListNodes(ctx context.Context) ([]NodeBriefInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(ctx, &nodes)
	if err != nil {
		return nil, err
	}

	nodeInfos := make([]NodeBriefInfo, len(nodes.Items))

	// Loop through each node and calculate resources from pods
	for i := range nodes.Items {
		node := &nodes.Items[i]

		// 获取节点上的所有 Pods（通过索引）
		podList := &corev1.PodList{}
		if err := nc.List(ctx, podList, indexer.MatchingPodsByNodeName(node.Name)); err != nil {
			klog.Errorf("Failed to get pods for node %s: %v", node.Name, err)
			// 继续处理，但 pods 为空
		}

		// 计算节点上所有 Pods 使用的资源
		usedResources := make(corev1.ResourceList)
		workloadCount := 0

		for j := range podList.Items {
			pod := &podList.Items[j]
			podResources := utils.CalculateRequsetsByContainers(pod.Spec.Containers)
			usedResources = utils.SumResources(usedResources, podResources)

			if pod.Namespace == config.GetConfig().Namespaces.Job {
				// 只计算特定命名空间的 pods
				workloadCount++
			}
		}

		// 获取节点的供应商信息
		vendor := ""
		if vendorLabel, exists := node.Labels["crater.raids-lab.io/instance-type"]; exists {
			vendor = vendorLabel
		}

		// 获取 GPU 驱动版本
		gpuDriver := ""
		if driver, exists := node.Labels["nvidia.com/cuda.driver-version.full"]; exists {
			gpuDriver = driver
		}

		nodeInfos[i] = NodeBriefInfo{
			Name:          node.Name,
			Role:          getNodeRole(node),
			Arch:          node.Status.NodeInfo.Architecture,
			Status:        getNodeStatus(node),
			Vendor:        vendor,
			Taints:        node.Spec.Taints,
			Capacity:      node.Status.Capacity,
			Allocatable:   node.Status.Allocatable,
			Used:          usedResources,
			Workloads:     workloadCount,
			Annotations:   node.Annotations,
			KernelVersion: node.Status.NodeInfo.KernelVersion,
			GPUDriver:     gpuDriver,
		}
	}

	return nodeInfos, nil
}

// GetNode 获取指定 Node 的信息
func (nc *NodeClient) GetNode(ctx context.Context, name string) (ClusterNodeDetail, error) {
	node := &corev1.Node{}
	if err := nc.Get(ctx, client.ObjectKey{Name: name}, node); err != nil {
		return ClusterNodeDetail{}, err
	}

	// 获取节点上的所有 Pods（用于计算已使用资源）
	podList := &corev1.PodList{}
	usedResources := make(corev1.ResourceList)
	if err := nc.List(ctx, podList, indexer.MatchingPodsByNodeName(node.Name)); err != nil {
		klog.Errorf("Failed to get pods for node %s: %v", node.Name, err)
		// 继续处理，但 usedResources 为空
	} else {
		// 计算节点上所有 Pods 使用的资源
		for j := range podList.Items {
			pod := &podList.Items[j]
			podResources := utils.CalculateRequsetsByContainers(pod.Spec.Containers)
			usedResources = utils.SumResources(usedResources, podResources)
		}
	}

	// 获取 GPU 驱动版本
	gpuDriver := ""
	if driver, exists := node.Labels["nvidia.com/cuda.driver-version.full"]; exists {
		gpuDriver = driver
	}

	nodeInfo := ClusterNodeDetail{
		Name:                    node.Name,
		Role:                    string(getNodeRole(node)),
		Status:                  getNodeStatus(node),
		Taint:                   taintsToString(node.Spec.Taints),
		Time:                    node.CreationTimestamp.String(),
		Address:                 node.Status.Addresses[0].Address,
		Os:                      node.Status.NodeInfo.OperatingSystem,
		OsVersion:               node.Status.NodeInfo.OSImage,
		Arch:                    node.Status.NodeInfo.Architecture,
		KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		KernelVersion:           node.Status.NodeInfo.KernelVersion,
		Capacity:                node.Status.Capacity,
		Allocatable:             node.Status.Allocatable,
		Used:                    usedResources,
		GPUDriver:               gpuDriver,
	}
	return nodeInfo, nil
}

func (nc *NodeClient) UpdateNodeunschedule(ctx context.Context, name, reason, operator string) error {
	node, err := nc.KubeClient.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// 确保 Annotations 不为 nil
	if node.Annotations == nil {
		node.Annotations = make(map[string]string)
	}

	// 记录原状态
	wasUnschedulable := node.Spec.Unschedulable
	reasonKey := "crater.raids.io/unschedulable-reason"
	operatorKey := "crater.raids.io/unschedulable-operator"

	// 切换节点的 Unschedulable 状态
	node.Spec.Unschedulable = !node.Spec.Unschedulable

	// 添加或删除注解，记录操作原因和操作员
	if wasUnschedulable {
		// 恢复调度：删除原因和操作员注解
		delete(node.Annotations, reasonKey)
		delete(node.Annotations, operatorKey)
	} else {
		// 禁止调度：添加原因和操作员注解
		if reason != "" {
			node.Annotations[reasonKey] = reason
		}
		if operator != "" {
			node.Annotations[operatorKey] = formatOperatorInfo(ctx, operator)
		}
	}

	_, err = nc.KubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

func (nc *NodeClient) GetPodsForNode(ctx context.Context, nodeName string) ([]Pod, error) {
	// Get Pods for the node, which is a costly operation
	podList := &corev1.PodList{}
	if err := nc.List(ctx, podList, indexer.MatchingPodsByNodeName(nodeName)); err != nil {
		klog.Errorf("Failed to get pods for node %s: %v", nodeName, err)
		return nil, err
	}

	// Initialize the return value
	pods := make([]Pod, len(podList.Items))
	for i := range podList.Items {
		pod := &podList.Items[i]
		pods[i] = Pod{
			Name:            pod.Name,
			Namespace:       pod.Namespace,
			IP:              pod.Status.PodIP,
			CreateTime:      pod.CreationTimestamp,
			Status:          pod.Status.Phase,
			OwnerReference:  pod.OwnerReferences,
			Resources:       utils.CalculateRequsetsByContainers(pod.Spec.Containers),
			Locked:          false,
			LockedTimestamp: metav1.Time{},
		}
		if len(pod.OwnerReferences) == 0 {
			continue
		}
		owner := pod.OwnerReferences[0]
		if owner.Kind != VCJOBKIND || owner.APIVersion != VCJOBAPIVERSION {
			continue
		}
		// VCJob Pod, Check if it is locked
		jobDB := query.Job
		job, err := jobDB.WithContext(ctx).Where(jobDB.JobName.Eq(owner.Name)).First()
		if err != nil {
			klog.Errorf("Get job %s failed, err: %v", owner.Name, err)
			continue
		}
		pods[i].Locked = job.LockedTimestamp.After(utils.GetLocalTime())
		pods[i].PermanentLocked = utils.IsPermanentTime(job.LockedTimestamp)
		pods[i].LockedTimestamp = metav1.NewTime(job.LockedTimestamp)
	}

	return pods, nil
}

// detectGPUCountFromAllocatable 从节点的 Allocatable 资源中检测 GPU 数量
func detectGPUCountFromAllocatable(node *corev1.Node) int {
	// 优先从 Allocatable 获取 GPU 数量
	if quantity, ok := node.Status.Allocatable["nvidia.com/gpu"]; ok {
		return int(quantity.Value())
	}

	// 如果没找到，尝试遍历 Allocatable 寻找其他加速卡资源 (如 MIG, AMD, Hygon 等)
	for resName, quantity := range node.Status.Allocatable {
		name := string(resName)
		if strings.HasPrefix(name, "nvidia.com/") ||
			strings.HasPrefix(name, "amd.com/") ||
			strings.HasPrefix(name, "hygon.com/") {
			val := int(quantity.Value())
			if val > 0 {
				return val // 找到一个即停止，假设一个节点主要使用一种加速卡
			}
		}
	}
	return 0
}

// populateGPUInfoFromLabels 从节点标签填充 GPU 信息
func populateGPUInfoFromLabels(gpuInfo *GPUInfo, node *corev1.Node) error {
	gpuCountValue, ok := node.Labels["nvidia.com/gpu.count"]
	if !ok {
		return nil
	}

	// 如果 Allocatable 中没有找到 GPU 数量，则尝试从标签中获取
	if gpuInfo.GPUCount == 0 {
		gpuCount := 0
		_, err := fmt.Sscanf(gpuCountValue, "%d", &gpuCount)
		if err != nil {
			return err
		}
		gpuInfo.GPUCount = gpuCount
		if gpuInfo.GPUCount > 0 {
			gpuInfo.HaveGPU = true
		}
	}

	gpuInfo.GPUMemory = node.Labels["nvidia.com/gpu.memory"]
	gpuInfo.GPUArch = node.Labels["nvidia.com/gpu.family"]
	gpuInfo.GPUDriver = node.Labels["nvidia.com/cuda.driver-version.full"]
	gpuInfo.CudaVersion = node.Labels["nvidia.com/cuda.runtime-version.full"]
	gpuInfo.GPUProduct = node.Labels["nvidia.com/gpu.product"]
	return nil
}

// populateGPUUtilFromPrometheus 从 Prometheus 查询填充 GPU 使用率和 Job 关联
func (nc *NodeClient) populateGPUUtilFromPrometheus(gpuInfo *GPUInfo, nodeName string) {
	jobPodsList := nc.PrometheusClient.GetJobPodsList()
	gpuUtilMap := nc.PrometheusClient.QueryNodeGPUUtil()
	for i := 0; i < len(gpuUtilMap); i++ {
		gpuUtil := &gpuUtilMap[i]
		if gpuUtil.Hostname != nodeName {
			continue
		}
		gpuInfo.GPUUtil[gpuUtil.Gpu] = gpuUtil.Util
		// 如果gpuUtil.pod在jobPodsList的value中，则将jobPodsList中的job加入gpuInfo.RelateJobs[gpuUtil.Gpu]
		curPod := gpuUtil.Pod
		for job, pods := range jobPodsList {
			for _, pod := range pods {
				if curPod == pod {
					gpuInfo.RelateJobs[gpuUtil.Gpu] = append(gpuInfo.RelateJobs[gpuUtil.Gpu], job)
					break
				}
			}
		}
	}
}

// calculateTotalUsedGPU 计算节点上所有 Pods 使用的 GPU 资源总量
func calculateTotalUsedGPU(podList *corev1.PodList) int {
	totalUsedGPU := 0
	for j := range podList.Items {
		pod := &podList.Items[j]
		podResources := utils.CalculateRequsetsByContainers(pod.Spec.Containers)
		for resName, quantity := range podResources {
			resNameStr := string(resName)
			if strings.HasPrefix(resNameStr, "nvidia.com/") ||
				strings.HasPrefix(resNameStr, "amd.com/") ||
				strings.HasPrefix(resNameStr, "hygon.com/") {
				totalUsedGPU += int(quantity.Value())
			}
		}
	}
	return totalUsedGPU
}

// markK8sAllocatedGPUs 标记从 Kubernetes 分配但没有 Job 关联的 GPU
func markK8sAllocatedGPUs(gpuInfo *GPUInfo, totalUsedGPU int) {
	if totalUsedGPU <= 0 || gpuInfo.GPUCount <= 0 {
		return
	}
	for i := 0; i < totalUsedGPU && i < gpuInfo.GPUCount; i++ {
		gpuId := fmt.Sprintf("%d", i)
		// 如果该GPU没有Job关联，添加特殊标记表示已从Kubernetes分配
		if jobs, exists := gpuInfo.RelateJobs[gpuId]; !exists || len(jobs) == 0 {
			gpuInfo.RelateJobs[gpuId] = []string{"__k8s_allocated__"}
		}
	}
}

func (nc *NodeClient) GetNodeGPUInfo(name string) (GPUInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return GPUInfo{}, err
	}

	// 初始化返回值
	gpuInfo := GPUInfo{
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

	// 查找目标节点并填充 GPU 信息
	for i := range nodes.Items {
		node := &nodes.Items[i]
		if node.Name != name {
			continue
		}

		// 从 Allocatable 检测 GPU 数量
		gpuInfo.GPUCount = detectGPUCountFromAllocatable(node)
		if gpuInfo.GPUCount > 0 {
			gpuInfo.HaveGPU = true
		}

		// 从标签填充 GPU 信息
		if err := populateGPUInfoFromLabels(&gpuInfo, node); err != nil {
			return GPUInfo{}, err
		}
		break
	}

	// 从 Prometheus 查询 GPU 使用率和 Job 关联
	nc.populateGPUUtilFromPrometheus(&gpuInfo, name)

	// 获取节点上的所有 Pods，计算实际使用的 GPU 资源
	podList := &corev1.PodList{}
	if err := nc.List(context.Background(), podList, indexer.MatchingPodsByNodeName(name)); err != nil {
		klog.Errorf("Failed to get pods for node %s: %v", name, err)
	} else {
		totalUsedGPU := calculateTotalUsedGPU(podList)
		markK8sAllocatedGPUs(&gpuInfo, totalUsedGPU)
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

// GetNodeMarks 获取指定节点的Labels、Annotations和Taints
func (nc *NodeClient) GetNodeMarks(ctx context.Context, name string) (NodeMarkInfo, error) {
	node := &corev1.Node{}
	if err := nc.Get(ctx, client.ObjectKey{Name: name}, node); err != nil {
		return NodeMarkInfo{}, err
	}

	nodeMarkInfo := NodeMarkInfo{
		Labels:      node.Labels,
		Annotations: node.Annotations,
		Taints:      node.Spec.Taints,
	}

	return nodeMarkInfo, nil
}

// AddNodeLabel 添加节点标签
func (nc *NodeClient) AddNodeLabel(ctx context.Context, nodeName, key, value string) error {
	node, err := nc.KubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}
	node.Labels[key] = value

	_, err = nc.KubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

// DeleteNodeLabel 删除节点标签
func (nc *NodeClient) DeleteNodeLabel(ctx context.Context, nodeName, key string) error {
	node, err := nc.KubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if node.Labels != nil {
		delete(node.Labels, key)
	}

	_, err = nc.KubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

// AddNodeAnnotation 添加节点注解
func (nc *NodeClient) AddNodeAnnotation(ctx context.Context, nodeName, key, value string) error {
	node, err := nc.KubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if node.Annotations == nil {
		node.Annotations = make(map[string]string)
	}
	node.Annotations[key] = value

	_, err = nc.KubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

// DeleteNodeAnnotation 删除节点注解
func (nc *NodeClient) DeleteNodeAnnotation(ctx context.Context, nodeName, key string) error {
	node, err := nc.KubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if node.Annotations != nil {
		delete(node.Annotations, key)
	}

	_, err = nc.KubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

// AddNodeTaint 添加节点污点
func (nc *NodeClient) AddNodeTaint(ctx context.Context, nodeName, key, value, effect, reason, operator string) error {
	node, err := nc.KubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// 确保 Annotations 不为 nil
	if node.Annotations == nil {
		node.Annotations = make(map[string]string)
	}

	now := metav1.Now()
	newTaint := corev1.Taint{
		Key:       key,
		Value:     value,
		Effect:    corev1.TaintEffect(effect),
		TimeAdded: &now,
	}

	// 检查污点是否已经存在
	for _, existingTaint := range node.Spec.Taints {
		if existingTaint.MatchTaint(&newTaint) {
			return fmt.Errorf("taint %s=%s:%s already exists on node %s", key, value, effect, nodeName)
		}
	}

	// 添加新的污点
	node.Spec.Taints = append(node.Spec.Taints, newTaint)

	// 如果是账户独占，记录原因和操作员
	if strings.HasPrefix(key, "crater.raids.io/account") && effect == "NoSchedule" {
		if reason != "" {
			node.Annotations["crater.raids.io/taint-reason-occupied"] = reason
		}
		if operator != "" {
			node.Annotations["crater.raids.io/taint-operator-occupied"] = formatOperatorInfo(ctx, operator)
		}
	}

	_, err = nc.KubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

// DeleteNodeTaint 删除节点污点
func (nc *NodeClient) DeleteNodeTaint(ctx context.Context, nodeName, key, value, effect string) error {
	node, err := nc.KubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	targetTaint := corev1.Taint{
		Key:    key,
		Value:  value,
		Effect: corev1.TaintEffect(effect),
	}

	// 从现有污点列表中删除指定的污点
	newTaints := []corev1.Taint{}
	found := false
	for _, existingTaint := range node.Spec.Taints {
		if !existingTaint.MatchTaint(&targetTaint) {
			newTaints = append(newTaints, existingTaint)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("taint %s=%s:%s not found on node %s", key, value, effect, nodeName)
	}

	// 更新污点列表
	node.Spec.Taints = newTaints

	// 如果删除的是账户独占污点，删除对应的注解
	if strings.HasPrefix(key, "crater.raids.io/account") && effect == "NoSchedule" && node.Annotations != nil {
		delete(node.Annotations, "crater.raids.io/taint-reason-occupied")
		delete(node.Annotations, "crater.raids.io/taint-operator-occupied")
	}

	_, err = nc.KubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}
