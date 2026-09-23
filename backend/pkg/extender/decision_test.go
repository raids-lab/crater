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
	"errors"
	"testing"

	. "github.com/bytedance/mockey"
	. "github.com/smartystreets/goconvey/convey"
	"k8s.io/utils/ptr"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/internal/service"
)

const (
	jobBlocker   = "job-blocker"
	causeQuota   = "queue quota exceeded"
	causeBlocked = "blocked by timed out job"
	fieldCause   = "cause"
	fieldSkip    = "skip"
)

// fieldValue reads one key from the structured log fields a vote carries.
func fieldValue(v vote, key string) any {
	for i := 0; i+1 < len(v.fields); i += 2 {
		if v.fields[i] == key {
			return v.fields[i+1]
		}
	}
	return nil
}

func TestDecideSkips(t *testing.T) {
	PatchConvey("pod groups without a vcjob owner", t, func() {
		stubJobNamespace()
		s := seededServer(newSnapshot())
		result := s.decide(t.Context(), nil)
		So(result.status, ShouldEqual, voteAbstain)
		So(fieldValue(result, fieldSkip), ShouldEqual, "not owned by a vcjob")

		req := vcjobRequest(jobA)
		req.PodGroup.OwnerReferences[0].Controller = ptr.To(false)
		result = s.decide(t.Context(), req)
		So(result.status, ShouldEqual, voteAbstain)
		So(fieldValue(result, "podGroup"), ShouldEqual, req.PodGroup.Name)
	})

	PatchConvey("jobs in another namespace", t, func() {
		stubJobNamespace()
		// A same-named view proves the scope guard runs before the name lookup, so names cannot collide.
		s := seededServer(newSnapshot(newView(jobA, rl("cpu", "1"))))
		req := vcjobRequest(jobA)
		req.Namespace = "other-namespace"

		result := s.decide(t.Context(), req)
		So(result.status, ShouldEqual, voteAbstain)
		So(fieldValue(result, fieldSkip), ShouldEqual, "outside the job namespace")
		So(s.accumulator.entries, ShouldBeEmpty)
	})

	PatchConvey("session state unavailable", t, func() {
		stubJobNamespace()
		s := seededServer(newSnapshot())
		s.session = nil
		s.configService = &service.ConfigService{}
		Mock((*service.ConfigService).GetSchedulerExtenderConfig).Return(nil, errors.New("db down")).Build()

		result := s.decide(t.Context(), vcjobRequest(jobA))
		So(result.status, ShouldEqual, voteAbstain)
		So(fieldValue(result, fieldSkip), ShouldEqual, "session state unavailable")
		So(result.err, ShouldBeError, "db down")
	})

	PatchConvey("plugin switched off", t, func() {
		stubJobNamespace()
		s := seededServer(nil)
		result := s.decide(t.Context(), vcjobRequest(jobA))
		So(result.status, ShouldEqual, voteAbstain)
		So(result.fields, ShouldBeEmpty)
	})

	PatchConvey("candidate not cached yet", t, func() {
		stubJobNamespace()
		s := seededServer(newSnapshot(newView(jobB, rl("cpu", "1"))))
		result := s.decide(t.Context(), vcjobRequest(jobA))
		So(result.status, ShouldEqual, voteReject)
		So(fieldValue(result, fieldCause), ShouldEqual, "vcjob not in cache yet")
		So(s.accumulator.entries, ShouldBeEmpty)
	})
}

func TestDecideQuota(t *testing.T) {
	PatchConvey("admitted usage plus demand above the limit", t, func() {
		stubJobNamespace()
		stubQuota(map[string]string{"cpu": "2"})
		s := seededServer(newSnapshot(newView(jobB, rl("cpu", "2"), admitted), newView(jobA, rl("cpu", "1"))))

		result := s.decide(t.Context(), vcjobRequest(jobA))
		So(result.status, ShouldEqual, voteReject)
		So(fieldValue(result, fieldCause), ShouldEqual, causeQuota)
		So(fieldValue(result, "quota"), ShouldEqual, "cpu used 3 limit 2")
		So(fieldValue(result, "userID"), ShouldEqual, testUserID)
		So(s.accumulator.entries, ShouldNotContainKey, jobA)
	})

	PatchConvey("exactly the limit passes and is reserved", t, func() {
		stubJobNamespace()
		stubQuota(map[string]string{"cpu": "2"})
		s := seededServer(newSnapshot(newView(jobB, rl("cpu", "1"), admitted), newView(jobA, rl("cpu", "1"))))

		result := s.decide(t.Context(), vcjobRequest(jobA))
		So(result.status, ShouldEqual, voteAbstain)
		So(result.fields, ShouldBeEmpty)
		So(s.accumulator.entries, ShouldContainKey, jobA)
	})

	PatchConvey("only admitted jobs of the same user count", t, func() {
		stubJobNamespace()
		stubQuota(map[string]string{"cpu": "2"})
		s := seededServer(newSnapshot(
			newView("job-other-user", rl("cpu", "5"), admitted, ownedBy(8)),
			newView("job-unadmitted", rl("cpu", "5")),
			newView(jobA, rl("cpu", "1")),
		))
		So(s.decide(t.Context(), vcjobRequest(jobA)).status, ShouldEqual, voteAbstain)
	})

	PatchConvey("jobs without a user skip the quota", t, func() {
		stubJobNamespace()
		stubQuota(map[string]string{"cpu": "1"})
		s := seededServer(newSnapshot(newView(jobA, rl("cpu", "4"), ownedBy(0))))
		So(s.decide(t.Context(), vcjobRequest(jobA)).status, ShouldEqual, voteAbstain)
	})

	PatchConvey("reservations accumulate within a session", t, func() {
		stubJobNamespace()
		stubQuota(map[string]string{"cpu": "1"})
		s := seededServer(newSnapshot(newView(jobA, rl("cpu", "1")), newView(jobB, rl("cpu", "1"))))

		So(s.decide(t.Context(), vcjobRequest(jobA)).status, ShouldEqual, voteAbstain)
		second := s.decide(t.Context(), vcjobRequest(jobB))
		So(second.status, ShouldEqual, voteReject)
		So(fieldValue(second, "quota"), ShouldEqual, "cpu used 2 limit 1")
	})

	PatchConvey("a job's own reservation is excluded", t, func() {
		stubJobNamespace()
		stubQuota(map[string]string{"cpu": "1"})
		s := seededServer(newSnapshot(newView(jobA, rl("cpu", "1"))))

		So(s.decide(t.Context(), vcjobRequest(jobA)).status, ShouldEqual, voteAbstain)
		So(s.decide(t.Context(), vcjobRequest(jobA)).status, ShouldEqual, voteAbstain)
	})
}

// blockedBy decides for the candidate in a round that holds only it and one potential blocker.
func blockedBy(t *testing.T, blocker, candidate *jobView) (vote, *Server) {
	t.Helper()
	s := seededServer(newSnapshot(blocker, candidate))
	result := s.decide(t.Context(), vcjobRequest(candidate.name))
	return result, s
}

func TestDecideBlockingScope(t *testing.T) {
	a100Blocker := func() *jobView {
		return newView(jobBlocker, rl("cpu", "1", gpuA100, "1"), timedOut, ownedBy(8))
	}

	PatchConvey("same queue and same card", t, func() {
		stubJobNamespace()
		result, s := blockedBy(t, a100Blocker(), newView(jobA, rl("cpu", "1", gpuA100, "1")))
		So(result.status, ShouldEqual, voteReject)
		So(fieldValue(result, fieldCause), ShouldEqual, causeBlocked)
		So(fieldValue(result, "blocker"), ShouldEqual, jobBlocker)
		So(fieldValue(result, "blockerPodGroupPhase"), ShouldEqual, scheduling.PodGroupPhase(""))
		So(fieldValue(result, "blockerResources"), ShouldEqual, "cpu=1, nvidia.com/a100=1")
		So(s.accumulator.entries, ShouldNotContainKey, jobA)
	})

	PatchConvey("cpu-only candidate still competes", t, func() {
		stubJobNamespace()
		result, _ := blockedBy(t, a100Blocker(), newView(jobA, rl("cpu", "1")))
		So(result.status, ShouldEqual, voteReject)
	})

	PatchConvey("different card is out of scope", t, func() {
		stubJobNamespace()
		result, _ := blockedBy(t, a100Blocker(), newView(jobA, rl("cpu", "1", gpuV100, "1")))
		So(result.status, ShouldEqual, voteAbstain)
	})

	PatchConvey("other queue is out of scope", t, func() {
		stubJobNamespace()
		result, _ := blockedBy(t, a100Blocker(), newView(jobA, rl("cpu", "1", gpuA100, "1"), inQueue("q-a2-u9")))
		So(result.status, ShouldEqual, voteAbstain)
	})

	PatchConvey("disjoint node constraints are out of scope", t, func() {
		stubJobNamespace()
		blocker := a100Blocker()
		onNodes("node-1")(blocker)
		result, _ := blockedBy(t, blocker, newView(jobA, rl("cpu", "1", gpuA100, "1"), onNodes("node-2")))
		So(result.status, ShouldEqual, voteAbstain)

		result, _ = blockedBy(t, blocker, newView(jobA, rl("cpu", "1", gpuA100, "1"), onNodes("node-1", "node-2")))
		So(result.status, ShouldEqual, voteReject)
	})

	PatchConvey("cpu-only blocker never blocks gpu jobs", t, func() {
		cpuBlocker := newView(jobBlocker, rl("cpu", "1"), timedOut, ownedBy(8))
		So(inBlockingScope(cpuBlocker, newView(jobA, rl("cpu", "1", gpuA100, "1"))), ShouldBeFalse)
		So(inBlockingScope(cpuBlocker, newView(jobA, rl("cpu", "1"))), ShouldBeTrue)
	})
}

func TestDecideBlockerEligibility(t *testing.T) {
	PatchConvey("admitted blocker stops timed out peers too", t, func() {
		stubJobNamespace()
		blocker := newView(jobBlocker, rl("cpu", "1"), timedOut, admitted, ownedBy(8))
		result, _ := blockedBy(t, blocker, newView(jobA, rl("cpu", "1"), timedOut))
		So(result.status, ShouldEqual, voteReject)
		So(fieldValue(result, "blockerPodGroupPhase"), ShouldEqual, scheduling.PodGroupInqueue)
	})

	PatchConvey("unadmitted blocker lets timed out peers through", t, func() {
		stubJobNamespace()
		blocker := newView(jobBlocker, rl("cpu", "1"), timedOut, ownedBy(8))
		result, _ := blockedBy(t, blocker, newView(jobA, rl("cpu", "1"), timedOut))
		So(result.status, ShouldEqual, voteAbstain)
	})

	PatchConvey("jobs that are not timed out or not waiting never block", t, func() {
		stubJobNamespace()
		result, _ := blockedBy(t, newView(jobBlocker, rl("cpu", "1"), ownedBy(8)), newView(jobA, rl("cpu", "1")))
		So(result.status, ShouldEqual, voteAbstain)

		result, _ = blockedBy(t, newView(jobBlocker, rl("cpu", "1"), timedOut, notWaiting, ownedBy(8)), newView(jobA, rl("cpu", "1")))
		So(result.status, ShouldEqual, voteAbstain)
	})

	PatchConvey("a job never blocks itself", t, func() {
		stubJobNamespace()
		s := seededServer(newSnapshot(newView(jobA, rl("cpu", "1"), timedOut)))
		So(s.decide(t.Context(), vcjobRequest(jobA)).status, ShouldEqual, voteAbstain)
	})

	PatchConvey("blocker held by its own quota has no power", t, func() {
		stubJobNamespace()
		stubQuota(map[string]string{"cpu": "1"})
		blocker := newView(jobBlocker, rl("cpu", "2"), timedOut, ownedBy(8))
		result, _ := blockedBy(t, blocker, newView(jobA, rl("cpu", "1")))
		So(result.status, ShouldEqual, voteAbstain)
	})

	PatchConvey("blocker larger than its queue has no power", t, func() {
		stubJobNamespace()
		blocker := newView(jobBlocker, rl("cpu", "2"), timedOut, ownedBy(8))
		snap := newSnapshot(blocker, newView(jobA, rl("cpu", "1")))
		addQueue(snap, publicQueue, "", rl("cpu", "1"))
		s := seededServer(snap)
		So(s.decide(t.Context(), vcjobRequest(jobA)).status, ShouldEqual, voteAbstain)
	})

	PatchConvey("ancestor capability is checked as well", t, func() {
		stubJobNamespace()
		blocker := newView(jobBlocker, rl("cpu", "2"), timedOut, ownedBy(8))
		snap := newSnapshot(blocker, newView(jobA, rl("cpu", "1")))
		addQueue(snap, publicQueue, "root", rl("cpu", "4"))
		addQueue(snap, "root", "", rl("cpu", "1"))
		s := seededServer(snap)
		So(s.decide(t.Context(), vcjobRequest(jobA)).status, ShouldEqual, voteAbstain)

		addQueue(snap, "root", "", rl("cpu", "8"))
		So(s.decide(t.Context(), vcjobRequest(jobA)).status, ShouldEqual, voteReject)
	})
}

func TestQueueCapability(t *testing.T) {
	PatchConvey("fitsCapability", t, func() {
		required := rl("cpu", "2")
		So(fitsCapability(required, nil), ShouldBeTrue)
		So(fitsCapability(required, rl("memory", "1Gi")), ShouldBeTrue)
		So(fitsCapability(required, rl("cpu", "2")), ShouldBeTrue)
		So(fitsCapability(required, rl("cpu", "1")), ShouldBeFalse)
	})

	PatchConvey("fitsQueueCapability", t, func() {
		view := newView(jobA, rl("cpu", "2"), withMinimum(rl("cpu", "4")))
		snap := newSnapshot(view)
		So(snap.fitsQueueCapability(view), ShouldBeTrue)

		addQueue(snap, publicQueue, "parent", rl("cpu", "4"))
		addQueue(snap, "parent", "", rl("cpu", "2"))
		So(snap.fitsQueueCapability(view), ShouldBeFalse)

		addQueue(snap, "parent", publicQueue, rl("cpu", "8"))
		So(snap.fitsQueueCapability(view), ShouldBeTrue)
	})
}
