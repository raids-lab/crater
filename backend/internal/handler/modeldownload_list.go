package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

type modelDownloadPageQuery struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=200"`
	Search   string `form:"search"`
	Sort     string `form:"sort"`
	Category string `form:"category"`
}

var modelDownloadSortFields = map[string]string{
	"name":      "model_downloads.name",
	"category":  "model_downloads.category",
	"status":    "model_downloads.status",
	"sizeBytes": "model_downloads.size_bytes",
	"createdAt": "model_downloads.created_at",
	"updatedAt": "model_downloads.updated_at",
}

var modelDownloadStatuses = map[string]struct{}{
	string(model.ModelDownloadStatusPending):     {},
	string(model.ModelDownloadStatusDownloading): {},
	string(model.ModelDownloadStatusPaused):      {},
	string(model.ModelDownloadStatusReady):       {},
	string(model.ModelDownloadStatusFailed):      {},
}

func modelDownloadSortClauses(sortValue string) []string {
	return pagination.SortClauses(sortValue, modelDownloadSortFields, "model_downloads.id DESC")
}

func bindModelDownloadPageQuery(c *gin.Context) (modelDownloadPageQuery, []string, error) {
	var request modelDownloadPageQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		return modelDownloadPageQuery{}, nil, bizerr.BadRequest.ParameterError.Wrap(
			err,
			"invalid model download list query",
		)
	}
	var err error
	request.Search, err = pagination.NormalizePageQuery(request.Search, request.Page, request.PageSize)
	if err != nil {
		return modelDownloadPageQuery{}, nil, err
	}
	if request.Category != "" && request.Category != CategoryModel && request.Category != CategoryDataset {
		return modelDownloadPageQuery{}, nil, bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("unsupported model download category %q", request.Category),
		)
	}
	request.Sort, err = pagination.DefaultAndValidateSort(request.Sort, "-updatedAt", modelDownloadSortFields)
	if err != nil {
		return modelDownloadPageQuery{}, nil, err
	}

	statuses := c.QueryArray("status")
	for _, status := range statuses {
		if _, ok := modelDownloadStatuses[status]; !ok {
			return modelDownloadPageQuery{}, nil, bizerr.BadRequest.ParameterError.New(
				fmt.Sprintf("unsupported model download status %q", status),
			)
		}
	}
	return request, statuses, nil
}

// ListDownloadsPage returns model download records using the shared page protocol.
//
//	@Summary	分页获取模型下载任务
//	@Tags		ModelDownload
//	@Produce	json
//	@Security	Bearer
//	@Param		page		query	int		false	"Page number"
//	@Param		page_size	query	int		false	"Page size, 1-200"
//	@Param		search		query	string	false	"Search download name"
//	@Param		category	query	string	false	"Filter by model or dataset"
//	@Param		status		query	string	false	"Filter by status; repeatable"
//	@Param		sort		query	string	false	"Sort fields"
//	@Success	200			{object}	resputil.Response[resputil.Page[ModelDownloadResp]]
//	@Router		/v1/model-download/models/downloads/page [get]
func (mgr *ModelDownloadMgr) ListDownloadsPage(c *gin.Context) {
	request, statuses, err := bindModelDownloadPageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	db := query.GetDB().WithContext(c).Model(&model.ModelDownload{})
	if request.Category != "" {
		db = db.Where("model_downloads.category = ?", request.Category)
	}
	if len(statuses) > 0 {
		db = db.Where("model_downloads.status IN ?", statuses)
	}
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		db = db.Where("LOWER(model_downloads.name) LIKE ? ESCAPE '\\'", pattern)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		klog.Errorf("failed to count model downloads: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count model downloads failed"))
		return
	}
	for _, clause := range modelDownloadSortClauses(request.Sort) {
		db = db.Order(clause)
	}
	downloads := make([]*model.ModelDownload, 0, request.PageSize)
	if err := db.Preload("Creator").
		Offset((request.Page - 1) * request.PageSize).
		Limit(request.PageSize).
		Find(&downloads).Error; err != nil {
		klog.Errorf("failed to list model downloads: %v", err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list model downloads failed"))
		return
	}

	token := util.GetToken(c)
	items := make([]ModelDownloadResp, len(downloads))
	for i, download := range downloads {
		items[i] = convertDownloadToResp(download, token)
	}
	if err := mgr.applyDownloadUserContext(c, downloads, items, token); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, resputil.NewPage(items, total, request.Page, request.PageSize))
}

// ListDownloadsSummary returns status totals independently from list filters.
//
//	@Summary	获取模型下载任务状态汇总
//	@Tags		ModelDownload
//	@Produce	json
//	@Security	Bearer
//	@Param		category	query		string	false	"Filter by model or dataset"
//	@Success	200			{object}	resputil.Response[map[string]int64]
//	@Router		/v1/model-download/models/downloads/summary [get]
func (mgr *ModelDownloadMgr) ListDownloadsSummary(c *gin.Context) {
	category := c.Query("category")
	if category != "" && category != CategoryModel && category != CategoryDataset {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("unsupported model download category %q", category),
		))
		return
	}
	summary, err := mgr.downloadStatusSummary(c, category)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "count download status summary failed"))
		return
	}
	resputil.Success(c, summary)
}
