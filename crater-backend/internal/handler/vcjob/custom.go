package vcjob

import (
	"context"
	"fmt"

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
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"
)

type (
	CreateCustomReq struct {
		CreateJobCommon `json:",inline"`
		Name            string          `json:"name" binding:"required"`
		Resource        v1.ResourceList `json:"resource"`
		Image           string          `json:"image" binding:"required"`
		Command         string          `json:"command" binding:"required"`
		WorkingDir      string          `json:"workingDir" binding:"required"`
	}
)

// CreateTrainingJob godoc
// @Summary Create a training job
// @Description Create a training job
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param CreateTrainingReq body any true "CreateTrainingReq"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/training [post]
func (mgr *VolcanojobMgr) CreateTrainingJob(c *gin.Context) {
	token := util.GetToken(c)
	if token.QueueName == util.QueueNameNull {
		resputil.Error(c, "Queue not specified", resputil.QueueNotFound)
		return
	}

	var req CreateCustomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Ingress base URL
	baseURL := fmt.Sprintf("%s-%s", token.Username, uuid.New().String()[:5])
	jobName := fmt.Sprintf("single-%s", baseURL)

	// useTensorBoard
	useTensorboard := fmt.Sprintf("%t", req.UseTensorBoard)

	// 2. Env Vars (Skip)
	// 3. TODO: Node Affinity for ARM64 Nodes

	// 4. Labels and Annotations
	namespace := config.GetConfig().Workspace.Namespace
	labels := map[string]string{
		LabelKeyTaskType: "training",
		LabelKeyTaskUser: token.Username,
		LabelKeyBaseURL:  baseURL,
	}
	annotations := map[string]string{
		AnnotationKeyTaskName:       req.Name,
		AnnotationKeyUseTensorBoard: useTensorboard,
	}

	// 5. Create the pod spec
	podSpec, err := GenerateCustomPodSpec(c, token.UserID, &req)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.UserNotFound)
		return
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
					Policies: []batch.LifecyclePolicy{
						{
							Action: bus.CompleteJobAction,
							Event:  bus.TaskCompletedEvent,
						},
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

func GenerateCustomPodSpec(
	ctx context.Context,
	userID uint,
	custom *CreateCustomReq,
) (podSpec v1.PodSpec, err error) {
	volumes, volumeMounts, err := GenerateVolumeMounts(ctx, userID, custom.VolumeMounts)
	if err != nil {
		return podSpec, err
	}
	affinity := GenerateNodeAffinity(custom.Selectors)

	podSpec = v1.PodSpec{
		Affinity: affinity,
		Volumes:  volumes,
		Containers: []v1.Container{
			{
				Name:       custom.Name,
				Image:      custom.Image,
				Command:    []string{"sh", "-c", custom.Command},
				WorkingDir: custom.WorkingDir,
				Resources: v1.ResourceRequirements{
					Limits:   custom.Resource,
					Requests: custom.Resource,
				},
				Env:   custom.Envs,
				Ports: []v1.ContainerPort{},
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
		RestartPolicy: v1.RestartPolicyNever,
	}
	return podSpec, nil
}
