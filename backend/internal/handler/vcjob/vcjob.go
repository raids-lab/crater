package vcjob

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

type VolcanojobMgr struct {
	client.Client
	kubeClient kubernetes.Interface
	mu         sync.Mutex // Add a mutex to protect the ingress creation
}

func NewVolcanojobMgr(cl client.Client, kc kubernetes.Interface) handler.Manager {
	return &VolcanojobMgr{
		Client:     cl,
		kubeClient: kc,
	}
}

func (mgr *VolcanojobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *VolcanojobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.GetUserJobs)
	g.GET("all", mgr.GetAllJobs)
	g.DELETE(":name", mgr.DeleteJob)

	g.GET(":name/log", mgr.GetJobLog)
	g.GET(":name/detail", mgr.GetJobDetail)
	g.GET(":name/yaml", mgr.GetJobYaml)

	// jupyter
	g.POST("jupyter", mgr.CreateJupyterJob)
	g.GET(":name/token", mgr.GetJupyterIngress)

	// training
	g.POST("training", mgr.CreateTrainingJob)
}

func (mgr *VolcanojobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.GetAllJobs)
}

const (
	VolcanoSchedulerName = "volcano"

	LabelKeyTaskType = "crater.raids.io/task-type"
	LabelKeyTaskUser = "crater.raids.io/task-user"
	LabelKeyBaseURL  = "crater.raids.io/base-url"

	AnnotationKeyTaskName       = "crater.raids.io/task-name"
	AnnotationKeyJupyter        = "crater.raids.io/jupyter-token"
	AnnotationKeyUseTensorBoard = "crater.raids.io/use-tensorboard"

	VolumeData  = "crater-workspace"
	VolumeCache = "crater-cache"

	JupyterPort     = 8888
	TensorBoardPort = 6006
)

type (
	JobActionReq struct {
		JobName string `uri:"name" binding:"required"`
	}

	JobIngressResp struct {
		BaseURL string `json:"baseURL"`
		Token   string `json:"token"`
	}
)

type (
	VolumeMount struct {
		SubPath   string `json:"subPath"`
		MountPath string `json:"mountPath"`
	}

	CreateJobCommon struct {
		VolumeMounts   []VolumeMount                `json:"volumeMounts"`
		Envs           []v1.EnvVar                  `json:"envs"`
		Selectors      []v1.NodeSelectorRequirement `json:"selectors"`
		UseTensorBoard bool                         `json:"useTensorBoard" binding:"required"`
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
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	job := &batch.Job{}
	namespace := config.GetConfig().Workspace.Namespace
	if err := mgr.Get(c, client.ObjectKey{Name: req.JobName, Namespace: namespace}, job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)
	if job.Labels[LabelKeyTaskUser] != token.Username {
		resputil.Error(c, "Job not found", resputil.NotSpecified)
		return
	}

	// 0. Preserve the job ingress base url
	baseURL := job.Labels[LabelKeyBaseURL]

	// 1. Delete the job
	if err := mgr.Delete(c, job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 2. Delete the service
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.JobName,
			Namespace: namespace,
		},
	}
	if err := mgr.Delete(context.Background(), svc); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 3. Delete the ingress
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	ingressClient := mgr.kubeClient.NetworkingV1().Ingresses(namespace)

	// Get the existing Ingress
	ingress, err := ingressClient.Get(c, config.GetConfig().Workspace.IngressName, metav1.GetOptions{})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Remove the path from the first rule
	// for i, path := range ingress.Spec.Rules[0].HTTP.Paths {
	// 	if strings.Contains(path.Path, baseURL) {
	// 		ingress.Spec.Rules[0].HTTP.Paths = append(ingress.Spec.Rules[0].HTTP.Paths[:i], ingress.Spec.Rules[0].HTTP.Paths[i+1:]...)
	// 		break
	// 	}
	// }

	// Remove both jupyter and tensorboard paths from the first rule
	newPaths := []networkingv1.HTTPIngressPath{}
	for _, path := range ingress.Spec.Rules[0].HTTP.Paths {
		if !strings.Contains(path.Path, baseURL) {
			newPaths = append(newPaths, path)
		}
	}
	ingress.Spec.Rules[0].HTTP.Paths = newPaths

	// Update the Ingress
	_, err = ingressClient.Update(context.Background(), ingress, metav1.UpdateOptions{})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, nil)
}

func (mgr *VolcanojobMgr) getPod(c *gin.Context, namespace, podName string) (*v1.Pod, error) {
	pod, err := mgr.kubeClient.CoreV1().Pods(namespace).Get(c, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pod, nil
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

// GetJobLog godoc
func (mgr *VolcanojobMgr) GetJobLog(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	job := &batch.Job{}
	namespace := config.GetConfig().Workspace.Namespace
	if err := mgr.Get(c, client.ObjectKey{Name: req.JobName, Namespace: namespace}, job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if job.Labels[LabelKeyTaskUser] != token.Username {
		resputil.Error(c, "Job not found", resputil.NotSpecified)
		return
	}

	// Get the logs of the job pod
	var podName string
	for i := range job.Spec.Tasks {
		task := &job.Spec.Tasks[i]
		podName = fmt.Sprintf("%s-%s-0", job.Name, task.Name)
		break
	}

	buf, err := mgr.getPodLog(c, namespace, podName)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, buf.String())
}

type (
	JobResp struct {
		Name              string         `json:"name"`
		JobName           string         `json:"jobName"`
		UserName          string         `json:"userName"`
		JobType           string         `json:"jobType"`
		Queue             string         `json:"queue"`
		Status            batch.JobPhase `json:"status"`
		CreationTimestamp metav1.Time    `json:"createdAt"`
		RunningTimestamp  metav1.Time    `json:"startedAt"`
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
	jobs := &batch.JobList{}
	if err := mgr.List(c, jobs, client.MatchingLabels{LabelKeyTaskUser: token.Username}); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	jobList := JobsToJobList(jobs)

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
	jobs := &batch.JobList{}
	if err := mgr.List(c, jobs, client.MatchingLabels{}); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	jobList := JobsToJobList(jobs)

	resputil.Success(c, jobList)
}

func JobsToJobList(jobs *batch.JobList) []JobResp {
	jobList := make([]JobResp, len(jobs.Items))
	for i := range jobs.Items {
		job := &jobs.Items[i]
		conditions := job.Status.Conditions
		var runningTimestamp metav1.Time
		for _, condition := range conditions {
			if condition.Status == batch.Running {
				runningTimestamp = *condition.LastTransitionTime
				break
			}
		}
		jobList[i] = JobResp{
			Name:              job.Annotations[AnnotationKeyTaskName],
			JobName:           job.Name,
			UserName:          job.Labels[LabelKeyTaskUser],
			JobType:           job.Labels[LabelKeyTaskType],
			Queue:             job.Spec.Queue,
			Status:            job.Status.State.Phase,
			CreationTimestamp: job.CreationTimestamp,
			RunningTimestamp:  runningTimestamp,
		}
	}
	sort.Slice(jobList, func(i, j int) bool {
		return jobList[i].CreationTimestamp.After(jobList[j].CreationTimestamp.Time)
	})
	return jobList
}

type (
	JobDetailResp struct {
		Name              string         `json:"name"`
		Namespace         string         `json:"namespace"`
		Username          string         `json:"username"`
		JobName           string         `json:"jobName"`
		Retry             string         `json:"retry"`
		Queue             string         `json:"queue"`
		Status            batch.JobPhase `json:"status"`
		CreationTimestamp metav1.Time    `json:"createdAt"`
		RunningTimestamp  metav1.Time    `json:"startedAt"`
		Duration          string         `json:"runtime"`
		PodDetails        []PodDetail    `json:"podDetails"`
		UseTensorBoard    bool           `json:"useTensorBoard"`
	}

	PodDetail struct {
		Name     string      `json:"name"`
		NodeName string      `json:"nodename"`
		IP       string      `json:"ip"`
		Port     string      `json:"port"`
		Resource string      `json:"resource"`
		Status   v1.PodPhase `json:"status"`
	}
)

// GetJobDetail godoc
// @Summary 获取jupyter详情
// @Description 调用k8s get crd
// @Tags vcjob-jupyter
// @Accept json
// @Produce json
// @Security Bearer
// @Param jobname query string true "vcjob-name"
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
	job := &batch.Job{}
	namespace := config.GetConfig().Workspace.Namespace
	if err := mgr.Get(c, client.ObjectKey{Name: req.JobName, Namespace: namespace}, job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if job.Labels[LabelKeyTaskUser] != token.Username {
		resputil.Error(c, "Job not found", resputil.NotSpecified)
		return
	}

	// 从job的annotations中获取useTensorBoard的值
	useTensorBoardStr, exists := job.Annotations[AnnotationKeyUseTensorBoard]
	useTensorBoard := false
	if exists {
		var err error
		useTensorBoard, err = strconv.ParseBool(useTensorBoardStr)
		if err != nil {
			resputil.Error(c, "Invalid useTensorBoard value", resputil.NotSpecified)
			return
		}
	}

	var jobDetail JobDetailResp
	conditions := job.Status.Conditions
	var runningTimestamp metav1.Time
	for _, condition := range conditions {
		if condition.Status == batch.Running {
			runningTimestamp = *condition.LastTransitionTime
			break
		}
	}

	retryAmount := 0
	for _, condition := range conditions {
		if condition.Status == batch.Restarting {
			retryAmount += 1
		}
	}

	var podName string
	PodDetails := []PodDetail{}
	for i := range job.Spec.Tasks {
		task := &job.Spec.Tasks[i]
		podName = fmt.Sprintf("%s-%s-0", job.Name, task.Name)
		pod, err := mgr.getPod(c, namespace, podName)
		if err != nil {
			continue
		}
		// assume one pod running one container
		if pod.Status.Phase == v1.PodRunning {
			portStr := ""
			for _, port := range pod.Spec.Containers[0].Ports {
				portStr += fmt.Sprintf("%s:%d,", port.Name, port.ContainerPort)
			}
			portStr = portStr[:len(portStr)-1]
			podDetail := PodDetail{
				Name:     pod.Name,
				NodeName: pod.Spec.NodeName,
				IP:       pod.Status.PodIP,
				Port:     portStr,
				Resource: model.ResourceListToJSON(pod.Spec.Containers[0].Resources.Requests),
				Status:   pod.Status.Phase,
			}
			PodDetails = append(PodDetails, podDetail)
		} else {
			podDetail := PodDetail{
				Name:   pod.Name,
				Status: pod.Status.Phase,
			}
			PodDetails = append(PodDetails, podDetail)
		}
	}

	jobDetail = JobDetailResp{
		Name:              job.Annotations[AnnotationKeyTaskName],
		JobName:           job.Name,
		Username:          job.Labels[LabelKeyTaskUser],
		Namespace:         namespace,
		Queue:             job.Spec.Queue,
		Status:            job.Status.State.Phase,
		CreationTimestamp: job.CreationTimestamp,
		RunningTimestamp:  runningTimestamp,
		Duration:          fmt.Sprintf("%.0fs", time.Since(runningTimestamp.Time).Seconds()),
		Retry:             fmt.Sprintf("%d", retryAmount),
		PodDetails:        PodDetails,
		UseTensorBoard:    useTensorBoard,
	}
	resputil.Success(c, jobDetail)
}

// GetJobYaml godoc
// @Summary 获取vcjob Yaml详情
// @Description 调用k8s get crd
// @Tags vcjob-jupyter
// @Accept json
// @Produce json
// @Security Bearer
// @Param jobname query string true "vcjob-name"
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
	job := &batch.Job{}
	namespace := config.GetConfig().Workspace.Namespace
	if err := mgr.Get(c, client.ObjectKey{Name: req.JobName, Namespace: namespace}, job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if job.Labels[LabelKeyTaskUser] != token.Username {
		resputil.Error(c, "Job not found", resputil.NotSpecified)
		return
	}

	JobYaml, err := yaml.Marshal(job)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, string(JobYaml))
}
