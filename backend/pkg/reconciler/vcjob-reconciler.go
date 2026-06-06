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
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/datatypes"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	"github.com/raids-lab/crater/internal/handler/vcjob"
	"github.com/raids-lab/crater/internal/service"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/pkg/alert"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/prequeuewatcher"

	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

// VcJobReconciler reconciles a AIJob object
type VcJobReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	log              logr.Logger
	prometheusClient monitor.PrometheusInterface // get monitor data
	kubeClient       kubernetes.Interface
	prequeueWatcher  *prequeuewatcher.PrequeueWatcher
	billingService   *service.BillingService
}

// NewVcJobReconciler returns a new reconcile.Reconciler
func NewVcJobReconciler(
	crClient client.Client,
	scheme *runtime.Scheme,
	prometheusClient monitor.PrometheusInterface,
	kubeClient kubernetes.Interface,
	prequeueWatcher *prequeuewatcher.PrequeueWatcher,
	billingService *service.BillingService,
) *VcJobReconciler {
	return &VcJobReconciler{
		Client:           crClient,
		Scheme:           scheme,
		log:              ctrl.Log.WithName("vcjob-reconciler"),
		prometheusClient: prometheusClient,
		kubeClient:       kubeClient,
		prequeueWatcher:  prequeueWatcher,
		billingService:   billingService,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VcJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("vcjob-reconciler").
		For(&batch.Job{}).
		WithOptions(controller.Options{}).
		Complete(r)
}

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
		// Cancel approval orders immediately as job is missing from K8s
		r.cancelPendingApprovalOrders(ctx, req.Name, "job not found in cluster")

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
		if record.Status == model.Prequeue {
			return ctrl.Result{}, nil
		}

		if record.Status == model.Deleted {
			if err = r.updateMissingJobProfile(ctx, req.Name, record); err != nil {
				logger.Error(err, "unable to update job profile data")
				return ctrl.Result{Requeue: true}, err
			}
			r.notifyPrequeue()
			return ctrl.Result{}, nil
		}

		// 如果数据库的纪录中，作业已经处于终止态，则无需将作业标记为被释放
		if record.Status == model.Deleted || record.Status == model.Freed ||
			record.Status == batch.Failed || record.Status == batch.Completed ||
			record.Status == batch.Aborted || record.Status == batch.Terminated {
			if record.Status == model.Deleted && r.billingService != nil {
				if settleErr := r.billingService.OnJobFinishedSettlement(ctx, record); settleErr != nil {
					logger.Error(settleErr, "billing final settlement hook failed for deleted job")
				}
			}
			if record.ProfileData == nil {
				if err = r.updateMissingJobProfile(ctx, req.Name, record); err != nil {
					logger.Error(err, "unable to update job profile data")
					return ctrl.Result{Requeue: true}, err
				}
			}
			return ctrl.Result{}, nil
		}

		// 作业被定时策略释放，进行性能数据收集
		podName := getPodNameFromJobTemplate(record.Attributes.Data())
		profileData := r.prometheusClient.QueryProfileData(types.NamespacedName{
			Namespace: config.GetConfig().Namespaces.Job,
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
		if r.billingService != nil {
			record.Status = model.Freed
			record.CompletedTimestamp = time.Now()
			if settleErr := r.billingService.OnJobFinishedSettlement(ctx, record); settleErr != nil {
				logger.Error(settleErr, "billing final settlement hook failed")
			}
		}
		r.notifyPrequeue()
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
		err = r.createOrUpdateJobRecord(ctx, newRecord)
		if err != nil {
			logger.Error(err, "unable to create job record")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// if job found: before updating, check previous status, and send email
	if oldRecord.AlertEnabled {
		alertMgr := alert.GetAlertMgr()

		// send email after pending
		if job.Status.State.Phase == batch.Running && oldRecord.Status != batch.Running {
			if err = alertMgr.JobRunningAlert(ctx, job.Name); err != nil {
				logger.Error(err, "fail to send email")
			}
		}

		// alert job failure
		if job.Status.State.Phase == batch.Failed && oldRecord.Status != batch.Failed {
			if err = alertMgr.JobFailureAlert(ctx, job.Name); err != nil {
				logger.Error(err, "fail to send email")
			}
		}

		// alert job complete
		if job.Status.State.Phase == batch.Completed && oldRecord.Status != batch.Completed {
			if err = alertMgr.JobCompleteAlert(ctx, job.Name); err != nil {
				logger.Error(err, "fail to send email")
			}
		}
	}

	// if job found, update the record
	updateRecord := r.generateUpdateJobModel(ctx, &job, oldRecord)
	_, err = j.WithContext(ctx).Where(j.JobName.Eq(job.Name)).Updates(updateRecord)
	if err != nil {
		logger.Error(err, "unable to update job record")
		return ctrl.Result{Requeue: true}, err
	}

	// Check if job is finished and cancel pending approval orders
	isJobActive := job.Status.State.Phase == batch.Running ||
		job.Status.State.Phase == batch.Pending

	if !isJobActive {
		if r.billingService != nil && (oldRecord.Status == batch.Running || oldRecord.Status == batch.Pending) {
			finalJob := *oldRecord
			finalJob.Status = job.Status.State.Phase
			if updateRecord.RunningTimestamp.IsZero() {
				finalJob.RunningTimestamp = oldRecord.RunningTimestamp
			} else {
				finalJob.RunningTimestamp = updateRecord.RunningTimestamp
			}
			finalJob.CompletedTimestamp = updateRecord.CompletedTimestamp
			if finalJob.CompletedTimestamp.IsZero() {
				if !job.Status.State.LastTransitionTime.IsZero() {
					finalJob.CompletedTimestamp = job.Status.State.LastTransitionTime.Time
				} else if !oldRecord.CompletedTimestamp.IsZero() {
					finalJob.CompletedTimestamp = oldRecord.CompletedTimestamp
				}
			}
			if settleErr := r.billingService.OnJobFinishedSettlement(ctx, &finalJob); settleErr != nil {
				logger.Error(settleErr, "billing final settlement hook failed")
			}
		}
		r.cancelPendingApprovalOrders(ctx, job.Name, fmt.Sprintf("job is not running (status: %s), order is canceled", job.Status.State.Phase))
	}

	if shouldActivatePrequeueOnPhaseChange(oldRecord.Status, job.Status.State.Phase) {
		r.notifyPrequeue()
	}

	return ctrl.Result{}, nil
}

func shouldActivatePrequeueOnPhaseChange(oldStatus, newStatus batch.JobPhase) bool {
	if !isReleasedJobPhase(newStatus) || isReleasedJobPhase(oldStatus) {
		return false
	}

	return oldStatus != model.Deleted && oldStatus != model.Freed && oldStatus != model.Prequeue
}

func isReleasedJobPhase(status batch.JobPhase) bool {
	return status == batch.Completed ||
		status == batch.Failed ||
		status == batch.Aborted ||
		status == batch.Terminated
}

func (r *VcJobReconciler) notifyPrequeue() {
	if r.prequeueWatcher == nil {
		return
	}
	r.prequeueWatcher.RequestFullScan()
}

func (r *VcJobReconciler) updateMissingJobProfile(ctx context.Context, jobName string, record *model.Job) error {
	if record == nil || record.ProfileData != nil || record.Attributes.Data() == nil {
		return nil
	}

	podName := getPodNameFromJobTemplate(record.Attributes.Data())
	profileData := r.prometheusClient.QueryProfileData(types.NamespacedName{
		Namespace: config.GetConfig().Namespaces.Job,
		Name:      podName,
	}, record.RunningTimestamp)

	info, err := query.Job.WithContext(ctx).Where(query.Job.JobName.Eq(jobName)).Updates(model.Job{
		ProfileData: ptr.To(datatypes.NewJSONType(profileData)),
	})
	if err != nil {
		return err
	}
	if info.RowsAffected == 0 {
		r.log.Info("job not found in database", "job", jobName)
	}
	return nil
}

func (r *VcJobReconciler) cancelPendingApprovalOrders(ctx context.Context, jobName, reason string) {
	ao := query.ApprovalOrder
	_, err := ao.WithContext(ctx).
		Where(
			ao.Name.Eq(jobName),
			ao.Status.Eq(string(model.ApprovalOrderStatusPending)),
			ao.Type.Eq(string(model.ApprovalOrderTypeJob)),
		).Updates(map[string]any{
		"status":       string(model.ApprovalOrderStatusCancelled),
		"review_notes": reason,
	})
	if err != nil {
		r.log.Error(err, "failed to cancel approval order", "job", jobName)
	}
}

func (r *VcJobReconciler) createOrUpdateJobRecord(ctx context.Context, record *model.Job) error {
	return query.Job.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "job_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"created_at",
			"updated_at",
			"deleted_at",
			"name",
			"user_id",
			"account_id",
			"job_type",
			"schedule_type",
			"waiting_tolerance_seconds",
			"status",
			"queue",
			"creation_timestamp",
			"running_timestamp",
			"completed_timestamp",
			"nodes",
			"resources",
			"attributes",
			"template",
			"alert_enabled",
			"reminded",
			"keep_when_low_resource_usage",
			"locked_timestamp",
			"profile_data",
			"schedule_data",
			"events",
			"terminated_states",
		}),
	}).Create(record)
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

	userName := job.Labels[crclient.LabelKeyTaskUser]
	user, err := u.WithContext(ctx).Where(u.Name.Eq(userName)).First()
	if err != nil {
		return nil, fmt.Errorf("unable to get user %s: %w", userName, err)
	}
	accountName := job.Labels[crclient.LalbeKeyTaskAccount]
	queue, err := q.WithContext(ctx).Where(q.Name.Eq(accountName)).First()
	if err != nil {
		queue, err = q.WithContext(ctx).Where(q.Name.Eq(job.Spec.Queue)).First()
		if err != nil {
			return nil, fmt.Errorf("unable to get queue %s and %s: %w", accountName, job.Spec.Queue, err)
		}
	}

	alertEnabled, err := strconv.ParseBool(job.Annotations[vcjob.AnnotationKeyAlertEnabled])
	if err != nil {
		alertEnabled = true
	}
	scheduleType := model.ScheduleTypeNormal
	if scheduleTypeInt, err := strconv.ParseInt(
		job.Annotations[vcjobservice.AnnotationKeyScheduleType], 10, 64,
	); err == nil {
		scheduleType = model.ScheduleType(scheduleTypeInt)
	}
	var waitingToleranceSeconds *int64
	if waitingToleranceSecondsInt, err := strconv.ParseInt(
		job.Annotations[vcjobservice.AnnotationKeyWaitingToleranceSeconds], 10, 64,
	); err == nil {
		waitingToleranceSeconds = ptr.To(waitingToleranceSecondsInt)
	}

	return &model.Job{
		Name:                    job.Annotations[vcjob.AnnotationKeyTaskName],
		JobName:                 job.Name,
		UserID:                  user.ID,
		AccountID:               queue.ID,
		JobType:                 model.JobType(job.Labels[crclient.LabelKeyTaskType]),
		ScheduleType:            ptr.To(scheduleType),
		WaitingToleranceSeconds: waitingToleranceSeconds,
		Status:                  job.Status.State.Phase,
		Queue:                   job.Spec.Queue,
		CreationTimestamp:       job.CreationTimestamp.Time,
		Resources:               datatypes.NewJSONType(resources),
		Attributes:              datatypes.NewJSONType(job),
		Template:                job.Annotations[vcjob.AnnotationKeyTaskTemplate],
		AlertEnabled:            alertEnabled,
	}, nil
}

//nolint:gocyclo // refactor later
func (r *VcJobReconciler) generateUpdateJobModel(ctx context.Context, job *batch.Job, oldRecord *model.Job) *model.Job {
	conditions := job.Status.Conditions
	status := job.Status.State.Phase
	if status == "" && (oldRecord.Status == batch.Pending || oldRecord.Status == model.Prequeue) {
		status = batch.Pending
	}

	var runningTimestamp time.Time
	var completedTimestamp time.Time
	for _, condition := range conditions {
		if condition.LastTransitionTime == nil {
			continue
		}
		switch condition.Status {
		case batch.Running:
			runningTimestamp = condition.LastTransitionTime.Time
		case batch.Completed, batch.Failed, batch.Aborted, batch.Terminated:
			completedTimestamp = condition.LastTransitionTime.Time
		}
	}
	if runningTimestamp.IsZero() && job.Status.State.Phase == batch.Running && !job.Status.State.LastTransitionTime.IsZero() {
		runningTimestamp = job.Status.State.LastTransitionTime.Time
	}
	if completedTimestamp.IsZero() &&
		(job.Status.State.Phase == batch.Completed || job.Status.State.Phase == batch.Failed ||
			job.Status.State.Phase == batch.Aborted || job.Status.State.Phase == batch.Terminated) &&
		!job.Status.State.LastTransitionTime.IsZero() {
		completedTimestamp = job.Status.State.LastTransitionTime.Time
	}

	nodes := make([]string, 0)
	for i := range job.Spec.Tasks {
		task := &job.Spec.Tasks[i]
		replicas := task.Replicas
		for j := int32(0); j < replicas; j++ {
			podName := fmt.Sprintf("%s-%s-%d", job.Name, task.Name, j)
			var pod v1.Pod
			err := r.Get(ctx, types.NamespacedName{
				Namespace: config.GetConfig().Namespaces.Job,
				Name:      podName,
			}, &pod)
			if err != nil {
				continue
			}
			if pod.Status.Phase == v1.PodRunning {
				nodes = append(nodes, pod.Spec.NodeName)
			}
		}
	}
	scheduleType := model.ScheduleTypeNormal
	if scheduleTypeInt, err := strconv.ParseInt(
		job.Annotations[vcjobservice.AnnotationKeyScheduleType], 10, 64,
	); err == nil {
		scheduleType = model.ScheduleType(scheduleTypeInt)
	}
	var waitingToleranceSeconds *int64
	if waitingToleranceSecondsInt, err := strconv.ParseInt(
		job.Annotations[vcjobservice.AnnotationKeyWaitingToleranceSeconds], 10, 64,
	); err == nil {
		waitingToleranceSeconds = ptr.To(waitingToleranceSecondsInt)
	}

	// do not update nodes info if job is not running on any node
	if len(nodes) == 0 {
		var profilePtr *datatypes.JSONType[*monitor.ProfileData]
		var terminatedStatesPtr *datatypes.JSONType[[]v1.ContainerStateTerminated]
		var eventsPtr *datatypes.JSONType[[]v1.Event]

		if !completedTimestamp.IsZero() {
			// 作业进入了终止态
			if oldRecord.ProfileData == nil {
				// 进行性能数据收集
				profile := r.prometheusClient.QueryProfileData(types.NamespacedName{
					Namespace: job.Namespace,
					Name:      getPodNameFromJobTemplate(job),
				}, runningTimestamp)
				if profile != nil {
					profilePtr = ptr.To(datatypes.NewJSONType(profile))
				}
			}
			if isReleasedJobPhase(status) {
				events := r.getNewEventsForJob(ctx, job, oldRecord)
				if len(events) > 0 {
					eventsPtr = ptr.To(datatypes.NewJSONType(events))
				}
				terminatedStates := r.getTerminatedStates(ctx, job, oldRecord)
				if len(terminatedStates) > 0 {
					terminatedStatesPtr = ptr.To(datatypes.NewJSONType(terminatedStates))
				}
			}
		}

		return &model.Job{
			Status:                  status,
			RunningTimestamp:        runningTimestamp,
			CompletedTimestamp:      completedTimestamp,
			ProfileData:             profilePtr,
			Events:                  eventsPtr,
			TerminatedStates:        terminatedStatesPtr,
			ScheduleType:            ptr.To(scheduleType),
			WaitingToleranceSeconds: waitingToleranceSeconds,
		}
	}

	// 作业运行，采集调度数据和事件
	if status == batch.Running {
		var scheduleDataPtr *datatypes.JSONType[*model.ScheduleData]
		var eventsPtr *datatypes.JSONType[[]v1.Event]
		// 采集事件
		events := r.getNewEventsForJob(ctx, job, oldRecord)
		if len(events) > 0 {
			eventsPtr = ptr.To(datatypes.NewJSONType(events))
		}
		for i := range events {
			event := &events[i]
			if event.Reason == "Pulled" {
				// 解析事件消息，获取镜像拉取时间和大小
				msg := event.Message
				var scheduleData model.ScheduleData
				err := scheduleData.Init(msg)
				if err != nil {
					continue
				}
				scheduleDataPtr = ptr.To(datatypes.NewJSONType(&scheduleData))
				break
			}
		}
		return &model.Job{
			Status:                  status,
			RunningTimestamp:        runningTimestamp,
			CompletedTimestamp:      completedTimestamp,
			Nodes:                   datatypes.NewJSONType(nodes),
			ScheduleData:            scheduleDataPtr,
			Events:                  eventsPtr,
			ScheduleType:            ptr.To(scheduleType),
			WaitingToleranceSeconds: waitingToleranceSeconds,
		}
	}

	return &model.Job{
		Status:                  status,
		RunningTimestamp:        runningTimestamp,
		CompletedTimestamp:      completedTimestamp,
		Nodes:                   datatypes.NewJSONType(nodes),
		ScheduleType:            ptr.To(scheduleType),
		WaitingToleranceSeconds: waitingToleranceSeconds,
	}
}
