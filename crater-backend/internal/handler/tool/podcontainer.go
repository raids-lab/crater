package tool

import (
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	handler.Registers = append(handler.Registers, NewAPIServerMgr)
}

type APIServerMgr struct {
	name           string
	config         *rest.Config
	client         client.Client
	kubeClient     kubernetes.Interface
	serviceManager crclient.ServiceManagerInterface // Add serviceManager field
}

// PortMapping 结构体定义
type PortMapping struct {
	Name          string // 端口名称
	ContainerPort int32  // 容器内部端口
	ServicePort   int32  // Service 暴露的端口
	Prefix        string // Ingress 路径前缀
	ServiceName   string // 该条规则对应的 Service 名称
	IngressName   string // 该条规则对应的 Ingress 名称
}

type (
	// PodIngress represents an ingress rule for a pod
	// 获取时的结构
	PodIngress struct {
		Name   string `json:"name" binding:"required"` // Rule name
		Port   int32  `json:"port" binding:"required"` // Port to expose
		Prefix string `json:"prefix"`                  // Unique prefix for access
	}

	// 创建和删除时的结构
	PodIngressMgr struct {
		Name string `json:"name" binding:"required"` // Rule name
		Port int32  `json:"port" binding:"required"` // Port to expose
	}

	// PodIngressResp represents the response for ingress operations
	PodIngressResp struct {
		Ingresses []PodIngress `json:"ingresses"` // List of ingress rules
	}

	PodNodeport struct {
		Name          string `json:"name" binding:"required"`          // Rule name
		ContainerPort int32  `json:"containerPort" binding:"required"` // ContainPort to expose
		Address       string `json:"address"`                          // address for access
		NodePort      int32  `json:"nodePort" binding:"required"`      // NodePort to expose
		ServiceName   string
	}

	PodNodeportMgr struct {
		Name          string `json:"name" binding:"required"`          // Rule name
		ContainerPort int32  `json:"containerPort" binding:"required"` // ContatinerPort to expose
	}

	PodNodeportResp struct {
		NodePorts []PodNodeport `json:"nodeports"` // List of nodeport rules
	}
)

const (
	IngressLabelKey      = "ingress.crater.raids.io"
	NodePortLabelKey     = "nodeport.crater.raids.io"
	AnnotationKeyOpenSSH = "crater.raids.io/open-ssh"
	SSHContainerPort     = 22
)

func NewAPIServerMgr(conf *handler.RegisterConfig) handler.Manager {
	return &APIServerMgr{
		name:           "namespaces",
		config:         conf.KubeConfig,
		client:         conf.Client,
		kubeClient:     conf.KubeClient,
		serviceManager: conf.ServiceManager,
	}
}

func (mgr *APIServerMgr) GetName() string { return mgr.name }

func (mgr *APIServerMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *APIServerMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET(":namespace/pods/:name/events", mgr.GetPodEvents)

	g.GET(":namespace/pods/:name/containers", mgr.GetPodContainers)
	g.GET(":namespace/pods/:name/containers/:container/log", mgr.GetPodContainerLog)
	g.GET(":namespace/pods/:name/containers/:container/log/stream", mgr.StreamPodContainerLog)

	// New ingress routes
	g.GET(":namespace/pods/:name/ingresses", mgr.GetPodIngresses)
	g.POST(":namespace/pods/:name/ingresses", mgr.CreatePodIngress)
	g.DELETE(":namespace/pods/:name/ingresses", mgr.DeletePodIngress)

	// New nodeport routes
	g.GET(":namespace/pods/:name/nodeports", mgr.GetPodNodeports)
	g.POST(":namespace/pods/:name/nodeports", mgr.CreatePodNodeport)
	g.DELETE(":namespace/pods/:name/nodeports", mgr.DeletePodNodeport)
}

// GetJobNameFromPod retrieves the job name from a pod's owner references
func (mgr *APIServerMgr) GetJobNameFromPod(c *gin.Context, namespace, podName string) (string, error) {
	// Fetch the pod details
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: namespace, Name: podName}, &pod); err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	// Loop through the owner references to find the Job
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "Job" {
			return owner.Name, nil
		}
	}

	return "", fmt.Errorf("pod does not belong to a Job")
}

// GetPodIngresses retrieves the ingress rules for a pod
// GetPodIngresses godoc
//
//	@Summary		获取Pod的Ingress规则
//	@Description	通过Pod注解获取相关的Ingress规则
//	@Tags			Pod
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			namespace	path		string								true	"命名空间"
//	@Param			name		path		string								true	"Pod名称"
//	@Success		200			{object}	resputil.Response[PodIngressResp]	"Pod Ingress规则列表"
//	@Failure		400			{object}	resputil.Response[any]				"请求参数错误"
//	@Failure		404			{object}	resputil.Response[any]				"Pod未找到"
//	@Failure		500			{object}	resputil.Response[any]				"其他错误"
//	@Router			/v1/namespaces/{namespace}/pods/{name}/ingresses [get]
func (mgr *APIServerMgr) GetPodIngresses(c *gin.Context) {
	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Fetch the pod
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Extract relevant labels for filtering
	selector := mgr.serviceManager.GenerateLabels(pod.Labels)

	// Fetch services matching the filtered labels
	services := &v1.ServiceList{}
	listOptions := client.MatchingLabels(selector)
	if err := mgr.client.List(c, services, listOptions, client.InNamespace(req.Namespace)); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Filter ClusterIP services
	clusterIPServices := make([]*v1.Service, 0, len(services.Items))
	for i := range services.Items {
		if services.Items[i].Spec.Type == v1.ServiceTypeClusterIP {
			clusterIPServices = append(clusterIPServices, &services.Items[i])
		}
	}

	// Fetch ingresses in the namespace
	ingresses := &networkingv1.IngressList{}
	if err := mgr.client.List(c, ingresses, client.InNamespace(req.Namespace)); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Match ingresses with services
	ingressRules := mgr.matchIngressesWithServices(clusterIPServices, ingresses)

	resputil.Success(c, PodIngressResp{Ingresses: ingressRules})
}

// Extracted helper function to reduce cyclomatic complexity
func (mgr *APIServerMgr) matchIngressesWithServices(
	clusterIPServices []*v1.Service,
	ingresses *networkingv1.IngressList,
) []PodIngress {
	var ingressRules []PodIngress
	for i := range ingresses.Items {
		ingress := &ingresses.Items[i]
		for j := range clusterIPServices {
			svc := clusterIPServices[j]
			if isIngressServiceMatch(ingress, svc) {
				for _, rule := range ingress.Spec.Rules {
					host := rule.Host // Extract the host from the ingress rule
					for _, httpPath := range rule.HTTP.Paths {
						if httpPath.Backend.Service.Name != svc.Name {
							continue
						}
						var containerPort int32
						for _, svcPort := range svc.Spec.Ports {
							if svcPort.Port == httpPath.Backend.Service.Port.Number {
								if svcPort.TargetPort.Type == intstr.Int {
									containerPort = svcPort.TargetPort.IntVal
								}
								break
							}
						}
						// Retrieve port.Name from the ingress annotation
						portName, ok := ingress.Annotations[crclient.AnnotationKeyPortName] // Use the constant
						if !ok {
							continue
						}

						// Construct the full ingress path
						ingressPath := fmt.Sprintf("https://%s%s", host, httpPath.Path)
						ingressRules = append(ingressRules, PodIngress{
							Name:   portName,
							Port:   containerPort,
							Prefix: ingressPath,
						})
					}
				}
			}
		}
	}
	return ingressRules
}

// 检查 Ingress 是否与指定的 Service 匹配
func isIngressServiceMatch(ingress *networkingv1.Ingress, svc *v1.Service) bool {
	for _, rule := range ingress.Spec.Rules {
		for _, httpPath := range rule.HTTP.Paths {
			if httpPath.Backend.Service.Name == svc.Name {
				return true
			}
		}
	}
	return false
}

// CreatePodIngress creates a new ingress rule for a pod
// CreatePodIngress godoc
//
//	@Summary		创建新的Pod Ingress规则
//	@Description	为指定Pod创建新的Ingress规则，规则名称和端口号必须唯一
//	@Tags			Pod
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			namespace	path		string							true	"命名空间"
//	@Param			name		path		string							true	"Pod名称"
//	@Param			body		body		PodIngressMgr					true	"Ingress规则内容"
//	@Success		200			{object}	resputil.Response[PodIngress]	"成功创建的Ingress规则"
//	@Failure		400			{object}	resputil.Response[any]			"请求参数错误或规则冲突"
//	@Failure		404			{object}	resputil.Response[any]			"Pod未找到"
//	@Failure		500			{object}	resputil.Response[any]			"其他错误"
//	@Router			/v1/namespaces/{namespace}/pods/{name}/ingresses [post]
func (mgr *APIServerMgr) CreatePodIngress(c *gin.Context) {
	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var ingressMgr PodIngressMgr
	if err := c.ShouldBindJSON(&ingressMgr); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Fetch the pod
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Use CreateIngress to create ingress and service
	podSelector := pod.Labels
	port := &v1.ServicePort{
		Name:       ingressMgr.Name,
		Port:       ingressMgr.Port,
		TargetPort: intstr.FromInt(int(ingressMgr.Port)),
		Protocol:   v1.ProtocolTCP,
	}

	username := pod.Labels[crclient.LabelKeyTaskUser]

	ingressPath, err := mgr.serviceManager.CreateIngress(
		c,
		[]metav1.OwnerReference{
			*metav1.NewControllerRef(&pod, v1.SchemeGroupVersion.WithKind("Pod")),
		},
		podSelector,
		port,
		config.GetConfig().Host,
		username,
	)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create ingress: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, map[string]string{"ingressPath": ingressPath})
}

// DeletePodIngress deletes an ingress rule for a pod
// DeletePodIngress godoc
//
//	@Summary		删除Pod的Ingress规则
//	@Description	根据规则名称删除指定的Ingress规则，同时删除关联的Service和Ingress
//	@Tags			Pod
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			namespace	path		string						true	"命名空间"
//	@Param			name		path		string						true	"Pod名称"
//	@Param			body		body		PodIngressMgr				true	"要删除的Ingress规则"
//	@Success		200			{object}	resputil.Response[string]	"Ingress规则删除成功"
//	@Failure		400			{object}	resputil.Response[any]		"请求参数错误或Ingress规则未找到"
//	@Failure		404			{object}	resputil.Response[any]		"Pod未找到"
//	@Failure		500			{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/namespaces/{namespace}/pods/{name}/ingresses [delete]
func (mgr *APIServerMgr) DeletePodIngress(c *gin.Context) {
	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var ingressMgr PodIngressMgr
	if err := c.ShouldBindJSON(&ingressMgr); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Fetch the pod
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to fetch pod: %v", err), resputil.NotSpecified)
		return
	}

	// Retrieve the service to delete
	serviceToDelete, err := mgr.getServiceForPort(c, req.Namespace, pod.Labels, ingressMgr.Port)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Retrieve the ingress associated with the service
	ingressToDelete, err := mgr.getIngressForService(c, req.Namespace, serviceToDelete.Name)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Delete the Service
	if err := mgr.deleteService(c, req.Namespace, serviceToDelete.Name); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to delete service: %v", err), resputil.NotSpecified)
		return
	}

	// Delete the Ingress
	if err := mgr.deleteIngress(c, req.Namespace, ingressToDelete.Name); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to delete ingress: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, "Ingress rule and associated resources deleted successfully")
}

func (mgr *APIServerMgr) getServiceForPort(c *gin.Context, namespace string, labels map[string]string, port int32) (*v1.Service, error) {
	selector := mgr.serviceManager.GenerateLabels(labels)

	// Fetch services matching the filtered labels
	services := &v1.ServiceList{}
	listOptions := client.MatchingLabels(selector)
	if err := mgr.client.List(c, services, listOptions, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	// Find the service matching the specified port
	for i := range services.Items {
		svc := &services.Items[i]
		if svc.Spec.Type == v1.ServiceTypeClusterIP {
			for _, svcPort := range svc.Spec.Ports {
				if svcPort.Port == port {
					return svc, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no matching service found for port: %d", port)
}

func (mgr *APIServerMgr) getIngressForService(c *gin.Context, namespace, serviceName string) (*networkingv1.Ingress, error) {
	// Fetch ingresses in the namespace
	ingresses := &networkingv1.IngressList{}
	if err := mgr.client.List(c, ingresses, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	// Find the ingress associated with the service
	for i := range ingresses.Items {
		ing := &ingresses.Items[i]
		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil && path.Backend.Service.Name == serviceName {
					return ing, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no matching ingress found for service: %s", serviceName)
}

// GetPodNodeports retrieves the NodePort rules for a pod
// GetPodNodeports godoc
//
//	@Summary		获取Pod的NodePort规则
//	@Description	通过Pod的labels选择相关的Service并获取NodePort规则
//	@Tags			Pod
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			namespace	path		string								true	"命名空间"
//	@Param			name		path		string								true	"Pod名称"
//	@Success		200			{object}	resputil.Response[PodNodeportResp]	"Pod NodePort规则列表"
//	@Failure		400			{object}	resputil.Response[any]				"请求参数错误"
//	@Failure		404			{object}	resputil.Response[any]				"Pod未找到"
//	@Failure		500			{object}	resputil.Response[any]				"其他错误"
//	@Router			/v1/namespaces/{namespace}/pods/{name}/nodeports [get]
func (mgr *APIServerMgr) GetPodNodeports(c *gin.Context) {
	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Determine selector based on job type
	selector := mgr.serviceManager.GenerateLabels(pod.Labels) // Use serviceManager from APIServerMgr

	// Fetch services matching the filtered labels
	services := &v1.ServiceList{}
	listOptions := client.MatchingLabels(selector)
	if err := mgr.client.List(c, services, listOptions, client.InNamespace(req.Namespace)); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	var nodeportRules []PodNodeport
	for i := range services.Items {
		svc := &services.Items[i]
		if svc.Spec.Type == v1.ServiceTypeNodePort {
			for _, port := range svc.Spec.Ports {
				if port.NodePort != 0 {
					// Retrieve port.Name from the service annotation
					portName, ok := svc.Annotations[crclient.AnnotationKeyPortName]
					if !ok {
						portName = svc.Name // Fallback to service name if annotation is missing
					}

					nodeportRules = append(nodeportRules, PodNodeport{
						Name:          portName,
						NodePort:      port.NodePort,
						Address:       pod.Status.HostIP,
						ContainerPort: port.TargetPort.IntVal,
					})
				}
			}
		}
	}

	resputil.Success(c, PodNodeportResp{NodePorts: nodeportRules})
}

// CreatePodNodeport creates a new NodePort rule for a pod
// CreatePodNodeport godoc
//
//	@Summary		创建新的Pod NodePort规则
//	@Description	为指定Pod创建新的NodePort规则，规则名称和端口号必须唯一
//	@Tags			Pod
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			namespace	path		string							true	"命名空间"
//	@Param			name		path		string							true	"Pod名称"
//	@Param			body		body		PodNodeportMgr					true	"NodePort规则内容"
//	@Success		200			{object}	resputil.Response[PodNodeport]	"成功创建的NodePort规则"
//	@Failure		400			{object}	resputil.Response[any]			"请求参数错误或规则冲突"
//	@Failure		404			{object}	resputil.Response[any]			"Pod未找到"
//	@Failure		500			{object}	resputil.Response[any]			"其他错误"
//	@Router			/v1/namespaces/{namespace}/pods/{name}/nodeports [post]
func (mgr *APIServerMgr) CreatePodNodeport(c *gin.Context) {
	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var nodeportMgr PodNodeportMgr
	if err := c.ShouldBindJSON(&nodeportMgr); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Fetch the pod
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Use CreateNodePort to create a NodePort service
	podSelector := pod.Labels
	port := &v1.ServicePort{
		Name:       nodeportMgr.Name,
		Port:       nodeportMgr.ContainerPort,
		TargetPort: intstr.FromInt(int(nodeportMgr.ContainerPort)),
		Protocol:   v1.ProtocolTCP,
	}

	host, nodePort, err := mgr.serviceManager.CreateNodePort(
		c,
		[]metav1.OwnerReference{
			*metav1.NewControllerRef(&pod, v1.SchemeGroupVersion.WithKind("Pod")),
		},
		podSelector,
		port,
		pod.Labels[crclient.LabelKeyTaskUser],
	)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create nodeport service: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, map[string]any{
		"host":     host,
		"nodePort": nodePort,
	})
}

// DeletePodNodeport deletes a NodePort rule for a pod
// DeletePodNodeport godoc
//
//	@Summary		删除Pod的NodePort规则
//	@Description	根据规则名称删除指定的NodePort规则，同时删除关联的Service
//	@Tags			Pod
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			namespace	path		string						true	"命名空间"
//	@Param			name		path		string						true	"Pod名称"
//	@Param			body		body		PodNodeportMgr				true	"要删除的NodePort规则"
//	@Success		200			{object}	resputil.Response[string]	"NodePort规则删除成功"
//	@Failure		400			{object}	resputil.Response[any]		"请求参数错误或NodePort规则未找到"
//	@Failure		404			{object}	resputil.Response[any]		"Pod未找到"
//	@Failure		500			{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/namespaces/{namespace}/pods/{name}/nodeports [delete]
func (mgr *APIServerMgr) DeletePodNodeport(c *gin.Context) {
	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var nodeportMgr PodNodeportMgr
	if err := c.ShouldBindJSON(&nodeportMgr); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Fetch the pod
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to fetch pod: %v", err), resputil.NotSpecified)
		return
	}

	// Determine selector based on job type
	selector := mgr.serviceManager.GenerateLabels(pod.Labels)

	// Fetch services matching the filtered labels
	services := &v1.ServiceList{}
	listOptions := client.MatchingLabels(selector)
	if err := mgr.client.List(c, services, listOptions, client.InNamespace(req.Namespace)); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to list services: %v", err), resputil.NotSpecified)
		return
	}

	// Find the service matching the specified container port
	var serviceToDelete *v1.Service
	for i := range services.Items {
		svc := &services.Items[i]
		if svc.Spec.Type == v1.ServiceTypeNodePort {
			for _, port := range svc.Spec.Ports {
				if port.TargetPort.IntVal == nodeportMgr.ContainerPort {
					serviceToDelete = svc
					break
				}
			}
		}
		if serviceToDelete != nil {
			break
		}
	}

	if serviceToDelete == nil {
		resputil.Error(c, fmt.Sprintf("No matching service found for container port: %d", nodeportMgr.ContainerPort), resputil.NotSpecified)
		return
	}

	// Delete the Service
	if err := mgr.deleteService(c, req.Namespace, serviceToDelete.Name); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to delete service: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, "NodePort rule and associated service deleted successfully")
}

func (mgr *APIServerMgr) deleteResource(c *gin.Context, namespace, name, resourceType string, obj client.Object) error {
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: namespace, Name: name}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			log.Printf("%s %s in namespace %s not found, nothing to delete", resourceType, name, namespace)
			return nil
		}
		return fmt.Errorf("failed to fetch %s: %w", resourceType, err)
	}

	if err := mgr.client.Delete(c, obj); err != nil {
		return fmt.Errorf("failed to delete %s: %w", resourceType, err)
	}

	log.Printf("%s %s in namespace %s successfully deleted", resourceType, name, namespace)
	return nil
}

func (mgr *APIServerMgr) deleteService(c *gin.Context, namespace, serviceName string) error {
	return mgr.deleteResource(c, namespace, serviceName, "Service", &v1.Service{})
}

func (mgr *APIServerMgr) deleteIngress(c *gin.Context, namespace, ingressName string) error {
	return mgr.deleteResource(c, namespace, ingressName, "Ingress", &networkingv1.Ingress{})
}

func (mgr *APIServerMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.PUT(":namespace/pods/:name/resources", mgr.UpdatePodResources)
	g.PUT(":namespace/pods/:name/containers/:container/resources", mgr.UpdatePodResources)
}

type (
	PodContainerReq struct {
		Namespace string `uri:"namespace" binding:"required"`
		PodName   string `uri:"name" binding:"required"`
	}

	ContainerStatus struct {
		Name            string            `json:"name"`
		Image           string            `json:"image"`
		State           v1.ContainerState `json:"state"`
		Resources       v1.ResourceList   `json:"resources,omitempty"`
		RestartCount    int32             `json:"restartCount"`
		IsInitContainer bool              `json:"isInitContainer"`
		Node            string            `json:"node"`
	}

	PodContainersResp struct {
		Containers []ContainerStatus `json:"containers"`
	}
)

// GetPodContainers godoc
//
//	@Summary		获取Pod的容器列表
//	@Description	获取Pod的容器列表
//	@Tags			Pod
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			namespace	path		string					true	"命名空间"
//	@Param			name		path		string					true	"Pod名称"
//	@Success		200			{object}	resputil.Response[any]	"Pod容器列表"
//	@Failure		400			{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500			{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/namespaces/{namespace}/pods/{name}/containers [get]
func (mgr *APIServerMgr) GetPodContainers(c *gin.Context) {
	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		if strings.Contains(err.Error(), "unknown namespace") {
			// try to get the pod from the kube client
			if podPtr, err := mgr.kubeClient.CoreV1().Pods(req.Namespace).Get(c, req.PodName, metav1.GetOptions{}); err != nil {
				resputil.Error(c, err.Error(), resputil.NotSpecified)
				return
			} else {
				pod = *podPtr
			}
		} else {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
	}

	// TODO(liyilong): Check if the user has permission to view the pod.

	var resourceRequestMap = make(map[string]v1.ResourceList)
	for i := range pod.Spec.InitContainers {
		container := &pod.Spec.InitContainers[i]
		resourceRequestMap[container.Name] = container.Resources.Requests
	}
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		resourceRequestMap[container.Name] = container.Resources.Requests
	}

	containers := make([]ContainerStatus, len(pod.Spec.Containers)+len(pod.Spec.InitContainers))
	for i := range pod.Status.InitContainerStatuses {
		cs := &pod.Status.InitContainerStatuses[i]
		containers[i] = ContainerStatus{
			Name:            cs.Name,
			Image:           cs.Image,
			State:           cs.State,
			Resources:       resourceRequestMap[cs.Name],
			RestartCount:    cs.RestartCount,
			IsInitContainer: true,
			Node:            pod.Spec.NodeName,
		}
	}
	for i := range pod.Status.ContainerStatuses {
		cs := &pod.Status.ContainerStatuses[i]
		containers[len(pod.Spec.InitContainers)+i] = ContainerStatus{
			Name:            cs.Name,
			Image:           cs.Image,
			State:           cs.State,
			Resources:       resourceRequestMap[cs.Name],
			RestartCount:    cs.RestartCount,
			IsInitContainer: false,
			Node:            pod.Spec.NodeName,
		}
	}
	resputil.Success(c, PodContainersResp{Containers: containers})
}

type (
	PodContainerLogURIReq struct {
		// from uri
		Namespace     string `uri:"namespace" binding:"required"`
		PodName       string `uri:"name" binding:"required"`
		ContainerName string `uri:"container" binding:"required"`
	}

	PodContainerLogQueryReq struct {
		// from query
		TailLines  *int64 `form:"tailLines"`
		Timestamps bool   `form:"timestamps"`
		Follow     bool   `form:"follow"`
		Previous   bool   `form:"previous"`
	}
)

// GetPodEvents godoc
//
//	@Summary		获取Pod的事件
//	@Description	获取Pod的事件
//	@Tags			Pod
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			namespace	path		string					true	"命名空间"
//	@Param			name		path		string					true	"任务名称"
//	@Success		200			{object}	resputil.Response[any]	"Pod事件列表"
//	@Failure		400			{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		404			{object}	resputil.Response[any]	"任务未找到"
//	@Failure		500			{object}	resputil.Response[any]	"其他错误"
//	@Router			/v1/namespaces/{namespace}/pods/{name}/events [get]
func (mgr *APIServerMgr) GetPodEvents(c *gin.Context) {
	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// get events
	events, err := mgr.kubeClient.CoreV1().Events(req.Namespace).List(c, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", req.PodName),
		TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
	})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, events.Items)
}

type (
	EditPodResourceURIReq struct {
		Namespace string `uri:"namespace" binding:"required"`
		PodName   string `uri:"name"      binding:"required"`
		Container string `uri:"container"` // 可选
	}

	EditPodResourceRequest struct {
		Resources v1.ResourceList `json:"resources" binding:"required"`
	}
)

// UpdatePodResource godoc
//
//	@Summary		edit pod's resources(cpu, mem)
//	@Description	edit pod's resources(cpu, mem)
//	@Tags			Operations
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	resputil.Response[any]	"Success"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/namespaces/{namespace}/pods/{name}/containers/{container}/resources [put]
func (mgr *APIServerMgr) UpdatePodResources(c *gin.Context) {
	// URI 绑定
	var uri EditPodResourceURIReq
	if err := c.ShouldBindUri(&uri); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Body 绑定
	var req EditPodResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{
		Namespace: uri.Namespace,
		Name:      uri.PodName,
	}, &pod); err != nil {
		resputil.Error(c, fmt.Sprintf("fetch pod: %v", err), resputil.NotSpecified)
		return
	}

	if !isJobPod(&pod) {
		resputil.Error(c, fmt.Sprintf("%s is not job pod", pod.Name), resputil.NotSpecified)
		return
	}

	jobName := pod.OwnerReferences[0].Name
	if !checkUserPermissionForJob(c, jobName) {
		resputil.Error(c, fmt.Sprintf("no permission for pod: %s", pod.Name), resputil.NotSpecified)
		return
	}

	var containerName *string
	if uri.Container != "" {
		containerName = &uri.Container
	}

	if err := mgr.EditPodResource(c, &pod, containerName, req.Resources); err != nil {
		resputil.Error(c, fmt.Sprintf("Edit resources: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, "Successfully updated pod resource")
}

func isJobPod(pod *v1.Pod) bool {
	if len(pod.OwnerReferences) == 0 {
		return false
	}

	owner := pod.OwnerReferences[0]
	return owner.Kind == "Job" && owner.APIVersion == "batch.volcano.sh/v1alpha1"
}

func checkUserPermissionForJob(c *gin.Context, jobName string) bool {
	token := util.GetToken(c)
	if token.RolePlatform == model.RoleAdmin {
		return true
	}

	jobDB := query.Job

	job, err := jobDB.WithContext(c).Where(jobDB.Name.Eq(jobName)).First()
	if err != nil {
		return false
	}

	if job.AccountID != token.AccountID {
		return false
	}
	if job.UserID != token.UserID {
		return false
	}

	return true
}

func (mgr *APIServerMgr) EditPodResource(
	c *gin.Context,
	pod *v1.Pod,
	containerName *string,
	resources v1.ResourceList,
) error {
	ctx := c.Request.Context()

	// 确定要修改的 container 索引
	idx := 0
	if containerName != nil {
		found := false
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Name == *containerName {
				idx = i
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("container %q not found in pod %s/%s",
				*containerName, pod.Namespace, pod.Name)
		}
	}

	// 只改目标容器的 CPU/Memory
	ctr := &pod.Spec.Containers[idx]
	if qty, ok := resources[v1.ResourceCPU]; ok {
		ctr.Resources.Requests[v1.ResourceCPU] = qty
		ctr.Resources.Limits[v1.ResourceCPU] = qty
	}
	if qty, ok := resources[v1.ResourceMemory]; ok {
		ctr.Resources.Requests[v1.ResourceMemory] = qty
		ctr.Resources.Limits[v1.ResourceMemory] = qty
	}

	// 使用 Update 提交整个 Pod 对象
	updated, err := mgr.kubeClient.CoreV1().Pods(pod.Namespace).
		Update(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update pod %s/%s container[%d] resources: %w",
			pod.Namespace, pod.Name, idx, err)
	}

	klog.Infof("Updated pod %s/%s container[%d]=%q resources=%+v",
		updated.Namespace, updated.Name, idx, updated.Spec.Containers[idx].Name,
		updated.Spec.Containers[idx].Resources)

	return nil
}
