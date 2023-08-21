// /*
// Copyright 2017 The Kubernetes Authors.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	aijobapi "k8s.io/ai-task-controller/pkg/apis/aijob/v1alpha1"

	"github.com/sirupsen/logrus"
	"k8s.io/ai-task-controller/pkg/constants"
	"k8s.io/ai-task-controller/pkg/control"
	commonutil "k8s.io/ai-task-controller/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "aijob-controller"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

// JobController is the controller implementation for AIJob resources
type JobController struct {
	kubeClientSet kubeclientset.Interface
	// podControl is used to add or delete pods.
	podControl      control.PodControlInterface
	jobPendingQueue sync.Map

	// common
	client.Client
	Scheme    *runtime.Scheme
	Log       logr.Logger
	recorder  record.EventRecorder
	apiReader client.Reader
}

// todo: add gangscheduling
type GangSchedulingSetupFunc func(jc *JobController)

// NewJobController returns a new aijob controller
func NewJobController(
	mgr manager.Manager,
	// gangSchedulingSetupFunc GangSchedulingSetupFunc,
) *JobController {

	// Create clients
	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	recorder := mgr.GetEventRecorderFor(controllerName)

	controller := &JobController{
		kubeClientSet:   kubeClientSet,
		podControl:      control.RealPodControl{KubeClient: kubeClientSet, Recorder: recorder},
		jobPendingQueue: sync.Map{},
		// common
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		recorder:  recorder,
		apiReader: mgr.GetAPIReader(),
		Log:       log.Log,
	}

	// todo: gangSchedulingSetupFunc
	return controller
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
// 负责对Job的状态转移进行同步
// 1. 对于新创建的Job，加入到Pendingqueue里面
// 2. 对于运行的job，通过pod的状态去更新Job的状态
func (jc *JobController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	logger := jc.Log.WithValues(aijobapi.AIJobPlural, req.NamespacedName)

	aijob := &aijobapi.AIJob{}
	err := jc.Get(ctx, req.NamespacedName, aijob) // client.Get
	if err != nil {
		logger.Info(err.Error(), "unable to fetch AIJob", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if reconciliation is needed
	jobKey, err := KeyFunc(aijob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get jobKey for job object %#v: %v", aijob, err))
	}

	// Push aijob into PendingQueue if it is Pending or Suspended
	// 对刚创建的Job，如果是Pending状态、Suspended状态，则加入到PendingQueue里面
	if aijob.Status.Phase == aijobapi.Pending || aijob.Status.Phase == aijobapi.Suspended {
		jc.addJobToQueue(jobKey, aijob)
	} else if aijob.Status.Phase == aijobapi.Running {
		// 对于Running的job，如果pod结束了，则更新job的状态
		err = jc.UpdateJobStatus(aijob)
		if err != nil {
			logger.Error(err, "UpdateJobStatus error")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil

	// Check if aijob needs reconcile Status
}

// UpdateJobStatus updates the job status and job conditions
func (jc *JobController) UpdateJobStatus(aijob *aijobapi.AIJob) error {
	pods, err := jc.GetPodsForJob(aijob)
	if err != nil {
		return fmt.Errorf("failed to get pods for job %s/%s: %v", aijob.Namespace, aijob.Name, err)
	}
	podFailed := false
	podSucceeded := false
	// podActive := false
	msg := ""
	// 1. pod变成Failed
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodFailed {
			podFailed = true
			msg = fmt.Sprintf("AIJob %s is failed because Pod %s is failed because %s", aijob.Namespace, aijob.Name, pod.Name, pod.Status.Message)
		} else if pod.Status.Phase == corev1.PodSucceeded {
			podSucceeded = true
			msg = fmt.Sprintf("AIJob %s/%s complete successfully, Pod %s is succeeded.", aijob.Namespace, aijob.Name, pod.Name)
		} else if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
			// podActive = true
		}
	}
	oldStatus := aijob.Status.DeepCopy()
	jobStatus := &aijob.Status
	if jobStatus.Phase == aijobapi.Running {
		if podFailed {
			jc.recorder.Event(aijob, corev1.EventTypeNormal, commonutil.JobFailedReason, msg)
			if jobStatus.CompletionTime == nil {
				now := metav1.Now()
				jobStatus.CompletionTime = &now
			}
			err := commonutil.UpdateJobConditionsAndStatus(jobStatus, aijobapi.Failed, aijobapi.JobFailed, commonutil.JobFailedReason, msg)
			if err != nil {
				commonutil.LoggerForJob(aijob).Infof("Append job condition error: %v", err)
				return err
			}
			quotaInfo := GetQuotaInfo(aijob.Namespace, aijob.Namespace)
			quotaInfo.DeleteJob(aijob)
		} else if podSucceeded {
			jc.recorder.Event(aijob, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
			if jobStatus.CompletionTime == nil {
				now := metav1.Now()
				jobStatus.CompletionTime = &now
			}
			err := commonutil.UpdateJobConditionsAndStatus(jobStatus, aijobapi.Succeeded,
				aijobapi.JobSucceeded, commonutil.JobSucceededReason, msg)
			if err != nil {
				commonutil.LoggerForJob(aijob).Infof("Append job condition error: %v", err)
				return err
			}
			quotaInfo := GetQuotaInfo(aijob.Namespace, aijob.Namespace)
			quotaInfo.DeleteJob(aijob)
		}
	}
	// No need to update the job status if the status hasn't changed since last time.
	if !reflect.DeepEqual(*oldStatus, jobStatus) {
		return jc.UpdateJobStatusInAPIServer(aijob)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (jc *JobController) SetupWithManager(mgr ctrl.Manager, controllerThreads int) error {
	c, err := controller.New(jc.ControllerName(), mgr, controller.Options{
		Reconciler:              jc,
		MaxConcurrentReconciles: controllerThreads,
	})
	if err != nil {
		return err
	}

	// using onOwnerCreateFunc is easier to set defaults
	// 只对Create的操作进行处理，加入到jobPendingQueue中
	if err = c.Watch(
		&source.Kind{Type: &aijobapi.AIJob{}},
		&handler.EnqueueRequestForObject{},
		predicate.Funcs{
			CreateFunc: jc.onJobCreateFunc(),
			UpdateFunc: jc.onJobUpdateFunc(),
			DeleteFunc: jc.onJobDeleteFunc(),
		},
	); err != nil {
		return err
	}

	// inject watching for job related pod
	if err = c.Watch(
		&source.Kind{Type: &corev1.Pod{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &aijobapi.AIJob{},
		}, predicate.Funcs{
			CreateFunc: jc.onPodCreateFunc(),
			UpdateFunc: jc.onPodUpdateFunc(),
			DeleteFunc: func(e event.DeleteEvent) bool {
				// Ignore delete events for which there is no corresponding object
				// in the cache (for example if the manager is restarted).
				return false
			},
		}); err != nil {
		return err
	}

	return nil
}

// RunScheduling checks for jobPendingQueue, and schedule the expected jobs
func (jc *JobController) RunScheduling(stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		case <-time.After(1 * time.Second):

			waitingList := &JobPQ{}

			jc.jobPendingQueue.Range(func(key, value interface{}) bool {
				//1. todo:筛选出符合调度条件的job
				job := value.(*aijobapi.AIJob)
				//2. 插入到pq里面
				waitingList.PushJob(job)
				return true
			})

			//3. 从pq里面取出job
			for waitingList.Len() > 0 {
				job := waitingList.PopJob()
				key, _ := KeyFunc(job)
				//4. 调度job
				if err := jc.scheduleJob(job); err == nil {
					// 删除PendingQueue里面的job
					jc.jobPendingQueue.Delete(key)
				} else {
					klog.Errorf("schedule job %s failed: %v", key, err)
				}
			}
		}
	}
}

func (jc *JobController) scheduleJob(job *aijobapi.AIJob) error {
	// 1. 获取job的quotainfo, todo: 定义quotaInfo的key格式
	quotaInfo := GetQuotaInfo(job.Namespace, job.Namespace)
	if quotaInfo == nil {
		return fmt.Errorf("quotaInfo %s for job %s not found", job.Namespace, job.Name)
	}
	// 2. 创建pod
	err := jc.createNewPod(job)
	if err != nil {
		return fmt.Errorf("create pod for job %s failed: %v", job.Name, err)
	}
	// 3. 修改job的状态
	commonutil.UpdateJobConditionsAndStatus(&job.Status, aijobapi.Running, aijobapi.JobRunning, "", "")

	// 4. 扣除quota
	quotaInfo.AddJob(job)

	return nil
}

// resolveControllerRef returns the job referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching job
// of the correct Kind.
func (jc *JobController) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) metav1.Object {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != jc.GetAPIGroupVersionKind().Kind {
		return nil
	}
	job, err := jc.GetJobFromInformerCache(namespace, controllerRef.Name)
	if err != nil {
		return nil
	}
	if job.GetUID() != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return job
}

func (jc *JobController) ControllerName() string {
	return controllerName
}

func (jc *JobController) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return aijobapi.SchemeGroupVersion.WithKind(aijobapi.AIJobKind)
}

func (jc *JobController) GetAPIGroupVersion() schema.GroupVersion {
	return aijobapi.SchemeGroupVersion
}

func (jc *JobController) GetGroupNameLabelValue() string {
	return aijobapi.SchemeGroupVersion.Group
}

func (jc *JobController) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &aijobapi.AIJob{}
	err := jc.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logrus.Error(err, "pytorch job not found", "namespace", namespace, "name", name)
		} else {
			logrus.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (jc *JobController) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &aijobapi.AIJob{}
	err := jc.apiReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logrus.Error(err, "pytorch job not found", "namespace", namespace, "name", name)
		} else {
			logrus.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (jc *JobController) GenOwnerReference(obj metav1.Object) *metav1.OwnerReference {
	boolPtr := func(b bool) *bool { return &b }
	controllerRef := &metav1.OwnerReference{
		APIVersion:         jc.GetAPIGroupVersion().String(),
		Kind:               jc.GetAPIGroupVersionKind().Kind,
		Name:               obj.GetName(),
		UID:                obj.GetUID(),
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

func (jc *JobController) GenLabels(jobName string) map[string]string {
	jobName = strings.Replace(jobName, "/", "-", -1)
	return map[string]string{
		constants.OperatorNameLabel: jc.ControllerName(),
		constants.JobNameLabel:      jobName,
	}
}

// onJobCreateFunc modify creation condition.
func (jc *JobController) onJobCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		aijob, ok := e.Object.(*aijobapi.AIJob)
		if !ok {
			return false
		}
		jc.Scheme.Default(aijob)

		// 过滤掉已经结束的job
		if aijob.Status.Phase == aijobapi.Failed || aijob.Status.Phase == aijobapi.Succeeded {
			return false
		}
		return true
	}
}

// onJobCreateFunc modify creation condition.
func (jc *JobController) onJobUpdateFunc() func(event.UpdateEvent) bool {
	return func(e event.UpdateEvent) bool {
		aijob, ok := e.ObjectNew.(*aijobapi.AIJob)
		if !ok {
			return false
		}
		jc.Scheme.Default(aijob)

		// 过滤掉已经结束的job
		if aijob.Status.Phase == aijobapi.Failed || aijob.Status.Phase == aijobapi.Succeeded {
			return false
		}
		return true
	}
}

// onJobCreateFunc modify creation condition.
func (jc *JobController) onJobDeleteFunc() func(event.DeleteEvent) bool {
	return func(e event.DeleteEvent) bool {
		aijob, ok := e.Object.(*aijobapi.AIJob)
		if !ok {
			return false
		}
		jc.Scheme.Default(aijob)

		key, err := KeyFunc(aijob)
		if err != nil {
			return false
		}
		// 从队列中删除
		jc.removeJobFromQueue(key)
		// quota清除
		quotaInfo := GetQuotaInfo(aijob.Namespace, aijob.Namespace)
		if quotaInfo != nil {
			quotaInfo.DeleteJob(aijob)
		}
		// 清理对应的pod
		pods, err := jc.GetPodsForJob(aijob)
		if err != nil {
			return false
		}
		for _, pod := range pods {
			jc.podControl.DeletePod(pod.Namespace, pod.Name, pod)
		}
		return false
	}
}

// UpdateJobStatusInAPIServer updates the status of this generic training job in APIServer
func (jc *JobController) UpdateJobStatusInAPIServer(job *aijobapi.AIJob) error {
	return jc.Status().Update(context.Background(), job)
}
