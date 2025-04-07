package crclient

import (
	"context"
	"fmt"
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

// ServiceManagerInterface 接口定义
type ServiceManagerInterface interface {
	// CreateNodePort 创建一个 NodePort 类型的 Service，并返回 host 和分配的 NodePort
	CreateNodePort(
		ctx context.Context,
		ownerReferences []metav1.OwnerReference,
		podSelector map[string]string,
		port *v1.ServicePort,
	) (host string, nodePort int32, err error)

	// CreateIngressWithPrefix 创建一个 ClusterIP 类型的 Service，并创建带有前缀的 Ingress
	CreateIngressWithPrefix(
		ctx context.Context,
		ownerReferences []metav1.OwnerReference,
		podSelector map[string]string,
		port *v1.ServicePort,
		host, prefix string,
	) (ingressPath string, err error)

	// CreateIngress 创建一个 ClusterIP 类型的 Service，并创建 Ingress
	CreateIngress(
		ctx context.Context,
		ownerReferences []metav1.OwnerReference,
		podSelector map[string]string,
		port *v1.ServicePort,
		host string,
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

// CreateNodePort 实现
func (s *serviceManagerImpl) CreateNodePort(
	ctx context.Context,
	ownerReferences []metav1.OwnerReference,
	podSelector map[string]string,
	port *v1.ServicePort,
) (host string, nodePort int32, err error) {
	if port == nil {
		return "", 0, fmt.Errorf("port and ownerRef cannot be nil")
	}
	// 生成唯一的 Service 名称
	serviceName := fmt.Sprintf("np-%s", uuid.New().String()[:8])
	namespace := s.config.Workspace.Namespace

	// 创建 NodePort 类型的 Service
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Namespace:       namespace,
			OwnerReferences: ownerReferences,
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
	host, prefix string,
) (ingressPath string, err error) {
	if port == nil {
		return "", fmt.Errorf("port and ownerRef cannot be nil")
	}
	namespace := s.config.Workspace.Namespace

	// 首先创建 ClusterIP 服务
	serviceName := fmt.Sprintf("svc-%s", uuid.New().String()[:8])
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Namespace:       namespace,
			OwnerReferences: ownerReferences,
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

	// 确保前缀以 / 开始
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	// 确保前缀以 / 结束
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// 创建 Ingress
	ingressName := fmt.Sprintf("ing-%s", uuid.New().String()[:8])
	pathType := networkingv1.PathTypePrefix

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect":    "true",
				"nginx.ingress.kubernetes.io/proxy-body-size": "20480m",
			},
			OwnerReferences: ownerReferences,
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
					SecretName: "crater-tls-secret",
				},
			},
		},
	}

	if err = s.client.Create(ctx, ingress); err != nil {
		return "", fmt.Errorf("failed to create ingress: %w", err)
	}

	ingressPath = fmt.Sprintf("https://%s%s", host, prefix)
	return ingressPath, nil
}

// CreateIngress 实现
func (s *serviceManagerImpl) CreateIngress(
	ctx context.Context,
	ownerReferences []metav1.OwnerReference,
	podSelector map[string]string,
	port *v1.ServicePort,
	host string,
) (ingressPath string, err error) {
	// 为兼容 API，生成一个随机前缀
	prefix := fmt.Sprintf("/%s/", uuid.New().String()[:8])
	return s.CreateIngressWithPrefix(ctx, ownerReferences, podSelector, port, host, prefix)
}
