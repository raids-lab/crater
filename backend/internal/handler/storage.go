package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/ceph"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/constants"
	"github.com/raids-lab/crater/pkg/patrol"
	"github.com/raids-lab/crater/pkg/storagequota"
)

const (
	toolboxCapabilityTimeout = 20 * time.Second
	capabilityCacheTTL       = time.Minute
	quotaCompensationTimeout = 45 * time.Second
	quotaCompensationRetries = 2
	cephStorageBackend       = "cephfs"
	unknownStorageBackend    = "unknown"
	userStorageScope         = "user"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewStorageMgr)
}

type StorageMgr struct {
	name       string
	kubeClient kubernetes.Interface
	kubeConfig *rest.Config

	capabilityMu        sync.Mutex
	cachedCapabilities  StorageCapabilities
	capabilityExpiresAt time.Time
}

type StorageCapabilities struct {
	Backend                string   `json:"backend"`
	Configured             bool     `json:"configured"`
	QuotaEnabled           bool     `json:"quota_enabled"`
	PVCName                string   `json:"pvc_name"`
	PVCNamespace           string   `json:"pvc_namespace,omitempty"`
	PVName                 string   `json:"pv_name,omitempty"`
	CSIDriver              string   `json:"csi_driver,omitempty"`
	QuotaProvider          string   `json:"quota_provider"`
	StorageServerAvailable bool     `json:"storage_server_available"`
	ToolboxAvailable       bool     `json:"toolbox_available"`
	UsageReadable          bool     `json:"usage_readable"`
	QuotaReadable          bool     `json:"quota_readable"`
	QuotaWritable          bool     `json:"quota_writable"`
	Reasons                []string `json:"reasons,omitempty"`
}

type StorageCapabilitySummary struct {
	QuotaEnabled  bool `json:"quota_enabled"`
	UsageReadable bool `json:"usage_readable"`
	QuotaReadable bool `json:"quota_readable"`
}

type SetUserSpaceQuotaRequest struct {
	Quota int64 `json:"quota" binding:"required"`
}

type quotaUpdateState struct {
	UserID        uint
	LogicalPath   string
	PreviousQuota int64
	AppliedQuota  int64
	CephChanged   bool
}

func NewStorageMgr(conf *RegisterConfig) Manager {
	return &StorageMgr{name: "storage", kubeClient: conf.KubeClient, kubeConfig: conf.KubeConfig}
}

func (mgr *StorageMgr) GetName() string { return mgr.name }

func (mgr *StorageMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *StorageMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/capabilities", mgr.GetCapabilitySummary)
	g.GET("/dirsize/*path", mgr.GetDirectorySize)
	g.GET("/my-quota", mgr.GetMyQuota)
}

func (mgr *StorageMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/capabilities", mgr.GetCapabilities)
	g.GET("/user-spaces", mgr.GetAllUserSpaceSizes)
	g.POST("/user-spaces/refresh", mgr.RefreshUserSpaceSizes)
	g.PUT("/user-spaces/:user/quota", mgr.SetUserSpaceQuota)
}

// GetCapabilitySummary godoc
// @Summary Get storage quota capability summary
// @Description Return non-sensitive storage capability flags for the current user
// @Tags Storage
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[StorageCapabilitySummary]
// @Router /v1/storage/capabilities [get]
func (mgr *StorageMgr) GetCapabilitySummary(c *gin.Context) {
	capabilities := mgr.getCachedCapabilities(false)
	resputil.Success(c, StorageCapabilitySummary{
		QuotaEnabled:  capabilities.QuotaEnabled,
		UsageReadable: capabilities.UsageReadable,
		QuotaReadable: capabilities.QuotaReadable,
	})
}

// GetCapabilities godoc
// @Summary Get detailed storage quota capabilities
// @Description Probe storage quota providers and return diagnostics to platform administrators
// @Tags Storage
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[StorageCapabilities]
// @Router /v1/admin/storage/capabilities [get]
func (mgr *StorageMgr) GetCapabilities(c *gin.Context) {
	resputil.Success(c, mgr.getCachedCapabilities(c.Query("refresh") == "true"))
}

func (mgr *StorageMgr) getCachedCapabilities(force bool) StorageCapabilities {
	mgr.capabilityMu.Lock()
	defer mgr.capabilityMu.Unlock()
	if !force && time.Now().Before(mgr.capabilityExpiresAt) {
		return mgr.cachedCapabilities
	}
	mgr.cachedCapabilities = mgr.detectCapabilities()
	mgr.capabilityExpiresAt = time.Now().Add(capabilityCacheTTL)
	return mgr.cachedCapabilities
}

//nolint:gocyclo // Each capability is independently probed and reported.
func (mgr *StorageMgr) detectCapabilities() StorageCapabilities {
	cfg := config.GetConfig()
	capability := StorageCapabilities{
		Backend:       unknownStorageBackend,
		Configured:    strings.TrimSpace(cfg.Storage.PVC.ReadWriteMany) != "",
		QuotaEnabled:  ceph.StorageQuotaEnabled(),
		PVCName:       strings.TrimSpace(cfg.Storage.PVC.ReadWriteMany),
		QuotaProvider: ceph.StorageQuotaProvider(),
	}
	if !capability.QuotaEnabled {
		capability.Reasons = append(capability.Reasons, "storage quota management is disabled")
		return capability
	}
	if !capability.Configured {
		capability.Reasons = append(capability.Reasons, "storage.pvc.readWriteMany is not configured")
		return capability
	}
	if mgr.kubeClient == nil {
		capability.Reasons = append(capability.Reasons, "kubernetes client is not available")
		return capability
	}

	pvName, pvcNamespace, driver, err := mgr.detectStoragePV(capability.PVCName)
	if err != nil {
		capability.Reasons = append(capability.Reasons, err.Error())
		return capability
	}
	capability.PVName, capability.PVCNamespace, capability.CSIDriver = pvName, pvcNamespace, driver
	if driver != ceph.StorageQuotaCephFSCSIDriver() {
		if driver != "" {
			capability.Backend = driver
		}
		capability.Reasons = append(capability.Reasons, fmt.Sprintf(
			"storage PVC CSI driver %q does not match configured CephFS driver %q",
			driver, ceph.StorageQuotaCephFSCSIDriver(),
		))
		return capability
	}
	capability.Backend = cephStorageBackend
	if capability.QuotaProvider == storagequota.ProviderDisabled {
		capability.Reasons = append(capability.Reasons, "storage quota provider is disabled")
		return capability
	}

	if capability.QuotaProvider != storagequota.ProviderToolbox {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		storageServerCapabilities, storageServerErr := ceph.GetStorageServerQuotaCapabilities(ctx)
		cancel()
		capability.StorageServerAvailable = storageServerErr == nil
		if storageServerErr != nil {
			capability.Reasons = append(capability.Reasons, fmt.Sprintf("storage-server is not available: %v", storageServerErr))
		} else {
			capability.UsageReadable = storageServerCapabilities.UsageReadable
			capability.QuotaReadable = storageServerCapabilities.QuotaReadable
			capability.QuotaWritable = storageServerCapabilities.QuotaWritable
			capability.Reasons = append(capability.Reasons, storageServerCapabilities.Reasons...)
		}
	}

	needsToolbox := capability.QuotaProvider == storagequota.ProviderToolbox ||
		(capability.QuotaProvider == storagequota.ProviderAuto &&
			(!capability.UsageReadable || !capability.QuotaReadable || !capability.QuotaWritable))
	if needsToolbox {
		ctx, cancel := context.WithTimeout(context.Background(), toolboxCapabilityTimeout)
		toolboxCapabilities, toolboxErr := ceph.GetToolboxQuotaCapabilities(
			ctx, mgr.kubeClient, mgr.kubeConfig, ceph.StorageQuotaRookNamespace(),
		)
		cancel()
		capability.ToolboxAvailable = toolboxErr == nil && toolboxCapabilities.UsageReadable
		if toolboxErr != nil {
			capability.Reasons = append(capability.Reasons, fmt.Sprintf("toolbox is not available: %v", toolboxErr))
		} else {
			capability.UsageReadable = capability.UsageReadable || toolboxCapabilities.UsageReadable
			capability.QuotaReadable = capability.QuotaReadable || toolboxCapabilities.QuotaReadable
			capability.QuotaWritable = capability.QuotaWritable || toolboxCapabilities.QuotaWritable
			capability.Reasons = append(capability.Reasons, toolboxCapabilities.Reasons...)
		}
	}
	return capability
}

//nolint:gocritic // The tuple is serialized as three independent diagnostic fields.
func (mgr *StorageMgr) detectStoragePV(pvcName string) (string, string, string, error) {
	pvcNamespace := strings.TrimSpace(config.GetConfig().Namespaces.Job)
	if pvcNamespace == "" {
		return "", "", "", bizerr.Internal.K8sServiceError.New("job namespace is not configured")
	}
	pvc, err := mgr.kubeClient.CoreV1().PersistentVolumeClaims(pvcNamespace).
		Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		return "", pvcNamespace, "", bizerr.Internal.K8sServiceError.Wrap(err, "failed to get storage PVC")
	}
	if pvc.Spec.VolumeName == "" {
		return "", pvc.Namespace, "", bizerr.Internal.K8sServiceError.New("storage PVC is not bound to a PV")
	}
	pv, err := mgr.kubeClient.CoreV1().PersistentVolumes().Get(
		context.TODO(), pvc.Spec.VolumeName, metav1.GetOptions{},
	)
	if err != nil {
		return pvc.Spec.VolumeName, pvc.Namespace, "", bizerr.Internal.K8sServiceError.Wrap(err, "failed to get storage PV")
	}
	if pv.Spec.CSI == nil {
		return pv.Name, pvc.Namespace, "", nil
	}
	return pv.Name, pvc.Namespace, pv.Spec.CSI.Driver, nil
}

func storagePrefixes() ceph.StoragePrefixConfig {
	cfg := config.GetConfig()
	return ceph.StoragePrefixConfig{
		User: cfg.Storage.Prefix.User, Account: cfg.Storage.Prefix.Account, Public: cfg.Storage.Prefix.Public,
	}
}

func (mgr *StorageMgr) authorizedLogicalPath(c *gin.Context) (string, error) {
	scope := strings.Trim(c.Param("path"), "/")
	if scope == "" || strings.Contains(scope, "/") {
		return "", bizerr.BadRequest.ParameterError.New("path must be one of user, account, or public")
	}
	token := util.GetToken(c)
	switch scope {
	case userStorageScope:
		var user model.User
		if err := query.GetDB().Select("id", "space").First(&user, token.UserID).Error; err != nil {
			return "", bizerr.NotFound.DataBaseNotFound.Wrap(err, "user was not found")
		}
		return "/user/" + user.Space, nil
	case "account":
		if token.AccountID == 0 || token.AccountID == model.DefaultAccountID || token.AccountAccessMode == model.AccessModeNA {
			return "", bizerr.Forbidden.PermissionDenied.New("account storage access is not allowed")
		}
		var account model.Account
		if err := query.GetDB().Select("id", "space").First(&account, token.AccountID).Error; err != nil {
			return "", bizerr.NotFound.DataBaseNotFound.Wrap(err, "account was not found")
		}
		return "/account/" + account.Space, nil
	case "public":
		if token.PublicAccessMode == model.AccessModeNA {
			return "", bizerr.Forbidden.PermissionDenied.New("public storage access is not allowed")
		}
		return "/public", nil
	default:
		return "", bizerr.BadRequest.ParameterError.New("path must be one of user, account, or public")
	}
}

// GetDirectorySize godoc
// @Summary Get permitted storage root usage
// @Description Read the current user's user, account, or public storage root usage
// @Tags Storage
// @Produce json
// @Security Bearer
// @Param path path string true "Storage scope: user, account, or public"
// @Success 200 {object} resputil.Response[any]
// @Failure 400 {object} resputil.Response[any]
// @Failure 403 {object} resputil.Response[any]
// @Router /v1/storage/dirsize/{path} [get]
func (mgr *StorageMgr) GetDirectorySize(c *gin.Context) {
	scope := strings.Trim(c.Param("path"), "/")
	logicalPath, err := mgr.authorizedLogicalPath(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	size, err := ceph.GetCephDirectorySize(
		mgr.kubeClient, mgr.kubeConfig, ceph.StorageQuotaRookNamespace(), logicalPath, storagePrefixes(),
	)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to read storage usage"))
		return
	}
	if scope == userStorageScope {
		token := util.GetToken(c)
		if err := patrol.UpdateUserSpaceSizeCache(
			c.Request.Context(), query.GetDB(), token.UserID, size,
		); err != nil {
			klog.Warningf("GetDirectorySize: failed to update usage cache for user %q: %v", token.Username, err)
		}
	}
	resputil.Success(c, gin.H{"size": size, "unit": "bytes", "formatted": formatSize(size)})
}

// GetMyQuota godoc
// @Summary Get the current user's enforced storage quota
// @Tags Storage
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any]
// @Router /v1/storage/my-quota [get]
func (mgr *StorageMgr) GetMyQuota(c *gin.Context) {
	logicalPath, err := mgr.authorizedUserPath(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	quota, err := ceph.GetCephDirectoryQuota(
		mgr.kubeClient, mgr.kubeConfig, ceph.StorageQuotaRookNamespace(), logicalPath, storagePrefixes(),
	)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to read the enforced storage quota"))
		return
	}
	resputil.Success(c, gin.H{"space_quota": quota, "space_quota_formatted": formatSize(quota)})
}

func (mgr *StorageMgr) authorizedUserPath(c *gin.Context) (string, error) {
	token := util.GetToken(c)
	var user model.User
	if err := query.GetDB().Select("id", "space").First(&user, token.UserID).Error; err != nil {
		return "", bizerr.NotFound.DataBaseNotFound.Wrap(err, "user was not found")
	}
	return "/user/" + user.Space, nil
}

// GetAllUserSpaceSizes godoc
// @Summary Get paginated user storage usage and quotas
// @Tags Storage
// @Produce json
// @Security Bearer
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size, from 1 to 100"
// @Success 200 {object} resputil.Response[any]
// @Router /v1/admin/storage/user-spaces [get]
func (mgr *StorageMgr) GetAllUserSpaceSizes(c *gin.Context) {
	page, pageErr := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, pageSizeErr := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageErr != nil || page < 1 || pageSizeErr != nil || pageSize < 1 || pageSize > 100 {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New(
			"page must be positive and pageSize must be between 1 and 100",
		))
		return
	}

	type userSpaceInfo struct {
		Username   string     `gorm:"column:username"`
		Size       int64      `gorm:"column:size"`
		UpdatedAt  *time.Time `gorm:"column:updated_at"`
		SpaceQuota int64      `gorm:"column:space_quota"`
	}
	db := query.GetDB().WithContext(c.Request.Context())
	var total int64
	if err := db.Model(&model.User{}).Count(&total).Error; err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to count users"))
		return
	}
	var rows []userSpaceInfo
	if err := db.Table("users").Select(
		"users.name AS username, users.space_quota, " +
			"COALESCE(user_space_sizes.size, -1) AS size, user_space_sizes.updated_at",
	).Joins("LEFT JOIN user_space_sizes ON user_space_sizes.user_id = users.id").
		Where("users.deleted_at IS NULL").Order("users.id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error; err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to query user storage usage"))
		return
	}

	// The list uses the database mirror to avoid one Kubernetes exec per row.
	// An explicit refresh reconciles both usage and quota from CephFS.
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]any{
			"user": row.Username, "size": row.Size, "quota": row.SpaceQuota,
			"unit": "bytes", "formatted": formatSize(row.Size),
			"quota_formatted": formatSize(row.SpaceQuota), "updated_at": row.UpdatedAt,
		})
	}
	resputil.Success(c, gin.H{
		"items": items, "total": total, "page": page, "pageSize": pageSize,
		"totalPages": (int(total) + pageSize - 1) / pageSize,
	})
}

// RefreshUserSpaceSizes godoc
// @Summary Refresh user storage usage and reconcile quota mirrors
// @Tags Storage
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[patrol.StorageUsageRefreshResult]
// @Router /v1/admin/storage/user-spaces/refresh [post]
func (mgr *StorageMgr) RefreshUserSpaceSizes(c *gin.Context) {
	capabilities := mgr.getCachedCapabilities(false)
	result, err := patrol.RefreshUserSpaceSizes(c.Request.Context(), &patrol.Clients{
		KubeClient: mgr.kubeClient, KubeConfig: mgr.kubeConfig,
	}, capabilities.QuotaReadable)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to refresh storage usage"))
		return
	}
	resputil.Success(c, result)
}

// SetUserSpaceQuota godoc
// @Summary Set a user's CephFS storage quota
// @Tags Storage
// @Accept json
// @Produce json
// @Security Bearer
// @Param user path string true "Username"
// @Param quota body SetUserSpaceQuotaRequest true "Space quota request"
// @Success 200 {object} resputil.Response[any]
// @Router /v1/admin/storage/user-spaces/{user}/quota [put]
func (mgr *StorageMgr) SetUserSpaceQuota(c *gin.Context) {
	username := c.Param("user")
	if username == "" {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("username is required"))
		return
	}
	var req SetUserSpaceQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "quota must be an integer number of bytes"))
		return
	}
	if req.Quota < -1 || req.Quota == 0 {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New(
			"quota must be -1 for unlimited or greater than zero",
		))
		return
	}

	auditDetails := map[string]any{"new_quota": req.Quota, "provider": ceph.StorageQuotaProvider()}
	state := quotaUpdateState{}
	err := query.GetDB().WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		return mgr.applyUserSpaceQuota(tx, username, req.Quota, &state, auditDetails)
	})
	if err != nil && state.CephChanged {
		auditDetails["transaction_failed_after_ceph_write"] = true
		compensationCtx, cancel := context.WithTimeout(
			context.WithoutCancel(c.Request.Context()), quotaCompensationTimeout,
		)
		compensationErr := mgr.compensateQuotaUpdateWithRetry(
			compensationCtx, &state, req.Quota, auditDetails,
		)
		cancel()
		auditDetails["rollback_succeeded"] = compensationErr == nil
		if compensationErr != nil {
			auditDetails["reconciliation_required"] = true
			klog.Errorf(
				"SetUserSpaceQuota: transaction failed for user %q and compensation requires reconciliation: %v",
				username, compensationErr,
			)
			err = errors.Join(err, compensationErr)
		}
	}
	if err != nil {
		statusErr := bizerr.Internal.FileSystemError.Wrap(err, "failed to update the storage quota")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			statusErr = bizerr.NotFound.DataBaseNotFound.Wrap(err, "user was not found")
		}
		RecordOperationLog(c, constants.OpTypeSetStorageQuota, username, constants.OpStatusFailed, err.Error(), auditDetails)
		resputil.HandleError(c, statusErr)
		return
	}

	auditDetails["ceph_applied"] = true
	RecordOperationLog(c, constants.OpTypeSetStorageQuota, username, constants.OpStatusSuccess, "", auditDetails)
	resputil.Success(c, gin.H{
		"user": username, "quota": state.AppliedQuota, "unit": "bytes",
		"quota_formatted": formatSize(state.AppliedQuota), "ceph_quota_set": true,
	})
}

func (mgr *StorageMgr) applyUserSpaceQuota(
	tx *gorm.DB,
	username string,
	requestedQuota int64,
	state *quotaUpdateState,
	auditDetails map[string]any,
) error {
	var user model.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", username).First(&user).Error; err != nil {
		return err
	}
	state.UserID = user.ID
	state.LogicalPath = "/user/" + user.Space
	oldQuota, err := ceph.GetCephDirectoryQuota(
		mgr.kubeClient, mgr.kubeConfig, ceph.StorageQuotaRookNamespace(), state.LogicalPath, storagePrefixes(),
	)
	if err != nil {
		return bizerr.Internal.FileSystemError.Wrap(err, "failed to read the current CephFS quota")
	}
	state.PreviousQuota = oldQuota
	auditDetails["old_quota"] = oldQuota
	if err := mgr.setCephQuota(state.LogicalPath, requestedQuota); err != nil {
		if errors.Is(err, ceph.ErrQuotaWriteOutcomeUnknown) {
			state.CephChanged = true
			auditDetails["write_outcome_uncertain"] = true
		}
		return err
	}
	state.CephChanged = true

	rollback := func(cause error) error {
		rollbackErr := mgr.setAndVerifyCephQuota(state.LogicalPath, oldQuota)
		auditDetails["rollback_succeeded"] = rollbackErr == nil
		if rollbackErr != nil {
			return bizerr.Internal.FileSystemError.Wrap(
				errors.Join(cause, rollbackErr),
				"quota update failed and CephFS rollback also failed",
			)
		}
		state.CephChanged = false
		return cause
	}
	if err := mgr.verifyCephQuota(state.LogicalPath, requestedQuota); err != nil {
		return rollback(err)
	}

	result := tx.Model(&model.User{}).Where("id = ?", user.ID).Update("space_quota", requestedQuota)
	if result.Error != nil {
		return rollback(bizerr.Internal.DatabaseError.Wrap(result.Error, "failed to update the database quota mirror"))
	}
	if result.RowsAffected != 1 {
		return rollback(bizerr.Internal.DatabaseError.New(fmt.Sprintf(
			"database quota mirror update affected %d rows", result.RowsAffected,
		)))
	}
	state.AppliedQuota = requestedQuota
	return nil
}

func (mgr *StorageMgr) setCephQuota(logicalPath string, quota int64) error {
	if err := ceph.SetCephDirectoryQuota(
		mgr.kubeClient, mgr.kubeConfig, ceph.StorageQuotaRookNamespace(), logicalPath, storagePrefixes(), quota,
	); err != nil {
		return bizerr.Internal.FileSystemError.Wrap(err, "failed to apply the CephFS quota")
	}
	return nil
}

func (mgr *StorageMgr) verifyCephQuota(logicalPath string, quota int64) error {
	readback, err := ceph.GetCephDirectoryQuota(
		mgr.kubeClient, mgr.kubeConfig, ceph.StorageQuotaRookNamespace(), logicalPath, storagePrefixes(),
	)
	if err != nil {
		return bizerr.Internal.FileSystemError.Wrap(err, "failed to verify the CephFS quota")
	}
	if readback != quota {
		return bizerr.Internal.FileSystemError.New(fmt.Sprintf(
			"CephFS quota readback is %d, expected %d", readback, quota,
		))
	}
	return nil
}

func (mgr *StorageMgr) setAndVerifyCephQuota(logicalPath string, quota int64) error {
	if err := mgr.setCephQuota(logicalPath, quota); err != nil {
		return err
	}
	return mgr.verifyCephQuota(logicalPath, quota)
}

func (mgr *StorageMgr) compensateQuotaUpdateWithRetry(
	ctx context.Context,
	state *quotaUpdateState,
	requestedQuota int64,
	auditDetails map[string]any,
) error {
	var compensationErr error
	for attempt := 1; attempt <= quotaCompensationRetries; attempt++ {
		auditDetails["compensation_attempts"] = attempt
		compensationErr = mgr.compensateQuotaUpdate(ctx, state, requestedQuota)
		if compensationErr == nil {
			state.CephChanged = false
			return nil
		}
		if ctx.Err() != nil {
			break
		}
	}
	return compensationErr
}

func (mgr *StorageMgr) compensateQuotaUpdate(
	ctx context.Context,
	state *quotaUpdateState,
	requestedQuota int64,
) error {
	return query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, state.UserID).Error; err != nil {
			return err
		}
		currentQuota, err := ceph.GetCephDirectoryQuota(
			mgr.kubeClient, mgr.kubeConfig, ceph.StorageQuotaRookNamespace(), state.LogicalPath, storagePrefixes(),
		)
		if err != nil {
			return bizerr.Internal.FileSystemError.Wrap(err, "failed to read the CephFS quota during compensation")
		}
		if currentQuota != requestedQuota && currentQuota != state.PreviousQuota {
			return bizerr.Conflict.ResourceStatusError.New(fmt.Sprintf(
				"refusing to overwrite a newer CephFS quota %d during compensation", currentQuota,
			))
		}
		if currentQuota != state.PreviousQuota {
			if err := mgr.setAndVerifyCephQuota(state.LogicalPath, state.PreviousQuota); err != nil {
				return err
			}
		}
		updated := tx.Model(&model.User{}).Where("id = ?", user.ID).Update("space_quota", state.PreviousQuota)
		if updated.Error != nil {
			return bizerr.Internal.DatabaseError.Wrap(updated.Error, "failed to restore the database quota mirror")
		}
		if updated.RowsAffected != 1 {
			return bizerr.Internal.DatabaseError.New(fmt.Sprintf(
				"database quota mirror restore affected %d rows", updated.RowsAffected,
			))
		}
		return nil
	})
}

func formatSize(bytes int64) string {
	if bytes < 0 {
		return "Unlimited"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
