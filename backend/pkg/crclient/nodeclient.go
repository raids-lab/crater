package crclient

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/utils"
)

const (
	VCJOBAPIVERSION = "batch.volcano.sh/v1alpha1"
	VCJOBKIND       = "Job"
)

type BriefResource struct {
	CPU string `json:"cpu"`
	Mem string `json:"memory"`
	GPU string `json:"gpu"`
}

type ClusterNodeInfo struct {
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	Role      string            `json:"role"`
	Labels    map[string]string `json:"labels"`
	IsReady   string            `json:"isReady"`
	Taint     string            `json:"taint"`
	Capacity  BriefResource     `json:"capacity"`
	Allocated BriefResource     `json:"allocated"`
	PodCount  int               `json:"podCount"`
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
	Name                    string `json:"name"`
	Role                    string `json:"role"`
	IsReady                 string `json:"isReady"`
	Taint                   string `json:"taint"`
	Time                    string `json:"time"`
	Address                 string `json:"address"`
	Os                      string `json:"os"`
	OsVersion               string `json:"osVersion"`
	Arch                    string `json:"arch"`
	KubeletVersion          string `json:"kubeletVersion"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
	GPUMemory               string `json:"gpuMemory"`
	GPUCount                int    `json:"gpuCount"`
	GPUArch                 string `json:"gpuArch"`
}

type ListNodeResp struct {
	Rows []ClusterNodeInfo `json:"rows"`
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

// 判断是否会有网络、内存、磁盘等压力问题
// 判断是否被占用
func isNodeReady(node *corev1.Node) string {
	if node.Spec.Unschedulable {
		return "Unschedulable"
	}

	if !isNodeConditionsReady(node) {
		return StatusFalse
	}

	if isNodeOccupied(node) {
		return "occupied"
	}

	return StatusTrue
}

func isNodeOccupied(node *corev1.Node) bool {
	for _, taint := range node.Spec.Taints {
		taintStr := taintToString(taint)
		if strings.Contains(taintStr, "crater.raids.io/account=") && strings.HasSuffix(taintStr, ":NoSchedule") {
			return true
		}
	}
	return false
}

func isNodeConditionsReady(node *corev1.Node) bool {
	var (
		isReady          bool
		diskPressure     bool
		memoryPressure   bool
		pidPressure      bool
		networkAvailable = true
	)

	for _, condition := range node.Status.Conditions {
		switch condition.Type {
		case corev1.NodeReady:
			isReady = (condition.Status == corev1.ConditionTrue)
		case corev1.NodeDiskPressure:
			diskPressure = (condition.Status == corev1.ConditionTrue)
		case corev1.NodeMemoryPressure:
			memoryPressure = (condition.Status == corev1.ConditionTrue)
		case corev1.NodePIDPressure:
			pidPressure = (condition.Status == corev1.ConditionTrue)
		case corev1.NodeNetworkUnavailable:
			networkAvailable = (condition.Status == corev1.ConditionFalse)
		}
	}

	if !isReady || diskPressure || memoryPressure || pidPressure || !networkAvailable {
		return false
	}

	return true
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

func formatCPULoad(cpu float32) string {
	return fmt.Sprintf("%.2f", cpu)
}

func formatMemoryLoad(mem int) string {
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
func (nc *NodeClient) getGPUCountOfNodes(nodes []corev1.Node) map[string]int {
	nodeMap := make(map[string]struct{})
	for i := range nodes {
		nodeMap[nodes[i].Name] = struct{}{}
	}
	gpuUtils := nc.PrometheusClient.QueryNodeGPUUtil()
	gpuCountMap := make(map[string]int)
	for i := range gpuUtils {
		gpuUtil := &gpuUtils[i]
		if _, ok := nodeMap[gpuUtil.Hostname]; ok {
			gpuCountMap[gpuUtil.Hostname]++
			continue
		}
		// try to check ip for ack
		// TODO(liyilong): remove hack code
		instanceSlice := strings.Split(gpuUtil.Instance, ":")
		if len(instanceSlice) != 2 {
			continue
		}
		ip := instanceSlice[0]
		for k := range nodeMap {
			if strings.Contains(k, ip) {
				gpuCountMap[k]++
				break
			}
		}
	}
	return gpuCountMap
}

// GetNodes 获取所有 Node 列表
func (nc *NodeClient) ListNodes() ([]ClusterNodeInfo, error) {
	var nodes corev1.NodeList

	err := nc.List(context.Background(), &nodes)
	if err != nil {
		return nil, err
	}

	nodeInfos := make([]ClusterNodeInfo, len(nodes.Items))
	CPUMap := nc.PrometheusClient.QueryNodeAllocatedCPU()
	MemMap := nc.PrometheusClient.QueryNodeAllocatedMemory()
	GPUMap := nc.PrometheusClient.QueryNodeAllocatedGPU()
	// 获取节点的 GPU 数量
	gpuCountMap := nc.getGPUCountOfNodes(nodes.Items)
	podCountMap := nc.PrometheusClient.QueryNodeRunningPodCount()

	// Loop through each node and print allocated resources
	for i := range nodes.Items {
		node := &nodes.Items[i]

		allocatedInfo := BriefResource{
			CPU: formatCPULoad(CPUMap[node.Name]),
			Mem: formatMemoryLoad(MemMap[node.Name]),
			GPU: fmt.Sprintf("%d", GPUMap[node.Name]),
		}

		cpuCapacity := node.Status.Capacity[corev1.ResourceCPU]
		memCapacity := node.Status.Capacity[corev1.ResourceMemory]
		gpuCapacity := gpuCountMap[node.Name]
		capacityInfo := BriefResource{
			CPU: cpuCapacity.String(),
			Mem: memCapacity.String(),
			GPU: fmt.Sprintf("%d", gpuCapacity),
		}

		podCount := 0
		if v, ok := podCountMap[node.Name]; ok {
			podCount = v
		}

		// 获取节点类型
		nodeType := node.Labels["crater.raids.io/nodetype"]
		nodeInfos[i] = ClusterNodeInfo{
			Type:      nodeType, // 添加节点类型
			Name:      node.Name,
			Taint:     taintsToString(node.Spec.Taints),
			Role:      getNodeRole(node),
			Labels:    node.Labels,
			IsReady:   isNodeReady(node),
			Capacity:  capacityInfo,
			Allocated: allocatedInfo,
			PodCount:  podCount,
		}
	}

	return nodeInfos, nil
}

// GetNode 获取指定 Node 的信息
func (nc *NodeClient) GetNode(ctx context.Context, name string) (ClusterNodeDetail, error) {
	node := &corev1.Node{}

	err := nc.Get(ctx, client.ObjectKey{
		Namespace: "",
		Name:      name,
	}, node)
	if err != nil {
		return ClusterNodeDetail{}, err
	}

	nodeInfo := ClusterNodeDetail{
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
	if err != nil {
		return err
	}
	var Taint corev1.Taint
	Taint, err1 := stringToTaint(taint)
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
func (nc *NodeClient) GetPodsForNode(ctx context.Context, nodeName string) ([]Pod, error) {
	// Get Pods for the node, which is a costly operation
	// TODO(zhangry): Add a cache for this? Or query from Prometheus?
	podList, err := nc.KubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
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
			logutils.Log.Errorf("Get job %s failed, err: %v", owner.Name, err)
			continue
		}
		pods[i].Locked = job.LockedTimestamp.After(utils.GetLocalTime())
		pods[i].PermanentLocked = utils.IsPermanentTime(job.LockedTimestamp)
		pods[i].LockedTimestamp = metav1.NewTime(job.LockedTimestamp)
	}

	return pods, nil
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
				return GPUInfo{}, err
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
