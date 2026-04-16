package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/fs"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	kbatchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
)

const (
	agentOpsAuditMetaKey = "_audit"

	defaultOpsWebSearchTimeoutSeconds = 10
	defaultOpsScriptTimeoutSeconds    = 300
	defaultOpsScriptMaxTimeoutSeconds = 1800
	defaultSandboxGrepMaxMatches      = 50
	defaultSandboxGrepRootPath        = "/"
)

var (
	sandboxGrepAllowedRoots = map[string]struct{}{
		"runbooks":          {},
		"collected-bundles": {},
		"mounted-config":    {},
	}

	distributedNetworkKeywords = []string{
		"nccl", "rdma", "ibv", "ib", "network", "cni", "multus",
		"connection", "timeout", "unreachable", "broken pipe",
	}
)

func ensureOpsAdmin(token util.JWTMessage, toolName string) error {
	if token.RolePlatform != model.RoleAdmin {
		return fmt.Errorf("%s requires admin privileges", toolName)
	}
	return nil
}

func normalizeAllowedDomains(domains []string) []string {
	result := make([]string, 0, len(domains))
	seen := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		normalized := strings.ToLower(strings.TrimSpace(domain))
		normalized = strings.TrimPrefix(normalized, "https://")
		normalized = strings.TrimPrefix(normalized, "http://")
		normalized = strings.TrimSuffix(normalized, "/")
		if idx := strings.Index(normalized, "/"); idx >= 0 {
			normalized = normalized[:idx]
		}
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	sort.Strings(result)
	return result
}

func isAllowedDomain(host string, allowedDomains []string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	for _, domain := range allowedDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func extractRunOpsScriptName(rawArgs json.RawMessage) string {
	var args struct {
		ScriptName string `json:"script_name"`
	}
	_ = json.Unmarshal(rawArgs, &args)
	return strings.TrimSpace(args.ScriptName)
}

func sanitizeSandboxRelativePath(rawPath string) (string, string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		trimmed = "runbooks/"
	}
	if strings.Contains(trimmed, "\x00") {
		return "", "", fmt.Errorf("invalid path")
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	if strings.HasPrefix(trimmed, "/") {
		return "", "", fmt.Errorf("absolute path is not allowed")
	}
	cleaned := filepath.Clean(trimmed)
	cleaned = strings.ReplaceAll(cleaned, "\\", "/")
	if cleaned == "." || cleaned == "" {
		return "", "", fmt.Errorf("path is required")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", "", fmt.Errorf("path traversal is not allowed")
	}
	root := cleaned
	if idx := strings.Index(cleaned, "/"); idx >= 0 {
		root = cleaned[:idx]
	}
	if _, ok := sandboxGrepAllowedRoots[root]; !ok {
		return "", "", fmt.Errorf("path must start with one of: runbooks/, collected-bundles/, mounted-config/")
	}
	return cleaned, root, nil
}

func isPathWithinRoot(path, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if path == root {
		return true
	}
	return strings.HasPrefix(path, root+string(os.PathSeparator))
}

func buildAuditMetadata(executionBackend, sandboxJobName, scriptName, resultArtifactRef string, egressDomains []string) map[string]any {
	meta := map[string]any{
		"execution_backend": executionBackend,
	}
	if sandboxJobName != "" {
		meta["sandbox_job_name"] = sandboxJobName
	}
	if scriptName != "" {
		meta["script_name"] = scriptName
	}
	if resultArtifactRef != "" {
		meta["result_artifact_ref"] = resultArtifactRef
	}
	if len(egressDomains) > 0 {
		meta["egress_domains"] = uniqueStrings(egressDomains)
	}
	return meta
}

func safeEventTimestamp(event corev1.Event) time.Time {
	if !event.EventTime.IsZero() {
		return event.EventTime.Time
	}
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.FirstTimestamp.IsZero() {
		return event.FirstTimestamp.Time
	}
	return event.CreationTimestamp.Time
}

func firstContainerImage(pod corev1.Pod) string {
	if len(pod.Spec.Containers) == 0 {
		return ""
	}
	return pod.Spec.Containers[0].Image
}

func matchAnyKeyword(line string, keywords []string) bool {
	lower := strings.ToLower(line)
	for _, keyword := range keywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func sanitizeForK8sName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "script"
	}
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if r == '-' || r == '_' || r == '.' {
			b.WriteRune('-')
		}
	}
	cleaned := strings.Trim(b.String(), "-")
	if cleaned == "" {
		return "script"
	}
	if len(cleaned) > 40 {
		return cleaned[:40]
	}
	return cleaned
}

func (mgr *AgentMgr) toolListStoragePVCs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolListStoragePVCs); err != nil {
		return nil, err
	}
	var args struct {
		Namespace     string `json:"namespace"`
		Status        string `json:"status"`
		LabelSelector string `json:"label_selector"`
		Limit         int64  `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	ns := strings.TrimSpace(args.Namespace)
	switch strings.ToLower(ns) {
	case "", "*", "all":
		ns = ""
	case "job":
		ns = pkgconfig.GetConfig().Namespaces.Job
	}
	opts := metav1.ListOptions{LabelSelector: strings.TrimSpace(args.LabelSelector)}
	if args.Limit > 0 {
		opts.Limit = args.Limit
	}
	statusFilter := strings.ToLower(strings.TrimSpace(args.Status))
	pvcList, err := mgr.kubeClient.CoreV1().PersistentVolumeClaims(ns).List(c, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pvc: %w", err)
	}
	items := make([]map[string]any, 0, len(pvcList.Items))
	for i := range pvcList.Items {
		pvc := pvcList.Items[i]
		phase := strings.TrimSpace(string(pvc.Status.Phase))
		if statusFilter != "" && strings.ToLower(phase) != statusFilter {
			continue
		}
		storageClass := ""
		if pvc.Spec.StorageClassName != nil {
			storageClass = strings.TrimSpace(*pvc.Spec.StorageClassName)
		}
		requested := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		capacity := pvc.Status.Capacity[corev1.ResourceStorage]
		items = append(items, map[string]any{
			"name":          pvc.Name,
			"namespace":     pvc.Namespace,
			"status":        phase,
			"phase":         phase,
			"storage_class": storageClass,
			"access_modes":  pvc.Spec.AccessModes,
			"volume_name":   pvc.Spec.VolumeName,
			"requested":     requested.String(),
			"capacity":      capacity.String(),
			"age":           time.Since(pvc.CreationTimestamp.Time).Round(time.Minute).String(),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		left := fmt.Sprintf("%s/%s", items[i]["namespace"], items[i]["name"])
		right := fmt.Sprintf("%s/%s", items[j]["namespace"], items[j]["name"])
		return left < right
	})
	return map[string]any{
		"items":              items,
		"pvcs":               items,
		"total":              len(items),
		"namespace":          ns,
		"status":             args.Status,
		agentOpsAuditMetaKey: buildAuditMetadata("backend_k8s_read", "", "", "", nil),
	}, nil
}

func (mgr *AgentMgr) toolGetPVCDetail(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolGetPVCDetail); err != nil {
		return nil, err
	}
	var args struct {
		Namespace string `json:"namespace"`
		PVCName   string `json:"pvc_name"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	pvcName := strings.TrimSpace(args.PVCName)
	if pvcName == "" {
		return nil, fmt.Errorf("pvc_name is required")
	}
	namespace := strings.TrimSpace(args.Namespace)
	if namespace == "" {
		namespace = pkgconfig.GetConfig().Namespaces.Job
	}
	pvc, err := mgr.kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(c, pvcName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pvc %s/%s: %w", namespace, pvcName, err)
	}
	storageClass := ""
	if pvc.Spec.StorageClassName != nil {
		storageClass = strings.TrimSpace(*pvc.Spec.StorageClassName)
	}
	requested := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	capacity := pvc.Status.Capacity[corev1.ResourceStorage]
	return map[string]any{
		"name":               pvc.Name,
		"namespace":          pvc.Namespace,
		"status":             string(pvc.Status.Phase),
		"storage_class":      storageClass,
		"access_modes":       pvc.Spec.AccessModes,
		"volume_name":        pvc.Spec.VolumeName,
		"requested":          requested.String(),
		"capacity":           capacity.String(),
		"conditions":         pvc.Status.Conditions,
		"annotations":        pvc.Annotations,
		"labels":             pvc.Labels,
		"creation_timestamp": pvc.CreationTimestamp,
		"resource_version":   pvc.ResourceVersion,
		"finalizers":         pvc.Finalizers,
		"deletion_timestamp": pvc.DeletionTimestamp,
		agentOpsAuditMetaKey: buildAuditMetadata("backend_k8s_read", "", "", "", nil),
	}, nil
}

func (mgr *AgentMgr) toolGetPVCEvents(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolGetPVCEvents); err != nil {
		return nil, err
	}
	var args struct {
		Namespace string `json:"namespace"`
		PVCName   string `json:"pvc_name"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	pvcName := strings.TrimSpace(args.PVCName)
	if pvcName == "" {
		return nil, fmt.Errorf("pvc_name is required")
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 30
	}
	if limit > 200 {
		limit = 200
	}
	namespace := strings.TrimSpace(args.Namespace)
	if namespace == "" {
		namespace = pkgconfig.GetConfig().Namespaces.Job
	}
	fieldSelector := fmt.Sprintf("involvedObject.kind=PersistentVolumeClaim,involvedObject.name=%s", pvcName)
	events, err := mgr.kubeClient.CoreV1().Events(namespace).List(c, metav1.ListOptions{FieldSelector: fieldSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to list pvc events: %w", err)
	}
	sort.Slice(events.Items, func(i, j int) bool {
		return safeEventTimestamp(events.Items[i]).After(safeEventTimestamp(events.Items[j]))
	})
	items := make([]map[string]any, 0, min(limit, len(events.Items)))
	for i := 0; i < len(events.Items) && i < limit; i++ {
		event := events.Items[i]
		items = append(items, map[string]any{
			"type":      event.Type,
			"reason":    event.Reason,
			"message":   event.Message,
			"count":     event.Count,
			"source":    event.Source.Component,
			"timestamp": safeEventTimestamp(event),
		})
	}
	return map[string]any{
		"namespace":          namespace,
		"pvc_name":           pvcName,
		"items":              items,
		"total":              len(items),
		agentOpsAuditMetaKey: buildAuditMetadata("backend_k8s_read", "", "", "", nil),
	}, nil
}

func (mgr *AgentMgr) toolInspectJobStorage(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolInspectJobStorage); err != nil {
		return nil, err
	}
	var args struct {
		JobName string `json:"job_name"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	jobName := strings.TrimSpace(args.JobName)
	if jobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}
	job, err := mgr.findScopedJob(c, token, jobName)
	if err != nil {
		return nil, err
	}
	if job.Attributes.Data() == nil {
		return nil, fmt.Errorf("job spec is not available")
	}

	namespace := job.Attributes.Data().Namespace
	if namespace == "" {
		namespace = pkgconfig.GetConfig().Namespaces.Job
	}
	volumes := make([]map[string]any, 0)
	volumeSeen := make(map[string]struct{})
	taskMounts := make([]map[string]any, 0)
	pvcClaims := make(map[string]struct{})
	for _, task := range job.Attributes.Data().Spec.Tasks {
		for _, volume := range task.Template.Spec.Volumes {
			kind := "other"
			detail := map[string]any{}
			switch {
			case volume.PersistentVolumeClaim != nil:
				kind = "persistent_volume_claim"
				claim := strings.TrimSpace(volume.PersistentVolumeClaim.ClaimName)
				detail["claim_name"] = claim
				detail["read_only"] = volume.PersistentVolumeClaim.ReadOnly
				if claim != "" {
					pvcClaims[claim] = struct{}{}
				}
			case volume.EmptyDir != nil:
				kind = "empty_dir"
				detail["medium"] = volume.EmptyDir.Medium
			case volume.HostPath != nil:
				kind = "host_path"
				detail["path"] = volume.HostPath.Path
			}
			key := fmt.Sprintf("%s/%s", task.Name, volume.Name)
			if _, ok := volumeSeen[key]; !ok {
				volumeSeen[key] = struct{}{}
				volumes = append(volumes, map[string]any{
					"task":   task.Name,
					"name":   volume.Name,
					"kind":   kind,
					"detail": detail,
				})
			}
		}
		for _, container := range task.Template.Spec.Containers {
			for _, mount := range container.VolumeMounts {
				taskMounts = append(taskMounts, map[string]any{
					"task":              task.Name,
					"container":         container.Name,
					"volume_name":       mount.Name,
					"mount_path":        mount.MountPath,
					"sub_path":          mount.SubPath,
					"read_only":         mount.ReadOnly,
					"mount_propagation": mount.MountPropagation,
				})
			}
		}
	}

	pvcStates := make([]map[string]any, 0, len(pvcClaims))
	for claim := range pvcClaims {
		pvc, pvcErr := mgr.kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(c, claim, metav1.GetOptions{})
		if pvcErr != nil {
			if k8serrors.IsNotFound(pvcErr) {
				pvcStates = append(pvcStates, map[string]any{
					"name":    claim,
					"status":  "NotFound",
					"message": "PVC not found in namespace",
				})
				continue
			}
			pvcStates = append(pvcStates, map[string]any{
				"name":    claim,
				"status":  "Error",
				"message": pvcErr.Error(),
			})
			continue
		}
		requested := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		pvcStates = append(pvcStates, map[string]any{
			"name":          pvc.Name,
			"status":        string(pvc.Status.Phase),
			"storage_class": pvc.Spec.StorageClassName,
			"volume_name":   pvc.Spec.VolumeName,
			"requested":     requested.String(),
		})
	}
	sort.Slice(pvcStates, func(i, j int) bool {
		left := fmt.Sprintf("%v", pvcStates[i]["name"])
		right := fmt.Sprintf("%v", pvcStates[j]["name"])
		return left < right
	})

	baseURL := job.JobName
	if job.Attributes.Data().Labels != nil {
		if v := strings.TrimSpace(job.Attributes.Data().Labels[crclient.LabelKeyBaseURL]); v != "" {
			baseURL = v
		}
	}
	pods, err := mgr.kubeClient.CoreV1().Pods(namespace).List(c, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, baseURL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods for job %s: %w", jobName, err)
	}
	podBindings := make([]map[string]any, 0, len(pods.Items))
	for _, pod := range pods.Items {
		podBindings = append(podBindings, map[string]any{
			"name":       pod.Name,
			"phase":      pod.Status.Phase,
			"node_name":  pod.Spec.NodeName,
			"pod_ip":     pod.Status.PodIP,
			"host_ip":    pod.Status.HostIP,
			"start_time": pod.Status.StartTime,
		})
	}

	return map[string]any{
		"job_name":           jobName,
		"namespace":          namespace,
		"volumes":            volumes,
		"mounts":             taskMounts,
		"pvc_states":         pvcStates,
		"pod_bindings":       podBindings,
		"total_volumes":      len(volumes),
		agentOpsAuditMetaKey: buildAuditMetadata("backend_k8s_read", "", "", "", nil),
	}, nil
}

func (mgr *AgentMgr) toolGetStorageCapacityOverview(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolStorageCapacity); err != nil {
		return nil, err
	}
	var args struct {
		Namespace string `json:"namespace"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	namespace := strings.TrimSpace(args.Namespace)
	switch strings.ToLower(namespace) {
	case "", "all", "*":
		namespace = ""
	case "job":
		namespace = pkgconfig.GetConfig().Namespaces.Job
	}
	pvcs, err := mgr.kubeClient.CoreV1().PersistentVolumeClaims(namespace).List(c, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pvc: %w", err)
	}

	byStatus := map[string]int{}
	byStorageClass := map[string]map[string]any{}
	totalRequested := int64(0)
	totalCapacity := int64(0)
	topPVCs := make([]map[string]any, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		status := string(pvc.Status.Phase)
		if status == "" {
			status = "Unknown"
		}
		byStatus[status]++

		storageClass := "<none>"
		if pvc.Spec.StorageClassName != nil && strings.TrimSpace(*pvc.Spec.StorageClassName) != "" {
			storageClass = strings.TrimSpace(*pvc.Spec.StorageClassName)
		}
		entry, ok := byStorageClass[storageClass]
		if !ok {
			entry = map[string]any{"count": 0, "requested_bytes": int64(0), "capacity_bytes": int64(0)}
			byStorageClass[storageClass] = entry
		}
		entry["count"] = entry["count"].(int) + 1

		requested := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		requestedBytes := requested.Value()
		capacity := pvc.Status.Capacity[corev1.ResourceStorage]
		capacityBytes := capacity.Value()
		totalRequested += requestedBytes
		totalCapacity += capacityBytes
		entry["requested_bytes"] = entry["requested_bytes"].(int64) + requestedBytes
		entry["capacity_bytes"] = entry["capacity_bytes"].(int64) + capacityBytes

		topPVCs = append(topPVCs, map[string]any{
			"namespace":       pvc.Namespace,
			"name":            pvc.Name,
			"status":          status,
			"storage_class":   storageClass,
			"requested":       requested.String(),
			"requested_bytes": requestedBytes,
			"capacity":        capacity.String(),
			"capacity_bytes":  capacityBytes,
		})
	}
	sort.Slice(topPVCs, func(i, j int) bool {
		return topPVCs[i]["requested_bytes"].(int64) > topPVCs[j]["requested_bytes"].(int64)
	})
	if len(topPVCs) > 20 {
		topPVCs = topPVCs[:20]
	}

	totalClasses := len(byStorageClass)
	highPressureClasses := 0
	for _, entry := range byStorageClass {
		requestedBytes, _ := entry["requested_bytes"].(int64)
		capacityBytes, _ := entry["capacity_bytes"].(int64)
		if capacityBytes > 0 && float64(requestedBytes)/float64(capacityBytes) >= 0.9 {
			highPressureClasses++
		}
	}
	summary := map[string]any{
		"total_pvc":              len(pvcs.Items),
		"total_requested_bytes":  totalRequested,
		"total_capacity_bytes":   totalCapacity,
		"total_requested_gib":    float64(totalRequested) / float64(1<<30),
		"total_capacity_gib":     float64(totalCapacity) / float64(1<<30),
		"high_pressure_clusters": highPressureClasses,
		"total_clusters":         totalClasses,
		"pending_pvcs":           byStatus[string(corev1.ClaimPending)],
	}

	return map[string]any{
		"namespace":             namespace,
		"total_pvc":             summary["total_pvc"],
		"total_requested_bytes": totalRequested,
		"total_capacity_bytes":  totalCapacity,
		"total_requested_gib":   summary["total_requested_gib"],
		"total_capacity_gib":    summary["total_capacity_gib"],
		"by_status":             byStatus,
		"by_storage_class":      byStorageClass,
		"top_requested_pvcs":    topPVCs,
		"summary":               summary,
		agentOpsAuditMetaKey:    buildAuditMetadata("backend_k8s_read", "", "", "", nil),
	}, nil
}

func nodeConditionStatus(node *corev1.Node, conditionType corev1.NodeConditionType) corev1.ConditionStatus {
	if node == nil {
		return corev1.ConditionUnknown
	}
	for _, condition := range node.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return corev1.ConditionUnknown
}

func (mgr *AgentMgr) toolGetNodeNetworkSummary(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolNodeNetwork); err != nil {
		return nil, err
	}
	var args struct {
		NodeName         string `json:"node_name"`
		IncludeAddresses bool   `json:"include_addresses"`
		Limit            int    `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}

	nodes := make([]corev1.Node, 0)
	nodeName := strings.TrimSpace(args.NodeName)
	if nodeName != "" {
		node, err := mgr.kubeClient.CoreV1().Nodes().Get(c, nodeName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get node %s: %w", nodeName, err)
		}
		nodes = append(nodes, *node)
	} else {
		nodeList, err := mgr.kubeClient.CoreV1().Nodes().List(c, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list nodes: %w", err)
		}
		nodes = nodeList.Items
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Name < nodes[j].Name
		})
		if args.Limit > 0 && args.Limit < len(nodes) {
			nodes = nodes[:args.Limit]
		}
	}

	summaryItems := make([]map[string]any, 0, len(nodes))
	degradedCount := 0
	readyCount := 0
	networkAlertCount := 0
	networkUnavailableCount := 0
	for _, node := range nodes {
		ready := nodeConditionStatus(&node, corev1.NodeReady) == corev1.ConditionTrue
		networkUnavailable := nodeConditionStatus(&node, corev1.NodeNetworkUnavailable) == corev1.ConditionTrue
		status := "healthy"
		alerts := make([]map[string]string, 0, 2)
		if !ready {
			status = "degraded"
			alerts = append(alerts, map[string]string{
				"severity": "critical",
				"message":  "NodeReady is not True",
			})
		}
		if networkUnavailable {
			status = "degraded"
			alerts = append(alerts, map[string]string{
				"severity": "warning",
				"message":  "NodeNetworkUnavailable is True",
			})
		}
		if node.Spec.Unschedulable && status == "healthy" {
			status = "warning"
		}
		if ready {
			readyCount++
		}
		if networkUnavailable {
			networkUnavailableCount++
		}
		if status != "healthy" {
			degradedCount++
		}
		networkAlertCount += len(alerts)
		item := map[string]any{
			"name":                 node.Name,
			"ready":                ready,
			"network_unavailable":  networkUnavailable,
			"unschedulable":        node.Spec.Unschedulable,
			"status":               status,
			"alerts":               alerts,
			"kubelet_version":      node.Status.NodeInfo.KubeletVersion,
			"kernel_version":       node.Status.NodeInfo.KernelVersion,
			"os_image":             node.Status.NodeInfo.OSImage,
			"container_runtime":    node.Status.NodeInfo.ContainerRuntimeVersion,
			"allocatable":          node.Status.Allocatable,
			"capacity":             node.Status.Capacity,
			"last_transition_time": node.CreationTimestamp,
		}
		if args.IncludeAddresses {
			addresses := make([]map[string]string, 0, len(node.Status.Addresses))
			for _, addr := range node.Status.Addresses {
				addresses = append(addresses, map[string]string{
					"type":    string(addr.Type),
					"address": addr.Address,
				})
			}
			item["addresses"] = addresses
		}
		summaryItems = append(summaryItems, item)
	}
	sort.Slice(summaryItems, func(i, j int) bool {
		return fmt.Sprintf("%v", summaryItems[i]["name"]) < fmt.Sprintf("%v", summaryItems[j]["name"])
	})

	return map[string]any{
		"total_nodes":               len(summaryItems),
		"ready_nodes":               readyCount,
		"degraded_nodes":            degradedCount,
		"network_unavailable_nodes": networkUnavailableCount,
		"network_alerts":            networkAlertCount,
		"items":                     summaryItems,
		"nodes":                     summaryItems,
		agentOpsAuditMetaKey:        buildAuditMetadata("backend_k8s_read", "", "", "", nil),
	}, nil
}

func matchesDistributedNetworkEvidence(text string, keywordFilter *regexp.Regexp) bool {
	if keywordFilter != nil && keywordFilter.MatchString(text) {
		return true
	}
	return matchAnyKeyword(text, distributedNetworkKeywords)
}

func totalGPURequests(resources corev1.ResourceList) int64 {
	total := int64(0)
	for name, quantity := range resources {
		resourceName := strings.ToLower(string(name))
		if strings.Contains(resourceName, "gpu") {
			total += quantity.Value()
		}
	}
	return total
}

func isLikelyDistributedJob(job *model.Job) bool {
	if job == nil {
		return false
	}
	switch job.JobType {
	case model.JobTypeDeepSpeed, model.JobTypeOpenMPI, model.JobTypeKubeRay:
		return true
	}
	if len(job.Nodes.Data()) > 1 {
		return true
	}
	if totalGPURequests(job.Resources.Data()) > 1 {
		return true
	}
	if job.Attributes.Data() != nil {
		if len(job.Attributes.Data().Spec.Tasks) > 1 {
			return true
		}
		replicas := int32(0)
		for _, task := range job.Attributes.Data().Spec.Tasks {
			if task.Replicas > 0 {
				replicas += task.Replicas
				continue
			}
			replicas++
		}
		if replicas > 1 {
			return true
		}
	}
	lowerName := strings.ToLower(strings.TrimSpace(job.Name + " " + job.JobName))
	return strings.Contains(lowerName, "dist") ||
		strings.Contains(lowerName, "mpi") ||
		strings.Contains(lowerName, "nccl") ||
		strings.Contains(lowerName, "rdma")
}

func (mgr *AgentMgr) diagnoseDistributedJobNetworkForJob(
	c *gin.Context,
	token util.JWTMessage,
	jobName string,
	tailLines int64,
	maxLogMatches int,
	keyword string,
) (map[string]any, error) {
	job, err := mgr.findScopedJob(c, token, jobName)
	if err != nil {
		return nil, err
	}
	if job.Attributes.Data() == nil {
		return nil, fmt.Errorf("job spec is unavailable")
	}
	namespace := job.Attributes.Data().Namespace
	if namespace == "" {
		namespace = pkgconfig.GetConfig().Namespaces.Job
	}

	baseURL := job.JobName
	if job.Attributes.Data().Labels != nil {
		if v := strings.TrimSpace(job.Attributes.Data().Labels[crclient.LabelKeyBaseURL]); v != "" {
			baseURL = v
		}
	}
	pods, err := mgr.kubeClient.CoreV1().Pods(namespace).List(c, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, baseURL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list job pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pods found for job %s", jobName)
	}

	var keywordFilter *regexp.Regexp
	if strings.TrimSpace(keyword) != "" {
		compiled, compileErr := regexp.Compile(keyword)
		if compileErr != nil {
			return nil, fmt.Errorf("invalid keyword regex: %w", compileErr)
		}
		keywordFilter = compiled
	}

	podSummary := make([]map[string]any, 0, len(pods.Items))
	nodeSet := make(map[string]struct{})
	eventMatches := make([]map[string]any, 0)
	logMatches := make([]map[string]any, 0)

	for _, pod := range pods.Items {
		if pod.Spec.NodeName != "" {
			nodeSet[pod.Spec.NodeName] = struct{}{}
		}
		restarts := int32(0)
		for _, cs := range pod.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}
		podSummary = append(podSummary, map[string]any{
			"name":            pod.Name,
			"node_name":       pod.Spec.NodeName,
			"phase":           pod.Status.Phase,
			"pod_ip":          pod.Status.PodIP,
			"host_ip":         pod.Status.HostIP,
			"restart_count":   restarts,
			"ready":           isPodReady(pod),
			"container_image": firstContainerImage(pod),
		})

		podEvents, eventErr := mgr.kubeClient.CoreV1().Events(namespace).List(c, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.kind=Pod,involvedObject.name=%s", pod.Name),
		})
		if eventErr == nil {
			sort.Slice(podEvents.Items, func(i, j int) bool {
				return safeEventTimestamp(podEvents.Items[i]).After(safeEventTimestamp(podEvents.Items[j]))
			})
			for _, event := range podEvents.Items {
				if !matchesDistributedNetworkEvidence(event.Reason+" "+event.Message, keywordFilter) {
					continue
				}
				eventMatches = append(eventMatches, map[string]any{
					"pod_name":  pod.Name,
					"type":      event.Type,
					"reason":    event.Reason,
					"message":   event.Message,
					"timestamp": safeEventTimestamp(event),
				})
				if len(eventMatches) >= 100 {
					break
				}
			}
		}

		if len(logMatches) >= maxLogMatches {
			continue
		}
		if len(pod.Spec.Containers) == 0 {
			continue
		}
		containerName := pod.Spec.Containers[0].Name
		stream, logErr := mgr.kubeClient.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			Container: containerName,
			TailLines: &tailLines,
		}).Stream(c)
		if logErr != nil {
			continue
		}
		scanner := bufio.NewScanner(stream)
		scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if !matchesDistributedNetworkEvidence(line, keywordFilter) {
				continue
			}
			logMatches = append(logMatches, map[string]any{
				"pod_name":    pod.Name,
				"container":   containerName,
				"log_snippet": line,
			})
			if len(logMatches) >= maxLogMatches {
				break
			}
		}
		_ = stream.Close()
	}

	issues := make([]string, 0)
	if len(nodeSet) > 1 && (len(eventMatches) > 0 || len(logMatches) > 0) {
		issues = append(issues, "Detected cross-node distributed job with network-related events/logs.")
	}
	if len(eventMatches) == 0 && len(logMatches) == 0 {
		issues = append(issues, "No explicit network keyword matches found in recent pod events/log tails.")
	}
	nodeNames := make([]string, 0, len(nodeSet))
	for nodeName := range nodeSet {
		nodeNames = append(nodeNames, nodeName)
	}
	sort.Strings(nodeNames)

	status := "healthy"
	severity := "info"
	if len(eventMatches) > 0 || len(logMatches) > 0 {
		status = "suspected_network_issue"
		severity = "warning"
	}
	if len(nodeNames) > 1 && (len(eventMatches) > 0 || len(logMatches) > 0) {
		severity = "high"
	}

	return map[string]any{
		"job_name":           jobName,
		"namespace":          namespace,
		"status":             status,
		"severity":           severity,
		"distributed_nodes":  nodeNames,
		"pod_summary":        podSummary,
		"event_matches":      eventMatches,
		"log_matches":        logMatches,
		"event_match_count":  len(eventMatches),
		"log_match_count":    len(logMatches),
		"issues":             issues,
		agentOpsAuditMetaKey: buildAuditMetadata("backend_k8s_read", "", "", "", nil),
	}, nil
}

func (mgr *AgentMgr) toolDiagnoseDistributedJobNetwork(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolDiagnoseJobNet); err != nil {
		return nil, err
	}
	var args struct {
		JobName       string `json:"job_name"`
		TailLines     int64  `json:"tail_lines"`
		Tail          int64  `json:"tail"`
		Keyword       string `json:"keyword"`
		MaxLogMatches int    `json:"max_log_matches"`
		Limit         int    `json:"limit"`
		LookbackHours int    `json:"lookback_hours"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	tailLines := args.TailLines
	if tailLines <= 0 {
		tailLines = args.Tail
	}
	if tailLines <= 0 {
		tailLines = 200
	}
	maxLogMatches := args.MaxLogMatches
	if maxLogMatches <= 0 {
		maxLogMatches = 50
	}

	jobName := strings.TrimSpace(args.JobName)
	if jobName != "" {
		return mgr.diagnoseDistributedJobNetworkForJob(
			c,
			token,
			jobName,
			tailLines,
			maxLogMatches,
			args.Keyword,
		)
	}

	lookbackHours := args.LookbackHours
	if lookbackHours <= 0 {
		lookbackHours = 24
	}
	jobLimit := args.Limit
	if jobLimit <= 0 {
		jobLimit = 10
	}
	if jobLimit > 20 {
		jobLimit = 20
	}

	cutoff := time.Now().Add(-time.Duration(lookbackHours) * time.Hour)
	jobs := make([]*model.Job, 0, jobLimit*4)
	err := query.GetDB().WithContext(c.Request.Context()).
		Where("status IN ?", []string{string(batch.Running), string(batch.Pending), string(batch.Failed)}).
		Where("running_timestamp >= ? OR creation_timestamp >= ?", cutoff, cutoff).
		Order("running_timestamp DESC").
		Order("creation_timestamp DESC").
		Limit(jobLimit * 4).
		Find(&jobs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list recent jobs for distributed network diagnosis: %w", err)
	}

	jobSummaries := make([]map[string]any, 0, jobLimit)
	highRiskJobs := 0
	for _, job := range jobs {
		if !isLikelyDistributedJob(job) {
			continue
		}
		diagnosis, diagnoseErr := mgr.diagnoseDistributedJobNetworkForJob(
			c,
			token,
			job.JobName,
			tailLines,
			maxLogMatches,
			args.Keyword,
		)
		if diagnoseErr != nil {
			jobSummaries = append(jobSummaries, map[string]any{
				"job_name": job.JobName,
				"status":   "error",
				"severity": "warning",
				"error":    diagnoseErr.Error(),
			})
		} else {
			jobSummaries = append(jobSummaries, map[string]any{
				"job_name":          diagnosis["job_name"],
				"status":            diagnosis["status"],
				"severity":          diagnosis["severity"],
				"distributed_nodes": diagnosis["distributed_nodes"],
				"issues":            diagnosis["issues"],
				"event_match_count": diagnosis["event_match_count"],
				"log_match_count":   diagnosis["log_match_count"],
			})
		}
		if severity, _ := jobSummaries[len(jobSummaries)-1]["severity"].(string); severity == "high" || severity == "critical" {
			highRiskJobs++
		}
		if len(jobSummaries) >= jobLimit {
			break
		}
	}

	return map[string]any{
		"jobs":                jobSummaries,
		"diagnosed_jobs":      len(jobSummaries),
		"high_risk_jobs":      highRiskJobs,
		"lookback_hours":      lookbackHours,
		"requested_job_limit": jobLimit,
		agentOpsAuditMetaKey:  buildAuditMetadata("backend_k8s_read", "", "", "", nil),
	}, nil
}

func isPodReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func buildDefaultWebSearchURLs(query string, domains []string, maxResults int) []string {
	escaped := url.QueryEscape(query)
	urls := make([]string, 0, len(domains))
	for _, domain := range domains {
		// Use site-native search endpoints as deterministic seeds.
		urls = append(urls, fmt.Sprintf("https://%s/search/?q=%s", domain, escaped))
		if len(urls) >= maxResults {
			break
		}
	}
	return urls
}

func stripHTML(input string) string {
	re := regexp.MustCompile(`(?s)<[^>]*>`)
	text := re.ReplaceAllString(input, " ")
	text = html.UnescapeString(text)
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

func summarizeTextByQuery(text, query string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 320
	}
	if len(text) <= maxChars {
		return text
	}
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	if lowerQuery != "" {
		if idx := strings.Index(lowerText, lowerQuery); idx >= 0 {
			start := idx - maxChars/3
			if start < 0 {
				start = 0
			}
			end := start + maxChars
			if end > len(text) {
				end = len(text)
			}
			return strings.TrimSpace(text[start:end])
		}
	}
	return strings.TrimSpace(text[:maxChars])
}

func extractHTMLTitle(htmlBody string) string {
	re := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	matches := re.FindStringSubmatch(htmlBody)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(stripHTML(matches[1]))
}

func newWebSearchClient(timeoutSeconds int, allowedDomains []string) *http.Client {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultOpsWebSearchTimeoutSeconds * time.Second
	}
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if req.URL == nil {
				return fmt.Errorf("redirect without url")
			}
			if !isAllowedDomain(req.URL.Hostname(), allowedDomains) {
				return fmt.Errorf("redirect target %s is not allowed", req.URL.Hostname())
			}
			return nil
		},
	}
}

func (mgr *AgentMgr) toolWebSearch(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolWebSearch); err != nil {
		return nil, err
	}
	var args struct {
		Query          string   `json:"query"`
		URLs           []string `json:"urls"`
		MaxResults     int      `json:"max_results"`
		Limit          int      `json:"limit"`
		TimeoutSeconds int      `json:"timeout_seconds"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}

	cfg := pkgconfig.GetConfig().Agent.Ops.WebSearch
	if !cfg.Enabled {
		return nil, fmt.Errorf("web_search is disabled by backend configuration")
	}
	allowedDomains := normalizeAllowedDomains(cfg.AllowedDomains)
	if len(allowedDomains) == 0 {
		return nil, fmt.Errorf("web_search allowed domain list is empty")
	}

	query := strings.TrimSpace(args.Query)
	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = args.Limit
	}
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxResults > 10 {
		maxResults = 10
	}
	urls := make([]string, 0, maxResults)
	if len(args.URLs) > 0 {
		for _, rawURL := range args.URLs {
			target := strings.TrimSpace(rawURL)
			if target == "" {
				continue
			}
			u, err := url.Parse(target)
			if err != nil {
				return nil, fmt.Errorf("invalid url %q: %w", target, err)
			}
			if u.Scheme != "https" && u.Scheme != "http" {
				return nil, fmt.Errorf("unsupported url scheme in %q", target)
			}
			if !isAllowedDomain(u.Hostname(), allowedDomains) {
				return nil, fmt.Errorf("url host %q is not in allowed domain list", u.Hostname())
			}
			urls = append(urls, target)
			if len(urls) >= maxResults {
				break
			}
		}
	} else {
		if query == "" {
			return nil, fmt.Errorf("query is required when urls are not provided")
		}
		urls = buildDefaultWebSearchURLs(query, allowedDomains, maxResults)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no search targets generated")
	}

	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultOpsWebSearchTimeoutSeconds
	}
	if args.TimeoutSeconds > 0 && args.TimeoutSeconds < timeoutSeconds {
		timeoutSeconds = args.TimeoutSeconds
	}
	client := newWebSearchClient(timeoutSeconds, allowedDomains)
	results := make([]map[string]any, 0, len(urls))
	egress := make([]string, 0, len(urls))

	for _, target := range urls {
		u, _ := url.Parse(target)
		if u == nil || !isAllowedDomain(u.Hostname(), allowedDomains) {
			return nil, fmt.Errorf("target url %q is not allowed", target)
		}
		egress = append(egress, u.Hostname())

		req, reqErr := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, target, nil)
		if reqErr != nil {
			results = append(results, map[string]any{
				"url":    target,
				"status": "error",
				"error":  reqErr.Error(),
			})
			continue
		}
		req.Header.Set("User-Agent", "crater-agent-ops-web-search/1.0")

		resp, err := client.Do(req)
		if err != nil {
			results = append(results, map[string]any{
				"url":    target,
				"status": "error",
				"error":  err.Error(),
			})
			continue
		}
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
		_ = resp.Body.Close()
		if readErr != nil {
			results = append(results, map[string]any{
				"url":         target,
				"status":      "error",
				"status_code": resp.StatusCode,
				"error":       readErr.Error(),
			})
			continue
		}
		body := string(bodyBytes)
		title := extractHTMLTitle(body)
		snippet := summarizeTextByQuery(stripHTML(body), query, 320)
		results = append(results, map[string]any{
			"url":          target,
			"status":       "ok",
			"status_code":  resp.StatusCode,
			"title":        title,
			"snippet":      snippet,
			"fetched_at":   time.Now(),
			"content_type": resp.Header.Get("Content-Type"),
		})
	}

	summaryParts := make([]string, 0, len(results))
	for _, result := range results {
		if result["status"] != "ok" {
			continue
		}
		title, _ := result["title"].(string)
		snippet, _ := result["snippet"].(string)
		if title != "" {
			summaryParts = append(summaryParts, title)
			continue
		}
		if snippet != "" {
			summaryParts = append(summaryParts, snippet)
		}
	}
	summary := strings.Join(summaryParts, " | ")
	if len(summary) > 600 {
		summary = summary[:600]
	}

	return map[string]any{
		"query":              query,
		"results":            results,
		"summary":            summary,
		agentOpsAuditMetaKey: buildAuditMetadata("backend_web_search_proxy", "", "", "", uniqueStrings(egress)),
	}, nil
}

func (mgr *AgentMgr) toolSandboxGrep(_ *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolSandboxGrep); err != nil {
		return nil, err
	}
	var args struct {
		Path         string `json:"path"`
		TargetPath   string `json:"target_path"`
		Pattern      string `json:"pattern"`
		IgnoreCase   bool   `json:"ignore_case"`
		MaxMatches   int    `json:"max_matches"`
		Limit        int    `json:"limit"`
		ContextLines int    `json:"context_lines"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	pattern := strings.TrimSpace(args.Pattern)
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	maxMatches := args.MaxMatches
	if maxMatches <= 0 {
		maxMatches = args.Limit
	}
	if maxMatches <= 0 {
		maxMatches = defaultSandboxGrepMaxMatches
	}
	if maxMatches > 1000 {
		maxMatches = 1000
	}

	targetPath := strings.TrimSpace(args.Path)
	if targetPath == "" {
		targetPath = strings.TrimSpace(args.TargetPath)
	}
	cleanedPath, rootSegment, err := sanitizeSandboxRelativePath(targetPath)
	if err != nil {
		return nil, err
	}
	rootAbs := filepath.Clean(filepath.Join(defaultSandboxGrepRootPath, rootSegment))
	targetAbs := filepath.Clean(filepath.Join(defaultSandboxGrepRootPath, filepath.FromSlash(cleanedPath)))
	if !isPathWithinRoot(targetAbs, rootAbs) {
		return nil, fmt.Errorf("path escapes allowed root")
	}

	info, err := os.Stat(targetAbs)
	if err != nil {
		return nil, fmt.Errorf("target path is not accessible in sandbox: %w", err)
	}
	regexPattern := pattern
	if args.IgnoreCase {
		regexPattern = "(?i)" + regexPattern
	}
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	matches := make([]map[string]any, 0, min(maxMatches, 200))
	scannedFiles := 0
	truncated := false
	searchFile := func(path string) error {
		realPath, evalErr := filepath.EvalSymlinks(path)
		if evalErr == nil {
			if !isPathWithinRoot(realPath, rootAbs) {
				return nil
			}
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer file.Close()
		scannedFiles++
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()
			if !regex.MatchString(line) {
				continue
			}
			matches = append(matches, map[string]any{
				"file":        strings.TrimPrefix(path, defaultSandboxGrepRootPath),
				"line_number": lineNumber,
				"line":        line,
			})
			if len(matches) >= maxMatches {
				truncated = true
				return io.EOF
			}
		}
		return nil
	}

	if info.IsDir() {
		walkErr := filepath.WalkDir(targetAbs, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if d.Type()&fs.ModeSymlink != 0 {
				realPath, evalErr := filepath.EvalSymlinks(path)
				if evalErr != nil || !isPathWithinRoot(realPath, rootAbs) {
					return nil
				}
			}
			if err := searchFile(path); err != nil {
				if err == io.EOF {
					return io.EOF
				}
			}
			return nil
		})
		if walkErr != nil && walkErr != io.EOF {
			return nil, fmt.Errorf("failed to scan files: %w", walkErr)
		}
	} else {
		if err := searchFile(targetAbs); err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
	}

	return map[string]any{
		"path":          cleanedPath,
		"pattern":       pattern,
		"matches":       matches,
		"total_matches": len(matches),
		"truncated":     truncated,
		"scanned_files": scannedFiles,
		agentOpsAuditMetaKey: buildAuditMetadata(
			"sandbox_grep_restricted",
			"",
			"",
			fmt.Sprintf("sandbox://%s", cleanedPath),
			nil,
		),
	}, nil
}

func normalizeOpsScriptAllowlist(scripts []string) map[string]struct{} {
	result := make(map[string]struct{}, len(scripts))
	for _, script := range scripts {
		s := strings.TrimSpace(script)
		if s == "" {
			continue
		}
		result[s] = struct{}{}
	}
	return result
}

func (mgr *AgentMgr) toolRunOpsScript(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolRunOpsScript); err != nil {
		return nil, err
	}
	var args struct {
		ScriptName     string         `json:"script_name"`
		ScriptArgs     map[string]any `json:"script_args"`
		TimeoutSeconds int            `json:"timeout_seconds"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	args.ScriptName = strings.TrimSpace(args.ScriptName)
	if args.ScriptName == "" {
		return nil, fmt.Errorf("script_name is required")
	}
	allowlist := normalizeOpsScriptAllowlist(pkgconfig.GetConfig().Agent.Ops.Sandbox.ScriptAllowlist)
	if _, ok := allowlist[args.ScriptName]; !ok {
		return nil, fmt.Errorf("script %q is not in sandbox.scriptAllowlist", args.ScriptName)
	}
	if args.ScriptArgs == nil {
		args.ScriptArgs = map[string]any{}
	}

	sandboxCfg := pkgconfig.GetConfig().Agent.Ops.Sandbox
	namespace := strings.TrimSpace(sandboxCfg.Namespace)
	image := strings.TrimSpace(sandboxCfg.Image)
	if namespace == "" {
		return nil, fmt.Errorf("agent.ops.sandbox.namespace is required")
	}
	if image == "" {
		return nil, fmt.Errorf("agent.ops.sandbox.image is required")
	}

	defaultTimeout := sandboxCfg.DefaultTimeoutSeconds
	if defaultTimeout <= 0 {
		defaultTimeout = defaultOpsScriptTimeoutSeconds
	}
	maxTimeout := sandboxCfg.MaxTimeoutSeconds
	if maxTimeout <= 0 {
		maxTimeout = defaultOpsScriptMaxTimeoutSeconds
	}
	timeoutSeconds := args.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultTimeout
	}
	if timeoutSeconds > maxTimeout {
		timeoutSeconds = maxTimeout
	}

	argsJSON, err := json.Marshal(args.ScriptArgs)
	if err != nil {
		return nil, fmt.Errorf("invalid script_args: %w", err)
	}
	backoffLimit := int32(0)
	ttlSeconds := int32(3600)
	activeDeadline := int64(timeoutSeconds + 60)
	scriptPart := sanitizeForK8sName(args.ScriptName)
	jobName := fmt.Sprintf("ops-%s-%d-%d", scriptPart, time.Now().Unix(), rand.Intn(1000))

	job := &kbatchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"crater.raids.io/managed-by": "agent-ops",
				"crater.raids.io/tool-name":  agentToolRunOpsScript,
				"crater.raids.io/script":     scriptPart,
			},
			Annotations: map[string]string{
				"crater.raids.io/ops-script-name": args.ScriptName,
			},
		},
		Spec: kbatchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			ActiveDeadlineSeconds:   &activeDeadline,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"crater.raids.io/managed-by": "agent-ops",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: strings.TrimSpace(sandboxCfg.ServiceAccount),
					Containers: []corev1.Container{
						{
							Name:    "ops-script",
							Image:   image,
							Command: []string{"/ops-tools/run-ops-script"},
							Args: []string{
								"--script", args.ScriptName,
								"--timeout-seconds", fmt.Sprintf("%d", timeoutSeconds),
								"--args-json-env", "CRATER_OPS_SCRIPT_ARGS_JSON",
							},
							Env: []corev1.EnvVar{
								{Name: "CRATER_OPS_SCRIPT_ARGS_JSON", Value: string(argsJSON)},
							},
						},
					},
				},
			},
		},
	}
	if _, err := mgr.kubeClient.BatchV1().Jobs(namespace).Create(c, job, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("failed to create sandbox job: %w", err)
	}
	resultRef := fmt.Sprintf("k8s://%s/job/%s", namespace, jobName)
	return map[string]any{
		"status":              "submitted",
		"script_name":         args.ScriptName,
		"sandbox_job_name":    jobName,
		"namespace":           namespace,
		"timeout_seconds":     timeoutSeconds,
		"result_artifact_ref": resultRef,
		agentOpsAuditMetaKey: buildAuditMetadata(
			"k8s_sandbox_job",
			jobName,
			args.ScriptName,
			resultRef,
			nil,
		),
	}, nil
}

// min is a local helper to keep compatibility with older stdlib targets.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
