package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	utils "github.com/raids-lab/crater/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type AIJobMgr struct {
	pvcClient      *crclient.PVCClient
	logClient      *crclient.LogClient
	taskController *aitaskctl.TaskController
}

func NewAIJobMgr(taskController *aitaskctl.TaskController, pvcClient *crclient.PVCClient, logClient *crclient.LogClient) Manager {
	return &AIJobMgr{
		pvcClient:      pvcClient,
		logClient:      logClient,
		taskController: taskController,
	}
}

func (mgr *AIJobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AIJobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("", mgr.Create)
	g.GET("", mgr.ListByType)
	g.GET("/getQuota", mgr.GetQuota) // should be split
	g.GET("/jobStats", mgr.GetJobStats)
	g.DELETE("/:id", mgr.Delete)
	g.GET("/:id", mgr.Get)
	g.POST("/updateSLO/:id", mgr.UpdateSLO)
	g.GET("/getLogs/:id", mgr.GetLogs)
	g.GET("/getToken/:id", mgr.GetToken)
}

func (mgr *AIJobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/type", mgr.ListTaskByTaskType)
	g.GET("/stats", mgr.GetAllJobStats)
	g.GET("/:id", mgr.Get)
	g.DELETE("/:id", mgr.Delete)
}

func (mgr *AIJobMgr) rolePermit(token *util.ReqContext, job *model.AIJob) bool {
	ok := false
	if token.PlatformRole == model.RoleAdmin {
		ok = true
	} else if job.ProjectID == token.ProjectID {
		if token.ProjectRole == model.RoleAdmin {
			ok = true
		} else if token.UserID == job.UserID {
			ok = true
		}
	}
	return ok
}

// FromCustomResourceList converts CustomResourceList to v1.ResourceList
func FromCustomResourceList(crl map[string]string) (v1.ResourceList, error) {
	rl := v1.ResourceList{}
	for k, v := range crl {
		quantity, err := resource.ParseQuantity(v)
		if err != nil {
			return nil, err
		}
		rl[v1.ResourceName(k)] = quantity
	}
	return rl, nil
}

// ToCustomResourceList converts v1.ResourceList to CustomResourceList
func ToCustomResourceList(rl v1.ResourceList) map[string]string {
	crl := map[string]string{}
	for k, v := range rl {
		crl[string(k)] = v.String()
	}
	return crl
}

type (
	DirMount struct {
		Volume    string `json:"volume"`
		MountPath string `json:"mountPath"`
		SubPath   string `json:"subPath"`
	}
	CreateJobReq struct {
		Name            string                `json:"taskName" binding:"required"`
		SLO             uint                  `json:"slo"`
		TaskType        string                `json:"taskType" binding:"required"`
		ResourceRequest map[string]any        `json:"resourceRequest" binding:"required"`
		Image           string                `json:"image" binding:"required"`
		WorkingDir      string                `json:"workingDir"`
		ShareDirs       map[string][]DirMount `json:"shareDirs"`
		Command         string                `json:"command" binding:"required"`
		GPUModel        string                `json:"gpuModel"`
		SchedulerName   string                `json:"schedulerName"`
	}
	CreateTaskResp struct {
		TaskID uint
	}
)

func ParseCustomizeTask(req *CreateJobReq) error {
	switch req.TaskType {
	case models.JupyterTask:
		req.SLO = 1
		req.Command = "start.sh jupyter lab --allow-root --NotebookApp.base_url=/jupyter/%s/"
		req.WorkingDir = "/home/%s"
	case models.TrainingTask:
		return nil
	default:
		return fmt.Errorf("unsupported tasktype: %q", req.TaskType)
	}

	return nil
}

func StructToJSONString(m any) (string, error) {
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

type CreateResp struct {
	TaskID uint
}

// 用户创建任务 godoc
// @Summary 创建任务
// @Description 创建任务并获取任务id.
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param CreateJobReq body CreateJobReq true "任务结构体"
// @Success 200 {object} resputil.Response[any]
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/aijobs [post]
func (mgr *AIJobMgr) Create(c *gin.Context) {
	logutils.Log.Infof("Job Create, url: %s", c.Request.URL)
	var req CreateJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		logutils.Log.Error(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}
	token, _ := util.GetToken(c)

	if err := ParseCustomizeTask(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("customizing job attributes failed, err %v", err), resputil.NotSpecified)
		return
	}

	var reqStr, shareDirsStr string
	var err error

	reqStr, err = StructToJSONString(req.ResourceRequest)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("parse job attributes failed, err %v", err), resputil.NotSpecified)
		return
	}

	shareDirsStr, err = StructToJSONString(req.ShareDirs)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("parse job attributes failed, err %v", err), resputil.NotSpecified)
		return
	}

	jsonMap := map[string]string{
		"SLO":           fmt.Sprint(req.SLO),
		"Image":         req.Image,
		"WorkingDir":    req.WorkingDir,
		"ShareDirs":     shareDirsStr,
		"Command":       req.Command,
		"GPUModel":      req.GPUModel,
		"SchedulerName": req.SchedulerName,
	}

	extraStr := models.MapToJSONString(jsonMap)

	job := model.AIJob{
		Name:            req.Name,
		UserID:          token.UserID,
		ProjectID:       token.ProjectID,
		TaskType:        req.TaskType,
		Status:          model.JobInitial,
		ResourceRequest: reqStr,
		Extra:           &extraStr,
	}

	jobQueryModel := query.AIJob
	err = jobQueryModel.WithContext(c).Create(&job)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("create job failed, err %v", err), resputil.NotSpecified)
		return
	}
	mgr.notifyTaskUpdate(job.ID, fmt.Sprintf("%d-%d", job.UserID, job.ProjectID), utils.CreateTask)

	logutils.Log.Infof("create job success, taskID: %d", job.ID)
	resp := CreateResp{
		TaskID: job.ID,
	}
	resputil.Success(c, resp)
}

type (
	ListTaskReq struct {
		PageIndex *int   `form:"pageIndex" binding:"required"`
		PageSize  int    `form:"pageSize" binding:"required"`
		TaskType  string `form:"taskType" binding:"required"`
	}
	ListTaskResp = payload.ListResp[*model.AIJob]
)

// 用户查询任务列表 godoc
// @Summary 用户查询任务列表
// @Description 根据任务状态和分页要求查询用户下的任务
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param ListTaskReq query ListTaskReq true "分页参数和筛选任务状态"
// @Success 200 {object} resputil.Response[any] "总数和任务数组"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/aijobs [get]
func (mgr *AIJobMgr) ListByType(c *gin.Context) {
	var req ListTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate list parameters failed, err %v", err), resputil.NotSpecified)
		return
	}

	token, _ := util.GetToken(c)

	jobQueryModel := query.AIJob
	jobQueryExec := jobQueryModel.WithContext(c)

	jobQueryExec = jobQueryExec.Where(jobQueryModel.UserID.Eq(token.UserID), jobQueryModel.TaskType.Eq(req.TaskType))

	jobs, count, err := jobQueryExec.FindByPage((*req.PageIndex)*req.PageSize, req.PageSize)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list job failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := ListTaskResp{
		Count: count,
		Rows:  jobs,
	}
	resputil.Success(c, resp)
}

type (
	GetTaskResp struct {
		*model.AIJob
	}
)

// 用户获取指定任务 godoc
// @Summary 用户获取指定任务
// @Description 检查用户id获取指定id的job
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path uint true "job id"
// @Success 200 {object} resputil.Response[any] "返回任务结构体"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/aijobs/{id} [get]
// @Router /v1/admin/aijobs/{id} [get]
func (mgr *AIJobMgr) Get(c *gin.Context) {
	logutils.Log.Infof("job Get, url: %s", c.Request.URL)
	id, err := util.GetParamID(c, "id")
	if err != nil {
		resputil.Error(c, fmt.Sprintf("validate get parameters failed, err %v", err), resputil.NotSpecified)
		return
	}

	token, _ := util.GetToken(c)

	jobQueryModel := query.AIJob

	job, err := jobQueryModel.WithContext(c).Where(jobQueryModel.ID.Eq(id)).First()

	if err != nil {
		resputil.Error(c, fmt.Sprintf("get job failed, err %v", err), resputil.NotSpecified)
		return
	}

	if !mgr.rolePermit(&token, job) {
		resputil.HTTPError(c, http.StatusForbidden, "forbidden", resputil.NotSpecified)
		return
	}

	resp := GetTaskResp{
		job,
	}
	logutils.Log.Infof("get job success, taskID: %d", id)
	resputil.Success(c, resp)
}

// Delete godoc
// @Summary Delete an AIJob by ID
// @Description Delete an AI job by its unique identifier.
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path uint true "AI job ID"
// @Success 200 {object} resputil.Response[any]
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/aijobs/{id} [delete]
// @Router /v1/admin/aijobs/{id} [delete]
func (mgr *AIJobMgr) Delete(c *gin.Context) {
	id, err := util.GetParamID(c, "id")
	if err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.NotSpecified)
		return
	}
	token, _ := util.GetToken(c)

	a := query.AIJob
	job, err := a.WithContext(c).Where(a.ID.Eq(id)).First()
	if err != nil {
		resputil.HTTPError(c, http.StatusNotFound, err.Error(), resputil.NotSpecified)
		return
	}

	// 检查该请求是否有权限删除任务

	if !mgr.rolePermit(&token, job) {
		resputil.HTTPError(c, http.StatusForbidden, "forbidden", resputil.NotSpecified)
		return
	}

	// 通知任务控制器，删除任务
	mgr.notifyTaskUpdate(id, fmt.Sprintf("%d-%d", job.UserID, job.ProjectID), utils.DeleteTask)

	// 从数据库中删除任务
	_, err = a.WithContext(c).Delete(job)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete job failed, err %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, "")
}

func (mgr *AIJobMgr) notifyTaskUpdate(taskID uint, userName string, op utils.TaskOperation) {
	mgr.taskController.TaskUpdated(utils.TaskUpdateChan{
		TaskID:    taskID,
		UserName:  userName,
		Operation: op,
	})
}

type GetLogResp struct {
	Logs []string `json:"logs"`
}

// 获取任务日志 godoc
// @Summary 获取任务日志
// @Description 通过指定任务id查询对应pod获取日志
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path uint true "任务id"
// @Success 200 {object} resputil.Response[any] "日志信息"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/aijobs/getLogs/{id} [get]
func (mgr *AIJobMgr) GetLogs(c *gin.Context) {
	logutils.Log.Infof("Job Get, url: %s", c.Request.URL)
	id, err := util.GetParamID(c, "id")
	if err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.NotSpecified)
		return
	}

	jobQueryModel := query.AIJob
	job, err := jobQueryModel.WithContext(c).Where(jobQueryModel.ID.Eq(id)).First()

	if err != nil {
		resputil.Error(c, fmt.Sprintf("get job failed, err %v", err), resputil.NotSpecified)
		return
	}
	// get log
	pods, err := mgr.logClient.GetPodsWithLabel(job.Project.Namespace, job.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get job log failed, err %v", err), resputil.NotSpecified)
		return
	}
	var logs []string
	for i := range pods {
		pod := &pods[i]
		podLog, err := mgr.logClient.GetPodLogs(pod)
		if err != nil {
			resputil.Error(c, fmt.Sprintf("get job log failed, err %v", err), resputil.NotSpecified)
			return
		}
		logs = append(logs, podLog)
	}
	resp := GetLogResp{
		Logs: logs,
	}
	logutils.Log.Infof("get job logs success, jobID: %d", job.ID)
	resputil.Success(c, resp)
}

type (
	UpdateTaskReq struct {
		SLO uint `json:"slo" binding:"required"` // change the slo of the job
	}
)

// 更新任务slo godoc
// @Summary 更新任务slo
// @Description 根据传入的id和字段值更新任务slo
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path uint true "任务id"
// @Param slo body UpdateTaskReq true "任务slo"
// @Success 200 {object} resputil.Response[any] "null"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/aijobs/updateSLO/{id} [post]
func (mgr *AIJobMgr) UpdateSLO(c *gin.Context) {
	logutils.Log.Infof("Job Update, url: %s", c.Request.URL)
	var id uint
	var err error
	var job *model.AIJob
	if id, err = util.GetParamID(c, "id"); err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.NotSpecified)
		return
	}

	var req UpdateTaskReq
	if err = c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, err %v", err), resputil.NotSpecified)
		return
	}

	token, _ := util.GetToken(c)

	jobQueryModel := query.AIJob
	jobQueryExec := jobQueryModel.WithContext(c)
	job, err = jobQueryExec.Where(jobQueryModel.ID.Eq(id)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get job failed, err %v", err), resputil.NotSpecified)
		return
	}

	if !mgr.rolePermit(&token, job) {
		resputil.HTTPError(c, http.StatusForbidden, "forbidden", resputil.NotSpecified)
		return
	}

	jsonMap := models.JSONStringToMap(*job.Extra)
	jsonMap["slo"] = fmt.Sprint(req.SLO)

	extraStr := model.MapToJSONString(jsonMap)

	ret, err := jobQueryExec.Where(jobQueryModel.ID.Eq(id)).Update(jobQueryModel.Extra, extraStr)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get job failed, err %v", err), resputil.NotSpecified)
		return
	}

	mgr.notifyTaskUpdate(job.ID, fmt.Sprintf("%d-%d", job.UserID, job.ProjectID), utils.UpdateTask)
	logutils.Log.Infof("update job success, taskID: %d", job.ID)
	resputil.Success(c, ret)
}

type GetQuotaResp struct {
	User     string          `json:"user"`
	Hard     v1.ResourceList `json:"hard"`
	HardUsed v1.ResourceList `json:"hardUsed"`
	SoftUsed v1.ResourceList `json:"softUsed"`
}

// 获取配额 godoc
// @Summary 获取用户当前配额
// @Description 目前只查了数据库确认用户情况，实际直接从sync map获取
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "用户quota描述结构体"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/aijobs/getQuota [get]
func (mgr *AIJobMgr) GetQuota(c *gin.Context) {
	token, _ := util.GetToken(c)

	userQueryModel := query.User
	user, err := userQueryModel.WithContext(c).Where(userQueryModel.ID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get job failed, err %v", err), resputil.NotSpecified)
		return
	}

	quotaInfo := mgr.taskController.GetQuotaInfoSnapshotByUsername(user.Name)
	if quotaInfo == nil {
		resputil.Error(c, fmt.Sprintf("get user:%v quota failed", user.Name), resputil.NotSpecified)
		return
	}
	resp := GetQuotaResp{
		Hard:     quotaInfo.Hard,
		HardUsed: quotaInfo.HardUsed,
		SoftUsed: quotaInfo.SoftUsed,
	}
	resputil.Success(c, resp)
}

type (
	TaskStatusCount struct {
		Status uint8
		Count  int
	}
	GetTaskStatsResp struct {
		TaskCount []TaskStatusCount `json:"taskCount"`
	}
)

// 获取各类状态任务统计 godoc
// @Summary 获取用户各类任务状态统计情况
// @Description 数据库查表count group后返回
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "状态统计列表"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/aijobs/jobStats [get]
func (mgr *AIJobMgr) GetJobStats(c *gin.Context) {
	token, _ := util.GetToken(c)

	jobQueryModel := query.AIJob

	jobQueryExec := jobQueryModel.WithContext(c)

	var stats []TaskStatusCount

	jobQueryExec = jobQueryExec.Where(jobQueryModel.UserID.Eq(token.UserID), jobQueryModel.DeletedAt.IsNull())
	err := jobQueryExec.Select(jobQueryModel.Status, jobQueryModel.Status.Count()).Group(jobQueryModel.Status).Scan(&stats)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get job count statistic failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := GetTaskStatsResp{
		TaskCount: stats,
	}
	resputil.Success(c, resp)
}

type GetTokenResp struct {
	Name  string `json:"name"`  // 任务名称
	Token string `json:"token"` // jupyter token
}

// GetToken godoc
// @Summary get token for access jupyter lab
// @Description get token from db or pods logs
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path uint true "jupyter对应任务id"
// @Success 200 {object} resputil.Response[any] "端口和token结构体"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/aijobs/getToken/{id} [get]
func (mgr *AIJobMgr) GetToken(c *gin.Context) {
	logutils.Log.Infof("Job Token Get, url: %s", c.Request.URL)
	id, err := util.GetParamID(c, "id")
	if err != nil {
		resputil.Error(c, fmt.Sprintf("validate get parameters failed, err %v", err), resputil.NotSpecified)
		return
	}

	jobQueryModel := query.AIJob
	jobQueryExec := jobQueryModel.WithContext(c)
	job, err := jobQueryExec.Where(jobQueryModel.ID.Eq(id)).First()

	if err != nil {
		resputil.Error(c, fmt.Sprintf("get job failed, err %v", err), resputil.NotSpecified)
		return
	}
	if job.Status != model.JobRunning {
		resp := GetTokenResp{
			Name:  job.Name,
			Token: "",
		}
		logutils.Log.Infof("job token not ready, taskID: %d", id)
		resputil.Success(c, resp)
		return
	}

	jsonMap := models.JSONStringToMap(*job.Extra)

	if jsonMap["Token"] != "" {
		resp := GetTokenResp{
			Name:  job.Name,
			Token: jsonMap["Token"],
		}
		logutils.Log.Infof("get job token success, taskID: %d", id)
		resputil.Success(c, resp)
		return
	}

	// get log
	pods, err := mgr.logClient.GetPodsWithLabel(job.Project.Namespace, job.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get job log failed, err %v", err), resputil.NotSpecified)
		return
	}
	var token string
	re := regexp.MustCompile(`\?token=([a-zA-Z0-9]+)`)
	for i := range pods {
		pod := &pods[i]
		podLog, getPodLogsErr := mgr.logClient.GetPodLogs(pod)
		if getPodLogsErr != nil {
			resputil.Error(c, fmt.Sprintf("get job log failed, err %v", getPodLogsErr), resputil.NotSpecified)
			return
		}
		matches := re.FindStringSubmatch(podLog)
		if len(matches) >= 2 {
			token = matches[1]
			break
		}
	}

	// Save token to db
	jsonMap["Token"] = token

	extraStr := model.MapToJSONString(jsonMap)
	if _, err := jobQueryExec.Where(jobQueryModel.ID.Eq(id)).Update(jobQueryModel.Extra, extraStr); err != nil {
		resputil.Error(c, fmt.Sprintf("update job extra failed, err %v", err), resputil.NotSpecified)
		return
	}

	resp := GetTokenResp{
		Name:  job.Name,
		Token: token,
	}
	logutils.Log.Infof("get job token success, taskID: %d", id)
	resputil.Success(c, resp)
}

type (
	ListTaskByTypeReq struct {
		// 分页参数
		PageIndex *int `form:"page_index" binding:"required"`
		PageSize  int  `form:"page_size" binding:"required"`
		// 筛选、排序参数
		TaskType string `form:"taskType" binding:"required"`
	}
)

// ListTaskByTaskType godoc
// @Summary 管理员获取指定类型任务列表
// @Description 查询某类型的全部任务
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param ListTaskByTypeReq query ListTaskByTypeReq true "分页参数和筛选类型"
// @Success 200 {object} resputil.Response[any] "总数和jobs数组"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/aijobs/type [get]
func (mgr *AIJobMgr) ListTaskByTaskType(c *gin.Context) {
	logutils.Log.Infof("Task List, url: %s", c.Request.URL)
	var req ListTaskByTypeReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate list parameters failed, err %v", err), resputil.NotSpecified)
		return
	}

	jobQueryModel := query.AIJob
	jobQueryExec := jobQueryModel.WithContext(c)

	jobQueryExec = jobQueryExec.Where(jobQueryModel.TaskType.Eq(req.TaskType)).Order(jobQueryModel.CreatedAt.Desc())

	jobs, count, err := jobQueryExec.FindByPage((*req.PageIndex)*req.PageSize, req.PageSize)

	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := ListTaskResp{
		Count: count,
		Rows:  jobs,
	}
	resputil.Success(c, resp)
}

// 获取各类状态任务统计 godoc
// @Summary 管理员获取各类任务状态统计情况
// @Description 数据库查表count group后返回
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "状态统计列表"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/aijobs/stats [get]
func (mgr *AIJobMgr) GetAllJobStats(c *gin.Context) {
	jobQueryModel := query.AIJob

	jobQueryExec := jobQueryModel.WithContext(c)

	var stats []TaskStatusCount

	jobQueryExec = jobQueryExec.Where(jobQueryModel.DeletedAt.IsNull())
	err := jobQueryExec.Select(jobQueryModel.Status, jobQueryModel.Status.Count()).Group(jobQueryModel.Status).Scan(&stats)

	if err != nil {
		resputil.Error(c, fmt.Sprintf("get job count statistic failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := GetTaskStatsResp{
		TaskCount: stats,
	}
	resputil.Success(c, resp)
}
