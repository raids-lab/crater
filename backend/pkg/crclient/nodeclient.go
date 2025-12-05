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

	"github.com/raids-lab/crater/dao/model"
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
	Address       string                   `json:"address"`
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
	// 管理员接口返回的字段（omitempty 表示字段为空时不序列化）
	UserName        string `json:"userName,omitempty"`        // 用户昵称（用于显示）
	UserID          uint   `json:"userID,omitempty"`          // 用户ID（用于跳转）
	UserRealName    string `json:"userRealName,omitempty"`    // 用户真实名称（用于tooltip）
	AccountName     string `json:"accountName,omitempty"`     // 账户昵称（用于显示）
	AccountID       uint   `json:"accountID,omitempty"`       // 账户ID（用于跳转）
	AccountRealName string `json:"accountRealName,omitempty"` // 账户真实名称（用于tooltip）
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

// GPUDeviceInfo 表示一种类型的 GPU 设备信息
type GPUDeviceInfo struct {
	ResourceName   string `json:"resourceName"`   // 资源名称，如 "nvidia.com/gpu"
	Label          string `json:"label"`          // 显示名称（从数据库获取，如 "NVIDIA GPU"）
	Product        string `json:"product"`        // 具体型号（从节点标签获取，如 "Tesla V100"，可选）
	VendorDomain   string `json:"vendorDomain"`   // 供应商域名，如 "nvidia.com"
	Count          int    `json:"count"`          // 数量
	Memory         string `json:"memory"`         // 显存
	Arch           string `json:"arch"`           // 架构
	Driver         string `json:"driver"`         // 驱动版本
	RuntimeVersion string `json:"runtimeVersion"` // 运行时版本（CUDA/ROCm 等）
}

// GPUInfo 节点的 GPU 信息（支持多种类型的 GPU）
type GPUInfo struct {
	Name       string              `json:"name"`
	HaveGPU    bool                `json:"haveGPU"`
	GPUCount   int                 `json:"gpuCount"`   // 总 GPU 数量
	GPUDevices []GPUDeviceInfo     `json:"gpuDevices"` // 多种类型的 GPU 设备列表
	GPUUtil    map[string]float32  `json:"gpuUtil"`
	RelateJobs map[string][]string `json:"relateJobs"`

	// 以下字段保留用于向后兼容（取第一个 GPU 设备的信息）
	GPUMemory   string `json:"gpuMemory"`
	GPUArch     string `json:"gpuArch"`
	GPUDriver   string `json:"gpuDriver"`
	CudaVersion string `json:"cudaVersion"`
	GPUProduct  string `json:"gpuProduct"`
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

// getNodeIP 提取节点IP地址，与节点详情页保持一致
// 优先使用第一个地址（与当前详情页实现一致）
func getNodeIP(node *corev1.Node) string {
	if len(node.Status.Addresses) == 0 {
		return ""
	}
	return node.Status.Addresses[0].Address
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

		// 获取节点IP地址
		nodeIP := getNodeIP(node)

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
			Address:       nodeIP,
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

	// 获取节点IP地址（使用统一的提取逻辑）
	nodeIP := getNodeIP(node)

	nodeInfo := ClusterNodeDetail{
		Name:                    node.Name,
		Role:                    string(getNodeRole(node)),
		Status:                  getNodeStatus(node),
		Taint:                   taintsToString(node.Spec.Taints),
		Time:                    node.CreationTimestamp.String(),
		Address:                 nodeIP,
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

// getPodsForNode 获取节点上的 Pod 列表（内部辅助方法）
func (nc *NodeClient) getPodsForNode(ctx context.Context, nodeName string) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	if err := nc.List(ctx, podList, indexer.MatchingPodsByNodeName(nodeName)); err != nil {
		klog.Errorf("Failed to get pods for node %s: %v", nodeName, err)
		return nil, err
	}
	return podList, nil
}

func (nc *NodeClient) GetPodsForNode(ctx context.Context, nodeName string) ([]Pod, error) {
	// Get Pods for the node, which is a costly operation
	podList, err := nc.getPodsForNode(ctx, nodeName)
	if err != nil {
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

// AdminGetPodsForNode 获取节点上的 Pod 列表（管理员接口）
// 返回包含作业和用户信息的 Pod 列表
func (nc *NodeClient) AdminGetPodsForNode(ctx context.Context, nodeName string) ([]Pod, error) {
	podList, err := nc.getPodsForNode(ctx, nodeName)
	if err != nil {
		return nil, err
	}

	// Initialize the return value
	pods := make([]Pod, len(podList.Items))
	jobDB := query.Job
	jobNamespace := config.GetConfig().Namespaces.Job

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

		// 如果是 crater 作业命名空间中的 VolcanoJob Pod，查询作业和用户信息
		if pod.Namespace == jobNamespace && len(pod.OwnerReferences) > 0 {
			for _, owner := range pod.OwnerReferences {
				if owner.Kind != VCJOBKIND || owner.APIVersion != VCJOBAPIVERSION {
					continue
				}

				job, err := jobDB.WithContext(ctx).
					Preload(jobDB.User).
					Preload(jobDB.Account).
					Where(jobDB.JobName.Eq(owner.Name)).
					First()
				if err != nil {
					klog.Errorf("Get job %s failed, err: %v", owner.Name, err)
					break
				}

				// 用户信息
				pods[i].UserID = job.UserID
				pods[i].UserRealName = job.User.Name
				pods[i].UserName = job.User.Nickname
				if pods[i].UserName == "" {
					pods[i].UserName = job.User.Name
				}

				// 账户信息
				pods[i].AccountID = job.AccountID
				pods[i].AccountRealName = job.Account.Name
				pods[i].AccountName = job.Account.Nickname
				if pods[i].AccountName == "" {
					pods[i].AccountName = job.Account.Name
				}

				// 检查锁定状态
				pods[i].Locked = job.LockedTimestamp.After(utils.GetLocalTime())
				pods[i].PermanentLocked = utils.IsPermanentTime(job.LockedTimestamp)
				pods[i].LockedTimestamp = metav1.NewTime(job.LockedTimestamp)
				break
			}
		}
	}

	return pods, nil
}

// getGPUResourcesFromDB 从数据库获取所有 GPU 类型的资源列表
func getGPUResourcesFromDB(ctx context.Context) ([]*model.Resource, error) {
	r := query.Resource
	gpuResources, err := r.WithContext(ctx).
		Where(r.Type.Eq(string(model.ResourceTypeGPU))).
		Find()
	if err != nil {
		return nil, err
	}
	return gpuResources, nil
}

// detectGPUDevicesFromAllocatable 从节点的 Allocatable 资源中检测所有类型的 GPU 设备
func detectGPUDevicesFromAllocatable(ctx context.Context, node *corev1.Node) []GPUDeviceInfo {
	var devices []GPUDeviceInfo

	// 从数据库查询所有 GPU 类型的资源
	gpuResources, err := getGPUResourcesFromDB(ctx)
	if err != nil {
		klog.Errorf("Failed to query GPU resources from database: %v", err)
		return devices
	}

	// 遍历数据库中的 GPU 资源，匹配节点的 Allocatable 资源
	for _, gpuResource := range gpuResources {
		if quantity, ok := node.Status.Allocatable[corev1.ResourceName(gpuResource.ResourceName)]; ok {
			count := int(quantity.Value())
			if count > 0 {
				vendorDomain := ""
				if gpuResource.VendorDomain != nil {
					vendorDomain = *gpuResource.VendorDomain
				}

				devices = append(devices, GPUDeviceInfo{
					ResourceName: gpuResource.ResourceName,
					Label:        gpuResource.Label, // 从数据库获取显示名称
					VendorDomain: vendorDomain,
					Count:        count,
				})
			}
		}
	}
	return devices
}

// populateGPUDeviceDetailsFromLabels 从节点标签填充每个 GPU 设备的详细信息
func populateGPUDeviceDetailsFromLabels(devices []GPUDeviceInfo, node *corev1.Node) []GPUDeviceInfo {
	for i := range devices {
		device := &devices[i]
		resourceName := device.ResourceName

		// 重置设备信息，确保不会使用之前设备的值
		device.Product = ""
		device.Memory = ""
		device.Arch = ""
		device.Driver = ""
		device.RuntimeVersion = ""

		// 根据 ResourceName 确定应该读取哪个厂商的标签
		switch {
		case strings.HasPrefix(resourceName, "nvidia.com/"):
			device.Product = node.Labels["nvidia.com/gpu.product"]
			device.Memory = node.Labels["nvidia.com/gpu.memory"]
			device.Arch = node.Labels["nvidia.com/gpu.family"]
			device.Driver = node.Labels["nvidia.com/cuda.driver-version.full"]
			device.RuntimeVersion = node.Labels["nvidia.com/cuda.runtime-version.full"]
		case strings.HasPrefix(resourceName, "amd.com/"):
			device.Product = node.Labels["amd.com/gpu.product"]
			device.Memory = node.Labels["amd.com/gpu.memory"]
			device.Arch = node.Labels["amd.com/gpu.family"]
			device.Driver = node.Labels["amd.com/gpu.driver-version"]
			device.RuntimeVersion = node.Labels["amd.com/rocm.version"]
		case strings.HasPrefix(resourceName, "hygon.com/"):
			device.Product = node.Labels["hygon.com/gpu.product"]
			device.Memory = node.Labels["hygon.com/gpu.memory"]
			device.Arch = node.Labels["hygon.com/gpu.family"]
			device.Driver = node.Labels["hygon.com/gpu.driver-version"]
			device.RuntimeVersion = node.Labels["hygon.com/gpu.runtime-version"]
		}

		// 如果节点标签中没有 Product，使用数据库的 Label 作为默认值
		if device.Product == "" {
			device.Product = device.Label
		}
	}
	return devices
}

// getJobNameFromPod 从 Pod 的 OwnerReference 获取作业名称
func getJobNameFromPod(pod *corev1.Pod) string {
	if len(pod.OwnerReferences) == 0 {
		return pod.Name
	}
	return pod.OwnerReferences[0].Name
}

// isPodRunning 判断 Pod 是否处于运行状态
func isPodRunning(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodPending ||
		pod.Status.Phase == corev1.PodRunning ||
		pod.Status.Phase == corev1.PodUnknown
}

// buildPodNameToJobNameMap 构建 Pod 名称到作业名称的映射（只处理运行中的 Pod）
func buildPodNameToJobNameMap(podList *corev1.PodList) map[string]string {
	podToJob := make(map[string]string)
	for i := range podList.Items {
		pod := &podList.Items[i]
		if !isPodRunning(pod) {
			continue
		}
		podToJob[pod.Name] = getJobNameFromPod(pod)
	}
	return podToJob
}

// isMonitoringPod 判断是否为监控类 Pod（不是真正使用 GPU 的作业）
func isMonitoringPod(podName, namespace string) bool {
	monitoringKeywords := []string{"dcgm-exporter", "node-exporter", "prometheus", "grafana", "gpu-operator", "device-plugin"}
	monitoringNamespaces := []string{"gpu-operator", "monitoring", "prometheus", "kube-system", "nvidia-gpu-operator"}

	podNameLower := strings.ToLower(podName)
	for _, keyword := range monitoringKeywords {
		if strings.Contains(podNameLower, keyword) {
			return true
		}
	}

	namespaceLower := strings.ToLower(namespace)
	for _, ns := range monitoringNamespaces {
		if namespaceLower == ns {
			return true
		}
	}
	return false
}

// markGPUAllocationsWithPrometheus 使用 Prometheus 数据标记 GPU 分配情况
func (nc *NodeClient) markGPUAllocationsWithPrometheus(gpuInfo *GPUInfo, podToJob map[string]string) bool {
	if nc.PrometheusClient == nil {
		return false
	}

	gpuPodMapping := nc.PrometheusClient.QueryNodeGPUPodMapping(gpuInfo.Name)
	if len(gpuPodMapping) == 0 {
		return false
	}

	jobNamespace := config.GetConfig().Namespaces.Job

	for gpuId := range gpuPodMapping {
		gpuUtil := gpuPodMapping[gpuId]
		podName, podNamespace := gpuUtil.Pod, gpuUtil.Namespace
		if podName == "" {
			continue
		}

		// 过滤监控类 Pod 和非作业命名空间的 Pod
		if isMonitoringPod(podName, podNamespace) {
			continue
		}
		if podNamespace != "" && podNamespace != jobNamespace {
			continue
		}

		// 获取作业名称
		jobName := podToJob[podName]
		if jobName == "" {
			jobName = nc.PrometheusClient.GetPodOwner(podName)
			if jobName == "" {
				jobName = podName
			}
		}

		// 设置 GPU 关联的作业（避免重复）
		if jobs := gpuInfo.RelateJobs[gpuId]; len(jobs) == 0 || jobs[len(jobs)-1] != jobName {
			gpuInfo.RelateJobs[gpuId] = append(gpuInfo.RelateJobs[gpuId], jobName)
		}
		gpuInfo.GPUUtil[gpuId] = gpuUtil.Util
	}

	return true
}

// markK8sAllocatedGPUsFromPods 根据 Pod 资源请求标记 GPU 分配（Prometheus 不可用时的回退方案）
func markK8sAllocatedGPUsFromPods(ctx context.Context, gpuInfo *GPUInfo, podList *corev1.PodList, podToJob map[string]string) {
	if gpuInfo.GPUCount <= 0 || len(podList.Items) == 0 {
		return
	}

	gpuResources, err := getGPUResourcesFromDB(ctx)
	if err != nil {
		klog.Errorf("Failed to query GPU resources from database: %v", err)
		return
	}

	// 构建 GPU 资源名称集合
	gpuResourceNames := make(map[string]bool)
	for _, res := range gpuResources {
		gpuResourceNames[res.ResourceName] = true
	}

	// 统计每种 GPU 资源类型的分配情况
	type allocation struct {
		jobName string
		count   int
	}
	allocations := make(map[string][]allocation)

	for i := range podList.Items {
		pod := &podList.Items[i]
		if !isPodRunning(pod) {
			continue
		}

		for resName, quantity := range utils.CalculateRequsetsByContainers(pod.Spec.Containers) {
			resNameStr := string(resName)
			if !gpuResourceNames[resNameStr] || quantity.Value() <= 0 {
				continue
			}

			jobName := podToJob[pod.Name]
			if jobName == "" {
				jobName = pod.Name
			}
			allocations[resNameStr] = append(allocations[resNameStr], allocation{jobName, int(quantity.Value())})
		}
	}

	// 按资源类型顺序分配 GPU ID
	gpuIndex := 0
	for i := range gpuInfo.GPUDevices {
		device := &gpuInfo.GPUDevices[i]
		allocated := 0

		for _, alloc := range allocations[device.ResourceName] {
			for j := 0; j < alloc.count && allocated < device.Count; j++ {
				gpuInfo.RelateJobs[fmt.Sprintf("%d", gpuIndex+allocated)] = []string{alloc.jobName}
				allocated++
			}
		}
		gpuIndex += device.Count
	}
}

func (nc *NodeClient) GetNodeGPUInfo(ctx context.Context, name string) (GPUInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(ctx, &nodes)
	if err != nil {
		return GPUInfo{}, err
	}

	// 初始化返回值
	gpuInfo := GPUInfo{
		Name:        name,
		HaveGPU:     false,
		GPUCount:    0,
		GPUDevices:  []GPUDeviceInfo{},
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

		// 从 Allocatable 检测所有类型的 GPU 设备
		gpuInfo.GPUDevices = detectGPUDevicesFromAllocatable(ctx, node)

		// 从标签填充每个 GPU 设备的详细信息
		gpuInfo.GPUDevices = populateGPUDeviceDetailsFromLabels(gpuInfo.GPUDevices, node)

		// 计算总 GPU 数量
		for i := range gpuInfo.GPUDevices {
			gpuInfo.GPUCount += gpuInfo.GPUDevices[i].Count
		}

		if gpuInfo.GPUCount > 0 {
			gpuInfo.HaveGPU = true
		}

		// 向后兼容：如果有 GPU 设备，将第一个设备的信息填充到旧字段
		if len(gpuInfo.GPUDevices) > 0 {
			firstDevice := gpuInfo.GPUDevices[0]
			gpuInfo.GPUProduct = firstDevice.Product
			gpuInfo.GPUMemory = firstDevice.Memory
			gpuInfo.GPUArch = firstDevice.Arch
			gpuInfo.GPUDriver = firstDevice.Driver
			gpuInfo.CudaVersion = firstDevice.RuntimeVersion
		}

		break
	}

	// 获取节点上的所有 Pod
	podList := &corev1.PodList{}
	if err := nc.List(ctx, podList, indexer.MatchingPodsByNodeName(name)); err != nil {
		klog.Errorf("Failed to get pods for node %s: %v", name, err)
		return gpuInfo, nil
	}

	// 构建 Pod 名称到作业名称的映射
	podToJob := buildPodNameToJobNameMap(podList)

	// 优先尝试使用 Prometheus 获取真实的 GPU 分配情况
	if !nc.markGPUAllocationsWithPrometheus(&gpuInfo, podToJob) {
		// Prometheus 不可用时，回退到基于 K8s 资源请求的方式，处理非英伟达加速卡的情况
		markK8sAllocatedGPUsFromPods(ctx, &gpuInfo, podList, podToJob)
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
