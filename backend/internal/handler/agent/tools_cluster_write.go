package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
)

func agentClusterWriteErrorf(format string, args ...any) error {
	return bizerr.Internal.ServiceError.New(fmt.Sprintf(strings.ReplaceAll(format, "%w", "%v"), args...))
}

type nodeLifecycleArgs struct {
	NodeName string `json:"node_name"`
	Reason   string `json:"reason"`
}

type deletePodArgs struct {
	Name               string `json:"name"`
	PodName            string `json:"pod_name"`
	Namespace          string `json:"namespace"`
	Force              bool   `json:"force"`
	GracePeriodSeconds *int64 `json:"grace_period_seconds"`
}

type restartWorkloadArgs struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type scaleWorkloadArgs struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  *int32 `json:"replicas"`
}

type nodeLabelArgs struct {
	NodeName string `json:"node_name"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

type nodeTaintArgs struct {
	NodeName string `json:"node_name"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	Effect   string `json:"effect"`
	Reason   string `json:"reason"`
}

type commandExecutionArgs struct {
	Command string `json:"command"`
	Reason  string `json:"reason"`
}

func defaultJobNamespace(raw string) string {
	if namespace := strings.TrimSpace(raw); namespace != "" {
		return namespace
	}
	return strings.TrimSpace(pkgconfig.GetConfig().Namespaces.Job)
}

func normalizeRestartWorkloadKind(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "deployment", "deploy", "deployments":
		return "deployment", nil
	case "statefulset", "sts", "statefulsets":
		return "statefulset", nil
	case "daemonset", "ds", "daemonsets":
		return "daemonset", nil
	default:
		return "", agentClusterWriteErrorf("unsupported workload kind %q", kind)
	}
}

func normalizeScalableWorkloadKind(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "deployment", "deploy", "deployments":
		return "deployment", nil
	case "statefulset", "sts", "statefulsets":
		return "statefulset", nil
	case "replicaset", "rs", "replicasets":
		return "replicaset", nil
	default:
		return "", agentClusterWriteErrorf("unsupported scalable workload kind %q", kind)
	}
}

func (mgr *AgentMgr) getNodeForLifecycleAction(c *gin.Context, nodeName string) (bool, error) {
	node, err := mgr.kubeClient.CoreV1().Nodes().Get(c, nodeName, metav1.GetOptions{})
	if err != nil {
		return false, agentClusterWriteErrorf("failed to get node %s: %w", nodeName, err)
	}
	return node.Spec.Unschedulable, nil
}

func (mgr *AgentMgr) annotateDrainReason(c *gin.Context, nodeName, reason string) string {
	if strings.TrimSpace(reason) == "" {
		return ""
	}
	node, err := mgr.kubeClient.CoreV1().Nodes().Get(c, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Sprintf("failed to update drain reason annotation: %v", err)
	}
	if node.Annotations == nil {
		node.Annotations = map[string]string{}
	}
	node.Annotations["crater.raids.io/drained-reason"] = strings.TrimSpace(reason)
	if _, err = mgr.kubeClient.CoreV1().Nodes().Update(c, node, metav1.UpdateOptions{}); err != nil {
		return fmt.Sprintf("failed to persist drain reason annotation: %v", err)
	}
	return ""
}

func (mgr *AgentMgr) toolCordonNode(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolCordonNode); err != nil {
		return nil, err
	}
	var args nodeLifecycleArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentClusterWriteErrorf("invalid args: %w", err)
	}
	nodeName := strings.TrimSpace(args.NodeName)
	if nodeName == "" {
		return nil, agentClusterWriteErrorf("node_name is required")
	}
	unschedulable, err := mgr.getNodeForLifecycleAction(c, nodeName)
	if err != nil {
		return nil, err
	}
	if unschedulable {
		return map[string]any{
			"node_name":      nodeName,
			"unschedulable":  true,
			"status":         "noop",
			"requested_tool": agentToolCordonNode,
			"message":        "node is already unschedulable",
		}, nil
	}
	if _, err := mgr.nodeClient.UpdateNodeunschedule(c, nodeName, strings.TrimSpace(args.Reason), token.Username); err != nil {
		return nil, agentClusterWriteErrorf("failed to cordon node %s: %w", nodeName, err)
	}
	return map[string]any{
		"node_name":      nodeName,
		"unschedulable":  true,
		"status":         "cordoned",
		"reason":         strings.TrimSpace(args.Reason),
		"requested_tool": agentToolCordonNode,
	}, nil
}

func (mgr *AgentMgr) toolUncordonNode(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolUncordonNode); err != nil {
		return nil, err
	}
	var args nodeLifecycleArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentClusterWriteErrorf("invalid args: %w", err)
	}
	nodeName := strings.TrimSpace(args.NodeName)
	if nodeName == "" {
		return nil, agentClusterWriteErrorf("node_name is required")
	}
	unschedulable, err := mgr.getNodeForLifecycleAction(c, nodeName)
	if err != nil {
		return nil, err
	}
	if !unschedulable {
		return map[string]any{
			"node_name":      nodeName,
			"unschedulable":  false,
			"status":         "noop",
			"requested_tool": agentToolUncordonNode,
			"message":        "node is already schedulable",
		}, nil
	}
	if _, err := mgr.nodeClient.UpdateNodeunschedule(c, nodeName, strings.TrimSpace(args.Reason), token.Username); err != nil {
		return nil, agentClusterWriteErrorf("failed to uncordon node %s: %w", nodeName, err)
	}
	return map[string]any{
		"node_name":      nodeName,
		"unschedulable":  false,
		"status":         "uncordoned",
		"reason":         strings.TrimSpace(args.Reason),
		"requested_tool": agentToolUncordonNode,
	}, nil
}

func (mgr *AgentMgr) toolDrainNode(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolDrainNode); err != nil {
		return nil, err
	}
	var args nodeLifecycleArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentClusterWriteErrorf("invalid args: %w", err)
	}
	nodeName := strings.TrimSpace(args.NodeName)
	if nodeName == "" {
		return nil, agentClusterWriteErrorf("node_name is required")
	}
	if err := mgr.nodeClient.DrainNode(c, nodeName, token.Username); err != nil {
		return nil, agentClusterWriteErrorf("failed to drain node %s: %w", nodeName, err)
	}
	warning := mgr.annotateDrainReason(c, nodeName, args.Reason)
	return map[string]any{
		"node_name":      nodeName,
		"unschedulable":  true,
		"status":         "draining",
		"reason":         strings.TrimSpace(args.Reason),
		"warning":        warning,
		"requested_tool": agentToolDrainNode,
	}, nil
}

func (mgr *AgentMgr) toolDeletePod(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolDeletePod); err != nil {
		return nil, err
	}
	var args deletePodArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentClusterWriteErrorf("invalid args: %w", err)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		name = strings.TrimSpace(args.PodName)
	}
	if name == "" {
		return nil, agentClusterWriteErrorf("name is required")
	}
	namespace := defaultJobNamespace(args.Namespace)
	if namespace == "" {
		return nil, agentClusterWriteErrorf("namespace is required")
	}
	deleteOpts := metav1.DeleteOptions{}
	if args.Force {
		zero := int64(0)
		deleteOpts.GracePeriodSeconds = &zero
	} else if args.GracePeriodSeconds != nil {
		deleteOpts.GracePeriodSeconds = args.GracePeriodSeconds
	}
	if err := mgr.kubeClient.CoreV1().Pods(namespace).Delete(c, name, deleteOpts); err != nil {
		return nil, agentClusterWriteErrorf("failed to delete pod %s/%s: %w", namespace, name, err)
	}
	return map[string]any{
		"name":                 name,
		"namespace":            namespace,
		"force":                args.Force,
		"grace_period_seconds": deleteOpts.GracePeriodSeconds,
		"status":               "deleted",
		"requested_tool":       agentToolDeletePod,
	}, nil
}

func (mgr *AgentMgr) toolRestartWorkload(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolRestartWL); err != nil {
		return nil, err
	}
	var args restartWorkloadArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentClusterWriteErrorf("invalid args: %w", err)
	}
	kind, err := normalizeRestartWorkloadKind(args.Kind)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return nil, agentClusterWriteErrorf("name is required")
	}
	namespace := defaultJobNamespace(args.Namespace)
	if namespace == "" {
		return nil, agentClusterWriteErrorf("namespace is required")
	}
	restartedAt := time.Now().UTC().Format(time.RFC3339)

	switch kind {
	case "deployment":
		deployment, err := mgr.kubeClient.AppsV1().Deployments(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, agentClusterWriteErrorf("failed to get deployment %s/%s: %w", namespace, name, err)
		}
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = map[string]string{}
		}
		deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := mgr.kubeClient.AppsV1().Deployments(namespace).Update(c, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, agentClusterWriteErrorf("failed to restart deployment %s/%s: %w", namespace, name, err)
		}
	case "statefulset":
		statefulSet, err := mgr.kubeClient.AppsV1().StatefulSets(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, agentClusterWriteErrorf("failed to get statefulset %s/%s: %w", namespace, name, err)
		}
		if statefulSet.Spec.Template.Annotations == nil {
			statefulSet.Spec.Template.Annotations = map[string]string{}
		}
		statefulSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := mgr.kubeClient.AppsV1().StatefulSets(namespace).Update(c, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, agentClusterWriteErrorf("failed to restart statefulset %s/%s: %w", namespace, name, err)
		}
	case "daemonset":
		daemonSet, err := mgr.kubeClient.AppsV1().DaemonSets(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, agentClusterWriteErrorf("failed to get daemonset %s/%s: %w", namespace, name, err)
		}
		if daemonSet.Spec.Template.Annotations == nil {
			daemonSet.Spec.Template.Annotations = map[string]string{}
		}
		daemonSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := mgr.kubeClient.AppsV1().DaemonSets(namespace).Update(c, daemonSet, metav1.UpdateOptions{}); err != nil {
			return nil, agentClusterWriteErrorf("failed to restart daemonset %s/%s: %w", namespace, name, err)
		}
	default:
		return nil, agentClusterWriteErrorf("unsupported workload kind %q", kind)
	}

	return map[string]any{
		"kind":           kind,
		"name":           name,
		"namespace":      namespace,
		"restarted_at":   restartedAt,
		"status":         "restarted",
		"requested_tool": agentToolRestartWL,
	}, nil
}

func (mgr *AgentMgr) toolScaleWorkload(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolK8sScaleWL); err != nil {
		return nil, err
	}
	var args scaleWorkloadArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentClusterWriteErrorf("invalid args: %w", err)
	}
	kind, err := normalizeScalableWorkloadKind(args.Kind)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return nil, agentClusterWriteErrorf("name is required")
	}
	if args.Replicas == nil || *args.Replicas < 0 {
		return nil, agentClusterWriteErrorf("replicas must be a non-negative integer")
	}
	namespace := defaultJobNamespace(args.Namespace)
	if namespace == "" {
		return nil, agentClusterWriteErrorf("namespace is required")
	}

	switch kind {
	case "deployment":
		deployment, err := mgr.kubeClient.AppsV1().Deployments(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, agentClusterWriteErrorf("failed to get deployment %s/%s: %w", namespace, name, err)
		}
		deployment.Spec.Replicas = args.Replicas
		if _, err := mgr.kubeClient.AppsV1().Deployments(namespace).Update(c, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, agentClusterWriteErrorf("failed to scale deployment %s/%s: %w", namespace, name, err)
		}
	case "statefulset":
		statefulSet, err := mgr.kubeClient.AppsV1().StatefulSets(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, agentClusterWriteErrorf("failed to get statefulset %s/%s: %w", namespace, name, err)
		}
		statefulSet.Spec.Replicas = args.Replicas
		if _, err := mgr.kubeClient.AppsV1().StatefulSets(namespace).Update(c, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, agentClusterWriteErrorf("failed to scale statefulset %s/%s: %w", namespace, name, err)
		}
	case "replicaset":
		replicaSet, err := mgr.kubeClient.AppsV1().ReplicaSets(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, agentClusterWriteErrorf("failed to get replicaset %s/%s: %w", namespace, name, err)
		}
		replicaSet.Spec.Replicas = args.Replicas
		if _, err := mgr.kubeClient.AppsV1().ReplicaSets(namespace).Update(c, replicaSet, metav1.UpdateOptions{}); err != nil {
			return nil, agentClusterWriteErrorf("failed to scale replicaset %s/%s: %w", namespace, name, err)
		}
	}

	return map[string]any{
		"kind":           kind,
		"name":           name,
		"namespace":      namespace,
		"replicas":       *args.Replicas,
		"status":         "scaled",
		"requested_tool": agentToolK8sScaleWL,
	}, nil
}

func (mgr *AgentMgr) toolLabelNode(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolK8sLabelNode); err != nil {
		return nil, err
	}
	var args nodeLabelArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentClusterWriteErrorf("invalid args: %w", err)
	}
	nodeName := strings.TrimSpace(args.NodeName)
	key := strings.TrimSpace(args.Key)
	if nodeName == "" {
		return nil, agentClusterWriteErrorf("node_name is required")
	}
	if key == "" {
		return nil, agentClusterWriteErrorf("key is required")
	}
	if err := mgr.nodeClient.AddNodeLabel(c, nodeName, key, strings.TrimSpace(args.Value)); err != nil {
		return nil, agentClusterWriteErrorf("failed to label node %s: %w", nodeName, err)
	}
	return map[string]any{
		"node_name":      nodeName,
		"key":            key,
		"value":          strings.TrimSpace(args.Value),
		"status":         "labeled",
		"requested_tool": agentToolK8sLabelNode,
	}, nil
}

func (mgr *AgentMgr) toolTaintNode(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolK8sTaintNode); err != nil {
		return nil, err
	}
	var args nodeTaintArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentClusterWriteErrorf("invalid args: %w", err)
	}
	nodeName := strings.TrimSpace(args.NodeName)
	key := strings.TrimSpace(args.Key)
	effect := strings.TrimSpace(args.Effect)
	if effect == "" {
		effect = string(v1.TaintEffectNoSchedule)
	}
	switch v1.TaintEffect(effect) {
	case v1.TaintEffectNoSchedule, v1.TaintEffectPreferNoSchedule, v1.TaintEffectNoExecute:
	default:
		return nil, agentClusterWriteErrorf("effect must be one of NoSchedule, PreferNoSchedule, NoExecute")
	}
	if nodeName == "" {
		return nil, agentClusterWriteErrorf("node_name is required")
	}
	if key == "" {
		return nil, agentClusterWriteErrorf("key is required")
	}
	if err := mgr.nodeClient.AddNodeTaint(
		c,
		nodeName,
		key,
		strings.TrimSpace(args.Value),
		effect,
		strings.TrimSpace(args.Reason),
		token.Username,
	); err != nil {
		return nil, agentClusterWriteErrorf("failed to taint node %s: %w", nodeName, err)
	}
	return map[string]any{
		"node_name":      nodeName,
		"key":            key,
		"value":          strings.TrimSpace(args.Value),
		"effect":         effect,
		"reason":         strings.TrimSpace(args.Reason),
		"status":         "tainted",
		"requested_tool": agentToolK8sTaintNode,
	}, nil
}

func (mgr *AgentMgr) toolRunKubectl(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolRunKubectl); err != nil {
		return nil, err
	}
	var args commandExecutionArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentClusterWriteErrorf("invalid args: %w", err)
	}
	command := strings.TrimSpace(args.Command)
	reason := strings.TrimSpace(args.Reason)
	if err := validateBackendKubectlWriteCommand(command, reason); err != nil {
		return nil, err
	}
	return runAgentShellCommand(c.Request.Context(), command, reason, agentToolRunKubectl)
}

func (mgr *AgentMgr) toolExecuteAdminCommand(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if err := ensureOpsAdmin(token, agentToolAdminCommand); err != nil {
		return nil, err
	}
	var args commandExecutionArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentClusterWriteErrorf("invalid args: %w", err)
	}
	command := strings.TrimSpace(args.Command)
	reason := strings.TrimSpace(args.Reason)
	if err := validateBackendAdminCommand(command, reason); err != nil {
		return nil, err
	}
	return runAgentShellCommand(c.Request.Context(), command, reason, agentToolAdminCommand)
}

func validateBackendKubectlWriteCommand(command, reason string) error {
	if command == "" {
		return agentClusterWriteErrorf("command is required")
	}
	if reason == "" {
		return agentClusterWriteErrorf("reason is required")
	}
	if hasShellControlOperator(command) {
		return agentClusterWriteErrorf("command chaining or redirection is not allowed")
	}
	tokens := strings.Fields(command)
	if len(tokens) < 2 || tokens[0] != "kubectl" {
		return agentClusterWriteErrorf("run_kubectl only accepts commands that start with kubectl")
	}
	verb := strings.ToLower(tokens[1])
	readOnly := map[string]struct{}{
		"get": {}, "describe": {}, "logs": {}, "top": {}, "version": {}, "api-resources": {}, "api-versions": {},
	}
	if _, ok := readOnly[verb]; ok {
		return agentClusterWriteErrorf("read-only kubectl commands are blocked here; use the dedicated read-only tools instead")
	}
	mutating := map[string]struct{}{
		"apply": {}, "create": {}, "delete": {}, "patch": {}, "replace": {}, "edit": {}, "set": {},
		"scale": {}, "rollout": {}, "cordon": {}, "uncordon": {}, "drain": {}, "taint": {}, "label": {}, "annotate": {},
	}
	if _, ok := mutating[verb]; !ok {
		return agentClusterWriteErrorf("kubectl operation %q is not allowed via run_kubectl", verb)
	}
	if verb == "rollout" && len(tokens) > 2 && strings.ToLower(tokens[2]) == "status" {
		return agentClusterWriteErrorf("kubectl rollout status is read-only; use the dedicated read-only tools instead")
	}
	return nil
}

func validateBackendAdminCommand(command, reason string) error {
	if command == "" {
		return agentClusterWriteErrorf("command is required")
	}
	if reason == "" {
		return agentClusterWriteErrorf("reason is required")
	}
	if hasShellControlOperator(command) {
		return agentClusterWriteErrorf("command chaining or redirection is not allowed")
	}
	tokens := strings.Fields(command)
	if len(tokens) == 0 {
		return agentClusterWriteErrorf("command is required")
	}
	allowed := map[string]struct{}{
		"helm": {}, "velero": {}, "istioctl": {}, "psql": {},
	}
	if _, ok := allowed[tokens[0]]; !ok {
		return agentClusterWriteErrorf("command must start with one of helm, velero, istioctl, psql")
	}
	return nil
}

func hasShellControlOperator(command string) bool {
	for _, pattern := range []string{"&&", "||", ";", "`", "$(", ">", "<", "\n", "\r"} {
		if strings.Contains(command, pattern) {
			return true
		}
	}
	return false
}

func runAgentShellCommand(parent context.Context, command, reason, toolName string) (any, error) {
	cfg := pkgconfig.GetConfig().Agent.Ops.Kubernetes
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Env = os.Environ()
	if kubeconfigPath := strings.TrimSpace(cfg.KubeconfigPath); kubeconfigPath != "" {
		cmd.Env = append(cmd.Env, "KUBECONFIG="+kubeconfigPath)
	}
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	text := string(output)
	truncated := false
	if len(text) > 50000 {
		text = text[:50000]
		truncated = true
	}
	result := map[string]any{
		"command":        command,
		"reason":         reason,
		"exit_code":      exitCode,
		"output":         text,
		"truncated":      truncated,
		"requested_tool": toolName,
	}
	if ctx.Err() == context.DeadlineExceeded {
		result["status"] = "timeout"
		return result, agentClusterWriteErrorf("command timed out after %s", timeout)
	}
	if err != nil {
		result["status"] = "error"
		return result, agentClusterWriteErrorf("command failed with exit code %d", exitCode)
	}
	result["status"] = "success"
	return result, nil
}
