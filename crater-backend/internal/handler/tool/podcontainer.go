package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/crclient"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	handler.Registers = append(handler.Registers, NewAPIServerMgr)
}

type APIServerMgr struct {
	name       string
	config     *rest.Config
	client     client.Client
	kubeClient kubernetes.Interface
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

const IngressLabelKey = "ingress.crater.raids.io"
const NodePortLabelKey = "nodeport.crater.raids.io"
const AnnotationKeyOpenSSH = "crater.raids.io/open-ssh"
const SSHContainerPort = 22

func NewAPIServerMgr(conf *handler.RegisterConfig) handler.Manager {
	return &APIServerMgr{
		name:       "namespaces",
		config:     conf.Kubeconfig,
		client:     conf.Client,
		kubeClient: conf.KubeClient,
	}
}

func (mgr *APIServerMgr) GetName() string { return mgr.name }

func (mgr *APIServerMgr) RegisterPublic(_ *gin.RouterGroup) {}

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

	// Retrieve annotations from the pod
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Extract ingress rules from pod annotations
	var ingressRules []PodIngress
	for key, value := range pod.Annotations {
		if strings.HasPrefix(key, IngressLabelKey) {
			// Parse the value (expected to be JSON format)
			var mapping PortMapping
			if err := json.Unmarshal([]byte(value), &mapping); err == nil {
				// Convert PortMapping to PodIngress
				ingress := PodIngress{
					Name:   mapping.Name,
					Port:   mapping.ContainerPort,
					Prefix: mapping.Prefix,
				}
				ingressRules = append(ingressRules, ingress)
			} else {
				log.Printf("Warning: failed to parse annotation %s: %v", key, err)
			}
		}
	}

	resputil.Success(c, PodIngressResp{Ingresses: ingressRules})
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

	// 获取 Pod 信息
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 校验是否已存在相同端口或规则名的 Ingress
	for key, value := range pod.Annotations {
		if strings.HasPrefix(key, IngressLabelKey) {
			var existingRule PodIngress
			if err := json.Unmarshal([]byte(value), &existingRule); err == nil {
				if existingRule.Port == ingressMgr.Port || existingRule.Name == ingressMgr.Name {
					resputil.BadRequestError(c, "Ingress rule with the same port or name already exists")
					return
				}
			}
		}
	}

	// 调用封装的函数创建和部署 Ingress 规则
	if err := mgr.ProcessPodIngressRule(c.Request.Context(), &pod, ingressMgr); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, ingressMgr)
}

func (mgr *APIServerMgr) ProcessPodIngressRule(ctx context.Context, pod *v1.Pod, ingressMgr PodIngressMgr) error {
	// 获取 "crater.raids.io/task-user" 作为前缀的一部分
	userName, ok := pod.Labels["crater.raids.io/task-user"]
	if !ok {
		return fmt.Errorf("label crater.raids.io/task-user not found")
	}

	// 创建 ingress 规则
	ingress := PodIngress{
		Name:   ingressMgr.Name,
		Port:   ingressMgr.Port,
		Prefix: fmt.Sprintf("/ingress/%s-%s", userName, uuid.New().String()[:5]), // 生成唯一前缀
	}

	// 将 ingress 转换为 crclient.PodIngress 类型
	crclientIngress := crclient.PodIngress{
		Name:   ingress.Name,
		Port:   ingress.Port,
		Prefix: ingress.Prefix,
	}

	// 确保 mgr.client 是 controller-runtime 的 client
	if mgr.client == nil {
		return fmt.Errorf("APIServerMgr client is nil")
	}

	// 调用 CreateCustomForwardingRule
	err := crclient.CreateCustomForwardingRule(ctx, mgr.client, pod, crclientIngress)
	if err != nil {
		return fmt.Errorf("failed to create ingress rule: %w", err)
	}

	return nil
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
	var ingressMgr PodIngressMgr
	if err := c.ShouldBindJSON(&ingressMgr); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// 获取Pod
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to fetch pod: %v", err), resputil.NotSpecified)
		return
	}

	// 从Annotation中获取规则
	annotationKey := fmt.Sprintf("%s/%s", IngressLabelKey, ingressMgr.Name)
	ruleData, exists := pod.Annotations[annotationKey]
	if !exists {
		resputil.Error(c, "Annotation for the specified rule not found", resputil.NotSpecified)
		return
	}

	// 解析规则
	var rule PortMapping
	if err := json.Unmarshal([]byte(ruleData), &rule); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to parse annotation data: %v", err), resputil.NotSpecified)
		return
	}

	// 删除关联的 Service
	if err := mgr.deleteService(c, req.Namespace, rule.ServiceName); err != nil {
		log.Printf("Warning: failed to delete service: %v", err)
	}

	// 删除关联的 Ingress
	if err := mgr.deleteIngress(c, req.Namespace, rule.IngressName); err != nil {
		log.Printf("Warning: failed to delete ingress: %v", err)
	}

	// 删除 Pod 的 Annotation
	delete(pod.Annotations, annotationKey)
	if err := mgr.client.Update(c, &pod); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to update pod annotations: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, "Ingress rule and associated resources deleted successfully")
}

// GetPodNodeports retrieves the NodePort rules for a pod
// GetPodNodeports godoc
// @Summary 获取Pod的NodePort规则
// @Description 通过Pod注解获取相关的NodePort规则
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

	var nodeportRules []PodNodeport
	for key, value := range pod.Annotations {
		if strings.HasPrefix(key, NodePortLabelKey) {
			var rule PodNodeport
			if err := json.Unmarshal([]byte(value), &rule); err == nil {
				// 获取 Service
				service := &v1.Service{}
				serviceName := fmt.Sprintf("%s-%s", req.PodName, rule.Name)
				if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: serviceName}, service); err == nil {
					// 获取 NodePort
					if len(service.Spec.Ports) > 0 && service.Spec.Ports[0].NodePort != 0 {
						rule.NodePort = service.Spec.Ports[0].NodePort
					}
				}

				// 同步最新 NodePort 和 HostIP
				rule.Address = pod.Status.HostIP
				nodeportRules = append(nodeportRules, rule)

				// 更新注解
				//nolint:gosec // 确定此处不会出现超过 int32 范围的情况
				if currentNodeport, _ := strconv.Atoi(pod.Annotations[key]); rule.NodePort != int32(currentNodeport) {
					nodeportJSON, _ := json.Marshal(rule)
					pod.Annotations[key] = string(nodeportJSON)
					// 异步更新 Pod 注解
					if err := mgr.client.Update(c, &pod); err != nil {
						log.Printf("Failed to asynchronously update pod annotations: %v", err)
					}
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
	// 使用 Gin 的参数绑定方法获取 URI 参数
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var nodeportMgr PodNodeportMgr
	// 使用 Gin 的参数绑定方法获取 JSON 参数
	if err := c.ShouldBindJSON(&nodeportMgr); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// 调用 ProcessPodNodeport
	nodeport, err := mgr.ProcessPodNodeport(c, req, nodeportMgr)
	if err != nil {
		if strings.Contains(err.Error(), "NodePort rule with the same name already exists") {
			resputil.BadRequestError(c, err.Error())
		} else {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
		}
		return
	}

	// 返回成功响应
	resputil.Success(c, nodeport)
}

// 获取 Pod
func (mgr *APIServerMgr) getPodByName(ctx context.Context, namespace, podName string) (*v1.Pod, error) {
	if namespace == "" || podName == "" {
		return nil, fmt.Errorf("namespace or pod name is empty")
	}

	var pod v1.Pod
	if err := mgr.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, &pod); err != nil {
		return nil, fmt.Errorf("failed to get Pod: %w", err)
	}

	// 确保 Pod 的 Annotations 字段被初始化
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	return &pod, nil
}

// 检查 NodePort 规则是否已存在
func getExistingNodePortRule(pod *v1.Pod, ruleName string) (*PodNodeport, bool) {
	for key := range pod.Annotations {
		if strings.HasPrefix(key, NodePortLabelKey) {
			var existingRule PodNodeport
			if err := json.Unmarshal([]byte(pod.Annotations[key]), &existingRule); err == nil {
				if existingRule.Name == ruleName {
					return &existingRule, true
				}
			}
		}
	}
	return nil, false
}

// 创建 NodePort Service
func (mgr *APIServerMgr) createNodePortService(ctx context.Context, pod *v1.Pod,
	containerPort int32) (nodePort int32, serviceName string, err error) {
	if containerPort == 0 {
		err = fmt.Errorf("invalid container port")
		return
	}

	nodePort, serviceName, err = crclient.CreateServiceForNodePort(ctx, mgr.client, pod, containerPort)
	if err != nil || nodePort == 0 || serviceName == "" {
		err = fmt.Errorf("failed to create NodePort service: %w", err)
	}
	return
}

// 更新 Pod Annotations
func updatePodAnnotations(pod *v1.Pod, nodeport *PodNodeport) {
	annotationKey := fmt.Sprintf("%s/%s", NodePortLabelKey, nodeport.Name)
	nodeportJSON, _ := json.Marshal(nodeport)
	pod.Annotations[annotationKey] = string(nodeportJSON)

	// SSH 特殊处理
	if nodeport.ContainerPort == SSHContainerPort {
		pod.Annotations[AnnotationKeyOpenSSH] = "true"
		fmt.Printf("Successfully changed AnnotationKeyOpenSSH from false to true, Pod: %s\n", pod.Name)
	}
}

// 处理 Pod NodePort 逻辑
func (mgr *APIServerMgr) ProcessPodNodeport(ctx context.Context, req PodContainerReq, nodeportMgr PodNodeportMgr) (*PodNodeport, error) {
	// 检查 mgr.client 是否已初始化
	if mgr.client == nil {
		return nil, fmt.Errorf("client is not initialized")
	}

	// 获取 Pod
	pod, err := mgr.getPodByName(ctx, req.Namespace, req.PodName)
	if err != nil {
		return nil, err
	}

	// 检查是否已有同名规则
	if existing, found := getExistingNodePortRule(pod, nodeportMgr.Name); found {
		return existing, nil
	}

	// 创建 NodePort Service
	nodePort, serviceName, err := mgr.createNodePortService(ctx, pod, nodeportMgr.ContainerPort)
	if err != nil {
		return nil, err
	}

	// 构建 NodePort 结构体
	nodeport := &PodNodeport{
		Name:          nodeportMgr.Name,
		ContainerPort: nodeportMgr.ContainerPort,
		NodePort:      nodePort,
		Address:       pod.Status.HostIP,
		ServiceName:   serviceName,
	}

	// 更新 Pod Annotations
	updatePodAnnotations(pod, nodeport)

	// 更新 Pod 信息
	if err := mgr.client.Update(ctx, pod); err != nil {
		return nil, fmt.Errorf("failed to update Pod annotations: %w", err)
	}

	return nodeport, nil
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
	var nodeportMgr PodNodeportMgr
	if err := c.ShouldBindJSON(&nodeportMgr); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var req PodContainerReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// 获取 Pod
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: req.Namespace, Name: req.PodName}, &pod); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to fetch pod: %v", err), resputil.NotSpecified)
		return
	}

	// 从 Annotation 中获取规则
	annotationKey := fmt.Sprintf("%s/%s", NodePortLabelKey, nodeportMgr.Name)
	ruleData, exists := pod.Annotations[annotationKey]
	if !exists {
		resputil.Error(c, "Annotation for the specified rule not found", resputil.NotSpecified)
		return
	}

	// 解析规则
	var rule PodNodeport
	if err := json.Unmarshal([]byte(ruleData), &rule); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to parse annotation data: %v", err), resputil.NotSpecified)
		return
	}

	// 删除关联的 Service
	if err := mgr.deleteService(c, req.Namespace, rule.ServiceName); err != nil {
		log.Printf("Warning: failed to delete service: %v", err)
	}

	// 删除 Pod 的 Annotation
	delete(pod.Annotations, annotationKey)
	if err := mgr.client.Update(c, &pod); err != nil {
		resputil.Error(c, fmt.Sprintf("Failed to update pod annotations: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, "NodePort rule and associated service deleted successfully")
}

func (mgr *APIServerMgr) deleteService(c *gin.Context, namespace, serviceName string) error {
	return mgr.deleteResource(c, namespace, serviceName, "Service", &v1.Service{})
}

func (mgr *APIServerMgr) deleteIngress(c *gin.Context, namespace, ingressName string) error {
	return mgr.deleteResource(c, namespace, ingressName, "Ingress", &networkingv1.Ingress{})
}

func (mgr *APIServerMgr) deleteResource(
	c *gin.Context,
	namespace string,
	name string,
	resourceType string,
	obj client.Object,
) error {
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
