package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	inferencesvc "github.com/raids-lab/crater/internal/service/inference"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/raids-lab/crater/pkg/prequeuewatcher"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewKthenaMgr)
}

type KthenaMgr struct {
	metrics             inferenceMetricsReader
	prequeue            *prequeuewatcher.PrequeueWatcher
	name                string
	client              client.Client
	kubeClient          kubernetes.Interface
	configService       *service.ConfigService
	admissionBans       *service.UserBanService
	billing             *service.BillingService
	quota               *service.PrequeueService
	conversationStore   *inferencesvc.ConversationStore
	proxyKthenaRouterFn func(
		ctx context.Context,
		method string,
		targetPath string,
		body []byte,
		headers http.Header,
	) ([]byte, error)
	downloaderImage string
	namespace       string
}

func NewKthenaMgr(conf *RegisterConfig) Manager {
	metrics, _ := conf.PrometheusClient.(inferenceMetricsReader)
	return &KthenaMgr{
		metrics:       metrics,
		name:          "kthena",
		prequeue:      conf.PrequeueWatcher,
		client:        conf.Client,
		kubeClient:    conf.KubeClient,
		configService: conf.ConfigService, admissionBans: conf.UserBanService, billing: conf.BillingService, quota: conf.PrequeueService,
		conversationStore: inferencesvc.NewConversationStore(query.GetDB()),
		namespace:         config.GetConfig().Namespaces.Job,
		downloaderImage:   config.GetConfig().Kthena.DownloaderImage,
	}
}

func (mgr *KthenaMgr) GetName() string { return mgr.name }

func (mgr *KthenaMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *KthenaMgr) RegisterProtected(g *gin.RouterGroup) {
	services := g.Group("inference-services", mgr.requireKthenaInferenceEnabled)
	services.POST("", mgr.UserCreateKthenaService)
	services.GET("capabilities", mgr.UserServingCapabilities)
	services.POST("preview", mgr.UserPreviewServing)
	services.GET("", mgr.UserListKthenaServices)
	services.GET(":name/conversations", mgr.UserListKthenaConversations)
	services.POST(":name/conversations", mgr.UserCreateKthenaConversation)
	// The collection endpoint creates a conversation and its first turn atomically from the client perspective.
	services.POST(":name/conversations/turns", mgr.UserCreateKthenaConversationTurnWithoutSession)
	services.POST(":name/conversations/:sessionID/turns", mgr.UserCreateKthenaConversationTurn)
	services.GET(":name/conversations/:sessionID", mgr.UserGetKthenaConversation)
	services.PATCH(":name/conversations/:sessionID", mgr.UserUpdateKthenaConversation)
	services.DELETE(":name/conversations/:sessionID", mgr.UserDeleteKthenaConversation)
	services.Any(":name/"+kthenaProxyPrefix+"/*path", mgr.UserProxyKthenaService)
	services.GET(":name", mgr.UserGetKthenaService)
	services.PATCH(":name/scale", mgr.UserScaleServing)
	services.PUT(":name", mgr.UserUpdateServing)
	services.GET(":name/yaml", mgr.UserGetKthenaServiceYaml)
	services.GET(":name/metrics", mgr.UserKthenaMetrics)
	services.GET(":name/logs", mgr.UserServingLogs)
	services.GET(":name/diagnostics", mgr.UserKthenaDiagnostics)
	services.POST(":name/reconcile", mgr.UserReconcileKthenaService)
	services.DELETE(":name", mgr.UserDeleteKthenaService)
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

// UserCreateKthenaService godoc
//
//	@Summary		Create inference service
//	@Description	Create a Kthena ModelServing with an owned ModelServer and ModelRoute.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		CreateKthenaReq	true	"Create inference service request"
//	@Success		200		{object}	resputil.Response[KthenaServiceResp]
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/kthena/inference-services [post]
func (mgr *KthenaMgr) UserCreateKthenaService(c *gin.Context) {
	var req CreateKthenaReq
	if err := bindServingRequest(c, &req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid request"))
		return
	}
	token := util.GetToken(c)
	if err := validateCreateKthenaReq(c.Request.Context(), &req, token); err != nil {
		resputil.HandleError(c, err)
		return
	}

	if err := mgr.validateServingSecrets(c.Request.Context(), &req, util.GetToken(c)); err != nil {
		resputil.HandleError(c, err)
		return
	}
	if err := validateServingCapability(&req); err != nil {
		resputil.HandleError(c, err)
		return
	}
	if err := mgr.checkServingAPIs(c.Request.Context()); err != nil {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.Wrap(err, "serving APIs unavailable"))
		return
	}
	if err := mgr.prepareKthenaScheduling(c.Request.Context(), token); err != nil {
		resputil.HandleError(c, err)
		return
	}
	if err := service.CheckWorkloadSubmission(c.Request.Context(),
		mgr.admissionBans,
		mgr.billing,
		mgr.quota,
		token.UserID,
		token.AccountID,
		model.ScheduleTypeNormal); err != nil {
		resputil.HandleError(c, err)
		return
	}
	obj, err := mgr.createNativeKthenaService(c.Request.Context(), &req, token)
	if err != nil {
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

	resp, err := mgr.inferenceServiceToResp(c.Request.Context(), obj)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(
			err,
			"created inference service but failed to parse response",
		))
		return
	}
	resputil.Success(c, resp)
}

// UserListKthenaServices godoc
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
func (mgr *KthenaMgr) UserListKthenaServices(c *gin.Context) {
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

// UserGetKthenaService godoc
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
func (mgr *KthenaMgr) UserGetKthenaService(c *gin.Context) {
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

// UserGetKthenaServiceYaml godoc
//
//	@Summary		Get inference service YAML
//	@Description	Get ModelServing, ModelServer and ModelRoute objects owned by the current user and account.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string	true	"Inference service name"
//	@Success		200		{object}	resputil.Response[any]
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/kthena/inference-services/{name}/yaml [get]
func (mgr *KthenaMgr) UserGetKthenaServiceYaml(c *gin.Context) {
	mgr.getKthenaService(c, false, true)
}

// AdminGetKthenaServiceYaml godoc
//
//	@Summary		Get inference service YAML as admin
//	@Description	Get raw Kthena ModelServing, ModelServer and ModelRoute objects for any inference service managed by Crater.
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

// UserDeleteKthenaService godoc
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
func (mgr *KthenaMgr) UserDeleteKthenaService(c *gin.Context) {
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
