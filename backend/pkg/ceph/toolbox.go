package ceph

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	cfgpkg "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/storagequota"
)

const (
	toolboxMountRoot    = "/mnt/crater-cephfs"
	cephUsageXattr      = "ceph.dir.rbytes"
	cephQuotaBytesXattr = "ceph.quota.max_bytes"
)

var errToolboxXattrNotFound = errors.New("CephFS extended attribute is not set")

func cephQuotaXattrValue(maxBytes int64) int64 {
	if maxBytes == -1 {
		return 0
	}
	return maxBytes
}

var volumeHandleUUIDPattern = regexp.MustCompile(
	`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
)

func GetToolboxQuotaCapabilities(
	ctx context.Context,
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace string,
) (storagequota.Capabilities, error) {
	var capabilities storagequota.Capabilities
	pod, storageRoot, err := resolveToolboxStorageRoot(ctx, clientset, config, namespace)
	if err != nil {
		return capabilities, err
	}

	if _, err := readToolboxXattr(ctx, clientset, config, pod, storageRoot, cephUsageXattr); err != nil {
		capabilities.Reasons = append(capabilities.Reasons, err.Error())
		return capabilities, nil
	}
	capabilities.UsageReadable = true

	if _, err := readToolboxXattr(ctx, clientset, config, pod, storageRoot, cephQuotaBytesXattr); err == nil ||
		errors.Is(err, errToolboxXattrNotFound) {
		capabilities.QuotaReadable = true
	}

	probeScript := fmt.Sprintf(`
set -eu
test_dir=$(mktemp -d %s/.crater-quota-capability-XXXXXX)
cleanup() {
  setfattr -n %s -v 0 "$test_dir" >/dev/null 2>&1 || true
  rmdir "$test_dir" >/dev/null 2>&1 || true
}
trap cleanup EXIT
setfattr -n %s -v 0 "$test_dir"
getfattr --only-values -n %s "$test_dir" >/dev/null
`, shellQuote(storageRoot), cephQuotaBytesXattr, cephQuotaBytesXattr, cephQuotaBytesXattr)
	if _, err := execToolbox(ctx, clientset, config, pod, []string{"sh", "-c", probeScript}); err != nil {
		capabilities.Reasons = append(capabilities.Reasons, fmt.Sprintf("toolbox quota write probe: %v", err))
		return capabilities, nil
	}

	capabilities.QuotaReadable = true
	capabilities.QuotaWritable = true
	return capabilities, nil
}

func getToolboxUsage(
	ctx context.Context,
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, relativePath string,
) (int64, error) {
	pod, storageRoot, err := resolveToolboxStorageRoot(ctx, clientset, config, namespace)
	if err != nil {
		return 0, err
	}
	targetPath, err := toolboxStoragePath(storageRoot, relativePath)
	if err != nil {
		return 0, err
	}
	targetPath, err = resolveToolboxTargetPath(ctx, clientset, config, pod, storageRoot, targetPath)
	if err != nil {
		return 0, err
	}
	value, err := readToolboxXattr(ctx, clientset, config, pod, targetPath, cephUsageXattr)
	if err != nil {
		return 0, err
	}
	usage, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse toolbox usage %q: %w", value, err)
	}
	return usage, nil
}

func getToolboxQuota(
	ctx context.Context,
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, relativePath string,
) (int64, error) {
	pod, storageRoot, err := resolveToolboxStorageRoot(ctx, clientset, config, namespace)
	if err != nil {
		return 0, err
	}
	targetPath, err := toolboxStoragePath(storageRoot, relativePath)
	if err != nil {
		return 0, err
	}
	targetPath, err = resolveToolboxTargetPath(ctx, clientset, config, pod, storageRoot, targetPath)
	if err != nil {
		return 0, err
	}
	value, err := readToolboxXattr(ctx, clientset, config, pod, targetPath, cephQuotaBytesXattr)
	if errors.Is(err, errToolboxXattrNotFound) {
		return -1, nil
	}
	if err != nil {
		return 0, err
	}
	quota, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse toolbox quota %q: %w", value, err)
	}
	return normalizeCephQuota(quota), nil
}

func setToolboxQuota(
	ctx context.Context,
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, relativePath string,
	maxBytes int64,
) error {
	pod, storageRoot, err := resolveToolboxStorageRoot(ctx, clientset, config, namespace)
	if err != nil {
		return err
	}
	targetPath, err := toolboxStoragePath(storageRoot, relativePath)
	if err != nil {
		return err
	}
	targetPath, err = resolveToolboxTargetPath(ctx, clientset, config, pod, storageRoot, targetPath)
	if err != nil {
		return err
	}
	script := fmt.Sprintf(
		"setfattr -n %s -v %s -- %s",
		cephQuotaBytesXattr,
		strconv.FormatInt(maxBytes, 10),
		shellQuote(targetPath),
	)
	if _, err := execToolbox(ctx, clientset, config, pod, []string{"sh", "-c", script}); err != nil {
		return fmt.Errorf("write CephFS quota through toolbox: %w", err)
	}
	return nil
}

//nolint:gocyclo // Resolving the toolbox mount validates each PV and CSI fallback explicitly.
func resolveToolboxStorageRoot(
	ctx context.Context,
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace string,
) (*corev1.Pod, string, error) {
	if clientset == nil || config == nil {
		return nil, "", fmt.Errorf("kubernetes client or REST config is unavailable")
	}
	pod, err := findToolboxPod(ctx, clientset, namespace, StorageQuotaToolboxLabelSelector())
	if err != nil {
		return nil, "", err
	}

	pvcNamespace := strings.TrimSpace(cfgpkg.GetConfig().Namespaces.Job)
	if pvcNamespace == "" {
		return nil, "", fmt.Errorf("job namespace is not configured")
	}
	pvc, err := clientset.CoreV1().PersistentVolumeClaims(pvcNamespace).
		Get(ctx, sharedStoragePVCName(), metav1.GetOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("get shared storage PVC %s/%s: %w", pvcNamespace, sharedStoragePVCName(), err)
	}
	pvName := strings.TrimSpace(pvc.Spec.VolumeName)
	if pvName == "" {
		return nil, "", fmt.Errorf("shared storage PVC is not bound")
	}
	pv, err := clientset.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("get shared storage PV %s: %w", pvName, err)
	}
	expectedDriver := StorageQuotaCephFSCSIDriver()
	if pv.Spec.CSI == nil || pv.Spec.CSI.Driver != expectedDriver {
		return nil, "", fmt.Errorf(
			"shared storage PV %s does not use configured CephFS CSI driver %q",
			pvName,
			expectedDriver,
		)
	}

	fsName := strings.TrimSpace(pv.Spec.CSI.VolumeAttributes["fsName"])
	if fsName == "" {
		fsName = StorageQuotaCephFSName()
	}
	if err := ensureToolboxCephFSMounted(ctx, clientset, config, pod, fsName); err != nil {
		return nil, "", err
	}

	if subvolumePath := strings.TrimSpace(pv.Spec.CSI.VolumeAttributes["subvolumePath"]); subvolumePath != "" {
		return pod, path.Join(toolboxMountRoot, strings.TrimLeft(subvolumePath, "/")), nil
	}

	volumeUUID := volumeHandleUUIDPattern.FindString(pv.Spec.CSI.VolumeHandle)
	if volumeUUID == "" {
		return nil, "", fmt.Errorf("cannot resolve CephFS subvolume path from PV %s", pvName)
	}
	volumeRoot := path.Join(toolboxMountRoot, "volumes/csi/csi-vol-"+volumeUUID)
	script := fmt.Sprintf(
		"find %s -mindepth 1 -maxdepth 1 -type d -print -quit",
		shellQuote(volumeRoot),
	)
	output, err := execToolbox(ctx, clientset, config, pod, []string{"sh", "-c", script})
	if err != nil {
		return nil, "", fmt.Errorf("resolve CephFS subvolume directory: %w", err)
	}
	storageRoot := strings.TrimSpace(output)
	if storageRoot == "" {
		return nil, "", fmt.Errorf("CephFS subvolume directory is empty under %s", volumeRoot)
	}
	return pod, storageRoot, nil
}

func findToolboxPod(
	ctx context.Context,
	clientset kubernetes.Interface,
	namespace, labelSelector string,
) (*corev1.Pod, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("list toolbox pods in %s with selector %q: %w", namespace, labelSelector, err)
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		return pod, nil
	}
	return nil, fmt.Errorf(
		"running Rook Ceph toolbox pod matching %q was not found in %s",
		labelSelector,
		namespace,
	)
}

func ensureToolboxCephFSMounted(
	ctx context.Context,
	clientset kubernetes.Interface,
	config *rest.Config,
	pod *corev1.Pod,
	fsName string,
) error {
	script := fmt.Sprintf(`
set -eu
mount_root=%s
fs_name=%s
mount_state=/run/crater-cephfs.mount-identity
cluster_fsid=$(ceph fsid)
expected_identity="${cluster_fsid}:${fs_name}"
if grep -qs " ${mount_root} " /proc/mounts; then
  mount_signature=$(awk -v mount_root="${mount_root}" '$5 == mount_root { print $1 ":" $3; exit }' /proc/self/mountinfo)
  mounted_identity=$(cat "${mount_state}" 2>/dev/null || true)
  if [ -n "${mount_signature}" ] && [ "${mounted_identity}" = "${expected_identity}:${mount_signature}" ]; then
    exit 0
  fi
  echo "refusing to reuse an unverified CephFS mount at ${mount_root}" >&2
  exit 1
fi
rm -f "${mount_state}"
mkdir -p "${mount_root}"
if command -v ceph-fuse >/dev/null 2>&1; then
  ceph-fuse --client_fs "${fs_name}" "${mount_root}"
else
  mount -t ceph :/ "${mount_root}" -o name=admin,fs="${fs_name}"
fi
grep -qs " ${mount_root} " /proc/mounts
mount_signature=$(awk -v mount_root="${mount_root}" '$5 == mount_root { print $1 ":" $3; exit }' /proc/self/mountinfo)
test -n "${mount_signature}"
state_tmp="${mount_state}.$$"
printf '%%s\n' "${expected_identity}:${mount_signature}" > "${state_tmp}"
mv "${state_tmp}" "${mount_state}"
`, shellQuote(toolboxMountRoot), shellQuote(fsName))
	if _, err := execToolbox(ctx, clientset, config, pod, []string{"sh", "-c", script}); err != nil {
		return fmt.Errorf("mount CephFS in toolbox: %w", err)
	}
	return nil
}

func readToolboxXattr(
	ctx context.Context,
	clientset kubernetes.Interface,
	config *rest.Config,
	pod *corev1.Pod,
	targetPath, attribute string,
) (string, error) {
	script := fmt.Sprintf(
		"getfattr --only-values -n %s -- %s",
		attribute,
		shellQuote(targetPath),
	)
	output, err := execToolbox(ctx, clientset, config, pod, []string{"sh", "-c", script})
	if err != nil {
		if isToolboxXattrNotFound(err) {
			return "", fmt.Errorf("%w: %s", errToolboxXattrNotFound, attribute)
		}
		return "", fmt.Errorf("read %s through toolbox: %w", attribute, err)
	}
	return strings.TrimSpace(output), nil
}

func isToolboxXattrNotFound(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such attribute") ||
		strings.Contains(message, "no data available") ||
		strings.Contains(message, "enodata")
}

func resolveToolboxTargetPath(
	ctx context.Context,
	clientset kubernetes.Interface,
	config *rest.Config,
	pod *corev1.Pod,
	storageRoot, targetPath string,
) (string, error) {
	script := fmt.Sprintf(
		"realpath -e -- %s; realpath -e -- %s",
		shellQuote(storageRoot),
		shellQuote(targetPath),
	)
	output, err := execToolbox(ctx, clientset, config, pod, []string{"sh", "-c", script})
	if err != nil {
		return "", fmt.Errorf("resolve toolbox storage path: %w", err)
	}
	resolvedPaths := strings.Split(strings.TrimSpace(output), "\n")
	if len(resolvedPaths) != 2 {
		return "", fmt.Errorf("resolve toolbox storage path returned %d paths", len(resolvedPaths))
	}
	return validateToolboxResolvedPath(resolvedPaths[0], resolvedPaths[1])
}

func validateToolboxResolvedPath(storageRoot, targetPath string) (string, error) {
	root := path.Clean(strings.TrimSpace(storageRoot))
	target := path.Clean(strings.TrimSpace(targetPath))
	if root == "." || !path.IsAbs(root) || !path.IsAbs(target) {
		return "", fmt.Errorf("toolbox storage paths must be absolute")
	}
	if target != root && !strings.HasPrefix(target, root+"/") {
		return "", fmt.Errorf("resolved path escapes toolbox storage root: %s", targetPath)
	}
	return target, nil
}

func toolboxStoragePath(storageRoot, relativePath string) (string, error) {
	root := path.Clean(storageRoot)
	relative := path.Clean(strings.Trim(strings.ReplaceAll(relativePath, "\\", "/"), "/"))
	if relative == "." || relative == "" {
		return root, nil
	}
	if relative == ".." || strings.HasPrefix(relative, "../") {
		return "", fmt.Errorf("path escapes toolbox storage root: %s", relativePath)
	}
	target := path.Join(root, relative)
	if target != root && !strings.HasPrefix(target, root+"/") {
		return "", fmt.Errorf("path escapes toolbox storage root: %s", relativePath)
	}
	return target, nil
}

func execToolbox(
	ctx context.Context,
	clientset kubernetes.Interface,
	config *rest.Config,
	pod *corev1.Pod,
	command []string,
) (string, error) {
	var stdout, stderr strings.Builder
	request := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return "", fmt.Errorf("register Kubernetes core scheme: %w", err)
	}
	request.VersionedParams(&corev1.PodExecOptions{
		Command: command,
		Stdout:  true,
		Stderr:  true,
	}, runtime.NewParameterCodec(scheme))

	executor, err := remotecommand.NewSPDYExecutor(config, http.MethodPost, request.URL())
	if err != nil {
		return "", fmt.Errorf("create toolbox executor: %w", err)
	}
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
