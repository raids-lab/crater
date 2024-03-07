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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"

	aijobapi "github.com/aisystem/ai-protal/pkg/apis/aijob/v1alpha1"
	util "github.com/aisystem/ai-protal/pkg/util"
)

// AIJobReconciler reconciles a AIJob object
type AIJobReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	log        logr.Logger
	statusChan chan<- util.JobStatusChan
}

// NewAIJobReconciler returns a new reconcile.Reconciler
func NewAIJobReconciler(client client.Client, scheme *runtime.Scheme, statusChan chan<- util.JobStatusChan) *AIJobReconciler {
	return &AIJobReconciler{
		Client:     client,
		Scheme:     scheme,
		log:        ctrl.Log.WithName("aijob-reconciler"),
		statusChan: statusChan,
	}
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

// Reconcile 主要用于同步AIJob的状态到数据库中
func (r *AIJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AIJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aijobapi.AIJob{}).
		WithEventFilter(r).
		Complete(r)
}

func (r *AIJobReconciler) Create(e event.CreateEvent) bool {
	job, ok := e.Object.(*aijobapi.AIJob)
	if !ok {
		return false
	}
	taskid := getTaskIDFromAIJob(job)
	if taskid == "" {
		return false
	}
	r.notifyJobStatus(job)
	return false
}

func (r *AIJobReconciler) Update(e event.UpdateEvent) bool {
	job, ok := e.ObjectNew.(*aijobapi.AIJob)
	if !ok {
		return false
	}
	taskid := getTaskIDFromAIJob(job)
	if taskid == "" {
		return false
	}
	r.notifyJobStatus(job)
	return false

}

func (r *AIJobReconciler) Delete(e event.DeleteEvent) bool {
	return false
}

func (r *AIJobReconciler) Generic(e event.GenericEvent) bool {
	return false
}

func getTaskIDFromAIJob(aijob *aijobapi.AIJob) string {
	return aijob.Labels[aijobapi.LabelKeyTaskID]
}

func (r *AIJobReconciler) notifyJobStatus(job *aijobapi.AIJob) {
	// Not include Pending and Init status, treat them as Pending (not started)
	if job.Status.Phase == aijobapi.Preempted || job.Status.Phase == aijobapi.Running ||
		job.Status.Phase == aijobapi.Succeeded || job.Status.Phase == aijobapi.Failed { // 是否需要加Pending状态？
		reason := ""
		if job.Status.Conditions != nil && len(job.Status.Conditions) > 0 {
			reason = job.Status.Conditions[len(job.Status.Conditions)-1].Message
		}
		c := util.JobStatusChan{
			TaskID:    getTaskIDFromAIJob(job),
			NewStatus: string(job.Status.Phase),
			Reason:    reason,
		}
		r.statusChan <- c
	}
}
