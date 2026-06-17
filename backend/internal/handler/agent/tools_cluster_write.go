package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
)

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
		return "", fmt.Errorf("unsupported workload kind %q", kind)
	}
}

func (mgr *AgentMgr) getNodeForLifecycleAction(c *gin.Context, nodeName string) (bool, error) {
	node, err := mgr.kubeClient.CoreV1().Nodes().Get(c, nodeName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get node %s: %w", nodeName, err)
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
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	nodeName := strings.TrimSpace(args.NodeName)
	if nodeName == "" {
		return nil, fmt.Errorf("node_name is required")
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
		return nil, fmt.Errorf("failed to cordon node %s: %w", nodeName, err)
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
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	nodeName := strings.TrimSpace(args.NodeName)
	if nodeName == "" {
		return nil, fmt.Errorf("node_name is required")
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
		return nil, fmt.Errorf("failed to uncordon node %s: %w", nodeName, err)
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
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	nodeName := strings.TrimSpace(args.NodeName)
	if nodeName == "" {
		return nil, fmt.Errorf("node_name is required")
	}
	if err := mgr.nodeClient.DrainNode(c, nodeName, token.Username); err != nil {
		return nil, fmt.Errorf("failed to drain node %s: %w", nodeName, err)
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
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		name = strings.TrimSpace(args.PodName)
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	namespace := defaultJobNamespace(args.Namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	deleteOpts := metav1.DeleteOptions{}
	if args.Force {
		zero := int64(0)
		deleteOpts.GracePeriodSeconds = &zero
	} else if args.GracePeriodSeconds != nil {
		deleteOpts.GracePeriodSeconds = args.GracePeriodSeconds
	}
	if err := mgr.kubeClient.CoreV1().Pods(namespace).Delete(c, name, deleteOpts); err != nil {
		return nil, fmt.Errorf("failed to delete pod %s/%s: %w", namespace, name, err)
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
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	kind, err := normalizeRestartWorkloadKind(args.Kind)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	namespace := defaultJobNamespace(args.Namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	restartedAt := time.Now().UTC().Format(time.RFC3339)

	switch kind {
	case "deployment":
		deployment, err := mgr.kubeClient.AppsV1().Deployments(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment %s/%s: %w", namespace, name, err)
		}
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = map[string]string{}
		}
		deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := mgr.kubeClient.AppsV1().Deployments(namespace).Update(c, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, fmt.Errorf("failed to restart deployment %s/%s: %w", namespace, name, err)
		}
	case "statefulset":
		statefulSet, err := mgr.kubeClient.AppsV1().StatefulSets(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get statefulset %s/%s: %w", namespace, name, err)
		}
		if statefulSet.Spec.Template.Annotations == nil {
			statefulSet.Spec.Template.Annotations = map[string]string{}
		}
		statefulSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := mgr.kubeClient.AppsV1().StatefulSets(namespace).Update(c, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, fmt.Errorf("failed to restart statefulset %s/%s: %w", namespace, name, err)
		}
	case "daemonset":
		daemonSet, err := mgr.kubeClient.AppsV1().DaemonSets(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get daemonset %s/%s: %w", namespace, name, err)
		}
		if daemonSet.Spec.Template.Annotations == nil {
			daemonSet.Spec.Template.Annotations = map[string]string{}
		}
		daemonSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := mgr.kubeClient.AppsV1().DaemonSets(namespace).Update(c, daemonSet, metav1.UpdateOptions{}); err != nil {
			return nil, fmt.Errorf("failed to restart daemonset %s/%s: %w", namespace, name, err)
		}
	default:
		return nil, fmt.Errorf("unsupported workload kind %q", kind)
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
