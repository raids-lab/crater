package vcjob

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/handler/tool"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/packer"
	"github.com/raids-lab/crater/pkg/utils"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	handler.Registers = append(handler.Registers, NewVolcanojobMgr)
}

type VolcanojobMgr struct {
	name          string
	client        client.Client
	kubeClient    kubernetes.Interface
	imagePacker   packer.ImagePackerInterface
	imageRegistry imageregistry.ImageRegistryInterface
}

func NewVolcanojobMgr(conf *handler.RegisterConfig) handler.Manager {
	return &VolcanojobMgr{
		name:          "vcjobs",
		client:        conf.Client,
		kubeClient:    conf.KubeClient,
		imagePacker:   conf.ImagePacker,
		imageRegistry: conf.ImageRegistry,
	}
}

func (mgr *VolcanojobMgr) GetName() string { return mgr.name }

func (mgr *VolcanojobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *VolcanojobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.GetUserJobs)
	g.GET("all", mgr.GetAllJobs)
	g.DELETE(":name", mgr.DeleteJob)

	g.GET(":name/detail", mgr.GetJobDetail)
	g.GET(":name/ssh", mgr.GetSSHPortDetail)
	g.GET(":name/yaml", mgr.GetJobYaml)
	g.GET(":name/pods", mgr.GetJobPods)
	g.GET(":name/template", mgr.GetJobTemplate)
	g.GET(":name/event", mgr.GetJobEvents)
	g.PUT(":name/alert", mgr.SetAlertEnabled)

	// jupyter
	g.POST("jupyter", mgr.CreateJupyterJob)
	g.GET(":name/token", mgr.GetJobToken)
	g.POST("jupyter/:name/snapshot", mgr.CreateJupyterSnapshot)

	// training
	g.POST("training", mgr.CreateTrainingJob)

	// tensorflow
	g.POST("tensorflow", mgr.CreateTensorflowJob)

	// pytorch
	g.POST("pytorch", mgr.CreatePytorchJob)
}

func (mgr *VolcanojobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.GetAllJobs)
	// delete job
	g.DELETE(":name", mgr.DeleteJobForAdmin)
}

const (
	VolcanoSchedulerName = "volcano"

	LabelKeyTaskType = "crater.raids.io/task-type"
	LabelKeyTaskUser = "crater.raids.io/task-user"
	LabelKeyBaseURL  = "crater.raids.io/base-url"

	AnnotationKeyTaskName     = "crater.raids.io/task-name"
	AnnotationKeyTaskTemplate = "crater.raids.io/task-template"
	AnnotationKeyJupyter      = "crater.raids.io/jupyter-token"
	AnnotationKeyOpenSSH      = "crater.raids.io/open-ssh"
	AnnotationKeyAlertEnabled = "crater.raids.io/alert-enabled"

	// VolumeData  = "crater-rw-workspace"
	VolumeCache = "crater-cache"
	JYCache     = "jycache"

	JupyterPort     = 8888
	TensorBoardPort = 6006
)

type (
	JobActionReq struct {
		JobName string `uri:"name" binding:"required"`
	}

	JobTokenResp struct {
		BaseURL   string `json:"baseURL"`
		Token     string `json:"token"`
		PodName   string `json:"podName"`
		Namespace string `json:"namespace"`
	}
)

type (
	VolumeMount struct {
		Type      VolumeType `json:"type"`
		DatasetID uint       `json:"datasetID"`
		SubPath   string     `json:"subPath"`
		MountPath string     `json:"mountPath"`
	}

	DatasetMount struct {
		DatasetID uint   `json:"datasetID"`
		MountPath string `json:"mountPath"`
	}

	CreateJobCommon struct {
		VolumeMounts   []VolumeMount                `json:"volumeMounts,omitempty"`
		DatasetMounts  []DatasetMount               `json:"datasetMounts,omitempty"`
		Envs           []v1.EnvVar                  `json:"envs,omitempty"`
		Selectors      []v1.NodeSelectorRequirement `json:"selectors,omitempty"`
		UseTensorBoard bool                         `json:"useTensorBoard"`
	}
)

// DeleteJob godoc
// @Summary Delete the job
// @Description Delete the job by client-go
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "Job Name"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name} [delete]
func (mgr *VolcanojobMgr) DeleteJob(c *gin.Context) {
	mgr.deleteJob(c, true)
}

//nolint:gocyclo // refactor later
func (mgr *VolcanojobMgr) deleteJob(c *gin.Context, shouldCheckOwner bool) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Get job record from database
	j := query.Job
	jobRecord, err := j.WithContext(c).Where(j.JobName.Eq(req.JobName)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// If should check owner, check whether the job belongs to the current user
	if shouldCheckOwner {
		token := util.GetToken(c)
		if jobRecord.UserID != token.UserID {
			resputil.Error(c, "You are not the owner of the job", resputil.NotSpecified)
			return
		}
	}

	shouldDeleteRecord := false
	shouldDeleteJob := false

	job := &batch.Job{}
	namespace := config.GetConfig().Workspace.Namespace
	if err := mgr.client.Get(c, client.ObjectKey{Name: req.JobName, Namespace: namespace}, job); err != nil {
		if errors.IsNotFound(err) {
			shouldDeleteRecord = true
		} else {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
	}

	// If should delete record is false, means the job currently exists
	if !shouldDeleteRecord {
		phase := job.Status.State.Phase
		if phase == batch.Failed || phase == batch.Completed ||
			phase == batch.Aborted || phase == batch.Terminated {
			// Job is not running or pending, delete the record directly
			shouldDeleteRecord = true
		}
		shouldDeleteJob = true
	}

	if shouldDeleteRecord {
		if _, err := j.WithContext(c).Where(j.JobName.Eq(req.JobName)).Delete(); err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
	} else {
		// update job status as deleted
		if _, err := j.WithContext(c).Where(j.JobName.Eq(req.JobName)).Updates(model.Job{
			Status:             model.Deleted,
			CompletedTimestamp: time.Now(),
		}); err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
	}

	// 直接删除 Job，OwnerReference 会自动删除 Ingress 和 Service
	if shouldDeleteJob {
		if err := mgr.client.Delete(c, job); err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
	}

	resputil.Success(c, nil)
}

// DeleteJobForAdmin godoc
// @Summary Admin delete the job
// @Description 管理员删除用户作业
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "Job Name"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/admin/vcjobs/{name} [delete]
func (mgr *VolcanojobMgr) DeleteJobForAdmin(c *gin.Context) {
	mgr.deleteJob(c, false)
}

func (mgr *VolcanojobMgr) getPodLog(c *gin.Context, namespace, podName string) (*bytes.Buffer, error) {
	logOptions := &v1.PodLogOptions{}
	logReq := mgr.kubeClient.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	logs, err := logReq.Stream(c)
	if err != nil {
		return nil, err
	}
	defer logs.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(logs)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

type GetJobLogResp struct {
	Logs map[string]string `json:"logs"`
}

type (
	JobResp struct {
		Name               string          `json:"name"`
		JobName            string          `json:"jobName"`
		Owner              string          `json:"owner"`
		JobType            string          `json:"jobType"`
		Queue              string          `json:"queue"`
		Status             string          `json:"status"`
		CreationTimestamp  metav1.Time     `json:"createdAt"`
		RunningTimestamp   metav1.Time     `json:"startedAt"`
		CompletedTimestamp metav1.Time     `json:"completedAt"`
		Nodes              []string        `json:"nodes"`
		Resources          v1.ResourceList `json:"resources"`
		KeepWhenLowUsage   bool            `json:"keepWhenLowUsage"`
	}
)

// GetUserJobs godoc
// @Summary Get the jobs of the user
// @Description Get the jobs of the user by client-go
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "Volcano Job List"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs [get]
func (mgr *VolcanojobMgr) GetUserJobs(c *gin.Context) {
	token := util.GetToken(c)

	// TODO: add indexer to list jobs by user
	j := query.Job
	jobs, err := j.WithContext(c).Preload(j.Account).Preload(j.User).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	jobList := convertJobResp(jobs)

	resputil.Success(c, jobList)
}

// GetAllJobs godoc
// @Summary Get all of the jobs
// @Description Get all jobs  by client-go
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "admin get Volcano Job List"
// @Failure 400 {object} resputil.Response[any] "admin Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/all [get]
func (mgr *VolcanojobMgr) GetAllJobs(c *gin.Context) {
	j := query.Job
	// TODO(liyilong): j.WithContext(c).Unscoped().Preload(j.Account).Preload(query.Job.User).Find()
	jobs, err := j.WithContext(c).Preload(j.Account).Preload(query.Job.User).Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	jobList := convertJobResp(jobs)

	resputil.Success(c, jobList)
}

func convertJobResp(jobs []*model.Job) []JobResp {
	jobList := make([]JobResp, len(jobs))
	for i := range jobs {
		job := jobs[i]
		jobList[i] = JobResp{
			Name:               job.Name,
			JobName:            job.JobName,
			Owner:              job.User.Nickname,
			JobType:            string(job.JobType),
			Queue:              job.Account.Nickname,
			Status:             string(job.Status),
			CreationTimestamp:  metav1.NewTime(job.CreationTimestamp),
			RunningTimestamp:   metav1.NewTime(job.RunningTimestamp),
			CompletedTimestamp: metav1.NewTime(job.CompletedTimestamp),
			Nodes:              job.Nodes.Data(),
			Resources:          job.Resources.Data(),
			KeepWhenLowUsage:   job.KeepWhenLowResourceUsage,
		}
	}
	sort.Slice(jobList, func(i, j int) bool {
		return jobList[i].CreationTimestamp.After(jobList[j].CreationTimestamp.Time)
	})
	return jobList
}

type (
	JobDetailResp struct {
		Name               string               `json:"name"`
		Namespace          string               `json:"namespace"`
		Username           string               `json:"username"`
		JobName            string               `json:"jobName"`
		JobType            model.JobType        `json:"jobType"`
		Queue              string               `json:"queue"`
		Resources          v1.ResourceList      `json:"resources"`
		Status             batch.JobPhase       `json:"status"`
		ProfileData        *monitor.ProfileData `json:"profileData"`
		CreationTimestamp  metav1.Time          `json:"createdAt"`
		RunningTimestamp   metav1.Time          `json:"startedAt"`
		CompletedTimestamp metav1.Time          `json:"completedAt"`
	}

	// SSHPortData 定义 SSH 端口信息的结构体
	SSHPortData struct {
		IP       string `json:"IP"`
		NodePort int32  `json:"nodePort"`
		Username string `json:"username"`
	}

	// SSHResp 定义返回的 SSH 信息的结构体
	SSHResp struct {
		Open bool        `json:"open"` // SSH 是否开启
		Data SSHPortData `json:"data"` // SSH 端口信息
	}

	PodDetail struct {
		Name      string          `json:"name"`
		Namespace string          `json:"namespace"`
		NodeName  *string         `json:"nodename"`
		IP        string          `json:"ip"`
		Port      string          `json:"port"`
		Resource  v1.ResourceList `json:"resource"`
		Phase     v1.PodPhase     `json:"phase"`
	}
)

// GetJobDetail godoc
// @Summary 获取jupyter详情
// @Description 调用k8s get crd
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "Job Name"
// @Success 200 {object} resputil.Response[any] "任务描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/detail [get]
func (mgr *VolcanojobMgr) GetJobDetail(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	// find from db
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	var profileData *monitor.ProfileData
	if job.ProfileData != nil {
		profileData = job.ProfileData.Data()
	}

	jobDetail := JobDetailResp{
		Name:               job.Name,
		Namespace:          job.Attributes.Data().Namespace,
		Username:           job.User.Nickname,
		JobName:            job.JobName,
		JobType:            job.JobType,
		Queue:              job.Account.Nickname,
		Status:             job.Status,
		Resources:          job.Resources.Data(),
		ProfileData:        profileData,
		CreationTimestamp:  metav1.NewTime(job.CreationTimestamp),
		RunningTimestamp:   metav1.NewTime(job.RunningTimestamp),
		CompletedTimestamp: metav1.NewTime(job.CompletedTimestamp),
	}
	resputil.Success(c, jobDetail)
}

func getJob(c context.Context, name string, token *util.JWTMessage) (*model.Job, error) {
	j := query.Job
	if token.RolePlatform == model.RoleAdmin {
		return j.WithContext(c).
			Preload(j.Account).
			Preload(j.User).
			Where(j.JobName.Eq(name)).
			First()
	} else {
		return j.WithContext(c).
			Preload(j.Account).
			Preload(j.User).
			Where(j.JobName.Eq(name)).
			Where(j.AccountID.Eq(token.AccountID)).
			Where(j.UserID.Eq(token.UserID)).
			First()
	}
}

// GetSSHPortDetail godoc
// @Summary 获取作业的SSH端口信息
// @Description 根据作业名称获取该作业的SSH端口相关信息
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "Job Name"
// @Success 200 {object} resputil.Response[SSHResp] "SSH端口信息"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/ssh [post]
func (mgr *VolcanojobMgr) GetSSHPortDetail(c *gin.Context) {
	var req JobActionReq
	// 解析 URI 参数
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	// 根据作业名称获取作业详情
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 检查作业是否处于 Running 状态
	if job.Status != batch.Running {
		resputil.Success(c, SSHResp{Open: false, Data: SSHPortData{}})
		return
	}

	// 获取作业的用户名
	username := job.User.Name

	// 构造 namespace 和 podName
	namespace := job.Attributes.Data().Namespace
	podName := fmt.Sprintf("%s-default0-0", req.JobName)

	// 获取 Pod 信息
	var pod v1.Pod
	if err := mgr.client.Get(c, client.ObjectKey{Namespace: namespace, Name: podName}, &pod); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 检查 annotations 中是否包含 "crater.raids.io/open-ssh: true"
	if pod.Annotations[AnnotationKeyOpenSSH] != "true" {
		resputil.Success(c, SSHResp{Open: false, Data: SSHPortData{}})
		return
	}

	// 查找 annotation 中的 nodeport 信息
	var sshPort SSHResp
	for key, value := range pod.Annotations {
		if strings.HasPrefix(key, NodePortLabelKey+"ssh") {
			var nodeportData tool.PodNodeport
			if err := json.Unmarshal([]byte(value), &nodeportData); err == nil {
				// 使用从 annotation 获取到的 IP 和 NodePort
				sshPort = SSHResp{
					Open: true, // 表示开启 SSH
					Data: SSHPortData{
						IP:       nodeportData.Address,
						NodePort: nodeportData.NodePort,
						Username: username,
					},
				}
			}
		}
	}

	// 返回 SSH 端口信息
	resputil.Success(c, sshPort)
}

// GetJobPods godoc
// @Summary 获取任务的Pod列表
// @Description 获取任务的Pod列表
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "Job Name"
// @Success 200 {object} resputil.Response[any] "Pod列表"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/pods [get]
func (mgr *VolcanojobMgr) GetJobPods(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	// find from db
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// every pod has label crater.raids.io/base-url: tf-liyilong-1314c
	// get pods with label selector
	vcjob := job.Attributes.Data()
	var podList = &v1.PodList{}
	if value, ok := vcjob.Labels[LabelKeyBaseURL]; !ok {
		resputil.Error(c, "label not found", resputil.NotSpecified)
		return
	} else {
		labels := client.MatchingLabels{LabelKeyBaseURL: value}
		err = mgr.client.List(c, podList, client.InNamespace(vcjob.Namespace), labels)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
	}

	PodDetails := []PodDetail{}
	for i := range podList.Items {
		pod := &podList.Items[i]

		// resource
		resources := utils.CalculateRequsetsByContainers(pod.Spec.Containers)

		portStr := ""
		for _, port := range pod.Spec.Containers[0].Ports {
			portStr += fmt.Sprintf("%s:%d,", port.Name, port.ContainerPort)
		}
		if portStr != "" {
			portStr = portStr[:len(portStr)-1]
		}
		podDetail := PodDetail{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			NodeName:  ptr.To(pod.Spec.NodeName),
			IP:        pod.Status.PodIP,
			Port:      portStr,
			Resource:  resources,
			Phase:     pod.Status.Phase,
		}
		PodDetails = append(PodDetails, podDetail)
	}

	resputil.Success(c, PodDetails)
}

// GetJobYaml godoc
// @Summary 获取vcjob Yaml详情
// @Description 调用k8s get crd
// @Tags vcjob-jupyter
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "Job Name"
// @Success 200 {object} resputil.Response[any] "任务yaml"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/yaml [get]
func (mgr *VolcanojobMgr) GetJobYaml(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	// find from db
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	vcjob := job.Attributes.Data()

	// prune useless field
	vcjob.ObjectMeta.ManagedFields = nil

	// utilize json omitempty tag to further prune
	jsonData, err := json.Marshal(vcjob)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	var prunedJob map[string]any
	err = json.Unmarshal(jsonData, &prunedJob)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// remove status field
	delete(prunedJob, "status")

	JobYaml, err := marshalYAMLWithIndent(prunedJob, 2)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, string(JobYaml))
}

func marshalYAMLWithIndent(v any, indent int) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(indent)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GetJobTemplate godoc
// @Summary 获取任务的 template
// @Description 获取任务的 template
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "Job Name"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/template [get]
func (mgr *VolcanojobMgr) GetJobTemplate(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	db := query.Job
	job, err := db.WithContext(c).Where(db.JobName.Eq(req.JobName)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, job.Template)
}

// GetJobEvents godoc
// @Summary 获取任务的事件
// @Description 获取任务的事件
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "Job Name"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/event [get]
func (mgr *VolcanojobMgr) GetJobEvents(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	// find from db
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	vcjob := job.Attributes.Data()

	// get job events
	jobEvents, err := mgr.kubeClient.CoreV1().Events(vcjob.Namespace).List(c, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", vcjob.Name),
		TypeMeta:      metav1.TypeMeta{Kind: "Job", APIVersion: "batch.volcano.sh/v1alpha1"},
	})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	events := jobEvents.Items
	containsPodEvents := false

	// get pod events
	var podList = &v1.PodList{}
	if value, ok := vcjob.Labels[LabelKeyBaseURL]; !ok {
		resputil.Error(c, "label not found", resputil.NotSpecified)
		return
	} else {
		labels := client.MatchingLabels{LabelKeyBaseURL: value}
		err = mgr.client.List(c, podList, client.InNamespace(vcjob.Namespace), labels)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
	}

	for i := range podList.Items {
		pod := &podList.Items[i]
		podEvents, err := mgr.kubeClient.CoreV1().Events(vcjob.Namespace).List(c, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		})
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
		// 如果存在 Pod 事件，则不返回 Job 事件
		if len(podEvents.Items) > 0 && !containsPodEvents {
			containsPodEvents = true
			events = []v1.Event{}
		}
		events = append(events, podEvents.Items...)
	}

	resputil.Success(c, events)
}

// SetAlertEnabled godoc
// @Summary set AlertEnabled of the job to the opposite value
// @Description set AlertEnabled of the job to the opposite value
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/alert [put]
func (mgr *VolcanojobMgr) SetAlertEnabled(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	jobDB := query.Job
	j, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.JobName)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	preStatus := j.AlertEnabled
	if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.JobName)).Update(jobDB.AlertEnabled, !preStatus); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	message := fmt.Sprintf("Set %s AlertEnabled to %t", req.JobName, !preStatus)
	resputil.Success(c, message)
}
