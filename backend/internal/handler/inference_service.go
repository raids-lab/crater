package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

const (
	inferenceServiceLabelManagedBy = "crater.raids.io/managed-by"
	inferenceServiceLabelUserID    = "crater.raids.io/user-id"
	inferenceServiceLabelAccountID = "crater.raids.io/account-id"
	inferenceServiceManagedByValue = "inference-service"

	inferenceServiceAnnotationUsername = "crater.raids.io/user"
	inferenceServiceAnnotationAccount  = "crater.raids.io/account"
	inferenceServiceAnnotationSource   = "crater.raids.io/model-source"
	inferenceServiceAnnotationModelID  = "crater.raids.io/platform-model-id"

	kthenaNamespace                       = "kthena-system"
	kthenaRouterService                   = "kthena-router"
	kthenaProxyPrefix                     = "openai"
	kthenaSchedulerName                   = "volcano"
	kthenaSpecTypeKey                     = "type"
	kthenaWorkloadServingAPIGroup         = "workload.serving.volcano.sh"
	kthenaNetworkingServingAPIGroup       = "networking.serving.volcano.sh"
	kthenaAPIVersion                      = "v1alpha1"
	kthenaKindModelBooster                = "ModelBooster"
	kthenaKindModelBoosterList            = "ModelBoosterList"
	kthenaKindModelServing                = "ModelServing"
	kthenaKindModelServingList            = "ModelServingList"
	kthenaKindModelRoute                  = "ModelRoute"
	kthenaKindModelRouteList              = "ModelRouteList"
	kthenaKindModelServer                 = "ModelServer"
	kthenaKindModelServerList             = "ModelServerList"
	kthenaBackendVLLM                     = "vLLM"
	kthenaDefaultCacheURI                 = "hostpath:///tmp/cache"
	kthenaDefaultWorkerMemory             = "4Gi"
	kthenaEnvValueKey                     = "value"
	kthenaResourceCPUKey                  = "cpu"
	kthenaResourceMemoryKey               = "memory"
	kthenaPhasePending                    = "Pending"
	kthenaPhaseReady                      = "Ready"
	kthenaPhaseActive                     = "Active"
	kthenaPhaseDegraded                   = "Degraded"
	kthenaPhaseProgressing                = "Progressing"
	kthenaDiagnosticLevelWarning          = "warning"
	kthenaMaxReplicas               int64 = 1_000_000
	kthenaRelatedResourceCapacity         = 4
	kthenaLogVerbosity                    = 4
	kthenaMaxDiagnostics                  = 10
	kthenaDefaultLogTailLines       int64 = 80

	inferenceModelSourcePlatform = "platform"
	inferenceModelSourceExternal = "external"
)

var (
	modelBoosterGVK = schema.GroupVersionKind{
		Group:   kthenaWorkloadServingAPIGroup,
		Version: kthenaAPIVersion,
		Kind:    kthenaKindModelBooster,
	}
	modelBoosterListGVK = schema.GroupVersionKind{
		Group:   kthenaWorkloadServingAPIGroup,
		Version: kthenaAPIVersion,
		Kind:    kthenaKindModelBoosterList,
	}
	modelServingListGVK = schema.GroupVersionKind{
		Group:   kthenaWorkloadServingAPIGroup,
		Version: kthenaAPIVersion,
		Kind:    kthenaKindModelServingList,
	}
	modelRouteListGVK = schema.GroupVersionKind{
		Group:   kthenaNetworkingServingAPIGroup,
		Version: kthenaAPIVersion,
		Kind:    kthenaKindModelRouteList,
	}
	modelServerListGVK = schema.GroupVersionKind{
		Group:   kthenaNetworkingServingAPIGroup,
		Version: kthenaAPIVersion,
		Kind:    kthenaKindModelServerList,
	}
	inferenceServiceNameRE = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewKthenaMgr)
}

type KthenaMgr struct {
	name                string
	client              client.Client
	kubeClient          kubernetes.Interface
	configService       *service.ConfigService
	conversationStore   *kthenaConversationStore
	proxyKthenaRouterFn func(
		ctx context.Context,
		method string,
		targetPath string,
		body []byte,
		headers http.Header,
	) ([]byte, error)
	namespace string
}

func NewKthenaMgr(conf *RegisterConfig) Manager {
	return &KthenaMgr{
		name:              "kthena",
		client:            conf.Client,
		kubeClient:        conf.KubeClient,
		configService:     conf.ConfigService,
		conversationStore: newKthenaConversationStore(query.GetDB()),
		namespace:         config.GetConfig().Namespaces.Job,
	}
}

func (mgr *KthenaMgr) GetName() string { return mgr.name }

func (mgr *KthenaMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *KthenaMgr) RegisterProtected(g *gin.RouterGroup) {
	services := g.Group("inference-services", mgr.requireKthenaInferenceEnabled)
	services.POST("", mgr.CreateKthenaService)
	services.GET("", mgr.ListKthenaServices)
	services.GET(":name/conversations", mgr.ListKthenaConversations)
	services.POST(":name/conversations", mgr.CreateKthenaConversation)
	// Register the static `turns` route before the parameterized session route
	// so an old client may send an empty sessionId in the JSON body.
	services.POST(":name/conversations/turns", mgr.CreateKthenaConversationTurnWithoutSession)
	services.POST(":name/conversations/:sessionID/turns", mgr.CreateKthenaConversationTurn)
	services.GET(":name/conversations/:sessionID", mgr.GetKthenaConversation)
	services.PATCH(":name/conversations/:sessionID", mgr.UpdateKthenaConversation)
	services.DELETE(":name/conversations/:sessionID", mgr.DeleteKthenaConversation)
	services.Any(":name/"+kthenaProxyPrefix+"/*path", mgr.ProxyKthenaService)
	services.GET(":name", mgr.GetKthenaService)
	services.GET(":name/yaml", mgr.GetKthenaServiceYaml)
	services.DELETE(":name", mgr.DeleteKthenaService)
}

func (mgr *KthenaMgr) RegisterAdmin(g *gin.RouterGroup) {
	services := g.Group("inference-services", mgr.requireKthenaInferenceEnabled)
	services.GET("", mgr.AdminListKthenaServices)
	services.GET(":name", mgr.AdminGetKthenaService)
	services.GET(":name/yaml", mgr.AdminGetKthenaServiceYaml)
	services.DELETE(":name", mgr.AdminDeleteKthenaService)
}

func (mgr *KthenaMgr) requireKthenaInferenceEnabled(c *gin.Context) {
	if mgr.configService == nil || !mgr.configService.IsKthenaInferenceEnabled(c.Request.Context()) {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("Kthena inference feature is disabled"))
		c.Abort()
		return
	}
	c.Next()
}

type KthenaWorkerReq struct {
	Image    string            `json:"image" binding:"required"`
	Replicas int64             `json:"replicas"`
	Pods     int64             `json:"pods"`
	CPU      string            `json:"cpu"`
	Memory   string            `json:"memory"`
	GPU      string            `json:"gpu"`
	GPUModel string            `json:"gpuModel"`
	Config   map[string]string `json:"config,omitempty"`
}

type CreateKthenaReq struct {
	Name            string                           `json:"name" binding:"required"`
	ModelSource     string                           `json:"modelSource"`
	PlatformModelID uint                             `json:"platformModelId"`
	ModelURI        string                           `json:"modelURI"`
	ServedModel     string                           `json:"servedModel"`
	BackendType     string                           `json:"backendType"`
	CacheURI        string                           `json:"cacheURI"`
	Replicas        int64                            `json:"replicas"`
	Env             map[string]string                `json:"env,omitempty"`
	Worker          KthenaWorkerReq                  `json:"worker" binding:"required"`
	Selectors       []corev1.NodeSelectorRequirement `json:"selectors,omitempty"`
	Tolerations     []corev1.Toleration              `json:"tolerations,omitempty"`
}

type KthenaServiceResp struct {
	Name              string             `json:"name"`
	Namespace         string             `json:"namespace"`
	Owner             string             `json:"owner"`
	UserInfo          model.UserInfo     `json:"userInfo"`
	ModelSource       string             `json:"modelSource"`
	PlatformModelID   uint               `json:"platformModelId"`
	ModelURI          string             `json:"modelURI"`
	ServedModel       string             `json:"servedModel"`
	BackendType       string             `json:"backendType"`
	CacheURI          string             `json:"cacheURI"`
	Replicas          int64              `json:"replicas"`
	WorkerImage       string             `json:"workerImage"`
	WorkerReplicas    int64              `json:"workerReplicas"`
	WorkerCPU         string             `json:"workerCPU"`
	WorkerMemory      string             `json:"workerMemory"`
	WorkerGPU         string             `json:"workerGPU"`
	WorkerGPUModel    string             `json:"workerGPUModel"`
	Env               map[string]string  `json:"env"`
	WorkerConfig      map[string]string  `json:"workerConfig"`
	Phase             string             `json:"phase"`
	Conditions        []map[string]any   `json:"conditions"`
	Resources         []KthenaResource   `json:"resources"`
	RuntimePods       []KthenaRuntimePod `json:"runtimePods"`
	Diagnostics       []KthenaDiagnostic `json:"diagnostics"`
	Access            KthenaAccess       `json:"access"`
	Labels            map[string]string  `json:"labels"`
	CreationTimestamp time.Time          `json:"createdAt"`
}

type KthenaResource struct {
	Kind       string           `json:"kind"`
	Name       string           `json:"name"`
	Namespace  string           `json:"namespace"`
	Phase      string           `json:"phase"`
	Ready      bool             `json:"ready"`
	Conditions []map[string]any `json:"conditions"`
}

type KthenaRuntimePod struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	NodeName        string `json:"nodeName"`
	PodIP           string `json:"podIP,omitempty"`
	HostIP          string `json:"hostIP,omitempty"`
	Phase           string `json:"phase"`
	Ready           bool   `json:"ready"`
	Restarts        int32  `json:"restarts"`
	ReadyContainers int    `json:"readyContainers"`
	TotalContainers int    `json:"totalContainers"`
}

type KthenaAccess struct {
	ModelName       string `json:"modelName"`
	ProxyBaseURL    string `json:"proxyBaseURL"`
	InternalBaseURL string `json:"internalBaseURL"`
	NodePortURL     string `json:"nodePortURL,omitempty"`
	RouterService   string `json:"routerService"`
	RouteName       string `json:"routeName,omitempty"`
	ServerName      string `json:"serverName,omitempty"`
}

type KthenaDiagnostic struct {
	Level     string    `json:"level"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	Pod       string    `json:"pod,omitempty"`
	Container string    `json:"container,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

type KthenaProxyReq struct {
	Model       string         `json:"model,omitempty"`
	Messages    []ChatMessage  `json:"messages,omitempty"`
	Prompt      any            `json:"prompt,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	MaxTokens   *int64         `json:"max_tokens,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
	Extra       map[string]any `json:"-"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CreateKthenaService godoc
//
//	@Summary		Create inference service
//	@Description	Create a Kthena ModelBooster-backed inference service.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		CreateKthenaReq	true	"Create inference service request"
//	@Success		200		{object}	resputil.Response[KthenaServiceResp]
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/kthena/inference-services [post]
func (mgr *KthenaMgr) CreateKthenaService(c *gin.Context) {
	var req CreateKthenaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid request"))
		return
	}
	token := util.GetToken(c)
	if err := validateCreateKthenaReq(c.Request.Context(), &req, token); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}

	obj := buildModelBoosterObject(&req, token, mgr.namespace)
	if err := mgr.client.Create(c.Request.Context(), obj); err != nil {
		if errors.IsAlreadyExists(err) {
			resputil.HandleError(c, bizerr.Conflict.ResourceAlreadyExists.Wrap(
				err,
				fmt.Sprintf("inference service %q already exists", req.Name),
			))
			return
		}
		klog.Errorf("create inference service failed: %v", err)
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "create inference service failed"))
		return
	}

	resp, err := mgr.modelBoosterToResp(c.Request.Context(), obj)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(
			err,
			"created inference service but failed to parse response",
		))
		return
	}
	resputil.Success(c, resp)
}

// ListKthenaServices godoc
//
//	@Summary		List my inference services
//	@Description	List Kthena inference services owned by the current user and account.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[[]KthenaServiceResp]
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/kthena/inference-services [get]
func (mgr *KthenaMgr) ListKthenaServices(c *gin.Context) {
	token := util.GetToken(c)
	services, err := mgr.listKthenaServices(
		c.Request.Context(),
		client.MatchingLabels{
			inferenceServiceLabelManagedBy: inferenceServiceManagedByValue,
			inferenceServiceLabelUserID:    strconv.FormatUint(uint64(token.UserID), 10),
			inferenceServiceLabelAccountID: strconv.FormatUint(uint64(token.AccountID), 10),
		},
	)
	if err != nil {
		klog.Errorf("list inference services failed: %v", err)
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "list inference services failed"))
		return
	}
	resputil.Success(c, services)
}

// AdminListKthenaServices godoc
//
//	@Summary		List all inference services
//	@Description	List all Kthena inference services managed by Crater.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[[]KthenaServiceResp]
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/kthena/inference-services [get]
func (mgr *KthenaMgr) AdminListKthenaServices(c *gin.Context) {
	services, err := mgr.listKthenaServices(
		c.Request.Context(),
		client.MatchingLabels{inferenceServiceLabelManagedBy: inferenceServiceManagedByValue},
	)
	if err != nil {
		klog.Errorf("admin list inference services failed: %v", err)
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "list inference services failed"))
		return
	}
	resputil.Success(c, services)
}

// GetKthenaService godoc
//
//	@Summary		Get inference service
//	@Description	Get a Kthena inference service owned by the current user and account.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string	true	"Inference service name"
//	@Success		200		{object}	resputil.Response[KthenaServiceResp]
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/kthena/inference-services/{name} [get]
func (mgr *KthenaMgr) GetKthenaService(c *gin.Context) {
	mgr.getKthenaService(c, false, false)
}

// AdminGetKthenaService godoc
//
//	@Summary		Get inference service as admin
//	@Description	Get any Kthena inference service managed by Crater.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string	true	"Inference service name"
//	@Success		200		{object}	resputil.Response[KthenaServiceResp]
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/kthena/inference-services/{name} [get]
func (mgr *KthenaMgr) AdminGetKthenaService(c *gin.Context) {
	mgr.getKthenaService(c, true, false)
}

// GetKthenaServiceYaml godoc
//
//	@Summary		Get inference service YAML
//	@Description	Get raw Kthena ModelBooster object for an inference service owned by the current user and account.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string	true	"Inference service name"
//	@Success		200		{object}	resputil.Response[any]
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/kthena/inference-services/{name}/yaml [get]
func (mgr *KthenaMgr) GetKthenaServiceYaml(c *gin.Context) {
	mgr.getKthenaService(c, false, true)
}

// AdminGetKthenaServiceYaml godoc
//
//	@Summary		Get inference service YAML as admin
//	@Description	Get raw Kthena ModelBooster object for any inference service managed by Crater.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string	true	"Inference service name"
//	@Success		200		{object}	resputil.Response[any]
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/kthena/inference-services/{name}/yaml [get]
func (mgr *KthenaMgr) AdminGetKthenaServiceYaml(c *gin.Context) {
	mgr.getKthenaService(c, true, true)
}

// DeleteKthenaService godoc
//
//	@Summary		Delete inference service
//	@Description	Delete a Kthena inference service owned by the current user and account.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string	true	"Inference service name"
//	@Success		200		{object}	resputil.Response[string]
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/kthena/inference-services/{name} [delete]
func (mgr *KthenaMgr) DeleteKthenaService(c *gin.Context) {
	mgr.deleteKthenaService(c, false)
}

// AdminDeleteKthenaService godoc
//
//	@Summary		Delete inference service as admin
//	@Description	Delete any Kthena inference service managed by Crater.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string	true	"Inference service name"
//	@Success		200		{object}	resputil.Response[string]
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/kthena/inference-services/{name} [delete]
func (mgr *KthenaMgr) AdminDeleteKthenaService(c *gin.Context) {
	mgr.deleteKthenaService(c, true)
}

func (mgr *KthenaMgr) listKthenaServices(
	ctx context.Context,
	labels client.MatchingLabels,
) ([]KthenaServiceResp, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(modelBoosterListGVK)
	if err := mgr.client.List(ctx, list, client.InNamespace(mgr.namespace), labels); err != nil {
		return nil, err
	}

	services := make([]KthenaServiceResp, 0, len(list.Items))
	for i := range list.Items {
		resp, err := mgr.modelBoosterToResp(ctx, &list.Items[i])
		if err != nil {
			klog.Warningf(
				"skip malformed ModelBooster %s/%s: %v",
				list.Items[i].GetNamespace(),
				list.Items[i].GetName(),
				err,
			)
			continue
		}
		services = append(services, *resp)
	}
	return services, nil
}

func (mgr *KthenaMgr) getKthenaService(c *gin.Context, admin, raw bool) {
	obj, ok := mgr.loadKthenaService(c, admin)
	if !ok {
		return
	}
	if raw {
		resputil.Success(c, obj.Object)
		return
	}
	resp, err := mgr.modelBoosterToResp(c.Request.Context(), obj)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, "failed to parse inference service"))
		return
	}
	resputil.Success(c, resp)
}

func (mgr *KthenaMgr) deleteKthenaService(c *gin.Context, admin bool) {
	obj, ok := mgr.loadKthenaService(c, admin)
	if !ok {
		return
	}
	if err := mgr.client.Delete(c.Request.Context(), obj); err != nil {
		klog.Errorf("delete inference service failed: %v", err)
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "delete inference service failed"))
		return
	}
	resputil.Success(c, "inference service deleted")
}

// ProxyKthenaService godoc
//
//	@Summary		Proxy OpenAI-compatible inference request
//	@Description	Proxy an OpenAI-compatible request to kthena-router for an inference service owned by the current user and account.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string				true	"Inference service name"
//	@Param			path	path		string				true	"OpenAI-compatible API path"
//	@Param			request	body		KthenaProxyReq	false	"OpenAI-compatible request body"
//	@Success		200		{object}	any
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		404		{object}	any				"Model route or runtime pod not found"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/kthena/inference-services/{name}/openai/{path} [post]
func (mgr *KthenaMgr) ProxyKthenaService(c *gin.Context) {
	obj, ok := mgr.loadKthenaService(c, false)
	if !ok {
		return
	}

	targetPath := strings.TrimPrefix(c.Param("path"), "/")
	if targetPath == "" {
		targetPath = "v1/chat/completions"
	}
	if !strings.HasPrefix(targetPath, "v1/") {
		targetPath = "v1/" + targetPath
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "failed to read request body"))
		return
	}
	body, err = withDefaultModel(body, servedModelFromModelBooster(obj))
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}

	rawResp, err := mgr.proxyKthenaRouter(c.Request.Context(), c.Request.Method, targetPath, body, c.Request.Header)
	if err != nil {
		klog.Errorf("proxy inference request failed: %v", err)
		// DoRaw preserves the router response body even for non-2xx responses.
		// Return that status and body to the caller so errors such as missing
		// runtime pods remain actionable instead of becoming a generic 500.
		if len(rawResp) > 0 {
			c.Data(kthenaProxyHTTPStatus(err), "application/json", rawResp)
			return
		}
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "proxy inference request failed"))
		return
	}
	c.Data(http.StatusOK, "application/json", rawResp)
}

func (mgr *KthenaMgr) proxyKthenaRouter(
	ctx context.Context,
	method string,
	targetPath string,
	body []byte,
	headers http.Header,
) ([]byte, error) {
	if mgr.proxyKthenaRouterFn != nil {
		return mgr.proxyKthenaRouterFn(ctx, method, targetPath, body, headers)
	}
	if mgr.kubeClient == nil {
		return nil, bizerr.Internal.K8sServiceError.New("kubernetes client is not initialized")
	}
	req := mgr.kubeClient.CoreV1().RESTClient().
		Verb(method).
		Namespace(kthenaNamespace).
		Resource("services").
		Name(kthenaRouterService + ":http").
		SubResource("proxy").
		Suffix(strings.Split(targetPath, "/")...)
	for key, values := range headers {
		canonicalKey := http.CanonicalHeaderKey(key)
		switch canonicalKey {
		case "Authorization", "Cookie", "Host", "Content-Length":
			continue
		}
		for _, value := range values {
			req.SetHeader(canonicalKey, value)
		}
	}
	req.SetHeader("Content-Type", "application/json")
	if len(body) > 0 {
		req.Body(body)
	}
	return req.DoRaw(ctx)
}

func kthenaProxyHTTPStatus(err error) int {
	type apiStatus interface {
		Status() metav1.Status
	}
	if statusErr, ok := err.(apiStatus); ok {
		statusCode := int(statusErr.Status().Code)
		if statusCode >= http.StatusBadRequest && statusCode <= 599 {
			return statusCode
		}
	}
	return http.StatusBadGateway
}

func (mgr *KthenaMgr) loadKthenaService(c *gin.Context, admin bool) (*unstructured.Unstructured, bool) {
	var req struct {
		Name string `uri:"name" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid service name"))
		return nil, false
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(modelBoosterGVK)
	if err := mgr.client.Get(c.Request.Context(), client.ObjectKey{Namespace: mgr.namespace, Name: req.Name}, obj); err != nil {
		if errors.IsNotFound(err) {
			resputil.HandleError(c, bizerr.NotFound.K8sResourceNotFound.Wrap(err, "inference service not found"))
			return nil, false
		}
		klog.Errorf("get inference service failed: %v", err)
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "get inference service failed"))
		return nil, false
	}
	if !admin && !canAccessKthenaService(c, obj) {
		resputil.HandleError(c, bizerr.NotFound.K8sResourceNotFound.New("inference service not found"))
		return nil, false
	}

	return obj, true
}

func canAccessKthenaService(c *gin.Context, obj *unstructured.Unstructured) bool {
	token := util.GetToken(c)
	labels := obj.GetLabels()
	return labels[inferenceServiceLabelUserID] == strconv.FormatUint(uint64(token.UserID), 10) &&
		labels[inferenceServiceLabelAccountID] == strconv.FormatUint(uint64(token.AccountID), 10)
}

//nolint:gocyclo // Each field is normalized and validated in a fixed order so API clients receive precise errors.
func validateCreateKthenaReq(ctx context.Context, req *CreateKthenaReq, token util.JWTMessage) error {
	req.Name = strings.TrimSpace(req.Name)
	req.ModelSource = strings.TrimSpace(req.ModelSource)
	req.ModelURI = strings.TrimSpace(req.ModelURI)
	req.ServedModel = strings.TrimSpace(req.ServedModel)
	req.BackendType = strings.TrimSpace(req.BackendType)
	req.CacheURI = strings.TrimSpace(req.CacheURI)
	req.Worker.Image = strings.TrimSpace(req.Worker.Image)

	if !inferenceServiceNameRE.MatchString(req.Name) || len(req.Name) > 63 {
		return bizerr.BadRequest.ParameterError.New(
			"name must be a valid Kubernetes name with lowercase letters, digits, or hyphens",
		)
	}
	if req.ModelSource == "" {
		req.ModelSource = inferenceModelSourcePlatform
	}
	if req.ModelSource == inferenceModelSourcePlatform {
		if req.PlatformModelID == 0 {
			return bizerr.BadRequest.MissingParameter.New("platform model is required")
		}
		dataset, err := loadAccessibleModelDataset(ctx, req.PlatformModelID, token)
		if err != nil {
			return err
		}
		req.ModelURI = datasetToKthenaModelURI(dataset)
		if req.ServedModel == "" {
			req.ServedModel = dataset.Name
		}
		req.CacheURI = datasetModelCacheURI()
	} else if req.ModelSource != inferenceModelSourceExternal {
		return bizerr.BadRequest.ParameterError.New("modelSource must be platform or external")
	}
	if req.ModelURI == "" {
		return bizerr.BadRequest.MissingParameter.New("modelURI is required")
	}
	if !strings.HasPrefix(req.ModelURI, "hf://") &&
		!strings.HasPrefix(req.ModelURI, "s3://") &&
		!strings.HasPrefix(req.ModelURI, "pvc://") &&
		!strings.HasPrefix(req.ModelURI, "ms://") {
		return bizerr.BadRequest.ParameterError.New("modelURI must start with hf://, s3://, pvc://, or ms://")
	}
	if req.BackendType == "" {
		req.BackendType = kthenaBackendVLLM
	}
	if req.CacheURI == "" {
		req.CacheURI = kthenaDefaultCacheURI
	}
	if !strings.HasPrefix(req.CacheURI, "hostpath://") && !strings.HasPrefix(req.CacheURI, "pvc://") {
		return bizerr.BadRequest.ParameterError.New("cacheURI must start with hostpath:// or pvc://")
	}
	if req.BackendType != kthenaBackendVLLM {
		// Kthena v1.0.0's published CRD accepts vLLM and
		// vLLMDisaggregated. CreateKthenaReq represents a single server worker,
		// so only the non-disaggregated vLLM shape can be generated here.
		return bizerr.BadRequest.ParameterError.New("backendType must be vLLM for Kthena v1.0.0")
	}
	if req.Replicas <= 0 {
		req.Replicas = 1
	}
	if req.Replicas > kthenaMaxReplicas {
		return bizerr.BadRequest.ParameterError.New("replicas cannot exceed 1000000")
	}
	if req.Worker.Image == "" {
		return bizerr.BadRequest.MissingParameter.New("worker.image is required")
	}
	if req.Worker.Replicas <= 0 {
		req.Worker.Replicas = 1
	}
	if req.Worker.Replicas > kthenaMaxReplicas {
		return bizerr.BadRequest.ParameterError.New("worker.replicas cannot exceed 1000000")
	}
	if req.Worker.Pods <= 0 {
		req.Worker.Pods = 1
	}
	if req.Worker.Pods > kthenaMaxReplicas {
		return bizerr.BadRequest.ParameterError.New("worker.pods cannot exceed 1000000")
	}
	if req.Worker.CPU == "" {
		req.Worker.CPU = "2"
	}
	if req.Worker.Memory == "" {
		req.Worker.Memory = kthenaDefaultWorkerMemory
	}
	if req.Worker.Config == nil {
		req.Worker.Config = map[string]string{}
	}
	if req.ServedModel == "" {
		req.ServedModel = inferServedModelName(req.ModelURI)
	}
	if _, ok := req.Worker.Config["served-model-name"]; !ok && req.ServedModel != "" {
		req.Worker.Config["served-model-name"] = req.ServedModel
	}
	return nil
}

func buildModelBoosterObject(req *CreateKthenaReq, token util.JWTMessage, namespace string) *unstructured.Unstructured {
	labels := map[string]string{
		inferenceServiceLabelManagedBy: inferenceServiceManagedByValue,
		inferenceServiceLabelUserID:    strconv.FormatUint(uint64(token.UserID), 10),
		inferenceServiceLabelAccountID: strconv.FormatUint(uint64(token.AccountID), 10),
	}
	annotations := map[string]string{
		inferenceServiceAnnotationUsername: token.Username,
		inferenceServiceAnnotationAccount:  token.AccountName,
		inferenceServiceAnnotationSource:   req.ModelSource,
	}
	if req.PlatformModelID > 0 {
		annotations[inferenceServiceAnnotationModelID] = strconv.FormatUint(uint64(req.PlatformModelID), 10)
	}

	env := make([]any, 0, len(req.Env))
	for name, value := range req.Env {
		env = append(env, map[string]any{"name": name, kthenaEnvValueKey: value})
	}

	resources := map[string]any{
		"requests": map[string]any{
			kthenaResourceCPUKey:    req.Worker.CPU,
			kthenaResourceMemoryKey: req.Worker.Memory,
		},
		"limits": map[string]any{
			kthenaResourceCPUKey:    req.Worker.CPU,
			kthenaResourceMemoryKey: req.Worker.Memory,
		},
	}
	if strings.TrimSpace(req.Worker.GPU) != "" && req.Worker.GPU != "0" {
		gpuModel := strings.TrimSpace(req.Worker.GPUModel)
		if gpuModel == "" {
			gpuModel = "nvidia.com/gpu"
		}
		resources["limits"].(map[string]any)[gpuModel] = req.Worker.GPU
	}

	workerConfig := make(map[string]any, len(req.Worker.Config))
	for key, value := range req.Worker.Config {
		workerConfig[key] = value
	}

	worker := map[string]any{
		kthenaSpecTypeKey: "server",
		"image":           req.Worker.Image,
		"replicas":        req.Worker.Replicas,
		"pods":            req.Worker.Pods,
		"config":          workerConfig,
		"resources":       resources,
	}
	if affinity := buildWorkerAffinity(req.Selectors); affinity != nil {
		worker["affinity"] = affinity
	}
	if len(req.Tolerations) > 0 {
		worker["tolerations"] = req.Tolerations
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(modelBoosterGVK)
	obj.SetName(req.Name)
	obj.SetNamespace(namespace)
	obj.SetLabels(labels)
	obj.SetAnnotations(annotations)
	obj.Object["spec"] = map[string]any{
		"backend": map[string]any{
			"name":            "backend1",
			kthenaSpecTypeKey: req.BackendType,
			"modelURI":        req.ModelURI,
			"cacheURI":        req.CacheURI,
			"replicas":        req.Replicas,
			"schedulerName":   kthenaSchedulerName,
			"env":             env,
			"workers":         []any{worker},
		},
	}
	return obj
}

func (mgr *KthenaMgr) modelBoosterToResp(
	ctx context.Context,
	obj *unstructured.Unstructured,
) (*KthenaServiceResp, error) {
	backend, ok, err := unstructured.NestedMap(obj.Object, "spec", "backend")
	if err != nil || !ok {
		return nil, bizerr.Internal.K8sServiceError.New("spec.backend is missing")
	}

	workers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "backend", "workers")
	var worker map[string]any
	if len(workers) > 0 {
		worker, _ = workers[0].(map[string]any)
	}
	if worker == nil {
		worker = map[string]any{}
	}

	configMap, _, _ := unstructured.NestedStringMap(worker, "config")
	envMap := envSliceToMap(backend["env"])
	conditions, _ := normalizeConditions(obj)
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if phase == "" {
		phase = conditionPhase(conditions)
	}
	servedModel := configMap["served-model-name"]
	if servedModel == "" {
		servedModel = inferServedModelName(stringValue(backend["modelURI"]))
	}

	resources := mgr.relatedKthenaResources(ctx, obj, servedModel)
	runtimePods := mgr.relatedKthenaRuntimePods(ctx, resources)
	access := mgr.buildKthenaAccess(ctx, obj, resources)
	diagnostics := mgr.relatedKthenaDiagnostics(ctx, resources)
	if phase == kthenaPhasePending {
		phase = aggregateInferencePhase(conditions, resources)
	}
	phase = runtimeAwareInferencePhase(phase, resources, runtimePods)
	if phase == kthenaPhaseDegraded && !hasReadyRuntimePod(runtimePods) {
		diagnostics = append(diagnostics, KthenaDiagnostic{
			Level:    kthenaDiagnosticLevelWarning,
			Reason:   "RuntimePodsUnavailable",
			Message:  kthenaKindModelServing + " reports ready, but no ready runtime pod was found",
			Resource: kthenaKindModelBooster + "/" + obj.GetNamespace() + "/" + obj.GetName(),
		})
	}
	owner, userInfo := kthenaServiceOwner(obj)

	return &KthenaServiceResp{
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		Owner:             owner,
		UserInfo:          userInfo,
		ModelSource:       modelSourceFromObject(obj),
		PlatformModelID:   platformModelIDFromObject(obj),
		ModelURI:          stringValue(backend["modelURI"]),
		ServedModel:       servedModel,
		BackendType:       stringValue(backend[kthenaSpecTypeKey]),
		CacheURI:          stringValue(backend["cacheURI"]),
		Replicas:          int64Value(backend["replicas"]),
		WorkerImage:       stringValue(worker["image"]),
		WorkerReplicas:    int64Value(worker["replicas"]),
		WorkerCPU:         nestedResourceValue(worker, kthenaResourceCPUKey),
		WorkerMemory:      nestedResourceValue(worker, kthenaResourceMemoryKey),
		WorkerGPU:         firstGPUResourceValue(worker),
		WorkerGPUModel:    firstGPUResourceName(worker),
		Env:               envMap,
		WorkerConfig:      configMap,
		Phase:             phase,
		Conditions:        conditions,
		Resources:         resources,
		RuntimePods:       runtimePods,
		Diagnostics:       diagnostics,
		Access:            access,
		Labels:            obj.GetLabels(),
		CreationTimestamp: obj.GetCreationTimestamp().Time,
	}, nil
}

// kthenaServiceOwner reads the creator identity persisted on the ModelBooster
// at creation time. Keeping this data on the Kubernetes object lets a list
// response expose UserInfo without an additional database lookup for each
// deployment.
func kthenaServiceOwner(obj *unstructured.Unstructured) (string, model.UserInfo) {
	if obj == nil {
		return "", model.UserInfo{}
	}
	username := strings.TrimSpace(obj.GetAnnotations()[inferenceServiceAnnotationUsername])
	return username, model.UserInfo{Username: username}
}

func (mgr *KthenaMgr) relatedKthenaResources(
	ctx context.Context,
	booster *unstructured.Unstructured,
	servedModel string,
) []KthenaResource {
	resources := make([]KthenaResource, 0, kthenaRelatedResourceCapacity)
	resources = append(resources, kthenaResourceFromObject(kthenaKindModelBooster, booster))
	resources = append(resources, mgr.listRelatedResources(ctx, modelServingListGVK, kthenaKindModelServing, booster, servedModel)...)
	resources = append(resources, mgr.listRelatedResources(ctx, modelServerListGVK, kthenaKindModelServer, booster, servedModel)...)
	resources = append(resources, mgr.listRelatedResources(ctx, modelRouteListGVK, kthenaKindModelRoute, booster, servedModel)...)
	return resources
}

func (mgr *KthenaMgr) listRelatedResources(
	ctx context.Context,
	gvk schema.GroupVersionKind,
	kind string,
	booster *unstructured.Unstructured,
	servedModel string,
) []KthenaResource {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	if err := mgr.client.List(ctx, list, client.InNamespace(booster.GetNamespace())); err != nil {
		klog.V(kthenaLogVerbosity).Infof(
			"list %s for inference service %s/%s failed: %v",
			kind,
			booster.GetNamespace(),
			booster.GetName(),
			err,
		)
		return nil
	}
	resources := make([]KthenaResource, 0, len(list.Items))
	for i := range list.Items {
		if isRelatedKthenaObject(&list.Items[i], booster, servedModel) {
			resources = append(resources, kthenaResourceFromObject(kind, &list.Items[i]))
		}
	}
	return resources
}

func (mgr *KthenaMgr) relatedKthenaDiagnostics(
	ctx context.Context,
	resources []KthenaResource,
) []KthenaDiagnostic {
	if mgr.kubeClient == nil {
		return nil
	}
	seen := map[string]struct{}{}
	diagnostics := make([]KthenaDiagnostic, 0)
	for _, resource := range resources {
		if resource.Kind != kthenaKindModelServing {
			continue
		}
		pods, err := mgr.kubeClient.CoreV1().Pods(resource.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("modelserving.volcano.sh/name=%s", resource.Name),
		})
		if err != nil {
			klog.V(kthenaLogVerbosity).Infof(
				"list pods for kthena %s %s/%s failed: %v",
				kthenaKindModelServing,
				resource.Namespace,
				resource.Name,
				err,
			)
			continue
		}
		for i := range pods.Items {
			pod := &pods.Items[i]
			key := string(pod.UID)
			if key == "" {
				key = pod.Namespace + "/" + pod.Name
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			diagnostics = append(diagnostics, mgr.diagnosticsFromPod(ctx, pod)...)
			diagnostics = append(diagnostics, mgr.diagnosticsFromPodEvents(ctx, pod)...)
		}
	}
	if len(diagnostics) == 0 {
		return nil
	}
	sort.SliceStable(diagnostics, func(i, j int) bool {
		return diagnostics[i].Timestamp.After(diagnostics[j].Timestamp)
	})
	if len(diagnostics) > kthenaMaxDiagnostics {
		return diagnostics[:kthenaMaxDiagnostics]
	}
	return diagnostics
}

func (mgr *KthenaMgr) relatedKthenaRuntimePods(
	ctx context.Context,
	resources []KthenaResource,
) []KthenaRuntimePod {
	if mgr.kubeClient == nil {
		return nil
	}
	seen := map[string]struct{}{}
	runtimePods := make([]KthenaRuntimePod, 0)
	for _, resource := range resources {
		if resource.Kind != kthenaKindModelServing {
			continue
		}
		pods, err := mgr.kubeClient.CoreV1().Pods(resource.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("modelserving.volcano.sh/name=%s", resource.Name),
		})
		if err != nil {
			klog.V(kthenaLogVerbosity).Infof(
				"list pods for kthena %s %s/%s failed: %v",
				kthenaKindModelServing,
				resource.Namespace,
				resource.Name,
				err,
			)
			continue
		}
		for i := range pods.Items {
			pod := &pods.Items[i]
			key := string(pod.UID)
			if key == "" {
				key = pod.Namespace + "/" + pod.Name
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			runtimePods = append(runtimePods, kthenaRuntimePodFromPod(pod))
		}
	}
	if len(runtimePods) == 0 {
		return nil
	}
	sort.SliceStable(runtimePods, func(i, j int) bool {
		return runtimePods[i].Name < runtimePods[j].Name
	})
	return runtimePods
}

func kthenaRuntimePodFromPod(pod *corev1.Pod) KthenaRuntimePod {
	readyContainers := 0
	totalContainers := len(pod.Status.ContainerStatuses)
	restarts := int32(0)
	for i := range pod.Status.ContainerStatuses {
		status := &pod.Status.ContainerStatuses[i]
		if status.Ready {
			readyContainers++
		}
		restarts += status.RestartCount
	}
	return KthenaRuntimePod{
		Name:            pod.Name,
		Namespace:       pod.Namespace,
		NodeName:        pod.Spec.NodeName,
		PodIP:           pod.Status.PodIP,
		HostIP:          pod.Status.HostIP,
		Phase:           string(pod.Status.Phase),
		Ready:           isPodReady(pod),
		Restarts:        restarts,
		ReadyContainers: readyContainers,
		TotalContainers: totalContainers,
	}
}

func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (mgr *KthenaMgr) diagnosticsFromPod(ctx context.Context, pod *corev1.Pod) []KthenaDiagnostic {
	if pod == nil {
		return nil
	}
	diagnostics := make([]KthenaDiagnostic, 0)
	resource := "Pod/" + pod.Namespace + "/" + pod.Name
	if pod.Status.Phase == corev1.PodPending {
		for _, condition := range pod.Status.Conditions {
			if condition.Type != corev1.PodScheduled || condition.Status != corev1.ConditionFalse {
				continue
			}
			diagnostics = append(diagnostics, KthenaDiagnostic{
				Level:     kthenaDiagnosticLevelWarning,
				Reason:    nonEmpty(condition.Reason, "PodSchedulingFailed"),
				Message:   condition.Message,
				Resource:  resource,
				Pod:       pod.Name,
				Timestamp: condition.LastTransitionTime.Time,
			})
		}
	}
	for i := range pod.Status.InitContainerStatuses {
		status := &pod.Status.InitContainerStatuses[i]
		logTail := mgr.containerLogTail(ctx, pod, status.Name, status.RestartCount)
		diagnostics = append(diagnostics, diagnosticsFromContainerStatus(pod, status, true, logTail)...)
	}
	for i := range pod.Status.ContainerStatuses {
		status := &pod.Status.ContainerStatuses[i]
		logTail := mgr.containerLogTail(ctx, pod, status.Name, status.RestartCount)
		diagnostics = append(diagnostics, diagnosticsFromContainerStatus(pod, status, false, logTail)...)
	}
	return diagnostics
}

func diagnosticsFromContainerStatus(
	pod *corev1.Pod,
	status *corev1.ContainerStatus,
	initContainer bool,
	logTail string,
) []KthenaDiagnostic {
	if pod == nil || status == nil {
		return nil
	}
	resource := "Pod/" + pod.Namespace + "/" + pod.Name
	containerKind := "container"
	reasonPrefix := "Runtime"
	if initContainer {
		containerKind = "initContainer"
		reasonPrefix = "InitContainer"
	}
	if waiting := status.State.Waiting; waiting != nil {
		return []KthenaDiagnostic{{
			Level:     kthenaDiagnosticLevelWarning,
			Reason:    nonEmpty(waiting.Reason, reasonPrefix+"Waiting"),
			Message:   nonEmpty(waiting.Message, fmt.Sprintf("%s %q is waiting", containerKind, status.Name)),
			Details:   logTail,
			Resource:  resource,
			Pod:       pod.Name,
			Container: status.Name,
		}}
	}
	if terminated := status.State.Terminated; terminated != nil && terminated.ExitCode != 0 {
		return []KthenaDiagnostic{{
			Level: "error",
			Reason: nonEmpty(
				terminated.Reason,
				reasonPrefix+"Failed",
			),
			Message: nonEmpty(
				terminated.Message,
				fmt.Sprintf("%s %q terminated with exit code %d", containerKind, status.Name, terminated.ExitCode),
			),
			Details:   logTail,
			Resource:  resource,
			Pod:       pod.Name,
			Container: status.Name,
			Timestamp: terminated.FinishedAt.Time,
		}}
	}
	return nil
}

func (mgr *KthenaMgr) containerLogTail(
	ctx context.Context,
	pod *corev1.Pod,
	container string,
	restartCount int32,
) string {
	if mgr.kubeClient == nil || pod == nil || container == "" {
		return ""
	}
	if logs := mgr.readContainerLogTail(ctx, pod.Namespace, pod.Name, container, false); logs != "" {
		return logs
	}
	if restartCount > 0 {
		return mgr.readContainerLogTail(ctx, pod.Namespace, pod.Name, container, true)
	}
	return ""
}

func (mgr *KthenaMgr) readContainerLogTail(
	ctx context.Context,
	namespace string,
	pod string,
	container string,
	previous bool,
) string {
	tailLines := kthenaDefaultLogTailLines
	req := mgr.kubeClient.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
		Container: container,
		Previous:  previous,
		TailLines: &tailLines,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		klog.V(kthenaLogVerbosity).Infof(
			"read logs for kthena pod %s/%s container %s previous=%t failed: %v",
			namespace,
			pod,
			container,
			previous,
			err,
		)
		return ""
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			klog.V(kthenaLogVerbosity).Infof(
				"close log stream for kthena pod %s/%s container %s failed: %v",
				namespace,
				pod,
				container,
				closeErr,
			)
		}
	}()
	data, err := io.ReadAll(stream)
	if err != nil {
		klog.V(kthenaLogVerbosity).Infof(
			"read log stream for kthena pod %s/%s container %s failed: %v",
			namespace,
			pod,
			container,
			err,
		)
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (mgr *KthenaMgr) diagnosticsFromPodEvents(ctx context.Context, pod *corev1.Pod) []KthenaDiagnostic {
	events, err := mgr.kubeClient.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
	})
	if err != nil {
		klog.V(kthenaLogVerbosity).Infof("list events for kthena pod %s/%s failed: %v", pod.Namespace, pod.Name, err)
		return nil
	}
	diagnostics := make([]KthenaDiagnostic, 0, len(events.Items))
	for i := range events.Items {
		event := &events.Items[i]
		// Pod names are reused when a ModelBooster is recreated. Events from a
		// previous pod with the same name can remain in the namespace and must
		// not make the replacement deployment appear unhealthy.
		if event.InvolvedObject.UID != pod.UID {
			continue
		}
		if event.Type != corev1.EventTypeWarning {
			continue
		}
		diagnostics = append(diagnostics, KthenaDiagnostic{
			Level:     kthenaDiagnosticLevelWarning,
			Reason:    nonEmpty(event.Reason, "PodWarning"),
			Message:   event.Message,
			Resource:  "Pod/" + pod.Namespace + "/" + pod.Name,
			Pod:       pod.Name,
			Timestamp: event.LastTimestamp.Time,
		})
	}
	return diagnostics
}

func isRelatedKthenaObject(obj, booster *unstructured.Unstructured, servedModel string) bool {
	name := obj.GetName()
	boosterName := booster.GetName()
	if name == boosterName || strings.HasPrefix(name, boosterName+"-") {
		return true
	}
	labels := obj.GetLabels()
	if labels["workload.serving.volcano.sh/model-name"] == boosterName {
		return true
	}
	for _, owner := range obj.GetOwnerReferences() {
		if owner.Kind == kthenaKindModelBooster && owner.Name == boosterName {
			return true
		}
	}
	if labels[inferenceServiceLabelManagedBy] == inferenceServiceManagedByValue &&
		labels[inferenceServiceLabelUserID] == booster.GetLabels()[inferenceServiceLabelUserID] &&
		labels[inferenceServiceLabelAccountID] == booster.GetLabels()[inferenceServiceLabelAccountID] {
		return true
	}
	if servedModel != "" && (name == servedModel || strings.Contains(name, sanitizeKubeName(servedModel))) {
		return true
	}
	return false
}

func kthenaResourceFromObject(kind string, obj *unstructured.Unstructured) KthenaResource {
	conditions, _ := normalizeConditions(obj)
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if phase == "" {
		phase = conditionPhase(conditions)
	}
	return KthenaResource{
		Kind:       kind,
		Name:       obj.GetName(),
		Namespace:  obj.GetNamespace(),
		Phase:      phase,
		Ready:      phase == kthenaPhaseReady || phase == kthenaPhaseActive,
		Conditions: conditions,
	}
}

func (mgr *KthenaMgr) buildKthenaAccess(
	ctx context.Context,
	booster *unstructured.Unstructured,
	resources []KthenaResource,
) KthenaAccess {
	access := KthenaAccess{
		ModelName:       booster.GetName(),
		ProxyBaseURL:    fmt.Sprintf("/v1/kthena/inference-services/%s/%s/v1", booster.GetName(), kthenaProxyPrefix),
		InternalBaseURL: fmt.Sprintf("http://%s.%s.svc.cluster.local/v1", kthenaRouterService, kthenaNamespace),
		RouterService:   fmt.Sprintf("%s/%s", kthenaNamespace, kthenaRouterService),
	}
	for _, resource := range resources {
		switch resource.Kind {
		case kthenaKindModelRoute:
			access.RouteName = resource.Name
		case kthenaKindModelServer:
			access.ServerName = resource.Name
		}
	}
	if mgr.kubeClient != nil {
		if svc, err := mgr.kubeClient.CoreV1().Services(kthenaNamespace).Get(ctx, kthenaRouterService, metav1.GetOptions{}); err == nil {
			access.NodePortURL = nodePortURL(svc)
		}
	}
	return access
}

func aggregateInferencePhase(conditions []map[string]any, resources []KthenaResource) string {
	if conditionPhase(conditions) == kthenaPhaseReady {
		return kthenaPhaseReady
	}
	hasRelated := false
	for _, resource := range resources {
		if resource.Kind == kthenaKindModelBooster {
			continue
		}
		hasRelated = true
		if !resource.Ready {
			return kthenaPhaseProgressing
		}
	}
	if hasRelated {
		return kthenaPhaseReady
	}
	return kthenaPhasePending
}

func runtimeAwareInferencePhase(
	phase string,
	resources []KthenaResource,
	runtimePods []KthenaRuntimePod,
) string {
	if phase != kthenaPhaseReady && phase != kthenaPhaseActive {
		return phase
	}
	hasModelServing := false
	for _, resource := range resources {
		if resource.Kind == kthenaKindModelServing {
			hasModelServing = true
			break
		}
	}
	if hasModelServing && !hasReadyRuntimePod(runtimePods) {
		return kthenaPhaseDegraded
	}
	return phase
}

func hasReadyRuntimePod(runtimePods []KthenaRuntimePod) bool {
	for _, pod := range runtimePods {
		if pod.Ready {
			return true
		}
	}
	return false
}

func normalizeConditions(obj *unstructured.Unstructured) ([]map[string]any, error) {
	raw, ok, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !ok {
		return []map[string]any{}, err
	}
	conditions := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if condition, ok := item.(map[string]any); ok {
			conditions = append(conditions, condition)
		}
	}
	return conditions, nil
}

func conditionPhase(conditions []map[string]any) string {
	for _, condition := range conditions {
		if stringValue(condition[kthenaSpecTypeKey]) == kthenaPhaseActive && stringValue(condition["status"]) == "True" {
			return kthenaPhaseReady
		}
	}
	if len(conditions) > 0 {
		return kthenaPhaseProgressing
	}
	return kthenaPhasePending
}

func loadAccessibleModelDataset(ctx context.Context, datasetID uint, token util.JWTMessage) (*model.Dataset, error) {
	d := query.Dataset
	dataset, err := d.WithContext(ctx).Where(d.ID.Eq(datasetID), d.Type.Eq(string(model.DataTypeModel))).First()
	if err != nil {
		return nil, bizerr.NotFound.DataBaseNotFound.New("selected platform model was not found")
	}
	if dataset.UserID == token.UserID {
		return dataset, nil
	}
	ud := query.UserDataset
	if _, err := ud.WithContext(ctx).Where(ud.UserID.Eq(token.UserID), ud.DatasetID.Eq(datasetID)).First(); err == nil {
		return dataset, nil
	}
	qd := query.AccountDataset
	if _, err := qd.WithContext(ctx).Where(qd.AccountID.Eq(token.AccountID), qd.DatasetID.Eq(datasetID)).First(); err == nil {
		return dataset, nil
	}
	return nil, bizerr.Forbidden.PermissionDenied.New("you do not have permission to use the selected platform model")
}

func datasetToKthenaModelURI(dataset *model.Dataset) string {
	url := strings.TrimSpace(dataset.URL)
	if strings.HasPrefix(url, "pvc://") {
		return url
	}
	return "pvc://" + datasetModelCacheMountPath() + "/" + strings.TrimLeft(url, "/")
}

func datasetModelCacheURI() string {
	pvcName := strings.TrimSpace(config.GetConfig().Storage.PVC.ReadWriteMany)
	if pvcName == "" {
		return kthenaDefaultCacheURI
	}
	return "pvc://" + pvcName
}

func datasetModelCacheMountPath() string {
	pvcName := strings.TrimSpace(config.GetConfig().Storage.PVC.ReadWriteMany)
	if pvcName == "" {
		return "/tmp/cache"
	}
	return "/" + strings.Trim(pvcName, "/")
}

func buildWorkerAffinity(selectors []corev1.NodeSelectorRequirement) map[string]any {
	if len(selectors) == 0 {
		return nil
	}
	return map[string]any{
		"nodeAffinity": map[string]any{
			"requiredDuringSchedulingIgnoredDuringExecution": map[string]any{
				"nodeSelectorTerms": []any{
					map[string]any{
						"matchExpressions": selectors,
					},
				},
			},
		},
	}
}

func inferServedModelName(modelURI string) string {
	if strings.Contains(modelURI, "://") {
		parts := strings.Split(modelURI, "://")
		modelURI = parts[len(parts)-1]
	}
	modelURI = strings.TrimSuffix(modelURI, "/")
	parts := strings.Split(modelURI, "/")
	if len(parts) == 0 {
		return modelURI
	}
	return parts[len(parts)-1]
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func nonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func int64Value(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case intstr.IntOrString:
		return int64(v.IntVal)
	case float64:
		return int64(v)
	case string:
		parsed, _ := strconv.ParseInt(v, 10, 64)
		return parsed
	default:
		return 0
	}
}

func servedModelFromModelBooster(obj *unstructured.Unstructured) string {
	workers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "backend", "workers")
	if len(workers) > 0 {
		if worker, ok := workers[0].(map[string]any); ok {
			configMap, _, _ := unstructured.NestedStringMap(worker, "config")
			if servedModel := strings.TrimSpace(configMap["served-model-name"]); servedModel != "" {
				return servedModel
			}
		}
	}
	modelURI, _, _ := unstructured.NestedString(obj.Object, "spec", "backend", "modelURI")
	return inferServedModelName(modelURI)
}

func modelSourceFromObject(obj *unstructured.Unstructured) string {
	source := obj.GetAnnotations()[inferenceServiceAnnotationSource]
	if source == "" {
		return inferenceModelSourceExternal
	}
	return source
}

func platformModelIDFromObject(obj *unstructured.Unstructured) uint {
	value := obj.GetAnnotations()[inferenceServiceAnnotationModelID]
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return uint(parsed)
}

func envSliceToMap(value any) map[string]string {
	env := map[string]string{}
	items, ok := value.([]any)
	if !ok {
		return env
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := stringValue(m["name"])
		if name == "" {
			continue
		}
		env[name] = stringValue(m[kthenaEnvValueKey])
	}
	return env
}

func nestedResourceValue(worker map[string]any, key string) string {
	if value, _, _ := unstructured.NestedString(worker, "resources", "limits", key); value != "" {
		return value
	}
	value, _, _ := unstructured.NestedString(worker, "resources", "requests", key)
	return value
}

func firstGPUResourceName(worker map[string]any) string {
	limits, ok, _ := unstructured.NestedMap(worker, "resources", "limits")
	if !ok {
		return ""
	}
	for key := range limits {
		if key != kthenaResourceCPUKey && key != kthenaResourceMemoryKey {
			return key
		}
	}
	return ""
}

func firstGPUResourceValue(worker map[string]any) string {
	name := firstGPUResourceName(worker)
	if name == "" {
		return "0"
	}
	return nestedResourceValue(worker, name)
}

func withDefaultModel(body []byte, modelName string) ([]byte, error) {
	if len(bytes.TrimSpace(body)) == 0 || strings.TrimSpace(modelName) == "" {
		return body, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, bizerr.BadRequest.InvalidRequest.New("request body must be valid JSON")
	}
	if strings.TrimSpace(stringValue(payload["model"])) == "" {
		payload["model"] = modelName
	}
	return json.Marshal(payload)
}

func sanitizeKubeName(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	lastHyphen := false
	for _, r := range value {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if valid {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func nodePortURL(svc *corev1.Service) string {
	if svc == nil || svc.Spec.Type != corev1.ServiceTypeLoadBalancer && svc.Spec.Type != corev1.ServiceTypeNodePort {
		return ""
	}
	for _, port := range svc.Spec.Ports {
		if port.NodePort <= 0 {
			continue
		}
		nodeHost := firstExternalOrInternalIP(svc)
		if nodeHost == "" {
			return fmt.Sprintf("http://<node-ip>:%d/v1", port.NodePort)
		}
		return fmt.Sprintf("http://%s:%d/v1", nodeHost, port.NodePort)
	}
	return ""
}

func firstExternalOrInternalIP(svc *corev1.Service) string {
	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			return ingress.IP
		}
		if ingress.Hostname != "" {
			return ingress.Hostname
		}
	}
	return ""
}
