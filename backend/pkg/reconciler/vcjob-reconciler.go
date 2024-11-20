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
	"time"

	"github.com/go-logr/logr"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler/vcjob"
	"github.com/raids-lab/crater/pkg/config"
	"gorm.io/datatypes"
	"gorm.io/gen"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

// VcJobReconciler reconciles a AIJob object
type VcJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    logr.Logger
}

// NewVcJobReconciler returns a new reconcile.Reconciler
func NewVcJobReconciler(crClient client.Client, scheme *runtime.Scheme) *VcJobReconciler {
	return &VcJobReconciler{
		Client: crClient,
		Scheme: scheme,
		log:    ctrl.Log.WithName("vcjob-reconciler"),
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

	// TODO(user): your logic here
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
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error(err, "unable to fetch job record")
			return ctrl.Result{Requeue: true}, err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("job not found in database")
			return ctrl.Result{}, nil
		}

		if record.Status == model.Deleted || record.Status == model.Freed {
			return ctrl.Result{}, nil
		}

		var info gen.ResultInfo
		info, err = j.WithContext(ctx).Where(j.JobName.Eq(req.Name)).Updates(model.Job{
			Status:             model.Freed,
			CompletedTimestamp: time.Now(),
			Nodes:              datatypes.NewJSONType([]string{}),
		})
		if err != nil {
			logger.Error(err, "unable to update job status to deleted")
			return ctrl.Result{Requeue: true}, err
		}
		if info.RowsAffected == 0 {
			logger.Info("job not found in database")
		}
		return ctrl.Result{}, nil
	}

	// create or update db record
	// if job not found, create a new record
	_, err = j.WithContext(ctx).Where(j.JobName.Eq(job.Name)).First()
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

	// if job found, update the record
	updateRecord := r.generateUpdateJobModel(ctx, &job)
	_, err = j.WithContext(ctx).Where(j.JobName.Eq(job.Name)).Updates(updateRecord)
	if err != nil {
		logger.Error(err, "unable to update job record")
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
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
	}, nil
}

func (r *VcJobReconciler) generateUpdateJobModel(ctx context.Context, job *batch.Job) *model.Job {
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
	return &model.Job{
		Status:             job.Status.State.Phase,
		RunningTimestamp:   runningTimestamp,
		CompletedTimestamp: completedTimestamp,
		Nodes:              datatypes.NewJSONType(nodes),
	}
}
