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
	"sync"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler/vcjob"
	"github.com/raids-lab/crater/internal/service"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/pkg/alert"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/utils"

	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

// VcJobReconciler reconciles a AIJob object
type VcJobReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	log              logr.Logger
	prometheusClient monitor.PrometheusInterface // get monitor data
	kubeClient       kubernetes.Interface
	billingService   *service.BillingService
	deletions        sync.Map // client.ObjectKey to vcJobDeletion
	profileQueue     *vcJobProfileQueue
}

// NewVcJobReconciler returns a new reconcile.Reconciler
func NewVcJobReconciler(
	crClient client.Client,
	scheme *runtime.Scheme,
	prometheusClient monitor.PrometheusInterface,
	kubeClient kubernetes.Interface,
	billingService *service.BillingService,
) *VcJobReconciler {
	return &VcJobReconciler{
		Client:           crClient,
		Scheme:           scheme,
		log:              ctrl.Log.WithName("vcjob-reconciler"),
		prometheusClient: prometheusClient,
		kubeClient:       kubeClient,
		billingService:   billingService,
		profileQueue:     newVCJobProfileQueue(),
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VcJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		return r.profileQueue.run(ctx, r.retryJobProfile)
	})); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named("vcjob-reconciler").
		For(&batch.Job{}, builder.WithPredicates(predicate.Funcs{DeleteFunc: r.observeVCJobDeletion})).
		Owns(&scheduling.PodGroup{}, builder.WithPredicates(podGroupPhaseChanged())).
		WithEventFilter(predicate.NewPredicateFuncs(shouldReconcileVCJob)).
		WithOptions(controller.Options{}).
		Complete(r)
}

// podGroupPhaseChanged keeps pod counter and condition churn out of the queue; only the admission
// phase carries information the database record does not already have.
func podGroupPhaseChanged() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldGroup, oldOK := e.ObjectOld.(*scheduling.PodGroup)
			newGroup, newOK := e.ObjectNew.(*scheduling.PodGroup)
			if !oldOK || !newOK {
				return true
			}
			return oldGroup.Status.Phase != newGroup.Status.Phase
		},
	}
}

func shouldReconcileVCJob(object client.Object) bool {
	return object.GetLabels()[crclient.LabelKeyTaskType] != string(model.JobTypeTensorboard)
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

	// PodGroup events may enqueue an auxiliary Job without carrying its labels.
	if err == nil && !shouldReconcileVCJob(&job) {
		return ctrl.Result{}, nil
	}

	if err != nil && !k8serrors.IsNotFound(err) {
		logger.Error(err, "unable to fetch VcJob")
		return ctrl.Result{}, nil
	}

	if k8serrors.IsNotFound(err) {
		return r.reconcileMissingVCJob(ctx, req.NamespacedName)
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
		if utils.IsJobPhaseTerminal(newRecord.Status) && !r.enqueueJobProfile(job.Namespace, newRecord) {
			return ctrl.Result{RequeueAfter: time.Minute}, nil
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
	updateRecord, err := r.generateUpdateJobModel(ctx, &job, oldRecord)
	if err != nil {
		logger.Error(err, "unable to resolve job status")
		return ctrl.Result{Requeue: true}, err
	}
	_, err = j.WithContext(ctx).Where(j.JobName.Eq(job.Name)).Updates(updateRecord)
	if err != nil {
		logger.Error(err, "unable to update job record")
		return ctrl.Result{Requeue: true}, err
	}

	// Check if job is finished and cancel pending approval orders
	isJobActive := isActiveJobStatus(updateRecord.Status)

	if !isJobActive {
		if r.billingService != nil && isActiveJobStatus(oldRecord.Status) {
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

	if !updateRecord.CompletedTimestamp.IsZero() {
		oldRecord.CompletedTimestamp = updateRecord.CompletedTimestamp
	}
	if !updateRecord.RunningTimestamp.IsZero() {
		oldRecord.RunningTimestamp = updateRecord.RunningTimestamp
	}
	if updateRecord.Attributes.Data() != nil {
		oldRecord.Attributes = updateRecord.Attributes
	}
	if utils.IsJobPhaseTerminal(updateRecord.Status) && !r.enqueueJobProfile(job.Namespace, oldRecord) {
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	return ctrl.Result{}, nil
}

func isActiveJobStatus(status batch.JobPhase) bool {
	return status == batch.Running || status == batch.Pending
}

// getPodGroup follows volcano's own lookup, including the pre-1.5 name without the job UID suffix.
func (r *VcJobReconciler) getPodGroup(ctx context.Context, job *batch.Job) (*scheduling.PodGroup, error) {
	for _, name := range []string{fmt.Sprintf("%s-%s", job.Name, job.UID), job.Name} {
		var group scheduling.PodGroup
		err := r.Get(ctx, types.NamespacedName{Namespace: job.Namespace, Name: name}, &group)
		if err == nil {
			return &group, nil
		}
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
	}
	return nil, nil
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
			"waiting_tolerance_seconds",
			"status",
			"pod_group_phase",
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
	var waitingToleranceSeconds *int64
	if waitingToleranceSecondsInt, err := strconv.ParseInt(
		job.Annotations[vcjobservice.AnnotationKeyWaitingToleranceSeconds], 10, 64,
	); err == nil {
		waitingToleranceSeconds = ptr.To(waitingToleranceSecondsInt)
	}

	podGroup, err := r.getPodGroup(ctx, job)
	if err != nil {
		return nil, err
	}
	var podGroupPhase scheduling.PodGroupPhase
	if podGroup != nil {
		podGroupPhase = podGroup.Status.Phase
	}
	key := client.ObjectKeyFromObject(job)
	pods, err := r.listVCJobPods(ctx, key, job)
	if err != nil {
		return nil, err
	}
	started, ended := observedVCJobTimestamps(job)
	started = resolveJobRunningTimestamp(started, time.Time{}, terminatedContainerStates(pods))
	attributes := keepVCJobProfilePods(key, job, job, pods)

	return &model.Job{
		Name:                    job.Annotations[vcjob.AnnotationKeyTaskName],
		JobName:                 job.Name,
		UserID:                  user.ID,
		AccountID:               queue.ID,
		JobType:                 model.JobType(job.Labels[crclient.LabelKeyTaskType]),
		WaitingToleranceSeconds: waitingToleranceSeconds,
		Status:                  job.Status.State.Phase,
		PodGroupPhase:           podGroupPhase,
		Queue:                   job.Spec.Queue,
		CreationTimestamp:       job.CreationTimestamp.Time,
		RunningTimestamp:        started,
		CompletedTimestamp:      ended,
		Resources:               datatypes.NewJSONType(resources),
		Attributes:              datatypes.NewJSONType(attributes),
		Template:                job.Annotations[vcjob.AnnotationKeyTaskTemplate],
		AlertEnabled:            alertEnabled,
	}, nil
}

func (r *VcJobReconciler) generateUpdateJobModel(
	ctx context.Context, job *batch.Job, oldRecord *model.Job,
) (*model.Job, error) {
	podGroup, err := r.getPodGroup(ctx, job)
	if err != nil {
		return nil, err
	}
	var podGroupPhase scheduling.PodGroupPhase
	if podGroup != nil {
		podGroupPhase = podGroup.Status.Phase
	}
	pods, err := r.listVCJobPods(ctx, client.ObjectKeyFromObject(job), job)
	if err != nil {
		return nil, err
	}
	record := r.buildUpdateJobModel(ctx, job, oldRecord, pods)
	// Updates skips the zero value, so a missing PodGroup leaves the last observed phase in place.
	record.PodGroupPhase = podGroupPhase
	return record, nil
}

func (r *VcJobReconciler) buildUpdateJobModel(
	ctx context.Context, job *batch.Job, oldRecord *model.Job, pods []v1.Pod,
) *model.Job {
	status := job.Status.State.Phase
	runningTimestamp, completedTimestamp := observedVCJobTimestamps(job)
	record := &model.Job{Status: status, CompletedTimestamp: completedTimestamp}
	var terminatedStates []v1.ContainerStateTerminated
	if utils.IsJobPhaseTerminal(status) {
		if oldRecord.TerminatedStates != nil {
			terminatedStates = oldRecord.TerminatedStates.Data()
		}
		if len(terminatedStates) == 0 {
			terminatedStates = terminatedContainerStates(pods)
			if len(terminatedStates) > 0 {
				record.TerminatedStates = ptr.To(datatypes.NewJSONType(terminatedStates))
			}
		}
	}
	record.RunningTimestamp = resolveJobRunningTimestamp(runningTimestamp, oldRecord.RunningTimestamp, terminatedStates)

	nodes := runningVCJobNodes(pods)
	if len(nodes) > 0 {
		record.Nodes = datatypes.NewJSONType(nodes)
	}
	if tolerance, err := strconv.ParseInt(job.Annotations[vcjobservice.AnnotationKeyWaitingToleranceSeconds], 10, 64); err == nil {
		record.WaitingToleranceSeconds = ptr.To(tolerance)
	}

	saved := oldRecord.Attributes.Data()
	base := saved
	if base == nil {
		base = job
	}
	attributes := keepVCJobProfilePods(client.ObjectKeyFromObject(job), base, job, pods)
	if attributes != saved {
		record.Attributes = datatypes.NewJSONType(attributes)
	}
	if (status == batch.Running && len(nodes) > 0) || (!completedTimestamp.IsZero() && utils.IsJobPhaseTerminal(status)) {
		podNames := compactProfilePodNames(append(profilePodNames(attributes), getPodNamesFromJobTemplate(job)...))
		events := r.getNewEventsForJob(ctx, job.Namespace, podNames, oldRecord)
		if len(events) > 0 {
			record.Events = ptr.To(datatypes.NewJSONType(events))
			if status == batch.Running {
				record.ScheduleData = scheduleDataFromEvents(events)
			}
		}
	}
	return record
}

func runningVCJobNodes(pods []v1.Pod) []string {
	var nodes []string
	for i := range pods {
		if pods[i].Status.Phase == v1.PodRunning {
			nodes = append(nodes, pods[i].Spec.NodeName)
		}
	}
	return nodes
}

func scheduleDataFromEvents(events []v1.Event) *datatypes.JSONType[*model.ScheduleData] {
	for i := range events {
		if events[i].Reason == "Pulled" {
			var data model.ScheduleData
			if err := data.Init(events[i].Message); err == nil {
				return ptr.To(datatypes.NewJSONType(&data))
			}
		}
	}
	return nil
}
