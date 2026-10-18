// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extender

import (
	"time"

	. "github.com/bytedance/mockey"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	schedulinginternal "volcano.sh/apis/pkg/apis/scheduling"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/service"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
)

const (
	publicQueue = "default"
	// testNamespace deliberately differs from any real config so an unmocked config.GetConfig shows up.
	testNamespace = "extender-test"
	testUserID    = uint(7)
	jobA          = "job-a"
	jobB          = "job-b"
	gpuA100       = "nvidia.com/a100"
	gpuV100       = "nvidia.com/v100"
)

var fixedNow = time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

// rl builds a resource list from name/quantity pairs.
func rl(pairs ...string) v1.ResourceList {
	list := v1.ResourceList{}
	for i := 0; i+1 < len(pairs); i += 2 {
		list[v1.ResourceName(pairs[i])] = apiresource.MustParse(pairs[i+1])
	}
	return list
}

func cpuOf(list v1.ResourceList) string {
	quantity := list[v1.ResourceCPU]
	return quantity.String()
}

// newView leaves jobPhase and podGroupPhase empty, so the default view is a job waiting for admission.
func newView(name string, res v1.ResourceList, mutators ...func(*jobView)) *jobView {
	view := &jobView{
		name:           name,
		userID:         testUserID,
		queue:          publicQueue,
		resources:      res,
		minimum:        res,
		resourceDomain: utils.GetResourceDomain(res),
		createdAt:      fixedNow,
	}
	for _, mutate := range mutators {
		mutate(view)
	}
	return view
}

func admitted(view *jobView) { view.podGroupPhase = scheduling.PodGroupInqueue }

func timedOut(view *jobView) {
	view.tolerance = 5 * time.Minute
	view.createdAt = fixedNow.Add(-10 * time.Minute)
}

func notWaiting(view *jobView) { view.jobPhase = batch.Running }

func ownedBy(userID uint) func(*jobView) {
	return func(view *jobView) { view.userID = userID }
}

func inQueue(queue string) func(*jobView) {
	return func(view *jobView) { view.queue = queue }
}

func onNodes(names ...string) func(*jobView) {
	return func(view *jobView) { view.nodes = sets.New(names...) }
}

func withMinimum(res v1.ResourceList) func(*jobView) {
	return func(view *jobView) { view.minimum = res }
}

func newSnapshot(views ...*jobView) *snapshot {
	snap := &snapshot{
		views:        views,
		byName:       make(map[string]*jobView, len(views)),
		queues:       map[string]*scheduling.Queue{},
		usageByOwner: map[ownerKey]v1.ResourceList{},
		now:          fixedNow,
	}
	for _, view := range views {
		snap.byName[view.name] = view
		if utils.IsPodGroupAdmitted(view.podGroupPhase) && !utils.IsJobPhaseTerminal(view.jobPhase) {
			key := ownerKey{userID: view.userID, queue: view.queue}
			snap.usageByOwner[key] = utils.SumResources(snap.usageByOwner[key], view.resources)
		}
	}
	return snap
}

func addQueue(snap *snapshot, name, parent string, capability v1.ResourceList) {
	snap.queues[name] = &scheduling.Queue{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       scheduling.QueueSpec{Parent: parent, Capability: capability},
	}
}

// seededServer freezes one scheduling round so decide never reaches the config service or the cache.
func seededServer(snap *snapshot) *Server {
	return &Server{
		accumulator: newSessionAccumulator(),
		logger:      logr.Discard(),
		session: &sessionState{
			settings: &settings{config: &model.SchedulerExtenderConfig{
				SchedulerExtenderEnabled:   true,
				QueueQuotaEnabled:          true,
				JobWaitingToleranceSeconds: model.DefaultJobWaitingToleranceSeconds,
			}},
			snap:    snap,
			builtAt: utils.GetLocalTime(),
		},
	}
}

func vcjobOwnerRef(job *batch.Job) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: batch.SchemeGroupVersion.String(),
		Kind:       vcJobKind,
		Name:       job.Name,
		UID:        job.UID,
		Controller: ptr.To(true),
	}
}

func vcjobRequest(name string) *requestJobInfo {
	job := &batch.Job{ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(name + "-uid")}}
	return &requestJobInfo{
		Name:      name,
		Namespace: testNamespace,
		PodGroup: &requestPodGroup{PodGroup: schedulinginternal.PodGroup{ObjectMeta: metav1.ObjectMeta{
			Name:            name + "-" + string(job.UID),
			OwnerReferences: []metav1.OwnerReference{vcjobOwnerRef(job)},
		}}},
	}
}

// stubJobNamespace points the scope guard at the fixtures; every round that reaches decide needs it.
func stubJobNamespace() {
	cfg := &config.Config{}
	cfg.Namespaces.Job = testNamespace
	Mock(config.GetConfig).Return(cfg).Build()
}

// stubQuota answers every queue with the same enabled limit; QueueQuotaSet cannot be built from here.
func stubQuota(quota map[string]string) {
	Mock((*service.QueueQuotaSet).Resolve).To(func(_ *service.QueueQuotaSet, name string) *service.ResolvedQueueQuota {
		return &service.ResolvedQueueQuota{Name: name, Enabled: true, Quota: quota}
	}).Build()
}

func newVCJob(name, namespace string, requests v1.ResourceList, mutators ...func(*batch.Job)) *batch.Job {
	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			UID:               types.UID(name + "-uid"),
			CreationTimestamp: metav1.NewTime(fixedNow.Add(-10 * time.Minute)),
			Annotations: map[string]string{
				vcjobservice.AnnotationKeyUserID:                  "7",
				vcjobservice.AnnotationKeyWaitingToleranceSeconds: "300",
			},
		},
		Spec: batch.JobSpec{
			Queue: publicQueue,
			Tasks: []batch.TaskSpec{{
				Replicas: 1,
				Template: v1.PodTemplateSpec{Spec: v1.PodSpec{Containers: []v1.Container{{
					Name:      "main",
					Resources: v1.ResourceRequirements{Requests: requests},
				}}}},
			}},
		},
	}
	for _, mutate := range mutators {
		mutate(job)
	}
	return job
}

func newPodGroup(job *batch.Job, phase scheduling.PodGroupPhase) *scheduling.PodGroup {
	return &scheduling.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:            job.Name + "-" + string(job.UID),
			Namespace:       job.Namespace,
			OwnerReferences: []metav1.OwnerReference{vcjobOwnerRef(job)},
		},
		Status: scheduling.PodGroupStatus{Phase: phase},
	}
}

func volcanoScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(batch.AddToScheme(scheme))
	utilruntime.Must(scheduling.AddToScheme(scheme))
	return scheme
}
