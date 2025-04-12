package crclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/pkg/config"
)

// 用于记录已分配的 ServicePort，避免冲突
var (
	usedPorts = make(map[int32]bool)
	portMutex sync.Mutex
)

// PortMapping 结构体定义
type PortMapping struct {
	Name          string // 端口名称
	ContainerPort int32  // 容器内部端口
	ServicePort   int32  // Service 暴露的端口
	Prefix        string // Ingress 路径前缀
	ServiceName   string // 该条规则对应的 Service 名称
	IngressName   string // 该条规则对应的 Ingress 名称
}

// PodIngress 结构体定义
type PodIngress struct {
	Name   string // 规则名称
	Port   int32  // 用户指定的容器端口
	Prefix string // 唯一前缀用于访问
}

// 常量定义
const (
	startingPort    = int32(81)                  // 起始端口
	maxPort         = int32(65535)               // 最大端口
	IngressLabelKey = "ingress.crater.raids.io/" // Annotation Ingress Key
	NotebookPort    = 8888
)

// getAvailablePort 从起始端口开始分配未使用的 ServicePort
func getAvailablePort() int32 {
	portMutex.Lock()
	defer portMutex.Unlock()

	for port := startingPort; port < maxPort; port++ {
		if !usedPorts[port] {
			usedPorts[port] = true
			return port
		}
	}
	return -1
}

// CreateService 创建新的 Service
func CreateService(ctx context.Context, kubeClient client.Client, pod *v1.Pod, mapping PortMapping) (string, error) {
	serviceName := fmt.Sprintf("%s-%s", pod.Name, uuid.New().String()[:5])

	// 检查 OwnerReferences 是否为空
	var ownerRef metav1.OwnerReference
	if len(pod.OwnerReferences) > 0 {
		ownerRef = metav1.OwnerReference{
			APIVersion:         pod.OwnerReferences[0].APIVersion,
			Kind:               pod.OwnerReferences[0].Kind,
			Name:               pod.OwnerReferences[0].Name,
			UID:                pod.OwnerReferences[0].UID,
			BlockOwnerDeletion: ptr.To(true),
		}
	} else {
		ownerRef = metav1.OwnerReference{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       pod.Name,
			UID:        pod.UID,
		}
	}

	// 创建 Service
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Namespace:       pod.Namespace,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
			Labels:          pod.Labels, // 使用 Pod 的 Labels 作为 Service 的 Labels
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       mapping.Name,
					Port:       mapping.ServicePort,
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromInt(int(mapping.ContainerPort)),
				},
			},
			Type:     v1.ServiceTypeClusterIP,
			Selector: pod.Labels, // 使用 Pod 的 labels 作为 Selector
		},
	}

	if err := kubeClient.Create(ctx, svc); err != nil {
		return "", fmt.Errorf("failed to create service: %w", err)
	}
	return serviceName, nil
}

// CreateServiceForNodePort 创建 NodePort 类型的 Service 并返回 NodePort 和 ServiceName
func CreateServiceForNodePort(ctx context.Context, kubeClient client.Client, pod *v1.Pod,
	containerPort int32) (nodePort int32, serviceName string, err error) {
	// 生成唯一的 Service 名称
	serviceName = fmt.Sprintf("%s-%s", pod.Name, uuid.New().String()[:5])

	// 构造 OwnerReference
	var ownerRef metav1.OwnerReference
	if len(pod.OwnerReferences) > 0 {
		ownerRef = metav1.OwnerReference{
			APIVersion:         pod.OwnerReferences[0].APIVersion,
			Kind:               pod.OwnerReferences[0].Kind,
			Name:               pod.OwnerReferences[0].Name,
			UID:                pod.OwnerReferences[0].UID,
			BlockOwnerDeletion: ptr.To(true),
		}
	} else {
		ownerRef = metav1.OwnerReference{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       pod.Name,
			UID:        pod.UID,
		}
	}

	// 创建 NodePort 类型的 Service
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Namespace:       pod.Namespace,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
			Labels:          pod.Labels, // 使用 Pod 的 Labels 作为 Service 的 Labels
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       fmt.Sprintf("port-%d", containerPort),
					Port:       containerPort,
					TargetPort: intstr.FromInt(int(containerPort)),
					Protocol:   v1.ProtocolTCP,
				},
			},
			Type:     v1.ServiceTypeNodePort,
			Selector: pod.Labels, // 使用 Pod 的 Labels 作为 Selector
		},
	}

	// 调用 Kubernetes API 创建 Service
	if err = kubeClient.Create(ctx, svc); err != nil {
		return 0, "", fmt.Errorf("failed to create NodePort service: %w", err)
	}

	// 获取分配的 NodePort
	for _, port := range svc.Spec.Ports {
		if port.Port == containerPort {
			nodePort = port.NodePort
			break
		}
	}

	// 如果未找到 NodePort，返回错误
	if nodePort == 0 {
		return 0, "", fmt.Errorf("failed to retrieve NodePort for service %s", serviceName)
	}

	return nodePort, serviceName, nil
}

// CreateIngress 创建新的 Ingress
func CreateIngress(ctx context.Context, kubeClient client.Client, pod *v1.Pod, serviceName string, mapping PortMapping) (string, error) {
	ingressName := fmt.Sprintf("%s-%s", pod.Name, uuid.New().String()[:5])

	// 检查 OwnerReferences 是否为空
	var ownerRef metav1.OwnerReference
	if len(pod.OwnerReferences) > 0 {
		ownerRef = metav1.OwnerReference{
			APIVersion:         pod.OwnerReferences[0].APIVersion,
			Kind:               pod.OwnerReferences[0].Kind,
			Name:               pod.OwnerReferences[0].Name,
			UID:                pod.OwnerReferences[0].UID,
			BlockOwnerDeletion: ptr.To(true),
		}
	} else {
		ownerRef = metav1.OwnerReference{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       pod.Name,
			UID:        pod.UID,
		}
	}

	// 创建 Ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: pod.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect":    "true",
				"nginx.ingress.kubernetes.io/proxy-body-size": "20480m",
			},
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: func(s string) *string { return &s }("nginx"),
			Rules: []networkingv1.IngressRule{
				{
					Host: config.GetConfig().Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     mapping.Prefix,
									PathType: func(s networkingv1.PathType) *networkingv1.PathType { return &s }(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: mapping.ServicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{config.GetConfig().Host}, // 需要 TLS 的主机名
					SecretName: "crater-tls-secret",               // TLS 证书的 Secret 名称
				},
			},
		},
	}

	if err := kubeClient.Create(ctx, ingress); err != nil {
		return "", fmt.Errorf("failed to create ingress: %w", err)
	}

	// 返回创建的 Ingress 名称
	return ingressName, nil
}

// 更新 Pod 的 Annotation
func UpdatePodAnnotation(ctx context.Context, mgr client.Client, pod *v1.Pod, mapping PortMapping) error {
	// 将 Ingress 转换为 JSON 字符串
	ingressJSON, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal ingress: %w", err)
	}

	// 设置 Pod 的 Annotation
	pod.Annotations[IngressLabelKey+mapping.Name] = string(ingressJSON)

	// 更新 Pod 的 Annotation
	if err := mgr.Update(ctx, pod); err != nil {
		return fmt.Errorf("failed to update pod annotation: %w", err)
	}

	return nil
}

// CreateCustomForwardingRule 增加自定义转发规则
func CreateCustomForwardingRule(ctx context.Context, kubeClient client.Client, pod *v1.Pod, ingressRule PodIngress) error {
	var servicePort int32
	// 特殊处理 jupyter-notebook
	if ingressRule.Port != NotebookPort {
		// 分配 ServicePort
		servicePort = getAvailablePort()
		if servicePort == -1 {
			return fmt.Errorf("no available ports for ServicePort")
		}
	} else {
		servicePort = 80
	}
	// 将 PodIngress 转换为 PortMapping
	mapping := PortMapping{
		Name:          ingressRule.Name,
		ContainerPort: ingressRule.Port,
		ServicePort:   servicePort,
		Prefix:        ingressRule.Prefix,
	}

	// 创建 Service
	serviceName, err := CreateService(ctx, kubeClient, pod, mapping)
	if err != nil {
		return err
	}
	mapping.ServiceName = serviceName // 记录 Service 名称

	// 创建 Ingress
	ingressName, err := CreateIngress(ctx, kubeClient, pod, serviceName, mapping)
	if err != nil {
		return err
	}
	mapping.IngressName = ingressName // 记录 Ingress 名称

	// 更新 Pod 的 Annotation，改为传入 mapping
	if err := UpdatePodAnnotation(ctx, kubeClient, pod, mapping); err != nil {
		return err
	}

	return nil
}

// DeleteCustomForwardingRule 删除自定义转发规则
func DeleteCustomForwardingRule(ctx context.Context, kubeClient client.Client, pod *v1.Pod, ingressName, serviceName string) error {
	// 删除 Ingress
	ingress := &networkingv1.Ingress{}
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: ingressName, Namespace: pod.Namespace}, ingress); err != nil {
		return fmt.Errorf("failed to get ingress: %w", err)
	}
	if err := kubeClient.Delete(ctx, ingress); err != nil {
		return fmt.Errorf("failed to delete ingress: %w", err)
	}

	// 删除 Service
	svc := &v1.Service{}
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: pod.Namespace}, svc); err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}
	if err := kubeClient.Delete(ctx, svc); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}
