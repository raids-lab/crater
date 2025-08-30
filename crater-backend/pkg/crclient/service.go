package crclient

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/pkg/config"
)

type CraterJobType string

const (
	CraterJobTypeTensorflow CraterJobType = "tensorflow"
	CraterJobTypePytorch    CraterJobType = "pytorch"
	CraterJobTypeJupyter    CraterJobType = "jupyter"
	CraterJobTypeCustom     CraterJobType = "custom"
)

const (
	LabelKeyBaseURL  = "crater.raids.io/base-url"
	LabelKeyTaskType = "crater.raids.io/task-type"
	LabelKeyTaskUser = "crater.raids.io/task-user"

	AnnotationKeyPortName = "crater.raids.io/port-name" // Annotation key for port name

	Poll    = 500 * time.Millisecond // Polling interval for checking service creation
	Timeout = 5 * time.Second        // Timeout for service creation

	// Added volcano keys
	LabelKeyTaskIndex = "volcano.sh/task-index"
	LabelKeyTaskSpec  = "volcano.sh/task-spec"
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
		host string,
		prefix string,
	) (ingressPath string, err error)

	// CreateIngress 创建一个 ClusterIP 类型的 Service，并创建 Ingress
	CreateIngress(
		ctx context.Context,
		ownerReferences []metav1.OwnerReference,
		podSelector map[string]string,
		port *v1.ServicePort,
		host, username string,
	) (ingressPath string, err error)

	// GenerateLabels based on TaskType
	GenerateLabels(podSelector map[string]string) map[string]string
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

// GenerateLabels based on TaskType
func (s *serviceManagerImpl) GenerateLabels(podSelector map[string]string) map[string]string {
	labels := map[string]string{
		LabelKeyBaseURL:  podSelector[LabelKeyBaseURL],
		LabelKeyTaskType: podSelector[LabelKeyTaskType],
		LabelKeyTaskUser: podSelector[LabelKeyTaskUser],
	}

	taskType := podSelector[LabelKeyTaskType]
	if taskType == string(CraterJobTypeTensorflow) || taskType == string(CraterJobTypePytorch) {
		if index, ok := podSelector[LabelKeyTaskIndex]; ok {
			labels[LabelKeyTaskIndex] = index
		}
		if spec, ok := podSelector[LabelKeyTaskSpec]; ok {
			labels[LabelKeyTaskSpec] = spec
		}
	}

	return labels
}

// CreateNodePort 实现
//
//nolint:gocyclo // Cyclomatic complexity is acceptable for this function
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
	serviceName := fmt.Sprintf("np-%s-%s", username, uuid.New().String()[:5])
	namespace := s.config.Workspace.Namespace

	labels := s.GenerateLabels(podSelector)

	const (
		nodePortStart = 30000
		nodePortEnd   = 32767
	)
	userPort := func(username string) int32 {
		h := fnv.New32a()
		h.Write([]byte(username))
		nodePortStart32 := int32(nodePortStart)
		portRange := uint32(nodePortEnd - nodePortStart + 1)

		hashVal := h.Sum32() % portRange
		return nodePortStart32 + int32(hashVal) // #nosec G115
	}(username)

	tryReserved := true
	var svc *v1.Service
	for {
		svc = &v1.Service{
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
				Ports: []v1.ServicePort{
					{
						Name:       port.Name,
						Protocol:   port.Protocol,
						Port:       port.Port,
						TargetPort: port.TargetPort,
						NodePort:   0,
					},
				},
				Type:     v1.ServiceTypeNodePort,
				Selector: podSelector,
			},
		}
		if tryReserved {
			svc.Spec.Ports[0].NodePort = userPort
		}
		err = s.client.Create(ctx, svc)
		if err == nil {
			break
		}
		if tryReserved && strings.Contains(err.Error(), "provided port is already allocated") {
			tryReserved = false
			continue
		}
		return "", 0, fmt.Errorf("failed to create NodePort service: %w", err)
	}

	var createdSvc v1.Service
	err = wait.PollUntilContextTimeout(ctx, Poll, Timeout, false, func(ctx context.Context) (bool, error) {
		if e := s.client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &createdSvc); e != nil {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return "", 0, fmt.Errorf("failed to get created service after retries: %w", err)
	}

	nodePort = createdSvc.Spec.Ports[0].NodePort

	podList := &v1.PodList{}
	if err = s.client.List(ctx, podList, client.InNamespace(namespace), client.MatchingLabels(podSelector)); err != nil {
		return "", nodePort, fmt.Errorf("failed to list pods: %w", err)
	}

	if len(podList.Items) == 0 {
		return "", nodePort, fmt.Errorf("no pods found matching selector")
	}

	pod := podList.Items[0]
	nodeName := pod.Spec.NodeName

	if nodeName == "" {
		return "", nodePort, fmt.Errorf("pod not assigned to a node yet")
	}

	node, err := s.kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return "", nodePort, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeExternalIP {
			host = addr.Address
			return host, nodePort, nil
		}
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			host = addr.Address
			return host, nodePort, nil
		}
	}

	return "", nodePort, fmt.Errorf("no suitable host IP found for node %s", nodeName)
}

// CreateIngressWithPrefix 实现
func (s *serviceManagerImpl) CreateIngressWithPrefix(
	ctx context.Context,
	ownerReferences []metav1.OwnerReference,
	podSelector map[string]string,
	port *v1.ServicePort,
	host string,
	prefix string,
) (ingressPath string, err error) {
	if port == nil {
		return "", fmt.Errorf("port and ownerRef cannot be nil")
	}
	namespace := s.config.Workspace.Namespace
	labels := s.GenerateLabels(podSelector)

	serviceName := fmt.Sprintf("svc-%s-%s", prefix, port.Name)
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

	ingressName := fmt.Sprintf("ing-%s-%s", prefix, port.Name)

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
				"nginx.ingress.kubernetes.io/ssl-redirect":          "true",
				"nginx.ingress.kubernetes.io/proxy-body-size":       "20480m",
				"nginx.ingress.kubernetes.io/proxy-connect-timeout": "300",
				"nginx.ingress.kubernetes.io/proxy-send-timeout":    "300",
				"nginx.ingress.kubernetes.io/proxy-read-timeout":    "300",
				AnnotationKeyPortName:                               port.Name,
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
					SecretName: s.config.Secrets.TLSSecretName,
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
	host,
	username string,
) (ingressPath string, err error) {
	if port == nil {
		return "", fmt.Errorf("port and ownerRef cannot be nil")
	}
	namespace := s.config.Workspace.Namespace
	labels := s.GenerateLabels(podSelector)

	randomPrefix := uuid.New().String()[:5]
	subdomain := fmt.Sprintf("%s.%s", randomPrefix, host)

	serviceName := fmt.Sprintf("svc-%s-%s-%s", username, randomPrefix, port.Name)
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

	ingressName := fmt.Sprintf("ing-%s-%s-%s", username, randomPrefix, port.Name)
	pathType := networkingv1.PathTypePrefix
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:            ingressName,
			Namespace:       namespace,
			OwnerReferences: ownerReferences,
			Labels:          labels,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect":          "true",
				"nginx.ingress.kubernetes.io/proxy-body-size":       "20480m",
				"nginx.ingress.kubernetes.io/proxy-connect-timeout": "300",
				"nginx.ingress.kubernetes.io/proxy-send-timeout":    "300",
				"nginx.ingress.kubernetes.io/proxy-read-timeout":    "300",
				AnnotationKeyPortName:                               port.Name,
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
					SecretName: s.config.Secrets.TLSForwardSecretName,
				},
			},
		},
	}

	if err = s.client.Create(ctx, ingress); err != nil {
		return "", fmt.Errorf("failed to create ingress: %w", err)
	}

	ingressPath = fmt.Sprintf("https://%s/", subdomain)
	return ingressPath, nil
}
