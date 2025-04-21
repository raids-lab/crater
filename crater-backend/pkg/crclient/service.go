package crclient

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/pkg/config"
)

const (
	LabelKeyBaseURL       = "crater.raids.io/base-url"
	LabelKeyTaskType      = "crater.raids.io/task-type"
	LabelKeyTaskUser      = "crater.raids.io/task-user"
	AnnotationKeyPortName = "crater.raids.io/port-name" // Annotation key for port name
)

// ServiceManagerInterface 接口定义
type ServiceManagerInterface interface {
	// CreateNodePort 创建一个 NodePort 类型的 Service，并返回 host 和分配的 NodePort
	CreateNodePort(
		ctx context.Context,
		ownerReferences []metav1.OwnerReference,
		podSelector map[string]string,
		port *v1.ServicePort,
		username string,
	) (host string, nodePort int32, err error)

	// CreateIngressWithPrefix 创建一个 ClusterIP 类型的 Service，并创建带有前缀的 Ingress
	CreateIngressWithPrefix(
		ctx context.Context,
		ownerReferences []metav1.OwnerReference,
		podSelector map[string]string,
		port *v1.ServicePort,
		host, prefix, username string,
	) (ingressPath string, err error)

	// CreateIngress 创建一个 ClusterIP 类型的 Service，并创建 Ingress
	CreateIngress(
		ctx context.Context,
		ownerReferences []metav1.OwnerReference,
		podSelector map[string]string,
		port *v1.ServicePort,
		host, username string,
	) (ingressPath string, err error)
}

// serviceManagerImpl 实现 ServiceManager 接口
type serviceManagerImpl struct {
	client     client.Client
	kubeClient kubernetes.Interface
	config     *config.Config
}

// NewServiceManager 创建新的 ServiceManager 实例
func NewServiceManager(cl client.Client, kubeClient kubernetes.Interface) ServiceManagerInterface {
	return &serviceManagerImpl{
		client:     cl,
		kubeClient: kubeClient,
		config:     config.GetConfig(),
	}
}

// Helper function to generate labels
func generateLabels(podSelector map[string]string, username string) map[string]string {
	return map[string]string{
		LabelKeyBaseURL:  podSelector[LabelKeyBaseURL],
		LabelKeyTaskType: podSelector[LabelKeyTaskType],
		LabelKeyTaskUser: username,
	}
}

// CreateNodePort 实现
func (s *serviceManagerImpl) CreateNodePort(
	ctx context.Context,
	ownerReferences []metav1.OwnerReference,
	podSelector map[string]string,
	port *v1.ServicePort,
	username string,
) (host string, nodePort int32, err error) {
	if port == nil {
		return "", 0, fmt.Errorf("port and ownerRef cannot be nil")
	}
	// 生成唯一的 Service 名称
	serviceName := fmt.Sprintf("%s-np-%s", username, uuid.New().String()[:5])
	namespace := s.config.Workspace.Namespace

	// Extract labels from podSelector
	labels := map[string]string{
		LabelKeyBaseURL:  podSelector[LabelKeyBaseURL],
		LabelKeyTaskType: podSelector[LabelKeyTaskType],
		LabelKeyTaskUser: podSelector[LabelKeyTaskUser],
	}

	// 创建 NodePort 类型的 Service
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Namespace:       namespace,
			OwnerReferences: ownerReferences,
			Labels:          labels,
			Annotations: map[string]string{
				AnnotationKeyPortName: port.Name,
			},
		},
		Spec: v1.ServiceSpec{
			Ports:    []v1.ServicePort{*port},
			Type:     v1.ServiceTypeNodePort,
			Selector: podSelector,
		},
	}

	// 调用 Kubernetes API 创建 Service
	if err = s.client.Create(ctx, svc); err != nil {
		return "", 0, fmt.Errorf("failed to create NodePort service: %w", err)
	}

	// 重新获取 Service 以获取分配的 NodePort
	if err = s.client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, svc); err != nil {
		return "", 0, fmt.Errorf("failed to get created service: %w", err)
	}

	// 获取分配的 NodePort
	nodePort = svc.Spec.Ports[0].NodePort

	// 获取集群节点的外部 IP
	nodes, err := s.kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", nodePort, fmt.Errorf("failed to list nodes: %w", err)
	}

	// 尝试获取节点的外部 IP
	for i := range nodes.Items {
		for _, addr := range nodes.Items[i].Status.Addresses {
			if addr.Type == v1.NodeExternalIP {
				host = addr.Address
				return host, nodePort, nil
			}
		}
	}

	// 如果没有外部 IP，使用第一个节点的内部 IP
	for i := range nodes.Items {
		for _, addr := range nodes.Items[i].Status.Addresses {
			if addr.Type == v1.NodeInternalIP {
				host = addr.Address
				return host, nodePort, nil
			}
		}
	}

	return "", nodePort, fmt.Errorf("no suitable host IP found")
}

// CreateIngressWithPrefix 实现
func (s *serviceManagerImpl) CreateIngressWithPrefix(
	ctx context.Context,
	ownerReferences []metav1.OwnerReference,
	podSelector map[string]string,
	port *v1.ServicePort,
	host,
	prefix,
	username string,
) (ingressPath string, err error) {
	if port == nil {
		return "", fmt.Errorf("port and ownerRef cannot be nil")
	}
	namespace := s.config.Workspace.Namespace

	// Generate labels
	labels := generateLabels(podSelector, username)

	// Create the ClusterIP service
	serviceName := fmt.Sprintf("%s-svc-%s", prefix, port.Name)
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Namespace:       namespace,
			OwnerReferences: ownerReferences,
			Labels:          labels,
		},
		Spec: v1.ServiceSpec{
			Ports:    []v1.ServicePort{*port},
			Type:     v1.ServiceTypeClusterIP,
			Selector: podSelector,
		},
	}

	if err = s.client.Create(ctx, svc); err != nil {
		return "", fmt.Errorf("failed to create service: %w", err)
	}

	// Validate the created service
	createdSvc := &v1.Service{}
	if err = s.client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSvc); err != nil {
		return "", fmt.Errorf("failed to fetch created service: %w", err)
	}
	if len(createdSvc.Spec.Selector) == 0 {
		return "", fmt.Errorf("service selector is empty or invalid")
	}

	// Create the Ingress
	ingressName := fmt.Sprintf("%s-ing-%s", prefix, port.Name)

	// Ensure prefix starts with "/"
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	prefix = fmt.Sprintf("/ingress%s", prefix)

	pathType := networkingv1.PathTypePrefix
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:            ingressName,
			Namespace:       namespace,
			OwnerReferences: ownerReferences,
			Labels:          labels,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect":    "true",
				"nginx.ingress.kubernetes.io/proxy-body-size": "20480m",
				AnnotationKeyPortName:                         port.Name,
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ptr.To("nginx"),
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     prefix,
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: port.Port,
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
					Hosts:      []string{host},
					SecretName: s.config.TLSSecretName,
				},
			},
		},
	}

	if err = s.client.Create(ctx, ingress); err != nil {
		return "", fmt.Errorf("failed to create ingress: %w", err)
	}

	// Validate the created ingress
	createdIngress := &networkingv1.Ingress{}
	if err = s.client.Get(ctx, types.NamespacedName{Name: ingressName, Namespace: namespace}, createdIngress); err != nil {
		return "", fmt.Errorf("failed to fetch created ingress: %w", err)
	}
	if len(createdIngress.Spec.Rules) == 0 || createdIngress.Spec.Rules[0].HTTP == nil {
		return "", fmt.Errorf("ingress rules are empty or invalid")
	}

	// Log the created service and ingress for debugging
	log.Printf("Created Service: %s, Selector: %v", createdSvc.Name, createdSvc.Spec.Selector)
	log.Printf("Created Ingress: %s, Rules: %v", createdIngress.Name, createdIngress.Spec.Rules)

	ingressPath = fmt.Sprintf("https://%s%s", host, prefix) // Construct the full ingress path
	return ingressPath, nil
}

// CreateIngress 实现
func (s *serviceManagerImpl) CreateIngress(
	ctx context.Context,
	ownerReferences []metav1.OwnerReference,
	podSelector map[string]string,
	port *v1.ServicePort,
	host,
	username string,
) (ingressPath string, err error) {
	if port == nil {
		return "", fmt.Errorf("port and ownerRef cannot be nil")
	}
	namespace := s.config.Workspace.Namespace

	// Generate labels
	labels := generateLabels(podSelector, username)

	// 生成随机8位字符串作为子域名前缀
	randomPrefix := uuid.New().String()[:5]

	// 构建新的五级域名
	subdomain := fmt.Sprintf("%s.%s", randomPrefix, host)

	// Create the ClusterIP service
	serviceName := fmt.Sprintf("%s-%s-svc-%s", username, randomPrefix, port.Name)
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Namespace:       namespace,
			OwnerReferences: ownerReferences,
			Labels:          labels,
			Annotations: map[string]string{
				AnnotationKeyPortName: port.Name,
			},
		},
		Spec: v1.ServiceSpec{
			Ports:    []v1.ServicePort{*port},
			Type:     v1.ServiceTypeClusterIP,
			Selector: podSelector,
		},
	}

	if err = s.client.Create(ctx, svc); err != nil {
		return "", fmt.Errorf("failed to create service: %w", err)
	}

	// Validate the created service
	createdSvc := &v1.Service{}
	if err = s.client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSvc); err != nil {
		return "", fmt.Errorf("failed to fetch created service: %w", err)
	}
	if len(createdSvc.Spec.Selector) == 0 {
		return "", fmt.Errorf("service selector is empty or invalid")
	}

	// Create the Ingress
	ingressName := fmt.Sprintf("%s-%s-ing-%s", username, randomPrefix, port.Name)

	pathType := networkingv1.PathTypePrefix
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:            ingressName,
			Namespace:       namespace,
			OwnerReferences: ownerReferences,
			Labels:          labels,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect":    "true",
				"nginx.ingress.kubernetes.io/proxy-body-size": "20480m",
				AnnotationKeyPortName:                         port.Name,
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ptr.To("nginx"),
			Rules: []networkingv1.IngressRule{
				{
					Host: subdomain,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: port.Port,
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
					Hosts:      []string{subdomain},
					SecretName: s.config.TLSForwardSecretName,
				},
			},
		},
	}

	if err = s.client.Create(ctx, ingress); err != nil {
		return "", fmt.Errorf("failed to create ingress: %w", err)
	}

	// Validate the created ingress
	createdIngress := &networkingv1.Ingress{}
	if err = s.client.Get(ctx, types.NamespacedName{Name: ingressName, Namespace: namespace}, createdIngress); err != nil {
		return "", fmt.Errorf("failed to fetch created ingress: %w", err)
	}
	if len(createdIngress.Spec.Rules) == 0 || createdIngress.Spec.Rules[0].HTTP == nil {
		return "", fmt.Errorf("ingress rules are empty or invalid")
	}

	// Log the created service and ingress for debugging
	log.Printf("Created Service: %s, Selector: %v", createdSvc.Name, createdSvc.Spec.Selector)
	log.Printf("Created Ingress: %s, Rules: %v", createdIngress.Name, createdIngress.Spec.Rules)

	ingressPath = fmt.Sprintf("https://%s/", subdomain) // 构建完整的访问路径
	return ingressPath, nil
}
