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
	"testing"
	"time"

	. "github.com/bytedance/mockey"
	. "github.com/smartystreets/goconvey/convey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
)

const (
	jobAdmitted = "job-admitted"
	jobPending  = "job-pending"
)

func TestNewJobView(t *testing.T) {
	PatchConvey("vcjob without a pod group", t, func() {
		view := newJobView(newVCJob(jobA, testNamespace, rl("cpu", "1")), nil)
		So(view.name, ShouldEqual, jobA)
		So(view.userID, ShouldEqual, testUserID)
		So(view.queue, ShouldEqual, publicQueue)
		So(view.resources.Cpu().String(), ShouldEqual, "1")
		So(view.minimum.Cpu().String(), ShouldEqual, "1")
		So(view.resourceDomain, ShouldEqual, utils.ResourceDomainCPUOnly)
		So(view.jobPhase, ShouldEqual, batch.JobPhase(""))
		So(view.podGroupPhase, ShouldEqual, scheduling.PodGroupPhase(""))
		So(view.createdAt, ShouldEqual, fixedNow.Add(-10*time.Minute))
		So(view.tolerance, ShouldEqual, 5*time.Minute)
	})

	PatchConvey("pod group supplies the admission phase and the minimum demand", t, func() {
		job := newVCJob(jobA, testNamespace, rl("cpu", "1"))
		group := newPodGroup(job, scheduling.PodGroupInqueue)
		group.Spec.MinResources = ptr.To(rl("cpu", "2"))
		view := newJobView(job, group)
		So(view.podGroupPhase, ShouldEqual, scheduling.PodGroupInqueue)
		So(view.minimum.Cpu().String(), ShouldEqual, "2")
		So(view.resources.Cpu().String(), ShouldEqual, "1")
	})
}

func TestIsTimedOut(t *testing.T) {
	PatchConvey("the tolerance boundary is exclusive", t, func() {
		view := newJobView(newVCJob(jobA, testNamespace, rl("cpu", "1"), func(job *batch.Job) {
			job.CreationTimestamp = metav1.NewTime(fixedNow.Add(-300 * time.Second))
		}), nil)
		So((&snapshot{now: fixedNow}).isTimedOut(view), ShouldBeFalse)
		So((&snapshot{now: fixedNow.Add(time.Second)}).isTimedOut(view), ShouldBeTrue)
	})

	PatchConvey("only a waiting job can time out", t, func() {
		snap := &snapshot{now: fixedNow}
		for _, phase := range []batch.JobPhase{"", batch.Pending} {
			view := newJobView(newVCJob(jobA, testNamespace, rl("cpu", "1"), func(job *batch.Job) {
				job.Status.State.Phase = phase
			}), nil)
			So(snap.isTimedOut(view), ShouldBeTrue)
		}
		for _, phase := range []batch.JobPhase{batch.Running, batch.Completed} {
			view := newJobView(newVCJob(jobA, testNamespace, rl("cpu", "1"), func(job *batch.Job) {
				job.Status.State.Phase = phase
			}), nil)
			So(snap.isTimedOut(view), ShouldBeFalse)
		}
	})

	PatchConvey("a job without the annotation never times out", t, func() {
		view := newJobView(newVCJob(jobA, testNamespace, rl("cpu", "1"), func(job *batch.Job) {
			delete(job.Annotations, vcjobservice.AnnotationKeyWaitingToleranceSeconds)
		}), nil)
		So((&snapshot{now: fixedNow.Add(time.Hour)}).isTimedOut(view), ShouldBeFalse)
	})
}

func TestAnnotationParsers(t *testing.T) {
	PatchConvey("annotationUserID", t, func() {
		So(annotationUserID(nil), ShouldEqual, 0)
		for _, raw := range []string{"", "abc", "-1"} {
			So(annotationUserID(map[string]string{vcjobservice.AnnotationKeyUserID: raw}), ShouldEqual, 0)
		}
		So(annotationUserID(map[string]string{vcjobservice.AnnotationKeyUserID: "42"}), ShouldEqual, 42)
	})

	PatchConvey("annotationTolerance", t, func() {
		So(annotationTolerance(nil), ShouldEqual, time.Duration(0))
		for _, raw := range []string{"", "abc", "0", "-5"} {
			So(annotationTolerance(map[string]string{
				vcjobservice.AnnotationKeyWaitingToleranceSeconds: raw,
			}), ShouldEqual, time.Duration(0))
		}
		So(annotationTolerance(map[string]string{
			vcjobservice.AnnotationKeyWaitingToleranceSeconds: "300",
		}), ShouldEqual, 5*time.Minute)
	})
}

func TestPodGroupOwnership(t *testing.T) {
	PatchConvey("controllerOwnerJobName", t, func() {
		job := newVCJob(jobA, testNamespace, rl("cpu", "1"))
		owner := vcjobOwnerRef(job)
		So(controllerOwnerJobName(nil), ShouldBeEmpty)
		So(controllerOwnerJobName([]metav1.OwnerReference{owner}), ShouldEqual, jobA)

		for _, mutate := range []func(*metav1.OwnerReference){
			func(ref *metav1.OwnerReference) { ref.Controller = nil },
			func(ref *metav1.OwnerReference) { ref.Controller = ptr.To(false) },
			func(ref *metav1.OwnerReference) { ref.Kind = "Deployment" },
			func(ref *metav1.OwnerReference) { ref.APIVersion = "apps/v1" },
		} {
			ref := owner
			mutate(&ref)
			So(controllerOwnerJobName([]metav1.OwnerReference{ref}), ShouldBeEmpty)
		}
	})

	PatchConvey("podGroupsByOwnerJob", t, func() {
		job := newVCJob(jobA, testNamespace, rl("cpu", "1"))
		first := newPodGroup(job, scheduling.PodGroupPending)
		second := newPodGroup(job, scheduling.PodGroupInqueue)
		second.Name = jobA + "-second"
		orphan := &scheduling.PodGroup{ObjectMeta: metav1.ObjectMeta{Name: "podgroup-plain-pod"}}

		groups := podGroupsByOwnerJob(&scheduling.PodGroupList{Items: []scheduling.PodGroup{*first, *second, *orphan}})
		So(len(groups), ShouldEqual, 1)
		So(groups[jobA].Name, ShouldEqual, second.Name)
	})
}

func TestBuildSnapshot(t *testing.T) {
	PatchConvey("joins jobs, pod groups and queues", t, func() {
		cfg := &config.Config{}
		cfg.Namespaces.Job = testNamespace
		Mock(config.GetConfig).Return(cfg).Build()
		Mock(utils.GetLocalTime).Return(fixedNow).Build()

		admittedJob := newVCJob(jobAdmitted, testNamespace, rl("cpu", "2"))
		// Created 100s ago with a 300s tolerance: only the frozen clock keeps it from timing out.
		pendingJob := newVCJob(jobPending, testNamespace, rl("cpu", "1"), func(job *batch.Job) {
			job.CreationTimestamp = metav1.NewTime(fixedNow.Add(-100 * time.Second))
		})
		foreignJob := newVCJob("job-foreign", "other-namespace", rl("cpu", "1"))
		reader := fake.NewClientBuilder().WithScheme(volcanoScheme()).WithObjects(
			admittedJob, pendingJob, foreignJob,
			newPodGroup(admittedJob, scheduling.PodGroupInqueue),
			newPodGroup(pendingJob, scheduling.PodGroupPending),
			&scheduling.Queue{ObjectMeta: metav1.ObjectMeta{Name: publicQueue}},
		).Build()

		snap, err := (&Server{reader: reader}).buildSnapshot(t.Context(), &settings{})
		So(err, ShouldBeNil)
		So(len(snap.views), ShouldEqual, 2)
		So(snap.byName, ShouldNotContainKey, "job-foreign")
		So(snap.now, ShouldEqual, fixedNow)
		So(snap.byName[jobAdmitted].podGroupPhase, ShouldEqual, scheduling.PodGroupInqueue)
		So(snap.byName[jobPending].podGroupPhase, ShouldEqual, scheduling.PodGroupPending)
		So(snap.isTimedOut(snap.byName[jobPending]), ShouldBeFalse)
		So(snap.queues, ShouldContainKey, publicQueue)
		So(snap.quotas, ShouldBeNil)
		usage := snap.usageByOwner[ownerKey{userID: testUserID, queue: publicQueue}]
		So(usage.Cpu().String(), ShouldEqual, "2")
	})
}
