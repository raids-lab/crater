package vcjob

import (
	"context"
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"
)

type (
	CreateJupyterReq struct {
		CreateJobCommon `json:",inline"`
		Name            string          `json:"name" binding:"required"`
		Resource        v1.ResourceList `json:"resource"`
		Image           string          `json:"image" binding:"required"`
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

	// useTensorBoard
	useTensorboard := fmt.Sprintf("%t", req.UseTensorBoard)

	// Command to start Jupyter
	commandSchema := "start.sh jupyter lab --allow-root --NotebookApp.base_url=/jupyter/%s/"
	command := fmt.Sprintf(commandSchema, baseURL)

	// 1. Volume Mounts
	volumes, volumeMounts, err := generateVolumeMounts(c, token.UserID, req.VolumeMounts)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.UserNotFound)
		return
	}

	// 2. Env Vars
	//nolint:gomnd // 4 is the number of default envs
	envs := make([]v1.EnvVar, len(req.Envs)+4)
	envs[0] = v1.EnvVar{Name: "GRANT_SUDO", Value: "1"}
	envs[1] = v1.EnvVar{Name: "CHOWN_HOME", Value: "1"}
	envs[2] = v1.EnvVar{Name: "NB_UID", Value: "1001"}
	envs[3] = v1.EnvVar{Name: "NB_USER", Value: token.Username}
	for i, env := range req.Envs {
		envs[i+4] = env
	}

	// 3. TODO: Node Affinity for ARM64 Nodes
	affinity := generateNodeAffinity(req.Selectors)

	// 4. Labels and Annotations
	namespace := config.GetConfig().Workspace.Namespace
	labels := map[string]string{
		LabelKeyTaskType: "jupyter",
		LabelKeyTaskUser: token.Username,
		LabelKeyBaseURL:  baseURL,
	}
	annotations := map[string]string{
		AnnotationKeyTaskName:       req.Name,
		AnnotationKeyUseTensorBoard: useTensorboard,
	}

	// 5. Create the pod spec
	podSpec := v1.PodSpec{
		Affinity: affinity,
		Volumes:  volumes,
		Containers: []v1.Container{
			{
				Name:    "jupyter-notebook",
				Image:   req.Image,
				Command: []string{"sh", "-c", command},
				Resources: v1.ResourceRequirements{
					Limits:   req.Resource,
					Requests: req.Resource,
				},
				WorkingDir: fmt.Sprintf("/home/%s", token.Username),

				Env: envs,
				Ports: []v1.ContainerPort{
					{ContainerPort: JupyterPort, Name: "notebook-port", Protocol: v1.ProtocolTCP},
				},
				SecurityContext: &v1.SecurityContext{
					AllowPrivilegeEscalation: lo.ToPtr(true),
					RunAsUser:                lo.ToPtr(int64(0)),
					RunAsGroup:               lo.ToPtr(int64(0)),
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: v1.TerminationMessageReadFile,
				VolumeMounts:             volumeMounts,
			},
		},
	}

	// 6. Create volcano job
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
						Spec: podSpec,
					},
				},
			},
		},
	}

	// 添加 TensorBoard 端口映射
	if req.UseTensorBoard {
		podSpec.Containers[0].Ports = append(podSpec.Containers[0].Ports, v1.ContainerPort{
			ContainerPort: TensorBoardPort, // TensorBoard 默认端口
			Name:          "tb-port",
			Protocol:      v1.ProtocolTCP,
		})
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

	// 更新 Service：如果使用TensorBoard，添加额外的端口映射
	if req.UseTensorBoard {
		svc.Spec.Ports = append(svc.Spec.Ports, v1.ServicePort{
			Name:       "tensorboard",
			Port:       81,
			Protocol:   v1.ProtocolTCP,
			TargetPort: intstr.FromInt(TensorBoardPort),
		})
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

	// req.UseTensorBoard为true, 用户希望暴露TensorBoard
	if req.UseTensorBoard {
		tensorboardPath := networkingv1.HTTPIngressPath{
			Path:     fmt.Sprintf("/tensorboard/%s", jobName),
			PathType: func(s networkingv1.PathType) *networkingv1.PathType { return &s }(networkingv1.PathTypePrefix),
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: jobName,
					Port: networkingv1.ServiceBackendPort{
						Number: 81,
					},
				},
			},
		}
		ingress.Spec.Rules[0].HTTP.Paths = append(ingress.Spec.Rules[0].HTTP.Paths, tensorboardPath)
	}

	// Update the Ingress
	_, err = ingressClient.Update(context.Background(), ingress, metav1.UpdateOptions{})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, job)
}

// GetJupyterIngress godoc
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
func (mgr *VolcanojobMgr) GetJupyterIngress(c *gin.Context) {
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

	buf, err := mgr.getPodLog(c, namespace, podName)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	re := regexp.MustCompile(`\?token=([a-zA-Z0-9]+)`)
	matches := re.FindStringSubmatch(buf.String())
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
