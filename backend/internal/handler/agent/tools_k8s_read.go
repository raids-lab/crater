package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
)

func isAdminToken(token util.JWTMessage) bool {
	return token.RolePlatform == model.RoleAdmin
}

func scopedJobNamespace(namespace string) string {
	if trimmed := strings.TrimSpace(namespace); trimmed != "" {
		return trimmed
	}
	return pkgconfig.GetConfig().Namespaces.Job
}

func userTaskLabelSelector(token util.JWTMessage) string {
	if strings.TrimSpace(token.Username) == "" {
		return ""
	}
	return fmt.Sprintf("%s=%s", crclient.LabelKeyTaskUser, token.Username)
}

func mergeLabelSelectors(parts ...string) string {
	merged := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		merged = append(merged, part)
	}
	return strings.Join(merged, ",")
}

func requireUserTaskSelector(token util.JWTMessage) (string, error) {
	if isAdminToken(token) {
		return "", nil
	}
	selector := userTaskLabelSelector(token)
	if selector == "" {
		return "", fmt.Errorf("user identity is unavailable")
	}
	return selector, nil
}

func ensureServiceVisibleToToken(service *v1.Service, token util.JWTMessage) error {
	if service == nil {
		return fmt.Errorf("service not found")
	}
	if isAdminToken(token) {
		return nil
	}
	if strings.TrimSpace(token.Username) == "" {
		return fmt.Errorf("user identity is unavailable")
	}
	if strings.TrimSpace(service.Labels[crclient.LabelKeyTaskUser]) != strings.TrimSpace(token.Username) {
		return fmt.Errorf("service %q does not belong to current user", service.Name)
	}
	return nil
}

func ensureIngressVisibleToToken(ingress *networkingv1.Ingress, token util.JWTMessage) error {
	if ingress == nil {
		return fmt.Errorf("ingress not found")
	}
	if isAdminToken(token) {
		return nil
	}
	if strings.TrimSpace(token.Username) == "" {
		return fmt.Errorf("user identity is unavailable")
	}
	if strings.TrimSpace(ingress.Labels[crclient.LabelKeyTaskUser]) != strings.TrimSpace(token.Username) {
		return fmt.Errorf("ingress %q does not belong to current user", ingress.Name)
	}
	return nil
}

func buildServiceSummary(item *v1.Service) map[string]any {
	ports := make([]map[string]any, 0, len(item.Spec.Ports))
	for _, port := range item.Spec.Ports {
		ports = append(ports, map[string]any{
			"name":        port.Name,
			"port":        port.Port,
			"target_port": port.TargetPort.String(),
			"protocol":    string(port.Protocol),
			"node_port":   port.NodePort,
		})
	}
	return map[string]any{
		"name":       item.Name,
		"namespace":  item.Namespace,
		"type":       string(item.Spec.Type),
		"cluster_ip": item.Spec.ClusterIP,
		"ports":      ports,
		"selector":   item.Spec.Selector,
	}
}

func buildEndpointsSummary(item *v1.Endpoints) map[string]any {
	subsets := make([]map[string]any, 0, len(item.Subsets))
	for _, subset := range item.Subsets {
		addresses := make([]map[string]any, 0, len(subset.Addresses))
		for _, addr := range subset.Addresses {
			targetRef := ""
			if addr.TargetRef != nil {
				targetRef = addr.TargetRef.Name
			}
			addresses = append(addresses, map[string]any{
				"ip":         addr.IP,
				"node_name":  addr.NodeName,
				"target_ref": targetRef,
			})
		}
		notReady := make([]map[string]any, 0, len(subset.NotReadyAddresses))
		for _, addr := range subset.NotReadyAddresses {
			targetRef := ""
			if addr.TargetRef != nil {
				targetRef = addr.TargetRef.Name
			}
			notReady = append(notReady, map[string]any{
				"ip":         addr.IP,
				"node_name":  addr.NodeName,
				"target_ref": targetRef,
			})
		}
		ports := make([]map[string]any, 0, len(subset.Ports))
		for _, port := range subset.Ports {
			ports = append(ports, map[string]any{
				"name":     port.Name,
				"port":     port.Port,
				"protocol": string(port.Protocol),
			})
		}
		subsets = append(subsets, map[string]any{
			"addresses":           addresses,
			"not_ready_addresses": notReady,
			"ports":               ports,
		})
	}
	return map[string]any{
		"name":      item.Name,
		"namespace": item.Namespace,
		"subsets":   subsets,
	}
}

func buildIngressSummary(item *networkingv1.Ingress) map[string]any {
	rules := make([]map[string]any, 0, len(item.Spec.Rules))
	for _, rule := range item.Spec.Rules {
		paths := make([]map[string]any, 0)
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				port := ""
				if path.Backend.Service != nil {
					if path.Backend.Service.Port.Number != 0 {
						port = fmt.Sprintf("%d", path.Backend.Service.Port.Number)
					} else {
						port = path.Backend.Service.Port.Name
					}
				}
				serviceName := ""
				if path.Backend.Service != nil {
					serviceName = path.Backend.Service.Name
				}
				pathType := ""
				if path.PathType != nil {
					pathType = string(*path.PathType)
				}
				paths = append(paths, map[string]any{
					"path":      path.Path,
					"path_type": pathType,
					"service":   serviceName,
					"port":      port,
				})
			}
		}
		rules = append(rules, map[string]any{
			"host":  rule.Host,
			"paths": paths,
		})
	}
	addresses := make([]string, 0, len(item.Status.LoadBalancer.Ingress))
	for _, ingress := range item.Status.LoadBalancer.Ingress {
		if ingress.Hostname != "" {
			addresses = append(addresses, ingress.Hostname)
		}
		if ingress.IP != "" {
			addresses = append(addresses, ingress.IP)
		}
	}
	return map[string]any{
		"name":      item.Name,
		"namespace": item.Namespace,
		"rules":     rules,
		"addresses": addresses,
	}
}

func (mgr *AgentMgr) toolK8sListPods(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	namespace := scopedJobNamespace(getToolArgString(args, "namespace", ""))
	limit := getToolArgInt(args, "limit", 50)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	fieldSelector := getToolArgString(args, "field_selector", "")
	nodeName := getToolArgString(args, "node_name", "")
	if nodeName != "" {
		if fieldSelector != "" {
			fieldSelector += ","
		}
		fieldSelector += "spec.nodeName=" + nodeName
	}

	labelSelector := getToolArgString(args, "label_selector", "")
	if !isAdminToken(token) {
		userSelector, err := requireUserTaskSelector(token)
		if err != nil {
			return nil, err
		}
		labelSelector = mergeLabelSelectors(userSelector, labelSelector)
	}

	pods, err := mgr.kubeClient.CoreV1().Pods(namespace).List(c, metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	items := make([]map[string]any, 0, min(limit, len(pods.Items)))
	for _, item := range pods.Items {
		restartCount := 0
		readyCount := 0
		startTime := ""
		for _, status := range item.Status.ContainerStatuses {
			restartCount += int(status.RestartCount)
			if status.Ready {
				readyCount++
			}
		}
		if item.Status.StartTime != nil {
			startTime = item.Status.StartTime.Time.Format(time.RFC3339)
		}
		items = append(items, map[string]any{
			"namespace":        item.Namespace,
			"name":             item.Name,
			"phase":            string(item.Status.Phase),
			"node_name":        item.Spec.NodeName,
			"pod_ip":           item.Status.PodIP,
			"ready_containers": readyCount,
			"containers":       len(item.Status.ContainerStatuses),
			"restart_count":    restartCount,
			"start_time":       startTime,
		})
		if len(items) >= limit {
			break
		}
	}
	return map[string]any{
		"namespace": namespace,
		"count":     len(items),
		"pods":      items,
	}, nil
}

func (mgr *AgentMgr) toolK8sGetService(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	namespace := scopedJobNamespace(getToolArgString(args, "namespace", ""))
	name := getToolArgString(args, "name", "")
	limit := getToolArgInt(args, "limit", 50)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	labelSelector := getToolArgString(args, "label_selector", "")
	if !isAdminToken(token) {
		userSelector, err := requireUserTaskSelector(token)
		if err != nil {
			return nil, err
		}
		labelSelector = mergeLabelSelectors(userSelector, labelSelector)
	}
	fieldSelector := getToolArgString(args, "field_selector", "")

	if name != "" {
		service, err := mgr.kubeClient.CoreV1().Services(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get service %q: %w", name, err)
		}
		if err := ensureServiceVisibleToToken(service, token); err != nil {
			return nil, err
		}
		return map[string]any{
			"count":    1,
			"services": []map[string]any{buildServiceSummary(service)},
		}, nil
	}

	services, err := mgr.kubeClient.CoreV1().Services(namespace).List(c, metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	items := make([]map[string]any, 0, min(limit, len(services.Items)))
	for _, service := range services.Items {
		items = append(items, buildServiceSummary(&service))
		if len(items) >= limit {
			break
		}
	}
	return map[string]any{
		"count":    len(items),
		"services": items,
	}, nil
}

func (mgr *AgentMgr) toolK8sGetEndpoints(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	namespace := scopedJobNamespace(getToolArgString(args, "namespace", ""))
	name := getToolArgString(args, "name", "")
	limit := getToolArgInt(args, "limit", 50)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	if name != "" {
		if !isAdminToken(token) {
			service, err := mgr.kubeClient.CoreV1().Services(namespace).Get(c, name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to verify service ownership for %q: %w", name, err)
			}
			if err := ensureServiceVisibleToToken(service, token); err != nil {
				return nil, err
			}
		}
		endpoints, err := mgr.kubeClient.CoreV1().Endpoints(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get endpoints %q: %w", name, err)
		}
		return map[string]any{
			"count":     1,
			"endpoints": []map[string]any{buildEndpointsSummary(endpoints)},
		}, nil
	}

	var endpointObjects []*v1.Endpoints
	if isAdminToken(token) {
		endpoints, err := mgr.kubeClient.CoreV1().Endpoints(namespace).List(c, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list endpoints: %w", err)
		}
		for idx := range endpoints.Items {
			endpointObjects = append(endpointObjects, &endpoints.Items[idx])
			if len(endpointObjects) >= limit {
				break
			}
		}
	} else {
		userSelector, err := requireUserTaskSelector(token)
		if err != nil {
			return nil, err
		}
		services, err := mgr.kubeClient.CoreV1().Services(namespace).List(c, metav1.ListOptions{
			LabelSelector: userSelector,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list owned services: %w", err)
		}
		for idx := range services.Items {
			serviceName := services.Items[idx].Name
			endpoints, getErr := mgr.kubeClient.CoreV1().Endpoints(namespace).Get(c, serviceName, metav1.GetOptions{})
			if getErr != nil {
				if k8serrors.IsNotFound(getErr) {
					continue
				}
				return nil, fmt.Errorf("failed to get endpoints for service %q: %w", serviceName, getErr)
			}
			endpointObjects = append(endpointObjects, endpoints)
			if len(endpointObjects) >= limit {
				break
			}
		}
	}

	items := make([]map[string]any, 0, len(endpointObjects))
	for _, endpoint := range endpointObjects {
		items = append(items, buildEndpointsSummary(endpoint))
	}
	return map[string]any{
		"count":     len(items),
		"endpoints": items,
	}, nil
}

func (mgr *AgentMgr) toolK8sGetIngress(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	namespace := scopedJobNamespace(getToolArgString(args, "namespace", ""))
	name := getToolArgString(args, "name", "")
	limit := getToolArgInt(args, "limit", 50)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	labelSelector := getToolArgString(args, "label_selector", "")
	if !isAdminToken(token) {
		userSelector, err := requireUserTaskSelector(token)
		if err != nil {
			return nil, err
		}
		labelSelector = mergeLabelSelectors(userSelector, labelSelector)
	}

	if name != "" {
		ingress, err := mgr.kubeClient.NetworkingV1().Ingresses(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get ingress %q: %w", name, err)
		}
		if err := ensureIngressVisibleToToken(ingress, token); err != nil {
			return nil, err
		}
		return map[string]any{
			"count":     1,
			"ingresses": []map[string]any{buildIngressSummary(ingress)},
		}, nil
	}

	ingresses, err := mgr.kubeClient.NetworkingV1().Ingresses(namespace).List(c, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}
	items := make([]map[string]any, 0, min(limit, len(ingresses.Items)))
	for idx := range ingresses.Items {
		items = append(items, buildIngressSummary(&ingresses.Items[idx]))
		if len(items) >= limit {
			break
		}
	}
	return map[string]any{
		"count":     len(items),
		"ingresses": items,
	}, nil
}
