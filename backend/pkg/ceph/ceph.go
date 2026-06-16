//nolint:errorlint,gocritic,gocyclo,mnd,staticcheck // Ceph path/capacity discovery is infra-heavy and intentionally centralized.
package ceph

import (
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"sync"

	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	cfgpkg "github.com/raids-lab/crater/pkg/config"
)

const (
	parentDir         = ".."
	cephFSMountRoot   = "/mnt/mycephfs"
	defaultCephFSName = "cephfs"
)

const UnknownSizeBytes int64 = -1

func AvailableBytes(totalBytes, usedBytes int64) int64 {
	if totalBytes < 0 || usedBytes < 0 {
		return UnknownSizeBytes
	}
	return totalBytes - usedBytes
}

var (
	// cephMountPathCache 缓存 CephFS 挂载路径
	cephMountPathCache  string
	cephMountPathPodUID string
	cephMountPathMu     sync.Mutex
)

func sharedStoragePVCName() string {
	storagePVCName := "crater-storage"
	if cfg := cfgpkg.GetConfig(); cfg != nil {
		if value := strings.TrimSpace(cfg.Storage.PVC.ReadWriteMany); value != "" {
			storagePVCName = value
		}
	}
	return storagePVCName
}

// FindCephToolboxPod 查找 Rook-Ceph Toolbox Pod
func FindCephToolboxPod(clientset kubernetes.Interface, namespace string) (*corev1.Pod, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=rook-ceph-tools",
	})
	if err != nil {
		return nil, fmt.Errorf("列出 Pod 失败: %v", err)
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return &pod, nil
		}
	}

	return nil, fmt.Errorf("未找到运行中的 Rook-Ceph Toolbox Pod")
}

// ExecInPod 在指定 Pod 中执行命令
func ExecInPod(clientset kubernetes.Interface, config *rest.Config, pod *corev1.Pod, command []string) (string, error) {
	var stdout, stderr strings.Builder
	err := ExecInPodStream(clientset, config, pod, command, &stdout, &stderr)
	if err != nil {
		return "", fmt.Errorf("执行命令失败: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// ExecInPodStream 在指定 Pod 中执行命令，并将 stdout/stderr 流式写入给定 writer。
func ExecInPodStream(
	clientset kubernetes.Interface,
	config *rest.Config,
	pod *corev1.Pod,
	command []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("添加 scheme 失败: %v", err)
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Command: command,
		Stdin:   false,
		Stdout:  true,
		Stderr:  true,
		TTY:     false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, http.MethodPost, req.URL())
	if err != nil {
		return fmt.Errorf("创建 executor 失败: %v", err)
	}

	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return err
	}

	return nil
}

// StoragePrefixConfig 存储路径前缀配置
type StoragePrefixConfig struct {
	User    string
	Account string
	Public  string
}

// ResolveCephFSPath 将逻辑路径解析为 CephFS 中的实际路径
// 逻辑路径格式: /user/{space}/... 或 /public/... 或 /account/{space}/...
// CephFS 实际路径格式: /mnt/mycephfs/volumes/csi/{pvc}/{prefix}/...
// 该函数通过在 toolbox pod 中执行 ls 命令自动查找 PVC 挂载路径
func ResolveCephFSPath(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, logicalPath string,
	prefixConfig StoragePrefixConfig,
) (string, error) {
	toolboxPod, err := FindCephToolboxPod(clientset, namespace)
	if err != nil {
		return "", err
	}

	trimmedPath := strings.TrimLeft(logicalPath, "/")
	parts := strings.SplitN(trimmedPath, "/", 3)

	if len(parts) < 1 {
		return "", fmt.Errorf("invalid path format: %s", logicalPath)
	}

	var storagePrefix string
	var remainingPath string

	switch parts[0] {
	case "user":
		if len(parts) < 2 {
			return "", fmt.Errorf("user path must include space name: %s", logicalPath)
		}
		storagePrefix = prefixConfig.User
		remainingPath = strings.Join(parts[1:], "/")
	case "account":
		storagePrefix = prefixConfig.Account
		if len(parts) > 1 {
			remainingPath = strings.Join(parts[1:], "/")
		}
	case "public":
		storagePrefix = prefixConfig.Public
		if len(parts) > 1 {
			remainingPath = strings.Join(parts[1:], "/")
		}
	default:
		return "", fmt.Errorf("unknown path type: %s", parts[0])
	}

	cephMountPath, err := findCephMountPath(clientset, config, toolboxPod)
	if err != nil {
		return "", err
	}

	var fullPath string
	if remainingPath != "" {
		// 确保路径使用正斜杠，不包含 ./ 或 ../
		// 构建干净的路径
		var pathParts []string

		// 添加存储前缀
		if storagePrefix != "" {
			pathParts = append(pathParts, storagePrefix)
		}

		// 添加剩余路径部分
		if remainingPath != "" {
			// 分割剩余路径并清理
			remainingParts := strings.Split(remainingPath, "/")
			for _, part := range remainingParts {
				if part != "" && part != "." {
					if part == parentDir {
						// 处理上级目录
						if len(pathParts) > 0 {
							pathParts = pathParts[:len(pathParts)-1]
						}
					} else {
						pathParts = append(pathParts, part)
					}
				}
			}
		}

		// 构建最终路径
		if len(pathParts) > 0 {
			cleanPath := strings.Join(pathParts, "/")
			fullPath = fmt.Sprintf("%s/%s", cephMountPath, cleanPath)
		} else {
			fullPath = cephMountPath
		}
	} else {
		fullPath = fmt.Sprintf("%s/%s", cephMountPath, storagePrefix)
	}

	// 确保最终路径使用正斜杠
	fullPath = strings.ReplaceAll(fullPath, "\\", "/")
	// 清理路径，移除所有 ./ 和 ../
	fullPath = path.Clean(fullPath)

	return fullPath, nil
}

// findCephMountPath 在 toolbox pod 中自动查找 CephFS 的 PVC 挂载路径
// 通过 Kubernetes API 获取名为 crater-storage 的 PVC 信息，然后构建路径
func findCephMountPath(clientset kubernetes.Interface, config *rest.Config, pod *corev1.Pod) (string, error) {
	cephMountPathMu.Lock()
	defer cephMountPathMu.Unlock()

	podUID := string(pod.UID)
	if cephMountPathCache != "" && cephMountPathPodUID == podUID {
		return cephMountPathCache, nil
	}

	fmt.Println("=== 开始发现 CephFS PVC 路径 ===")
	mountPath, err := discoverCephMountPath(clientset, config, pod)
	if err != nil {
		fmt.Printf("=== 路径发现失败: %v ===\n", err)
		return "", err
	}

	cephMountPathCache = mountPath
	cephMountPathPodUID = podUID
	fmt.Printf("=== 路径发现成功: %s ===\n", cephMountPathCache)

	return cephMountPathCache, nil
}

// discoverCephMountPath 实际执行路径发现
func discoverCephMountPath(clientset kubernetes.Interface, config *rest.Config, pod *corev1.Pod) (string, error) {
	storagePVCName := sharedStoragePVCName()

	// 查找共享存储 PVC
	fmt.Printf("1. 查找名为 %s 的 PVC...\n", storagePVCName)
	craterStoragePVC, err := clientset.CoreV1().PersistentVolumeClaims("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "metadata.name=" + storagePVCName,
	})
	if err != nil {
		fmt.Printf("查找 %s PVC 失败: %v\n", storagePVCName, err)
		return "", fmt.Errorf("查找 %s PVC 失败: %v", storagePVCName, err)
	}

	if len(craterStoragePVC.Items) == 0 {
		fmt.Printf("未找到名为 %s 的 PVC\n", storagePVCName)
		return "", fmt.Errorf("未找到名为 %s 的 PVC", storagePVCName)
	}

	pvc := craterStoragePVC.Items[0]
	pvName := pvc.Spec.VolumeName
	if pvName == "" {
		fmt.Printf("%s PVC 未绑定到 PV\n", storagePVCName)
		return "", fmt.Errorf("%s PVC 未绑定到 PV", storagePVCName)
	}

	fmt.Printf("找到 %s PVC，对应的 PV: %s\n", storagePVCName, pvName)

	// 获取对应的 PV
	pv, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("获取 PV %s 失败: %v\n", pvName, err)
		return "", fmt.Errorf("获取 PV %s 失败: %v", pvName, err)
	}

	// 检查是否是 cephfs PV
	if pv.Spec.CSI == nil || pv.Spec.CSI.Driver != "rook-ceph.cephfs.csi.ceph.com" {
		fmt.Printf("%s PV 不是 cephfs 类型\n", storagePVCName)
		return "", fmt.Errorf("%s PV 不是 cephfs 类型", storagePVCName)
	}

	fmt.Printf("%s PV 是 cephfs 类型\n", storagePVCName)

	if err := ensureCephFSMounted(clientset, config, pod, cephFSNameFromPV(pv)); err != nil {
		return "", err
	}

	if subvolumePath := strings.TrimSpace(pv.Spec.CSI.VolumeAttributes["subvolumePath"]); subvolumePath != "" {
		pvcPath := path.Clean(cephFSMountRoot + "/" + strings.TrimLeft(subvolumePath, "/"))
		fmt.Printf("2. 从 PV volumeAttributes.subvolumePath 直接获取 PVC 路径: %s\n", pvcPath)
		return pvcPath, nil
	}

	// 检查 PV 的 volumeHandle
	if pv.Spec.CSI.VolumeHandle == "" {
		fmt.Printf("%s PV 没有 volumeHandle\n", storagePVCName)
		return "", fmt.Errorf("%s PV 没有 volumeHandle", storagePVCName)
	}

	volumeHandle := pv.Spec.CSI.VolumeHandle
	fmt.Printf("crater-storage PV 的 volumeHandle: %s\n", volumeHandle)

	// 提取 UUID 部分
	parts := strings.Split(volumeHandle, "-")
	if len(parts) < 5 {
		fmt.Println("volumeHandle 格式不正确")
		return "", fmt.Errorf("volumeHandle 格式不正确")
	}

	// 从后往前查找 UUID 部分
	var uuid string
	for i := len(parts) - 5; i >= 0; i-- {
		if len(parts[i]) == 8 {
			uuidParts := parts[i : i+5]
			uuid = strings.Join(uuidParts, "-")
			if len(uuid) == 36 {
				break
			}
		}
	}

	if uuid == "" {
		fmt.Println("无法从 volumeHandle 中提取 UUID")
		return "", fmt.Errorf("无法从 volumeHandle 中提取 UUID")
	}

	fmt.Printf("提取 UUID: %s\n", uuid)

	// 构建 csi-vol- 名称
	csiVolName := "csi-vol-" + uuid
	fmt.Printf("构建 csi-vol- 名称: %s\n", csiVolName)

	// 构建 PVC 路径
	csiPath := fmt.Sprintf("%s/volumes/csi/%s", cephFSMountRoot, csiVolName)
	fmt.Printf("构建 PVC 路径: %s\n", csiPath)

	// 查找该路径下的唯一子文件夹
	fmt.Println("2. 正在查找子文件夹...")
	command := []string{"ls", csiPath}
	output, err := ExecInPod(clientset, config, pod, command)
	if err != nil {
		fmt.Printf("列出 PVC 目录失败: %v\n", err)
		return "", fmt.Errorf("列出 PVC 目录失败: %v", err)
	}

	fmt.Printf("PVC 目录内容: %s\n", output)

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var subDir string
	for _, line := range lines {
		if line != "." && line != ".." && !strings.HasPrefix(line, ".") {
			subDir = line
			fmt.Printf("找到子文件夹: %s\n", subDir)
			break
		}
	}

	if subDir == "" {
		fmt.Println("未找到子文件夹")
		return "", fmt.Errorf("未找到子文件夹")
	}

	// 构建完整的 PVC 挂载路径
	pvcPath := fmt.Sprintf("%s/%s", csiPath, subDir)
	// 确保路径使用正斜杠
	pvcPath = strings.ReplaceAll(pvcPath, "\\", "/")
	// 清理路径
	pvcPath = path.Clean(pvcPath)

	fmt.Printf("3. 构建完整 PVC 挂载路径: %s\n", pvcPath)

	return pvcPath, nil
}

func cephFSNameFromPV(pv *corev1.PersistentVolume) string {
	if pv != nil && pv.Spec.CSI != nil {
		if fsName := strings.TrimSpace(pv.Spec.CSI.VolumeAttributes["fsName"]); fsName != "" {
			return fsName
		}
	}
	return defaultCephFSName
}

func ensureCephFSMounted(
	clientset kubernetes.Interface,
	config *rest.Config,
	pod *corev1.Pod,
	fsName string,
) error {
	if strings.TrimSpace(fsName) == "" {
		fsName = defaultCephFSName
	}

	script := fmt.Sprintf(`
set -eu
mount_root=%s
fs_name=%s
if grep -qs " ${mount_root} " /proc/mounts; then
  exit 0
fi
mkdir -p "${mount_root}"
if command -v ceph-fuse >/dev/null 2>&1; then
  ceph-fuse --client_fs "${fs_name}" "${mount_root}" || ceph-fuse "${mount_root}"
else
  mount -t ceph :/ "${mount_root}" -o name=admin,fs="${fs_name}" || mount -t ceph :/ "${mount_root}" -o name=admin
fi
for i in 1 2 3 4 5; do
  if grep -qs " ${mount_root} " /proc/mounts; then
    exit 0
  fi
  sleep 1
done
echo "cephfs mount did not appear at ${mount_root}" >&2
exit 1
`, shellQuote(cephFSMountRoot), shellQuote(fsName))

	if _, err := ExecInPod(clientset, config, pod, []string{"sh", "-c", script}); err != nil {
		return fmt.Errorf("ensure CephFS mounted in toolbox failed: %v", err)
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// GetCephDirectorySize 获取 CephFS 目录大小
func GetCephDirectorySize(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, logicalPath string,
	prefixConfig StoragePrefixConfig,
) (int64, error) {
	toolboxPod, err := FindCephToolboxPod(clientset, namespace)
	if err != nil {
		return 0, err
	}

	fullPath, err := ResolveCephFSPath(clientset, config, namespace, logicalPath, prefixConfig)
	if err != nil {
		return 0, err
	}

	// 确保路径使用正斜杠
	fullPath = strings.ReplaceAll(fullPath, "\\", "/")

	command := []string{"getfattr", "-n", "ceph.dir.rbytes", fullPath}
	output, err := ExecInPod(clientset, config, toolboxPod, command)
	if err != nil {
		return 0, err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ceph.dir.rbytes=") {
			sizeStr := strings.Trim(strings.TrimPrefix(line, "ceph.dir.rbytes="), "\"")
			size, err := strconv.ParseInt(sizeStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("解析大小失败: %v", err)
			}
			return size, nil
		}
	}

	return 0, fmt.Errorf("未找到 ceph.dir.rbytes 信息: %s", output)
}

// SetCephDirectoryQuota 设置 CephFS 目录配额
func SetCephDirectoryQuota(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, logicalPath string,
	prefixConfig StoragePrefixConfig,
	maxBytes int64,
) error {
	toolboxPod, err := FindCephToolboxPod(clientset, namespace)
	if err != nil {
		return err
	}

	fullPath, err := ResolveCephFSPath(clientset, config, namespace, logicalPath, prefixConfig)
	if err != nil {
		return err
	}

	// 确保路径使用正斜杠
	fullPath = strings.ReplaceAll(fullPath, "\\", "/")

	// 检查路径是否存在
	checkCommand := []string{"ls", "-la", fullPath}
	checkOutput, err := ExecInPod(clientset, config, toolboxPod, checkCommand)
	if err != nil {
		return fmt.Errorf("检查路径失败: %v, 输出: %s", err, checkOutput)
	}

	// 构建 setfattr 命令
	// 注意：-1 表示无限制，需要设置为 0
	var command []string
	if maxBytes == -1 {
		// 移除配额限制（设置为 0）
		command = []string{"setfattr", "-n", "ceph.quota.max_bytes", "-v", "0", fullPath}
	} else {
		// 设置配额限制
		command = []string{"setfattr", "-n", "ceph.quota.max_bytes", "-v", fmt.Sprintf("%d", maxBytes), fullPath}
	}

	// 执行命令并获取详细输出
	output, err := ExecInPod(clientset, config, toolboxPod, command)
	if err != nil {
		return fmt.Errorf("设置配额失败: %v, 命令: %v, 输出: %s", err, command, output)
	}

	return nil
}

// GetCraterStorageCapacity 通过 Kubernetes API 读取 crater-storage PVC 的容量，
// 并通过 getfattr 读取已用量。
// 返回 (totalBytes, usedBytes, error)。
func GetCraterStorageCapacity(clientset kubernetes.Interface, config *rest.Config, namespace string) (int64, int64, error) {
	storagePVCName := sharedStoragePVCName()

	// 1. 从 K8s API 读取 PVC 容量（不需要 exec）
	// 使用提供的 namespace 参数，而不是空字符串
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: "metadata.name=" + storagePVCName,
	})
	if err != nil {
		return UnknownSizeBytes, UnknownSizeBytes, nil
	}

	if len(pvcs.Items) == 0 {
		// 如果在指定命名空间中找不到，尝试在所有命名空间中查找
		pvcs, err = clientset.CoreV1().PersistentVolumeClaims("").List(context.TODO(), metav1.ListOptions{
			FieldSelector: "metadata.name=" + storagePVCName,
		})
		if err != nil || len(pvcs.Items) == 0 {
			return UnknownSizeBytes, UnknownSizeBytes, nil
		}
	}

	pvc := pvcs.Items[0]
	var totalBytes int64
	if cap, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
		totalBytes = cap.Value()
	} else if req, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		totalBytes = req.Value()
	} else {
		totalBytes = UnknownSizeBytes
	}

	// 2. 通过 getfattr 读取 crater-storage PVC 对应 subvolume 的已用量
	toolboxPod, err := FindCephToolboxPod(clientset, namespace)
	if err != nil {
		return totalBytes, UnknownSizeBytes, nil
	}

	// crater-storage 只是 CephFS 下的一个 subvolume，不应直接统计整个 /mnt/mycephfs 根目录。
	mountPath, err := findCephMountPath(clientset, config, toolboxPod)
	if err != nil {
		return totalBytes, UnknownSizeBytes, nil
	}

	// 尝试获取 crater-storage subvolume 的已使用容量
	out, err := ExecInPod(clientset, config, toolboxPod, []string{"getfattr", "-n", "ceph.dir.rbytes", mountPath})
	if err != nil {
		return totalBytes, UnknownSizeBytes, nil
	}

	usedBytes := UnknownSizeBytes
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ceph.dir.rbytes=") {
			valStr := strings.Trim(strings.TrimPrefix(line, "ceph.dir.rbytes="), "\"")
			if v, parseErr := strconv.ParseInt(valStr, 10, 64); parseErr == nil {
				usedBytes = v
			} else {
				return totalBytes, UnknownSizeBytes, nil
			}
			break
		}
	}

	// 确保返回有效的值
	if usedBytes == UnknownSizeBytes {
		// 如果无法获取已用量，尝试使用 du 命令作为备选方案
		out, err := ExecInPod(clientset, config, toolboxPod, []string{"du", "-sb", mountPath})
		if err == nil {
			parts := strings.Fields(out)
			if len(parts) > 0 {
				if v, parseErr := strconv.ParseInt(parts[0], 10, 64); parseErr == nil {
					usedBytes = v
				}
			}
		}
	}

	return totalBytes, usedBytes, nil
}

// GetCephMountRoot 获取 CephFS 挂载根路径（用于全局统计）
func GetCephMountRoot(clientset kubernetes.Interface, config *rest.Config, namespace string) (string, error) {
	toolboxPod, err := FindCephToolboxPod(clientset, namespace)
	if err != nil {
		return "", err
	}
	return findCephMountPath(clientset, config, toolboxPod)
}

// GetAllUserSpaceSizes 获取所有用户空间的大小
func GetAllUserSpaceSizes(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace string,
	prefixConfig StoragePrefixConfig,
	page, pageSize int,
) (map[string]int64, int, error) {
	toolboxPod, err := FindCephToolboxPod(clientset, namespace)
	if err != nil {
		return nil, 0, err
	}

	// 直接构建用户空间根目录路径
	cephMountPath, err := findCephMountPath(clientset, config, toolboxPod)
	if err != nil {
		return nil, 0, err
	}

	// 构建用户空间根目录路径
	userRootPath := fmt.Sprintf("%s/%s", cephMountPath, prefixConfig.User)
	// 确保路径使用正斜杠
	userRootPath = strings.ReplaceAll(userRootPath, "\\", "/")

	// 列出用户空间根目录下的所有子目录（即用户空间）
	command := []string{"ls", userRootPath}
	output, err := ExecInPod(clientset, config, toolboxPod, command)
	if err != nil {
		return nil, 0, fmt.Errorf("列出用户空间目录失败: %v", err)
	}

	// 解析输出，获取所有用户空间名称
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// 过滤有效用户目录
	var validUsers []string
	for _, user := range lines {
		user = strings.TrimSpace(user)
		if user != "" && user != "." && user != ".." && !strings.HasPrefix(user, ".") {
			validUsers = append(validUsers, user)
		}
	}

	// 计算总用户数
	totalUsers := len(validUsers)

	// 计算分页范围
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= totalUsers {
		return make(map[string]int64), totalUsers, nil
	}
	if end > totalUsers {
		end = totalUsers
	}

	// 获取当前页的用户
	currentPageUsers := validUsers[start:end]

	// 对当前页的每个用户空间获取大小
	userSpaces := make(map[string]int64)
	for _, user := range currentPageUsers {
		// 构建用户空间的逻辑路径
		userPath := fmt.Sprintf("/user/%s", user)

		// 获取用户空间大小
		size, err := GetCephDirectorySize(clientset, config, namespace, userPath, prefixConfig)
		if err != nil {
			// 记录错误但继续处理其他用户
			fmt.Printf("获取用户 %s 空间大小失败: %v\n", user, err)
			continue
		}

		userSpaces[user] = size
	}

	return userSpaces, totalUsers, nil
}
