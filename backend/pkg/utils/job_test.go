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

package utils

import (
	"testing"

	. "github.com/bytedance/mockey"
	. "github.com/smartystreets/goconvey/convey"
	"k8s.io/apimachinery/pkg/util/sets"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

func TestNodeConstraintsOverlap(t *testing.T) {
	PatchConvey("empty constraints are wildcards", t, func() {
		So(NodeConstraintsOverlap(nil, nil), ShouldBeTrue)
		So(NodeConstraintsOverlap(sets.New[string](), sets.New("node-1")), ShouldBeTrue)
		So(NodeConstraintsOverlap(sets.New("node-1"), nil), ShouldBeTrue)
		So(NodeConstraintsOverlap(sets.New("node-1"), sets.New("node-2")), ShouldBeFalse)
		So(NodeConstraintsOverlap(sets.New("node-1", "node-2"), sets.New("node-2", "node-3")), ShouldBeTrue)
	})
}

func TestPhasePredicates(t *testing.T) {
	PatchConvey("a pod group is admitted once it leaves Pending", t, func() {
		for _, phase := range []scheduling.PodGroupPhase{
			scheduling.PodGroupInqueue, scheduling.PodGroupRunning,
			scheduling.PodGroupUnknown, scheduling.PodGroupCompleted,
		} {
			So(IsPodGroupAdmitted(phase), ShouldBeTrue)
		}
		for _, phase := range []scheduling.PodGroupPhase{"", scheduling.PodGroupPending} {
			So(IsPodGroupAdmitted(phase), ShouldBeFalse)
		}
	})

	PatchConvey("terminal phases are the four a job never leaves", t, func() {
		for _, phase := range []batch.JobPhase{batch.Completed, batch.Failed, batch.Aborted, batch.Terminated} {
			So(IsJobPhaseTerminal(phase), ShouldBeTrue)
		}
		for _, phase := range []batch.JobPhase{"", batch.Pending, batch.Running, batch.Completing, batch.Terminating} {
			So(IsJobPhaseTerminal(phase), ShouldBeFalse)
		}
	})

	PatchConvey("a job is in queue while volcano keeps it Pending after admission", t, func() {
		for _, jobPhase := range []batch.JobPhase{"", batch.Pending} {
			So(IsJobInqueue(jobPhase, scheduling.PodGroupInqueue), ShouldBeTrue)
			So(IsJobInqueue(jobPhase, scheduling.PodGroupRunning), ShouldBeTrue)
			So(IsJobInqueue(jobPhase, scheduling.PodGroupPending), ShouldBeFalse)
			So(IsJobInqueue(jobPhase, ""), ShouldBeFalse)
		}
		for _, jobPhase := range []batch.JobPhase{batch.Running, batch.Completed, batch.Failed} {
			So(IsJobInqueue(jobPhase, scheduling.PodGroupInqueue), ShouldBeFalse)
		}
	})
}
