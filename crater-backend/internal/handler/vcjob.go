package handler

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"
)

type VolcanojobMgr struct {
	client.Client
	kubeClient kubernetes.Interface
	mu         sync.Mutex // Add a mutex to protect the ingress creation
}

func NewVolcanojobMgr(cl client.Client, kc kubernetes.Interface) Manager {
	return &VolcanojobMgr{
		Client:     cl,
		kubeClient: kc,
	}
}

func (mgr *VolcanojobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *VolcanojobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("/jupyter", mgr.CreateJupyterJob)
	g.GET("", mgr.GetJobs)
	g.GET("/:name/token", mgr.GetJobIngress)
	g.DELETE("/:name", mgr.DeleteJob)
}

func (mgr *VolcanojobMgr) RegisterAdmin(_ *gin.RouterGroup) {}

const (
	VolcanoSchedulerName = "volcano"

	LabelKeyTaskType = "crater.raids.io/task-type"
	LabelKeyTaskUser = "crater.raids.io/task-user"
	LabelKeyBaseURL  = "crater.raids.io/base-url"

	AnnotationKeyTaskName = "crater.raids.io/task-name"
	AnnotationKeyJupyter  = "crater.raids.io/jupyter-token"

	VolumeData  = "crater-workspace"
	VolumeCache = "crater-cache"

	JupyterPort = 8888
)

type (
	VolumeMount struct {
		SubPath   string `json:"subPath"`
		MountPath string `json:"mountPath"`
	}

	CreateJupyterReq struct {
		Name         string            `json:"name"`
		Resource     v1.ResourceList   `json:"resource"`
		Image        string            `json:"image"`
		VolumeMounts []VolumeMount     `json:"volumeMounts"`
		NodeSelector map[string]string `json:"nodeSelector"`
	}
)

func (mgr *VolcanojobMgr) CreateJupyterJob(c *gin.Context) {
	token := util.GetToken(c)
	if token.QueueName == util.QueueNameNull {
		resputil.Error(c, "Queue not specified", resputil.QueueNotFound)
		return
	}

	var req CreateJupyterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Ingress base URL
	baseURL := fmt.Sprintf("%s-%s", token.Username, uuid.New().String()[:5])
	jobName := fmt.Sprintf("jupyter-%s", baseURL)

	// Command to start Jupyter
	commandSchema := "start.sh jupyter lab --allow-root --NotebookApp.base_url=/jupyter/%s/"
	command := fmt.Sprintf(commandSchema, baseURL)

	// Volume Mounts
	volumes := []v1.Volume{
		{
			Name: VolumeData,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: config.GetConfig().Workspace.PVCName,
				},
			},
		},
		{
			Name: VolumeCache,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium: v1.StorageMediumMemory,
				},
			},
		},
	}

	volumeMounts := make([]v1.VolumeMount, len(req.VolumeMounts)+2)
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, "User not found", resputil.UserNotFound)
		return
	}
	volumeMounts[0] = v1.VolumeMount{
		Name:      VolumeData,
		MountPath: "/home/" + user.Name,
		SubPath:   user.Space,
	}
	volumeMounts[1] = v1.VolumeMount{
		Name:      VolumeCache,
		MountPath: "/dev/shm",
	}

	for i, vm := range req.VolumeMounts {
		volumeMounts[i+2] = v1.VolumeMount{
			Name:      VolumeData,
			SubPath:   vm.SubPath,
			MountPath: vm.MountPath,
		}
	}

	namespace := config.GetConfig().Workspace.Namespace
	labels := map[string]string{
		LabelKeyTaskType: "jupyter",
		LabelKeyTaskUser: token.Username,
		LabelKeyBaseURL:  baseURL,
	}
	annotations := map[string]string{
		AnnotationKeyTaskName: req.Name,
	}

	// create volcano job
	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: batch.JobSpec{
			MinAvailable:  1,
			SchedulerName: VolcanoSchedulerName,
			Queue:         token.QueueName,
			Policies: []batch.LifecyclePolicy{
				{
					Action: bus.RestartJobAction,
					Event:  bus.PodEvictedEvent,
				},
			},
			Tasks: []batch.TaskSpec{
				{
					Replicas: 1,
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      labels,
							Annotations: annotations,
						},
						Spec: v1.PodSpec{
							NodeSelector: req.NodeSelector,
							Volumes:      volumes,
							Containers: []v1.Container{
								{
									Name:    "jupyter-notebook",
									Image:   req.Image,
									Command: []string{"/bin/bash", "-c", command},
									Resources: v1.ResourceRequirements{
										Limits:   req.Resource,
										Requests: req.Resource,
									},
									WorkingDir: fmt.Sprintf("/home/%s", token.Username),

									Env: []v1.EnvVar{
										{Name: "GRANT_SUDO", Value: "1"},
										{Name: "CHOWN_HOME", Value: "1"},
										{Name: "NB_UID", Value: "1001"},
										{Name: "NB_USER", Value: token.Username},
									},
									Ports: []v1.ContainerPort{
										{ContainerPort: JupyterPort, Name: "notebook-port", Protocol: v1.ProtocolTCP},
									},
									SecurityContext: &v1.SecurityContext{
										AllowPrivilegeEscalation: func(b bool) *bool { return &b }(true),
										RunAsUser:                func(i int64) *int64 { return &i }(0),
										RunAsGroup:               func(i int64) *int64 { return &i }(0),
									},
									TerminationMessagePath:   "/dev/termination-log",
									TerminationMessagePolicy: v1.TerminationMessageReadFile,
									VolumeMounts:             volumeMounts,
								},
							},
						},
					},
				},
			},
		},
	}
	if err = mgr.Create(c, &job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// create service
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Selector: labels,
			Ports: []v1.ServicePort{
				{
					Name:       jobName,
					Port:       80,
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromInt(JupyterPort),
				},
			},
			SessionAffinity: v1.ServiceAffinityNone,
			Type:            v1.ServiceTypeClusterIP,
		},
	}

	err = mgr.Create(c, svc)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 添加锁
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	// 创建 Ingress，转发 Jupyter 端口
	ingressClient := mgr.kubeClient.NetworkingV1().Ingresses(namespace)

	// Get the existing Ingress
	ingress, err := ingressClient.Get(c, config.GetConfig().Workspace.IngressName, metav1.GetOptions{})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Add a new path to the first rule
	newPath := networkingv1.HTTPIngressPath{
		Path:     fmt.Sprintf("/jupyter/%s", baseURL),
		PathType: func(s networkingv1.PathType) *networkingv1.PathType { return &s }(networkingv1.PathTypePrefix),
		Backend: networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{
				Name: jobName,
				Port: networkingv1.ServiceBackendPort{
					Number: 80,
				},
			},
		},
	}

	ingress.Spec.Rules[0].HTTP.Paths = append(ingress.Spec.Rules[0].HTTP.Paths, newPath)

	// Update the Ingress
	_, err = ingressClient.Update(context.Background(), ingress, metav1.UpdateOptions{})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, job)
}

type (
	JobActionReq struct {
		JobName string `uri:"name" binding:"required"`
	}

	JobIngressResp struct {
		BaseURL string `json:"baseURL"`
		Token   string `json:"token"`
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
	for i, path := range ingress.Spec.Rules[0].HTTP.Paths {
		if strings.Contains(path.Path, baseURL) {
			ingress.Spec.Rules[0].HTTP.Paths = append(ingress.Spec.Rules[0].HTTP.Paths[:i], ingress.Spec.Rules[0].HTTP.Paths[i+1:]...)
			break
		}
	}

	// Update the Ingress
	_, err = ingressClient.Update(context.Background(), ingress, metav1.UpdateOptions{})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, nil)
}

// GetJobIngress godoc
// @Summary Get the ingress base url and jupyter token of the job
// @Description Get the token of the job by logs
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param jobName path string true "Job Name"
// @Success 200 {object} resputil.Response[JobIngressResp] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/token [get]
func (mgr *VolcanojobMgr) GetJobIngress(c *gin.Context) {
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

	// Check if the job is running
	status := job.Status.State.Phase
	if status != batch.Running {
		resputil.Error(c, "Job not running", resputil.NotSpecified)
		return
	}

	baseURL := job.Labels[LabelKeyBaseURL]

	// Check if jupyter token has been cached in the job annotations
	jupyterToken, ok := job.Annotations[AnnotationKeyJupyter]
	if ok {
		resputil.Success(c, JobIngressResp{BaseURL: baseURL, Token: jupyterToken})
		return
	}

	// Get the logs of the job pod
	var podName string
	for i := range job.Spec.Tasks {
		task := &job.Spec.Tasks[i]
		podName = fmt.Sprintf("%s-%s-0", job.Name, task.Name)
		break
	}

	logOptions := &v1.PodLogOptions{}
	logReq := mgr.kubeClient.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	logs, err := logReq.Stream(context.TODO())
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	defer logs.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(logs)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	re := regexp.MustCompile(`\?token=([a-zA-Z0-9]+)`)
	podLog := buf.String()
	matches := re.FindStringSubmatch(podLog)
	if len(matches) >= 2 {
		jupyterToken = matches[1]
	} else {
		resputil.Error(c, "Jupyter token not found", resputil.NotSpecified)
		return
	}

	if jupyterToken == "" {
		resputil.Error(c, "Jupyter token not found", resputil.NotSpecified)
		return
	}

	// Cache the jupyter token in the job annotations
	job.Annotations[AnnotationKeyJupyter] = jupyterToken
	if err := mgr.Update(c, job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, JobIngressResp{BaseURL: baseURL, Token: jupyterToken})
}

type (
	JobResp struct {
		Name              string         `json:"name"`
		JobName           string         `json:"jobName"`
		Queue             string         `json:"queue"`
		Status            batch.JobPhase `json:"status"`
		CreationTimestamp metav1.Time    `json:"createdAt"`
		RunningTimestamp  metav1.Time    `json:"startedAt"`
	}
)

// GetJobs godoc
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
func (mgr *VolcanojobMgr) GetJobs(c *gin.Context) {
	token := util.GetToken(c)

	// TODO: add indexer to list jobs by user
	jobs := &batch.JobList{}
	if err := mgr.List(c, jobs, client.MatchingLabels{LabelKeyTaskUser: token.Username}); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

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
			Queue:             job.Spec.Queue,
			Status:            job.Status.State.Phase,
			CreationTimestamp: job.CreationTimestamp,
			RunningTimestamp:  runningTimestamp,
		}
	}

	// sort jobs by creation time in descending order
	sort.Slice(jobList, func(i, j int) bool {
		return jobList[i].CreationTimestamp.After(jobList[j].CreationTimestamp.Time)
	})

	resputil.Success(c, jobList)
}
