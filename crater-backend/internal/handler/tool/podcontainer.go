package tool

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
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

func (mgr *APIServerMgr) RegisterPublic(g *gin.RouterGroup) {
	// TODO(liyilong): 进行权限控制
	g.GET(":namespace/pods/:name/containers/:container/log/stream", mgr.StreamPodContainerLog)
}

func (mgr *APIServerMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET(":namespace/pods/:name/events", mgr.GetPodEvents)

	g.GET(":namespace/pods/:name/containers", mgr.GetPodContainers)
	g.GET(":namespace/pods/:name/containers/:container/log", mgr.GetPodContainerLog)
	g.GET(":namespace/pods/:name/containers/:container/terminal", mgr.GetPodContainerTerminal)

	// New ingress routes
	g.GET(":namespace/pods/:name/ingresses", mgr.GetPodIngresses)
	g.POST(":namespace/pods/:name/ingresses", mgr.CreatePodIngress)
	g.DELETE(":namespace/pods/:name/ingresses", mgr.DeletePodIngress)

	// New nodeport routes
	g.GET(":namespace/pods/:name/nodeports", mgr.GetPodNodeports)
	g.POST(":namespace/pods/:name/nodeports", mgr.CreatePodNodeport)
	g.DELETE(":namespace/pods/:name/nodeports", mgr.DeletePodNodeport)
}

// 实现流式日志函数
func (mgr *APIServerMgr) StreamPodContainerLog(c *gin.Context) {
	var req PodContainerLogURIReq
	if err := c.ShouldBindUri(&req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	var param PodContainerLogQueryReq
	if err := c.ShouldBindQuery(&param); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 创建日志请求，强制设置Follow为true
	logReq := mgr.kubeClient.CoreV1().Pods(req.Namespace).GetLogs(req.PodName, &v1.PodLogOptions{
		Container:  req.ContainerName,
		Follow:     true, // 强制为true
		Timestamps: param.Timestamps,
		TailLines:  param.TailLines,
	})

	// 使用流式方式获取日志
	stream, err := logReq.Stream(c)
	if err != nil {
		c.SSEvent("error", fmt.Sprintf("获取日志流失败: %v", err))
		c.Writer.Flush()
		return
	}
	defer stream.Close()

	// 设置连接关闭检测
	ctx := c.Request.Context()
	go func() {
		<-ctx.Done()
		stream.Close()
	}()

	// 读取并发送日志
	reader := bufio.NewReader(stream)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				c.SSEvent("error", fmt.Sprintf("读取日志出错: %v", err))
			}
			break
		}

		// 将日志行编码为base64并发送
		encoded := base64.StdEncoding.EncodeToString(line)
		c.SSEvent("message", encoded)
		c.Writer.Flush()
	}
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
// @Summary 获取Pod的Ingress规则
// @Description 通过Pod注解获取相关的Ingress规则
// @Tags Pod
// @Accept json
// @Produce json
// @Security Bearer
// @Param namespace path string true "命名空间"
// @Param name path string true "Pod名称"
// @Success 200 {object} resputil.Response[PodIngressResp] "Pod Ingress规则列表"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 404 {object} resputil.Response[any] "Pod未找到"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/namespaces/{namespace}/pods/{name}/ingresses [get]
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
// @Summary 创建新的Pod Ingress规则
// @Description 为指定Pod创建新的Ingress规则，规则名称和端口号必须唯一
// @Tags Pod
// @Accept json
// @Produce json
// @Security Bearer
// @Param namespace path string true "命名空间"
// @Param name path string true "Pod名称"
// @Param body body PodIngressMgr true "Ingress规则内容"
// @Success 200 {object} resputil.Response[PodIngress] "成功创建的Ingress规则"
// @Failure 400 {object} resputil.Response[any] "请求参数错误或规则冲突"
// @Failure 404 {object} resputil.Response[any] "Pod未找到"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/namespaces/{namespace}/pods/{name}/ingresses [post]
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
// @Summary 删除Pod的Ingress规则
// @Description 根据规则名称删除指定的Ingress规则，同时删除关联的Service和Ingress
// @Tags Pod
// @Accept json
// @Produce json
// @Security Bearer
// @Param namespace path string true "命名空间"
// @Param name path string true "Pod名称"
// @Param body body PodIngressMgr true "要删除的Ingress规则"
// @Success 200 {object} resputil.Response[string] "Ingress规则删除成功"
// @Failure 400 {object} resputil.Response[any] "请求参数错误或Ingress规则未找到"
// @Failure 404 {object} resputil.Response[any] "Pod未找到"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/namespaces/{namespace}/pods/{name}/ingresses [delete]
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

	// Construct service and ingress names
	serviceName := fmt.Sprintf("%s-svc-%s", pod.Labels[crclient.LabelKeyTaskUser], ingressMgr.Name)
	ingressName := fmt.Sprintf("%s-ing-%s", pod.Labels[crclient.LabelKeyTaskUser], uuid.New().String()[:5])

	// Delete the Service
	if err := mgr.deleteService(c, req.Namespace, serviceName); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to delete service: %v", err), resputil.NotSpecified)
		return
	}

	// Delete the Ingress
	if err := mgr.deleteIngress(c, req.Namespace, ingressName); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to delete ingress: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, "Ingress rule and associated resources deleted successfully")
}

// GetPodNodeports retrieves the NodePort rules for a pod
// GetPodNodeports godoc
// @Summary 获取Pod的NodePort规则
// @Description 通过Pod的labels选择相关的Service并获取NodePort规则
// @Tags Pod
// @Accept json
// @Produce json
// @Security Bearer
// @Param namespace path string true "命名空间"
// @Param name path string true "Pod名称"
// @Success 200 {object} resputil.Response[PodNodeportResp] "Pod NodePort规则列表"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 404 {object} resputil.Response[any] "Pod未找到"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/namespaces/{namespace}/pods/{name}/nodeports [get]
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
// @Summary 创建新的Pod NodePort规则
// @Description 为指定Pod创建新的NodePort规则，规则名称和端口号必须唯一
// @Tags Pod
// @Accept json
// @Produce json
// @Security Bearer
// @Param namespace path string true "命名空间"
// @Param name path string true "Pod名称"
// @Param body body PodNodeportMgr true "NodePort规则内容"
// @Success 200 {object} resputil.Response[PodNodeport] "成功创建的NodePort规则"
// @Failure 400 {object} resputil.Response[any] "请求参数错误或规则冲突"
// @Failure 404 {object} resputil.Response[any] "Pod未找到"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/namespaces/{namespace}/pods/{name}/nodeports [post]
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
// @Summary 删除Pod的NodePort规则
// @Description 根据规则名称删除指定的NodePort规则，同时删除关联的Service
// @Tags Pod
// @Accept json
// @Produce json
// @Security Bearer
// @Param namespace path string true "命名空间"
// @Param name path string true "Pod名称"
// @Param body body PodNodeportMgr true "要删除的NodePort规则"
// @Success 200 {object} resputil.Response[string] "NodePort规则删除成功"
// @Failure 400 {object} resputil.Response[any] "请求参数错误或NodePort规则未找到"
// @Failure 404 {object} resputil.Response[any] "Pod未找到"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/namespaces/{namespace}/pods/{name}/nodeports [delete]
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

	// Construct service name
	serviceName := fmt.Sprintf("%s-np-%s", pod.Labels[crclient.LabelKeyTaskUser], nodeportMgr.Name)

	// Delete the Service
	if err := mgr.deleteService(c, req.Namespace, serviceName); err != nil {
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

// DeletePodRule deletes a rule annotation from a Pod
func (mgr *APIServerMgr) DeletePodRule(
	c *gin.Context,
	labelKey string,
	ruleName string,
	targetStruct any,
) error {
	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		return fmt.Errorf("failed to bind URI: %w", err)
	}

	err := mgr.deletePodAnnotation(c, req, labelKey, ruleName, json.Unmarshal, targetStruct)
	if err != nil {
		return fmt.Errorf("failed to delete pod annotation: %w", err)
	}

	return nil
}

func (mgr *APIServerMgr) deletePodAnnotation(
	c *gin.Context,
	req PodContainerReq,
	labelPrefix string,
	ruleName string,
	decodeFunc func([]byte, any) error,
	targetStruct any,
) error {
	// Fetch the pod
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		return fmt.Errorf("failed to fetch pod: %w", err)
	}

	// Locate the annotation to delete
	annotationKey := ""
	for key, value := range pod.Annotations {
		if strings.HasPrefix(key, labelPrefix) {
			// Decode the annotation into the target struct
			if err := decodeFunc([]byte(value), targetStruct); err == nil {
				// 动态检查 targetStruct 类型
				switch rule := targetStruct.(type) {
				case *PodIngress:
					// 如果是 Ingress 规则，检查 Name 是否匹配
					if rule.Name == ruleName {
						annotationKey = key
						break
					}
				case *PodNodeport:
					// 如果是 NodePort 规则，检查 Name 是否匹配
					if rule.Name == ruleName {
						annotationKey = key
						break
					}
				default:
					return fmt.Errorf("unsupported rule type")
				}
			}
		}
	}

	if annotationKey == "" {
		return fmt.Errorf("rule '%s' not found under prefix '%s'", ruleName, labelPrefix)
	}

	// Delete the annotation
	delete(pod.Annotations, annotationKey)

	// Update the pod
	if err := mgr.client.Update(c, &pod); err != nil {
		return fmt.Errorf("failed to update pod annotations: %w", err)
	}

	return nil
}

func (mgr *APIServerMgr) RegisterAdmin(_ *gin.RouterGroup) {}

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
	}

	PodContainersResp struct {
		Containers []ContainerStatus `json:"containers"`
	}
)

// GetPodContainers godoc
// @Summary 获取Pod的容器列表
// @Description 获取Pod的容器列表
// @Tags Pod
// @Accept json
// @Produce json
// @Security Bearer
// @Param namespace path string true "命名空间"
// @Param name path string true "Pod名称"
// @Success 200 {object} resputil.Response[any] "Pod容器列表"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/namespaces/{namespace}/pods/{name}/containers [get]
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
			//nolint:govet // Ignore govet warning about shadowing err.
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

// GetPodContainerLog godoc
// @Summary 获取Pod容器日志
// @Description 获取Pod容器日志
// @Tags Pod
// @Accept json
// @Produce json
// @Security Bearer
// @Param namespace path string true "命名空间"
// @Param name path string true "Pod名称"
// @Param container path string true "容器名称"
// @Param page query int true "页码"
// @Param size query int true "每页数量"
// @Success 200 {object} resputil.Response[any] "Pod容器日志"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/namespaces/{namespace}/pods/{name}/containers/{container}/log [get]
func (mgr *APIServerMgr) GetPodContainerLog(c *gin.Context) {
	// Implementation for fetching and returning the pod container log
	var req PodContainerLogURIReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	var param PodContainerLogQueryReq
	if err := c.ShouldBindQuery(&param); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// 获取指定 Pod 的日志请求
	logReq := mgr.kubeClient.CoreV1().Pods(req.Namespace).GetLogs(req.PodName, &v1.PodLogOptions{
		Container:  req.ContainerName,
		Follow:     param.Follow,
		TailLines:  param.TailLines,
		Timestamps: param.Timestamps,
		Previous:   param.Previous,
	})

	// 获取日志内容
	logData, err := logReq.DoRaw(c)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to get log: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, logData)
}

type (
	PodContainerTerminalReq struct {
		// from uri
		Namespace     string `uri:"namespace" binding:"required"`
		PodName       string `uri:"name" binding:"required"`
		ContainerName string `uri:"container" binding:"required"`
	}
)

const (
	// WriteTimeout specifies the maximum duration for completing a write operation.
	WriteTimeout = 10 * time.Second
	// EndOfTransmission represents the signal for ending the transmission (Ctrl+D).
	EndOfTransmission = "\u0004"
)

// 首先定义终端大小消息的结构
type TerminalMessage struct {
	Op   string `json:"op"`   // 操作类型: "stdin", "stdout", "resize"等
	Data string `json:"data"` // 对于stdin/stdout是内容，对于resize是宽高
	Cols uint16 `json:"cols"` // 列数
	Rows uint16 `json:"rows"` // 行数
}

type streamHandler struct {
	ws       *websocket.Conn
	sizeChan chan remotecommand.TerminalSize
	doneChan chan struct{}
}

// 实现TerminalSizeQueue接口的Next方法
func (h *streamHandler) Next() *remotecommand.TerminalSize {
	select {
	case size := <-h.sizeChan:
		return &size
	case <-h.doneChan:
		return nil
	}
}

func (h *streamHandler) Write(p []byte) (int, error) {
	if err := h.ws.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil {
		// If setting the write deadline fails, return the error immediately.
		return 0, err
	}
	err := h.ws.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}

// References:
// - https://github.com/kubernetes/client-go/issues/554
// - https://github.com/juicedata/juicefs-csi-driver/pull/1053
func (h *streamHandler) Read(p []byte) (int, error) {
	_, message, err := h.ws.ReadMessage()
	if err != nil {
		// Returns "0x04" on error
		return copy(p, EndOfTransmission), err
	}

	// 尝试解析为终端消息
	var msg TerminalMessage
	if err := json.Unmarshal(message, &msg); err == nil {
		// 如果是resize操作
		if msg.Op == "resize" {
			h.sizeChan <- remotecommand.TerminalSize{
				Width:  msg.Cols,
				Height: msg.Rows,
			}
			return 0, nil
		}

		// 如果是stdin操作，使用Data字段
		if msg.Op == "stdin" {
			return copy(p, msg.Data), nil
		}
	}

	return copy(p, message), nil
}

func (mgr *APIServerMgr) GetPodContainerTerminal(c *gin.Context) {
	var req PodContainerTerminalReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	var upgrade = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	// Allow all origins in debug mode
	if gin.Mode() == gin.DebugMode {
		upgrade.CheckOrigin = func(_ *http.Request) bool {
			return true
		}
	}
	ws, err := upgrade.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(c)
	defer cancel()

	stream := &streamHandler{
		ws:       ws,
		sizeChan: make(chan remotecommand.TerminalSize),
		doneChan: make(chan struct{}),
	}
	defer close(stream.doneChan)

	// Reference: https://github.com/juicedata/juicefs-csi-driver/pull/1053
	request := mgr.kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(req.PodName).
		Namespace(req.Namespace).
		SubResource("exec")
	request.VersionedParams(&v1.PodExecOptions{
		Command:   []string{"sh", "-c", "bash || sh"},
		Container: req.ContainerName,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(mgr.config, "POST", request.URL())
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             stream,
		Stdout:            stream,
		Stderr:            stream,
		Tty:               true,
		TerminalSizeQueue: stream,
	})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
}

// GetPodEvents godoc
// @Summary 获取Pod的事件
// @Description 获取Pod的事件
// @Tags Pod
// @Accept json
// @Produce json
// @Security Bearer
// @Param namespace path string true "命名空间"
// @Param name path string true "任务名称"
// @Success 200 {object} resputil.Response[any] "Pod事件列表"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 404 {object} resputil.Response[any] "任务未找到"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/namespaces/{namespace}/pods/{name}/events [get]
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
