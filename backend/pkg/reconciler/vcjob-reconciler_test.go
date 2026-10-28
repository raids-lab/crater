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

package reconciler

import (
	"context"
	"errors"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/go-logr/logr"
	. "github.com/smartystreets/goconvey/convey"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
)

const (
	vcjobTestName      = "job-a"
	vcjobTestNamespace = "crater-workspace"
	vcjobTestUID       = "uid-1"
)

type failingGetClient struct {
	client.Client
	err error
}

func (c failingGetClient) Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
	return c.err
}

func podGroupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(scheduling.AddToScheme(scheme))
	return scheme
}

func vcjobWithPhase(phase batch.JobPhase) *batch.Job {
	return &batch.Job{
		ObjectMeta: metav1.ObjectMeta{Name: vcjobTestName, Namespace: vcjobTestNamespace, UID: vcjobTestUID},
		Status:     batch.JobStatus{State: batch.JobState{Phase: phase}},
	}
}

func podGroupNamed(name string, phase scheduling.PodGroupPhase) *scheduling.PodGroup {
	return &scheduling.PodGroup{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vcjobTestNamespace},
		Status:     scheduling.PodGroupStatus{Phase: phase},
	}
}

func reconcilerWith(objects ...client.Object) *VcJobReconciler {
	return &VcJobReconciler{
		Client: fake.NewClientBuilder().WithScheme(podGroupScheme()).WithObjects(objects...).Build(),
		log:    logr.Discard(),
	}
}

func TestPodGroupPhaseChanged(t *testing.T) {
	PatchConvey("only phase transitions pass the update filter", t, func() {
		pred := podGroupPhaseChanged()
		pending := podGroupNamed("pg", scheduling.PodGroupPending)
		inqueue := podGroupNamed("pg", scheduling.PodGroupInqueue)

		So(pred.Update(event.UpdateEvent{ObjectOld: pending, ObjectNew: inqueue}), ShouldBeTrue)
		So(pred.Update(event.UpdateEvent{ObjectOld: inqueue, ObjectNew: inqueue}), ShouldBeFalse)
		So(pred.Update(event.UpdateEvent{ObjectOld: &v1.Pod{}, ObjectNew: inqueue}), ShouldBeTrue)
		So(pred.Create(event.CreateEvent{Object: pending}), ShouldBeTrue)
		So(pred.Delete(event.DeleteEvent{Object: pending}), ShouldBeTrue)
	})
}

func TestIsActiveJobStatus(t *testing.T) {
	PatchConvey("counts pending and running", t, func() {
		for _, phase := range []batch.JobPhase{batch.Pending, batch.Running} {
			So(isActiveJobStatus(phase), ShouldBeTrue)
		}
		for _, phase := range []batch.JobPhase{"", batch.Completed, batch.Failed, batch.Aborted, model.Prequeue} {
			So(isActiveJobStatus(phase), ShouldBeFalse)
		}
	})
}

func TestGenerateUpdateJobModel(t *testing.T) {
	current := vcjobTestName + "-" + vcjobTestUID

	PatchConvey("raw vcjob phase and raw pod group phase are stored side by side", t, func() {
		admitted := reconcilerWith(podGroupNamed(current, scheduling.PodGroupInqueue))
		record, err := admitted.generateUpdateJobModel(t.Context(), vcjobWithPhase(batch.Pending), &model.Job{})
		So(err, ShouldBeNil)
		So(record.Status, ShouldEqual, batch.Pending)
		So(record.PodGroupPhase, ShouldEqual, scheduling.PodGroupInqueue)

		running := reconcilerWith(podGroupNamed(current, scheduling.PodGroupRunning))
		record, err = running.generateUpdateJobModel(t.Context(), vcjobWithPhase(batch.Running), &model.Job{})
		So(err, ShouldBeNil)
		So(record.Status, ShouldEqual, batch.Running)
		So(record.PodGroupPhase, ShouldEqual, scheduling.PodGroupRunning)

		record, err = reconcilerWith().generateUpdateJobModel(t.Context(), vcjobWithPhase(""), &model.Job{})
		So(err, ShouldBeNil)
		So(record.Status, ShouldBeEmpty)
		So(record.PodGroupPhase, ShouldBeEmpty)

		legacy := reconcilerWith(podGroupNamed(vcjobTestName, scheduling.PodGroupRunning))
		record, err = legacy.generateUpdateJobModel(t.Context(), vcjobWithPhase(batch.Pending), &model.Job{})
		So(err, ShouldBeNil)
		So(record.PodGroupPhase, ShouldEqual, scheduling.PodGroupRunning)
	})

	PatchConvey("lookup failures propagate instead of demoting the job", t, func() {
		lookupErr := errors.New("cache not synced")
		r := &VcJobReconciler{
			Client: failingGetClient{Client: reconcilerWith().Client, err: lookupErr},
			log:    logr.Discard(),
		}
		_, err := r.generateUpdateJobModel(t.Context(), vcjobWithPhase(""), &model.Job{})
		So(errors.Is(err, lookupErr), ShouldBeTrue)

		group, err := reconcilerWith().getPodGroup(t.Context(), vcjobWithPhase(""))
		So(group, ShouldBeNil)
		So(err, ShouldBeNil)
	})
}
