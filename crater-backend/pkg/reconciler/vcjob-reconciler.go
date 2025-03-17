/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reconciler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/datatypes"
	"gorm.io/gen"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/handler/tool"
	"github.com/raids-lab/crater/internal/handler/vcjob"
	"github.com/raids-lab/crater/pkg/alert"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/monitor"

	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

// VcJobReconciler reconciles a AIJob object
type VcJobReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	log              logr.Logger
	prometheusClient monitor.PrometheusInterface // get monitor data
}

// NewVcJobReconciler returns a new reconcile.Reconciler
func NewVcJobReconciler(
	crClient client.Client,
	scheme *runtime.Scheme,
	prometheusClient monitor.PrometheusInterface,
) *VcJobReconciler {
	return &VcJobReconciler{
		Client:           crClient,
		Scheme:           scheme,
		log:              ctrl.Log.WithName("vcjob-reconciler"),
		prometheusClient: prometheusClient,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VcJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batch.Job{}).
		WithOptions(controller.Options{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=aisystem.github.com,resources=aijobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aisystem.github.com,resources=aijobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aisystem.github.com,resources=aijobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AIJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile

// Reconcile 主要用于同步 VcJob 的状态到数据库中
//
//nolint:gocyclo // refactor later
func (r *VcJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	j := query.Job

	var job batch.Job
	err := r.Get(ctx, req.NamespacedName, &job)

	if err != nil && !k8serrors.IsNotFound(err) {
		logger.Error(err, "unable to fetch VcJob")
		return ctrl.Result{}, nil
	}

	if k8serrors.IsNotFound(err) {
		// set job status to deleted
		var record *model.Job
		record, err = j.WithContext(ctx).Where(j.JobName.Eq(req.Name)).First()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Info("job not found in database")
				return ctrl.Result{}, nil
			} else {
				logger.Error(err, "unable to fetch job record")
				return ctrl.Result{Requeue: true}, err
			}
		}

		if record.Status == model.Deleted || record.Status == model.Freed ||
			record.Status == batch.Failed || record.Status == batch.Completed ||
			record.Status == batch.Aborted || record.Status == batch.Terminated {
			if record.ProfileData == nil {
				podName := getPodNameFromJobTemplate(record.Attributes.Data())
				profileData := r.prometheusClient.QueryProfileData(types.NamespacedName{
					Namespace: config.GetConfig().Workspace.Namespace,
					Name:      podName,
				}, record.RunningTimestamp)
				var info gen.ResultInfo
				info, err = j.WithContext(ctx).Where(j.JobName.Eq(req.Name)).Updates(model.Job{
					ProfileData: ptr.To(datatypes.NewJSONType(profileData)),
				})
				if err != nil {
					logger.Error(err, "unable to update job profile data")
					return ctrl.Result{Requeue: true}, err
				}
				if info.RowsAffected == 0 {
					logger.Info("job not found in database")
				}
			}

			return ctrl.Result{}, nil
		}

		podName := getPodNameFromJobTemplate(record.Attributes.Data())
		profileData := r.prometheusClient.QueryProfileData(types.NamespacedName{
			Namespace: config.GetConfig().Workspace.Namespace,
			Name:      podName,
		}, record.RunningTimestamp)

		var info gen.ResultInfo
		info, err = j.WithContext(ctx).Where(j.JobName.Eq(req.Name)).Updates(model.Job{
			Status:             model.Freed,
			CompletedTimestamp: time.Now(),
			ProfileData:        ptr.To(datatypes.NewJSONType(profileData)),
		})
		if err != nil {
			logger.Error(err, "unable to update job status to freed")
			return ctrl.Result{Requeue: true}, err
		}
		if info.RowsAffected == 0 {
			logger.Info("job not found in database")
		}
		return ctrl.Result{}, nil
	}

	// create or update db record
	// if job not found, create a new record
	oldRecord, err := j.WithContext(ctx).Where(j.JobName.Eq(job.Name)).First()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(err, "unable to fetch job record")
		return ctrl.Result{Requeue: true}, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		var newRecord *model.Job
		newRecord, err = r.generateCreateJobModel(ctx, &job)
		if err != nil {
			logger.Error(err, "unable to generate create job model")
			return ctrl.Result{}, err
		}
		err = j.WithContext(ctx).Create(newRecord)
		if err != nil {
			logger.Error(err, "unable to create job record")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	updateRecord := r.generateUpdateJobModel(ctx, &job, oldRecord)

	// if job found: before updating, check previous status, and send email
	dbJob, _ := j.WithContext(ctx).Where(j.JobName.Eq(job.Name)).First()
	if dbJob.AlertEnabled {
		alertMgr := alert.GetAlertMgr()

		// send email after pending
		if job.Status.State.Phase == batch.Running && dbJob.Status != batch.Running {
			if err = alertMgr.JobRunningAlert(ctx, job.Name); err != nil {
				logger.Error(err, "fail to send email")
			}
			// 检查是否要打开 ssh 端口
			err = r.checkAndOpenSSH(ctx, &job, logger)
			if err != nil {
				return ctrl.Result{}, err
			}

			// 获取 Job，为后续从 Template 里获取需要添加的 ingress 和 nodeport 做准备
			var record *model.Job
			record, err = j.WithContext(ctx).Where(j.JobName.Eq(req.Name)).First()
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					logger.Info("job not found in database")
					return ctrl.Result{}, nil
				} else {
					logger.Error(err, "unable to fetch job record")
					return ctrl.Result{Requeue: true}, err
				}
			}

			// 处理需要打开的 ingress 端口
			ingressList := r.getIngressesFromJobTemplate(record)

			// 调用 checkAndOpenIngress 函数
			err = r.checkAndOpenIngress(ctx, &job, ingressList, logger)
			if err != nil {
				logger.Error(err, "Failed to open Ingress ports")
				return ctrl.Result{}, err
			}

			// 处理需要打开的 nodeport 端口
			nodePortList := r.getNodePortsFromJobTemplate(record)

			// 调用 checkAndOpenNodePort 函数
			err = r.checkAndOpenNodePort(ctx, &job, nodePortList, logger)
			if err != nil {
				logger.Error(err, "Failed to open NodePort ports")
				return ctrl.Result{}, err
			}
		}

		// alert job failure
		if job.Status.State.Phase == batch.Failed && dbJob.Status != batch.Failed {
			if err = alertMgr.JobFailureAlert(ctx, job.Name); err != nil {
				logger.Error(err, "fail to send email")
			}
		}
	}

	// if job found, update the record
	_, err = j.WithContext(ctx).Where(j.JobName.Eq(job.Name)).Updates(updateRecord)
	if err != nil {
		logger.Error(err, "unable to update job record")
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

// 检查是否需要打开 SSH 端口
func (r *VcJobReconciler) checkAndOpenSSH(ctx context.Context, job *batch.Job, logger logr.Logger) error {
	if job.Annotations[vcjob.AnnotationKeyOpenSSH] != "true" {
		return nil
	}

	logger.Info("Checking if SSH NodePort needs to be opened", "job", job.Name)

	// 初始化 APIServerMgr 实例
	var apiServerMgr *tool.APIServerMgr
	for _, register := range handler.Registers {
		mgrInterface := register(&handler.RegisterConfig{
			Client:     r.Client,
			KubeClient: kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		})
		if mgr, ok := mgrInterface.(*tool.APIServerMgr); ok {
			apiServerMgr = mgr
			break
		}
	}

	if apiServerMgr == nil {
		logger.Error(nil, "Failed to retrieve APIServerMgr instance")
		return fmt.Errorf("failed to retrieve APIServerMgr instance")
	}

	logger.Info("Successfully retrieved APIServerMgr instance")

	// 定义 SSH NodePort 的规则
	sshNodeportMgr := tool.PodNodeportMgr{
		Name:          "ssh",
		ContainerPort: 22,
	}

	sshReq := tool.PodContainerReq{
		Namespace: job.Namespace,
		PodName:   fmt.Sprintf("%s-default0-0", job.Name),
	}

	// 调用 ProcessPodNodeport
	nodePort, err := apiServerMgr.ProcessPodNodeport(ctx, sshReq, sshNodeportMgr)
	if err != nil {
		logger.Error(err, "Failed to open SSH NodePort")
		return fmt.Errorf("failed to open SSH NodePort: %w", err)
	}

	logger.Info("SSH NodePort successfully opened", "NodePort", nodePort)
	return nil
}

func (r *VcJobReconciler) generateCreateJobModel(ctx context.Context, job *batch.Job) (*model.Job, error) {
	resources := make(v1.ResourceList, 0)
	for i := range job.Spec.Tasks {
		task := &job.Spec.Tasks[i]
		replicas := task.Replicas
		for j := range task.Template.Spec.Containers {
			container := &task.Template.Spec.Containers[j]
			for name, quantity := range container.Resources.Requests {
				quantity.Mul(int64(replicas))
				if v, ok := resources[name]; !ok {
					resources[name] = quantity
				} else {
					v.Add(quantity)

					resources[name] = v
				}
			}
		}
	}
	u := query.User
	q := query.Account

	// get user and queue
	user, err := u.WithContext(ctx).Where(u.Name.Eq(job.Labels[vcjob.LabelKeyTaskUser])).First()
	if err != nil {
		return nil, fmt.Errorf("unable to get user %s: %w", job.Labels[vcjob.LabelKeyTaskUser], err)
	}
	queue, err := q.WithContext(ctx).Where(q.Name.Eq(job.Spec.Queue)).First()
	if err != nil {
		return nil, fmt.Errorf("unable to get queue %s: %w", job.Spec.Queue, err)
	}

	// receive alert email or not
	alertEnabled, err := strconv.ParseBool(job.Annotations[vcjob.AnnotationKeyAlertEnabled])
	if err != nil {
		alertEnabled = true
	}

	return &model.Job{
		Name:              job.Annotations[vcjob.AnnotationKeyTaskName],
		JobName:           job.Name,
		UserID:            user.ID,
		AccountID:         queue.ID,
		JobType:           model.JobType(job.Labels[vcjob.LabelKeyTaskType]),
		Status:            job.Status.State.Phase,
		CreationTimestamp: job.CreationTimestamp.Time,
		Resources:         datatypes.NewJSONType(resources),
		Attributes:        datatypes.NewJSONType(job),
		Template:          job.Annotations[vcjob.AnnotationKeyTaskTemplate],
		AlertEnabled:      alertEnabled,
	}, nil
}

func (r *VcJobReconciler) generateUpdateJobModel(ctx context.Context, job *batch.Job, oldRecord *model.Job) *model.Job {
	conditions := job.Status.Conditions
	var runningTimestamp time.Time
	var completedTimestamp time.Time
	for _, condition := range conditions {
		if condition.Status == batch.Running {
			runningTimestamp = condition.LastTransitionTime.Time
		} else if condition.Status == batch.Completed || condition.Status == batch.Failed {
			completedTimestamp = condition.LastTransitionTime.Time
		}
	}
	nodes := make([]string, 0)
	for i := range job.Spec.Tasks {
		task := &job.Spec.Tasks[i]
		replicas := task.Replicas
		for j := int32(0); j < replicas; j++ {
			podName := fmt.Sprintf("%s-%s-%d", job.Name, task.Name, j)
			var pod v1.Pod
			err := r.Get(ctx, types.NamespacedName{Namespace: config.GetConfig().Workspace.Namespace, Name: podName}, &pod)
			if err != nil {
				continue
			}
			if pod.Status.Phase == v1.PodRunning {
				nodes = append(nodes, pod.Spec.NodeName)
			}
		}
	}

	// do not update nodes info if job is not running on any node
	if len(nodes) == 0 {
		if !completedTimestamp.IsZero() && oldRecord.ProfileData == nil {
			profile := r.prometheusClient.QueryProfileData(types.NamespacedName{
				Namespace: job.Namespace,
				Name:      getPodNameFromJobTemplate(job),
			}, runningTimestamp)
			if profile != nil {
				return &model.Job{
					Status:             job.Status.State.Phase,
					RunningTimestamp:   runningTimestamp,
					CompletedTimestamp: completedTimestamp,
					ProfileData:        ptr.To(datatypes.NewJSONType(profile)),
				}
			}
		}
		return &model.Job{
			Status:             job.Status.State.Phase,
			RunningTimestamp:   runningTimestamp,
			CompletedTimestamp: completedTimestamp,
		}
	}

	return &model.Job{
		Status:             job.Status.State.Phase,
		RunningTimestamp:   runningTimestamp,
		CompletedTimestamp: completedTimestamp,
		Nodes:              datatypes.NewJSONType(nodes),
	}
}

func getPodNameFromJobTemplate(job *batch.Job) string {
	for i := range job.Spec.Tasks {
		task := &job.Spec.Tasks[i]
		if task.Replicas > 0 {
			podName := fmt.Sprintf("%s-%s-%d", job.Name, task.Name, 0)
			return podName
		}
	}
	return ""
}

// extractPortMappings 从 JobTemplate 提取端口映射（ingresses/nodeports）
func extractPortMappings(record *model.Job, key string, logger logr.Logger) []map[string]any {
	mappings := []map[string]any{}

	// 检查 record.Template 是否为空
	if record.Template == "" {
		logger.Info("record.Template is empty, skipping extraction", "key", key)
		return mappings
	}

	// 解析 JSON
	var templateData map[string]any
	if err := json.Unmarshal([]byte(record.Template), &templateData); err != nil {
		logger.Error(err, "Failed to parse job template JSON")
		return mappings
	}

	// 进入 data 层级
	data, dataExists := templateData["data"].(map[string]any)
	if !dataExists {
		logger.Info("'data' field not found in template, skipping extraction", "key", key)
		return mappings
	}

	// 获取指定 key 的数据
	portData, exists := data[key]
	if !exists {
		return mappings
	}

	portList, ok := portData.([]any)
	if !ok {
		logger.Info("Invalid format in job template", "key", key)
		return mappings
	}

	// 解析数据
	for _, port := range portList {
		portMap, ok := port.(map[string]any)
		if !ok {
			continue
		}
		mappings = append(mappings, portMap)
	}

	return mappings
}

// getIngressesFromJobTemplate 获取 ingresses 映射
func (r *VcJobReconciler) getIngressesFromJobTemplate(record *model.Job) []tool.PodIngressMgr {
	ingresses := []tool.PodIngressMgr{}
	mappings := extractPortMappings(record, "ingresses", r.log)

	for _, ingressMap := range mappings {
		name, nameOk := ingressMap["name"].(string)
		portFloat, portOk := ingressMap["port"].(float64)

		if nameOk && portOk {
			ingresses = append(ingresses, tool.PodIngressMgr{Name: name, Port: int32(portFloat)})
		}
	}
	return ingresses
}

// getNodePortsFromJobTemplate 获取 nodeports 映射
func (r *VcJobReconciler) getNodePortsFromJobTemplate(record *model.Job) []tool.PodNodeportMgr {
	nodePorts := []tool.PodNodeportMgr{}
	mappings := extractPortMappings(record, "nodeports", r.log)

	for _, portMap := range mappings {
		name, nameOk := portMap["name"].(string)
		portFloat, portOk := portMap["port"].(float64)

		if nameOk && portOk {
			nodePorts = append(nodePorts, tool.PodNodeportMgr{Name: name, ContainerPort: int32(portFloat)})
		}
	}
	return nodePorts
}

// 处理 Ingress 端口
func (r *VcJobReconciler) checkAndOpenIngress(ctx context.Context, job *batch.Job,
	ingressList []tool.PodIngressMgr, logger logr.Logger) error {
	if len(ingressList) == 0 {
		return nil
	}

	logger.Info("Processing Ingress rules", "job", job.Name, "ingressCount", len(ingressList))

	// 获取 Kubernetes 客户端
	cfg := ctrl.GetConfigOrDie()
	kubeClient, err := client.New(cfg, client.Options{})
	if err != nil {
		logger.Error(err, "Failed to create controller-runtime client")
		return err
	}

	// 读取 Pod 信息
	var pod v1.Pod
	err = kubeClient.Get(ctx, client.ObjectKey{
		Namespace: job.Namespace,
		Name:      fmt.Sprintf("%s-default0-0", job.Name),
	}, &pod)

	if err != nil {
		logger.Error(err, "Failed to fetch Pod from Kubernetes API", "PodName", fmt.Sprintf("%s-default0-0", job.Name))
		return err
	}

	// 确保 pod 有正确的标签
	if _, ok := pod.Labels["crater.raids.io/task-user"]; !ok {
		logger.Error(nil, "Pod is missing required label crater.raids.io/task-user", "PodName", pod.Name)
		return fmt.Errorf("label crater.raids.io/task-user not found in Pod %s", pod.Name)
	}

	// 初始化 APIServerMgr 实例
	var apiServerMgr *tool.APIServerMgr
	for _, register := range handler.Registers {
		mgrInterface := register(&handler.RegisterConfig{
			Client:     r.Client,
			KubeClient: kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		})
		if mgr, ok := mgrInterface.(*tool.APIServerMgr); ok {
			apiServerMgr = mgr
			break
		}
	}

	if apiServerMgr == nil {
		logger.Error(nil, "Failed to retrieve APIServerMgr instance for NodePort")
		return fmt.Errorf("failed to retrieve APIServerMgr instance")
	}

	// 逐个处理 Ingress 规则
	for _, ingress := range ingressList {
		err := apiServerMgr.ProcessPodIngressRule(ctx, &pod, ingress)
		if err != nil {
			logger.Error(err, "Failed to create Ingress forwarding rule", "Ingress", ingress.Name)
			return err
		}

		logger.Info("Successfully created Ingress forwarding rule", "Ingress", ingress.Name, "Port", ingress.Port)
	}

	return nil
}

func (r *VcJobReconciler) checkAndOpenNodePort(ctx context.Context, job *batch.Job,
	nodePortList []tool.PodNodeportMgr, logger logr.Logger) error {
	if len(nodePortList) == 0 {
		return nil
	}

	logger.Info("Processing NodePort rules", "job", job.Name, "nodePortCount", len(nodePortList))

	// 初始化 APIServerMgr 实例
	var apiServerMgr *tool.APIServerMgr
	for _, register := range handler.Registers {
		mgrInterface := register(&handler.RegisterConfig{
			Client:     r.Client,
			KubeClient: kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		})
		if mgr, ok := mgrInterface.(*tool.APIServerMgr); ok {
			apiServerMgr = mgr
			break
		}
	}

	if apiServerMgr == nil {
		logger.Error(nil, "Failed to retrieve APIServerMgr instance for NodePort")
		return fmt.Errorf("failed to retrieve APIServerMgr instance")
	}

	// 逐个处理 NodePort 规则
	for _, nodePort := range nodePortList {
		nodePortReq := tool.PodContainerReq{
			Namespace: job.Namespace,
			PodName:   fmt.Sprintf("%s-default0-0", job.Name),
		}

		_, err := apiServerMgr.ProcessPodNodeport(ctx, nodePortReq, nodePort)
		if err != nil {
			logger.Error(err, "Failed to open NodePort", "NodePort", nodePort.Name)
			return err
		}
		logger.Info("Successfully opened NodePort", "NodePort", nodePort.Name, "ContainerPort", nodePort.ContainerPort)
	}

	return nil
}
