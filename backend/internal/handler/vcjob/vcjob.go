package vcjob

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
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
	name           string
	client         client.Client
	config         *rest.Config
	kubeClient     kubernetes.Interface
	imagePacker    packer.ImagePackerInterface
	imageRegistry  imageregistry.ImageRegistryInterface
	serviceManager crclient.ServiceManagerInterface
}

func NewVolcanojobMgr(conf *handler.RegisterConfig) handler.Manager {
	return &VolcanojobMgr{
		name:           "vcjobs",
		client:         conf.Client,
		config:         conf.KubeConfig,
		kubeClient:     conf.KubeClient,
		imagePacker:    conf.ImagePacker,
		imageRegistry:  conf.ImageRegistry,
		serviceManager: conf.ServiceManager,
	}
}

func (mgr *VolcanojobMgr) GetName() string { return mgr.name }

func (mgr *VolcanojobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *VolcanojobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.GetUserJobs)
	g.GET("all", mgr.GetAllJobsInDays)
	g.DELETE(":name", mgr.DeleteJob)

	g.GET(":name/detail", mgr.GetJobDetail)
	g.GET(":name/yaml", mgr.GetJobYaml)
	g.GET(":name/pods", mgr.GetJobPods)
	g.GET(":name/template", mgr.GetJobTemplate)
	g.GET(":name/event", mgr.GetJobEvents)
	g.PUT(":name/alert", mgr.ToggleAlertState)

	// open ssh
	g.POST(":name/ssh", mgr.OpenSSH)

	// snapshot - 通用作业快照功能，适用于 Jupyter 和 Custom 类型作业
	g.POST(":name/snapshot", mgr.CreateSnapshot)

	// jupyter
	g.POST("jupyter", mgr.CreateJupyterJob)
	g.GET(":name/token", mgr.GetJobToken)

	// webide
	g.POST("webide", mgr.CreateWebIDEJob)
	g.GET(":name/secret", mgr.GetWebIDESecret)

	// training
	g.POST("training", mgr.CreateTrainingJob)

	// tensorflow
	g.POST("tensorflow", mgr.CreateTensorflowJob)

	// pytorch
	g.POST("pytorch", mgr.CreatePytorchJob)
}

func (mgr *VolcanojobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.GetAllJobsInDays)
	// delete job
	g.DELETE(":name", mgr.DeleteJobForAdmin)
}

const (
	VolcanoSchedulerName = "volcano"

	AnnotationKeyUser         = "crater.raids.io/user"          // 用户名，以小写字母开头
	AnnotationKeyTaskName     = "crater.raids.io/task-name"     // 任务名称（可能是中文）
	AnnotationKeyTaskTemplate = "crater.raids.io/task-template" // 任务模板
	AnnotationKeyJupyter      = "crater.raids.io/jupyter-token" // Jupyter token cache
	AnnotationKeyWebIDE       = "crater.raids.io/webide-token"  // WebIDE token cache
	AnnotationKeyAlertEnabled = "crater.raids.io/alert-enabled" // 是否开启告警
	AnnotationKeySSHEnabled   = "crater.raids.io/ssh-enabled"   // SSH 缓存，格式为 "ip:port"

	// VolumeData  = "crater-rw-workspace"
	VolumeCache = "crater-cache"
	JYCache     = "jycache"

	JupyterPort     = 8888
	TensorBoardPort = 6006
	SSHPort         = 22
)

type (
	JobActionReq struct {
		JobName string `uri:"name" binding:"required"`
	}

	JupyterTokenResp struct {
		BaseURL   string `json:"baseURL"`
		Token     string `json:"token"`
		PodName   string `json:"podName"`
		Namespace string `json:"namespace"`
		FullURL   string `json:"fullURL"`
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

	Forward struct {
		Type ForwardType `json:"type"`
		Name string      `json:"name"`
		Port int32       `json:"port"`
	}

	CreateJobCommon struct {
		Name          string                       `json:"name" binding:"required"`
		VolumeMounts  []VolumeMount                `json:"volumeMounts,omitempty"`
		DatasetMounts []DatasetMount               `json:"datasetMounts,omitempty"`
		Envs          []v1.EnvVar                  `json:"envs,omitempty"`
		Selectors     []v1.NodeSelectorRequirement `json:"selectors,omitempty"`
		Template      string                       `json:"template"`
		AlertEnabled  bool                         `json:"alertEnabled"`
		Forwards      []Forward                    `json:"forwards,omitempty"`
	}
)

// DeleteJob godoc
//
//	@Summary		Delete the job
//	@Description	Delete the job by client-go
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"Job Name"
//	@Success		200		{object}	resputil.Response[any]	"Success"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/{name} [delete]
func (mgr *VolcanojobMgr) DeleteJob(c *gin.Context) {
	mgr.deleteJob(c)
}

func (mgr *VolcanojobMgr) deleteJob(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Get job record from database
	token := util.GetToken(c)
	j := query.Job
	_, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	shouldDeleteRecord := false
	shouldDeleteJob := false

	job := &batch.Job{}
	namespace := config.GetConfig().Namespaces.Job
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
//
//	@Summary		Admin delete the job
//	@Description	管理员删除用户作业
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"Job Name"
//	@Success		200		{object}	resputil.Response[any]	"Success"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/admin/vcjobs/{name} [delete]
func (mgr *VolcanojobMgr) DeleteJobForAdmin(c *gin.Context) {
	mgr.deleteJob(c)
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
		UserInfo           model.UserInfo  `json:"userInfo"`
		JobType            string          `json:"jobType"`
		Queue              string          `json:"queue"`
		Status             string          `json:"status"`
		CreationTimestamp  metav1.Time     `json:"createdAt"`
		RunningTimestamp   metav1.Time     `json:"startedAt"`
		CompletedTimestamp metav1.Time     `json:"completedAt"`
		Nodes              []string        `json:"nodes"`
		Resources          v1.ResourceList `json:"resources"`
		Locked             bool            `json:"locked"`
		PermanentLocked    bool            `json:"permanentLocked"`
		LockedTimestamp    metav1.Time     `json:"lockedTimestamp"`
	}
)

// GetUserJobs godoc
//
//	@Summary		Get the jobs of the user
//	@Description	Get the jobs of the user by client-go
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"Volcano Job List"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs [get]
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

// GetAllJobsInDays godoc
//
//	@Summary		Get all of the jobs
//	@Description	返回指定天数内的所有作业，默认为14天
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			days	query		int						false	"Number of days to look back, default is 14"	default(14)
//	@Success		200		{object}	resputil.Response[any]	"admin get Volcano Job List"
//	@Failure		400		{object}	resputil.Response[any]	"admin Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/all [get]
func (mgr *VolcanojobMgr) GetAllJobsInDays(c *gin.Context) {
	type QueryParams struct {
		Days int `form:"days"`
	}

	var req QueryParams
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Use default value of 7 if days parameter is not provided or is zero
	days := 7
	if req.Days > 0 {
		days = req.Days
	}

	j := query.Job
	q := j.WithContext(c).Preload(j.Account).Preload(query.Job.User)

	// If days is -1, don't apply time filter (return all jobs)
	// Otherwise apply the time filter for the specified days
	if req.Days != -1 {
		now := time.Now()
		lookbackPeriod := now.AddDate(0, 0, -days)
		q = q.Where(j.CreatedAt.Gte(lookbackPeriod))
	}

	jobs, err := q.Find()
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
			Name:    job.Name,
			JobName: job.JobName,
			Owner:   job.User.Nickname,
			UserInfo: model.UserInfo{
				Username: job.User.Name,
				Nickname: job.User.Nickname,
			},
			JobType:            string(job.JobType),
			Queue:              job.Account.Nickname,
			Status:             string(job.Status),
			CreationTimestamp:  metav1.NewTime(job.CreationTimestamp),
			RunningTimestamp:   metav1.NewTime(job.RunningTimestamp),
			CompletedTimestamp: metav1.NewTime(job.CompletedTimestamp),
			Nodes:              job.Nodes.Data(),
			Resources:          job.Resources.Data(),
			Locked:             job.LockedTimestamp.After(utils.GetLocalTime()),
			PermanentLocked:    utils.IsPermanentTime(job.LockedTimestamp),
			LockedTimestamp:    metav1.NewTime(job.LockedTimestamp),
		}
	}
	sort.Slice(jobList, func(i, j int) bool {
		return jobList[i].CreationTimestamp.After(jobList[j].CreationTimestamp.Time)
	})
	return jobList
}

type (
	JobDetailResp struct {
		Name               string                        `json:"name"`
		Namespace          string                        `json:"namespace"`
		Username           string                        `json:"username"`
		Nickname           string                        `json:"nickname"`
		UserInfo           model.UserInfo                `json:"userInfo"`
		JobName            string                        `json:"jobName"`
		JobType            model.JobType                 `json:"jobType"`
		Queue              string                        `json:"queue"`
		Resources          v1.ResourceList               `json:"resources"`
		Status             batch.JobPhase                `json:"status"`
		ProfileData        *monitor.ProfileData          `json:"profileData"`
		ScheduleData       *model.ScheduleData           `json:"scheduleData"`
		Events             []v1.Event                    `json:"events"`
		TerminatedStates   []v1.ContainerStateTerminated `json:"terminatedStates"`
		CreationTimestamp  metav1.Time                   `json:"createdAt"`
		RunningTimestamp   metav1.Time                   `json:"startedAt"`
		CompletedTimestamp metav1.Time                   `json:"completedAt"`
	}

	// SSHPortData 定义 SSH 端口信息的结构体
	SSHPortData struct {
		IP       string `json:"IP"`
		NodePort int32  `json:"nodePort"`
		Username string `json:"username"`
	}

	// SSHInfo 定义 SSH 信息的结构体
	SSHInfo struct {
		IP   string `json:"ip"`
		Port string `json:"port"`
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
//
//	@Summary		获取jupyter详情
//	@Description	调用k8s get crd
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"Job Name"
//	@Success		200		{object}	resputil.Response[any]	"任务描述"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/{name}/detail [get]
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

	var scheduleData *model.ScheduleData
	if job.ScheduleData != nil {
		scheduleData = job.ScheduleData.Data()
	}

	var events []v1.Event
	if job.Events != nil {
		events = job.Events.Data()
	}

	var terminatedStates []v1.ContainerStateTerminated
	if job.TerminatedStates != nil {
		terminatedStates = job.TerminatedStates.Data()
	}

	jobDetail := JobDetailResp{
		Name:      job.Name,
		Namespace: job.Attributes.Data().Namespace,
		Username:  job.User.Name,
		Nickname:  job.User.Nickname,
		UserInfo: model.UserInfo{
			Username: job.User.Name,
			Nickname: job.User.Nickname,
		},
		JobName:            job.JobName,
		JobType:            job.JobType,
		Queue:              job.Account.Nickname,
		Status:             job.Status,
		Resources:          job.Resources.Data(),
		ProfileData:        profileData,
		ScheduleData:       scheduleData,
		Events:             events,
		TerminatedStates:   terminatedStates,
		CreationTimestamp:  metav1.NewTime(job.CreationTimestamp),
		RunningTimestamp:   metav1.NewTime(job.RunningTimestamp),
		CompletedTimestamp: metav1.NewTime(job.CompletedTimestamp),
	}
	resputil.Success(c, jobDetail)
}

// OpenSSH godoc
//
//	@Summary		开启 SSH
//	@Description	开启 SSH
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"Job Name"
//	@Success		200		{object}	resputil.Response[any]	"SSH开启成功"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/{name}/ssh [post]
func (mgr *VolcanojobMgr) OpenSSH(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if job.Status != batch.Running {
		resputil.Error(c, "job is not running", resputil.NotSpecified)
		return
	}

	vcjob := job.Attributes.Data()
	namespace := job.Attributes.Data().Namespace
	podName, podLabels := getPodNameAndLabelFromJob(vcjob)

	var pod v1.Pod
	if err = mgr.client.Get(c, client.ObjectKey{Namespace: namespace, Name: podName}, &pod); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// get container name
	containerName := pod.Spec.Containers[0].Name

	// 1. Check if AnnotationKeySSHEnabled is already set, get the value and return
	if v, ok := pod.Annotations[AnnotationKeySSHEnabled]; ok {
		// 解析主机名和端口号 // Value 格式为 "ip:port"
		splits := strings.Split(v, ":")
		if len(splits) != 2 {
			resputil.Error(c, "invalid ssh enabled value", resputil.NotSpecified)
			return
		}
		resputil.Success(c, SSHInfo{
			IP:   splits[0],
			Port: splits[1],
		})
		return
	}

	// 检查并创建sshd用户，然后启动sshd服务
	commands := []string{
		"sh",
		"-c",
		`if ! id "sshd" &>/dev/null; then
			echo "Creating SSHD user and group...";
			groupadd sshd;
			useradd -u 66 -g sshd -c "SSH daemon" -d /var/lib/sshd -s /sbin/nologin -m sshd;
			echo "SSHD user creation completed";
		else
			echo "SSHD user already exists, skipping creation";
		fi && 
		service ssh restart`,
	}

	if _, err = mgr.execCommandInPod(c, namespace, podName, containerName, commands); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to setup and start ssh service in pod: %v", err), resputil.ServiceSshdNotFound)
		return
	}

	// 3. Create service for SSH
	ip, port, err := mgr.serviceManager.CreateNodePort(
		c,
		[]metav1.OwnerReference{
			{
				APIVersion:         "batch.volcano.sh/v1alpha1",
				Kind:               "Job",
				Name:               vcjob.Name,
				UID:                vcjob.UID,
				BlockOwnerDeletion: ptr.To(true),
			},
		},
		podLabels,
		ptr.To(v1.ServicePort{
			Name:       "ssh",
			Port:       SSHPort,
			Protocol:   v1.ProtocolTCP,
			TargetPort: intstr.FromInt(SSHPort),
		}),
		job.User.Name, // Pass the username
	)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create ssh service: %v", err), resputil.NotSpecified)
		return
	}

	// 4. Update the pod annotation with the SSH information
	sshInfo := SSHInfo{
		IP:   ip,
		Port: fmt.Sprintf("%d", port),
	}
	sshInfoStr := fmt.Sprintf("%s:%d", ip, port)
	pod.Annotations[AnnotationKeySSHEnabled] = sshInfoStr
	if err := mgr.client.Update(c, &pod); err != nil {
		klog.Errorf("failed to update pod annotation: %v", err)
	}

	resputil.Success(c, sshInfo)
}

// GetJobPods godoc
//
//	@Summary		获取任务的Pod列表
//	@Description	获取任务的Pod列表
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"Job Name"
//	@Success		200		{object}	resputil.Response[any]	"Pod列表"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/{name}/pods [get]
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
	if value, ok := vcjob.Labels[crclient.LabelKeyBaseURL]; !ok {
		resputil.Error(c, "label not found", resputil.NotSpecified)
		return
	} else {
		labels := client.MatchingLabels{crclient.LabelKeyBaseURL: value}
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
//
//	@Summary		获取vcjob Yaml详情
//	@Description	调用k8s get crd
//	@Tags			vcjob-jupyter
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"Job Name"
//	@Success		200		{object}	resputil.Response[any]	"任务yaml"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/{name}/yaml [get]
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
	vcjob.ManagedFields = nil

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

// GetJobTemplate godoc
//
//	@Summary		获取任务的 template
//	@Description	获取任务的 template
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"Job Name"
//	@Success		200		{object}	resputil.Response[any]	"Success"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/{name}/template [get]
func (mgr *VolcanojobMgr) GetJobTemplate(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	token := util.GetToken(c)
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, job.Template)
}

// GetJobEvents godoc
//
//	@Summary		获取任务的事件
//	@Description	获取任务的事件
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"Job Name"
//	@Success		200		{object}	resputil.Response[any]	"Success"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/{name}/event [get]
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
	if value, ok := vcjob.Labels[crclient.LabelKeyBaseURL]; !ok {
		resputil.Error(c, "label not found", resputil.NotSpecified)
		return
	} else {
		labels := client.MatchingLabels{crclient.LabelKeyBaseURL: value}
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

// ToggleAlertState godoc
//
//	@Summary		set AlertEnabled of the job to the opposite value
//	@Description	set AlertEnabled of the job to the opposite value
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	resputil.Response[any]	"Success"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/{name}/alert [put]
func (mgr *VolcanojobMgr) ToggleAlertState(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	preStatus := job.AlertEnabled
	j := query.Job
	if _, err := j.WithContext(c).Where(j.JobName.Eq(req.JobName)).Update(j.AlertEnabled, !preStatus); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	message := fmt.Sprintf("Set %s AlertEnabled to %t", req.JobName, !preStatus)
	resputil.Success(c, message)
}
