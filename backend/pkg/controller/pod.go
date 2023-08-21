// Copyright 2019 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	aijobapi "k8s.io/ai-task-controller/pkg/apis/aijob/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

const (
	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is set in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"
	// exitedWithCodeReason is the normal reason when the pod is exited because of the exit code.
	exitedWithCodeReason = "ExitedWithCode"
	// podTemplateSchedulerNameReason is the warning reason when other scheduler name is set
	// in pod templates with gang-scheduling enabled
	podTemplateSchedulerNameReason = "SettedPodTemplateSchedulerName"
)

var (
	// Prometheus metrics
	createdPodsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "created_pods_total",
		Help: "The total number of created pods",
	})
	deletedPodsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "deleted_pods_total",
		Help: "The total number of deleted pods",
	})
	failedPodsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "failed_pods_total",
		Help: "The total number of failed pods",
	})
)

// onJobCreateFunc modify creation condition.
func (jc *JobController) onPodUpdateFunc() func(event.UpdateEvent) bool {
	return func(e event.UpdateEvent) bool {
		oldPod, _ := e.ObjectOld.(*corev1.Pod)
		newPod, _ := e.ObjectNew.(*corev1.Pod)
		if oldPod.Status.Phase != newPod.Status.Phase {
			if newPod.Status.Phase == corev1.PodFailed || newPod.Status.Phase == corev1.PodSucceeded {
				return true
			}
		}
		return false
	}
}

// onJobCreateFunc modify creation condition.
func (jc *JobController) onPodCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		pod, _ := e.Object.(*corev1.Pod)

		// 只有状态变成结束的时候触发
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return true
		}
		return false
	}
}

func (jc *JobController) createNewPod(job *aijobapi.AIJob) error {

	// todo: 考虑pod的replicas个数
	podTemplate := job.Spec.Template.DeepCopy()

	// Set name for the template.
	podTemplate.Name = GenGeneralName(job.Name, "0")

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}

	// set label for the template.
	for key, value := range job.Labels {
		podTemplate.Labels[key] = value
	}

	controllerRef := jc.GenOwnerReference(job)
	err := jc.podControl.CreatePodsWithControllerRef(job.Namespace, podTemplate, job, controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Pod is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the pod keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// pod when the expectation expires.
		return nil
	} else if err != nil {
		// Since error occurred(the informer won't observe this pod),
		// we decrement the expected number of creates
		// and wait until next reconciliation
		return err
	}
	return nil
}

func (jc *JobController) GetPodsForJob(obj interface{}) ([]*corev1.Pod, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	err = jc.List(context.Background(), podlist, client.MatchingLabels(jc.GenLabels(job.GetName())), client.InNamespace(job.GetNamespace()))
	if err != nil {
		return nil, err
	}

	return JobControlledPodList(podlist.Items, job), nil
}

func GenGeneralName(jobName string, index string) string {
	n := jobName + "-" + index
	return strings.Replace(n, "/", "-", -1)
}
