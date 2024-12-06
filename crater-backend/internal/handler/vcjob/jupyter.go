package vcjob

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"
)

const (
	ThreeDaySeconds int32 = 259200
	IngressLabelKey       = "ingress.crater.raids.io" // Annotation Ingress Key

)

type (
	CreateJupyterReq struct {
		CreateJobCommon `json:",inline"`
		Name            string          `json:"name" binding:"required"`
		Resource        v1.ResourceList `json:"resource"`
		Image           string          `json:"image" binding:"required"`
		Template        string          `json:"template"`
	}
)

func (mgr *VolcanojobMgr) CreateJupyterJob(c *gin.Context) {
	token := util.GetToken(c)

	var req CreateJupyterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	exceededResources := aitaskctl.CheckResourcesBeforeCreateJob(c, token.UserID, token.AccountID, req.Resource)
	if len(exceededResources) > 0 {
		resputil.Error(c, fmt.Sprintf("%v", exceededResources), resputil.NotSpecified)
		return
	}
	// Ingress base URL
	baseURL := fmt.Sprintf("%s-%s", token.Username, uuid.New().String()[:5])
	jobName := fmt.Sprintf("jupyter-%s", baseURL)

	// useTensorBoard
	useTensorboard := fmt.Sprintf("%t", req.UseTensorBoard)

	// Command to start Jupyter
	commandSchema := "start.sh jupyter lab --allow-root --notebook-dir=/home/%s --NotebookApp.base_url=/ingress/%s/"
	command := fmt.Sprintf(commandSchema, token.Username, baseURL)

	// 1. Volume Mounts
	volumes, volumeMounts, err := GenerateNewVolumeMounts(c, token.UserID, req.VolumeMounts)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 1.1 Support NGC images
	if !strings.Contains(req.Image, "jupyter") {
		volumes = append(volumes, v1.Volume{
			Name: "bash-script-volume",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: "jupyter-start-configmap",
					},
					//nolint:mnd // 0755 is the default mode
					DefaultMode: lo.ToPtr(int32(0755)),
				},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "bash-script-volume",
			MountPath: "/usr/bin/start.sh",
			ReadOnly:  false,
			SubPath:   "start.sh",
		})

		commandSchema := "/usr/bin/start.sh jupyter lab --allow-root --notebook-dir=/home/%s --NotebookApp.base_url=/ingress/%s/"
		command = fmt.Sprintf(commandSchema, token.Username, baseURL)
	}

	// 2. Env Vars
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	userAttr := user.Attributes.Data()
	if !config.GetConfig().ACT.StrictRegisterMode {
		userAttr.UID = lo.ToPtr("1001")
		userAttr.GID = lo.ToPtr("1001")
	}
	if userAttr.UID == nil || userAttr.GID == nil {
		resputil.Error(c, "UID or GID not found", resputil.NotSpecified)
		return
	}

	//nolint:mnd // 4 is the number of default envs
	envs := make([]v1.EnvVar, len(req.Envs)+5)
	envs[0] = v1.EnvVar{Name: "GRANT_SUDO", Value: "1"}
	envs[1] = v1.EnvVar{Name: "CHOWN_HOME", Value: "1"}
	envs[2] = v1.EnvVar{Name: "NB_UID", Value: *userAttr.UID}
	envs[3] = v1.EnvVar{Name: "NB_GID", Value: *userAttr.GID}
	envs[4] = v1.EnvVar{Name: "NB_USER", Value: token.Username}
	for i, env := range req.Envs {
		envs[i+4] = env
	}

	// 3. TODO: Node Affinity for ARM64 Nodes
	affinity := GenerateNodeAffinity(req.Selectors)

	// 4. Labels and Annotations
	namespace := config.GetConfig().Workspace.Namespace
	labels := map[string]string{
		LabelKeyTaskType: "jupyter",
		LabelKeyTaskUser: token.Username,
		LabelKeyBaseURL:  baseURL,
	}

	// 设置 notebook 的 Annotation 信息
	notebookIngress := crclient.PodIngress{
		Name:   "notebook",
		Port:   8888,                                 // Notebook 在容器内的默认端口
		Prefix: fmt.Sprintf("/ingress/%s/", baseURL), // 设置唯一的 URL 前缀
	}

	// 将 Ingress 转换为 JSON 字符串并添加到 annotations
	ingressJSON, err := json.Marshal(notebookIngress)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to marshal ingress: %v", err), resputil.NotSpecified)
		return
	}

	// 初始化 annotations 并添加 notebookIngress Annotation
	annotations := map[string]string{
		AnnotationKeyTaskName:                  req.Name,
		AnnotationKeyTaskTemplate:              req.Template,
		AnnotationKeyUseTensorBoard:            useTensorboard,
		IngressLabelKey + notebookIngress.Name: string(ingressJSON), // 添加 notebookIngress Annotation
	}

	// 5. Create the pod spec
	podSpec := v1.PodSpec{
		Affinity: affinity,
		Volumes:  volumes,
		Containers: []v1.Container{
			{
				Name:    "jupyter-notebook",
				Image:   req.Image,
				Command: []string{"bash", "-c", command},
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
			// 3 days
			TTLSecondsAfterFinished: lo.ToPtr(ThreeDaySeconds),
			MinAvailable:            1,
			SchedulerName:           VolcanoSchedulerName,
			Queue:                   token.AccountName,
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

	if err = mgr.client.Create(c, &job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// create service
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "batch.volcano.sh/v1alpha1",
					Kind:               "Job",
					Name:               jobName,
					UID:                job.UID,
					BlockOwnerDeletion: lo.ToPtr(true),
				},
			},
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

	err = mgr.client.Create(c, svc)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 创建 Ingress，添加 OwnerReference
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			// 每个作业的 Service 创建单独的 Ingress
			Name:      jobName,
			Namespace: namespace,
			Annotations: map[string]string{
				// 启用 SSL 重定向
				"nginx.ingress.kubernetes.io/ssl-redirect": "true",
				// 设置请求体的最大大小
				"nginx.ingress.kubernetes.io/proxy-body-size": "20480m",
			},
			// 添加 OwnerReference
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(&job, batch.SchemeGroupVersion.WithKind("Job")),
			},
		},
		Spec: networkingv1.IngressSpec{
			// 指定 Ingress 控制器为 nginx
			IngressClassName: func(s string) *string { return &s }("nginx"),
			Rules: []networkingv1.IngressRule{
				{
					Host: config.GetConfig().Host, // 设置 Host
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     fmt.Sprintf("/ingress/%s", baseURL),
									PathType: func(s networkingv1.PathType) *networkingv1.PathType { return &s }(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: jobName,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{config.GetConfig().Host}, // 需要 TLS 的主机名
					SecretName: "crater-tls-secret",               // TLS 证书的 Secret 名称
				},
			},
		},
	}

	if err = mgr.client.Create(c, ingress); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, job)
}

// GetJobToken godoc
// @Summary Get the ingress base url and jupyter token of the job
// @Description Get the token of the job by logs
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param jobName path string true "Job Name"
// @Success 200 {object} resputil.Response[JobTokenResp] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/token [get]
func (mgr *VolcanojobMgr) GetJobToken(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	j := query.Job
	job, err := j.WithContext(c).Where(j.JobName.Eq(req.JobName)).Where(j.UserID.Eq(token.UserID)).First()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if job.JobType != model.JobTypeJupyter {
		resputil.Error(c, "Job type is not Jupyter", resputil.NotSpecified)
		return
	}

	vcjob := &batch.Job{}
	namespace := config.GetConfig().Workspace.Namespace
	if err = mgr.client.Get(c, client.ObjectKey{Name: req.JobName, Namespace: namespace}, vcjob); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if vcjob.Labels[LabelKeyTaskUser] != token.Username {
		resputil.Error(c, "Job do not belong to the user", resputil.NotSpecified)
		return
	}

	// Check if the job is running
	status := vcjob.Status.State.Phase
	if status != batch.Running {
		resputil.Error(c, "Job not running", resputil.NotSpecified)
		return
	}

	baseURL := vcjob.Labels[LabelKeyBaseURL]

	// Get the logs of the job pod
	var podName string
	for i := range vcjob.Spec.Tasks {
		task := &vcjob.Spec.Tasks[i]
		podName = fmt.Sprintf("%s-%s-0", vcjob.Name, task.Name)
		break
	}

	// Check if jupyter token has been cached in the job annotations
	jupyterToken, ok := vcjob.Annotations[AnnotationKeyJupyter]
	if ok {
		resputil.Success(c, JobTokenResp{
			BaseURL:   baseURL,
			Token:     jupyterToken,
			PodName:   podName,
			Namespace: namespace,
		})
		return
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
	vcjob.Annotations[AnnotationKeyJupyter] = jupyterToken
	if err := mgr.client.Update(c, vcjob); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, JobTokenResp{
		BaseURL:   baseURL,
		Token:     jupyterToken,
		PodName:   podName,
		Namespace: namespace,
	})
}
