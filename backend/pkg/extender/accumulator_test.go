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
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/pkg/utils"
)

func TestSessionAccumulator(t *testing.T) {
	PatchConvey("reserve overwrites the standing entry", t, func() {
		clock := fixedNow
		Mock(utils.GetLocalTime).To(func() time.Time { return clock }).Build()
		acc := newSessionAccumulator()
		view := newView(jobA, rl("cpu", "1"))

		acc.reserve(view)
		clock = fixedNow.Add(10 * time.Second)
		acc.reserve(view)
		So(acc.entries, ShouldHaveLength, 1)
		So(acc.entries[jobA].at, ShouldEqual, clock)
		So(acc.entries[jobA].userID, ShouldEqual, testUserID)
		So(acc.entries[jobA].queue, ShouldEqual, publicQueue)
	})

	PatchConvey("sweep releases admitted, terminal, vanished and expired entries", t, func() {
		Mock(utils.GetLocalTime).Return(fixedNow).Build()
		kept := newView("job-kept", rl("cpu", "1"))
		admittedView := newView("job-admitted", rl("cpu", "1"), admitted)
		terminalView := newView("job-terminal", rl("cpu", "1"))
		terminalView.jobPhase = batch.Completed
		expired := newView("job-expired", rl("cpu", "1"))
		vanished := newView("job-vanished", rl("cpu", "1"))

		acc := newSessionAccumulator()
		for _, view := range []*jobView{kept, admittedView, terminalView, expired, vanished} {
			acc.reserve(view)
		}
		acc.entries[expired.name].at = fixedNow.Add(-reservationTTL - time.Second)

		acc.sweep(newSnapshot(kept, admittedView, terminalView, expired))
		So(acc.entries, ShouldHaveLength, 1)
		So(acc.entries, ShouldContainKey, kept.name)
	})

	PatchConvey("reservedExcluding filters by job, user and queue", t, func() {
		acc := newSessionAccumulator()
		So(acc.reservedExcluding(testUserID, publicQueue, jobA), ShouldNotBeNil)
		So(acc.reservedExcluding(testUserID, publicQueue, jobA), ShouldBeEmpty)

		acc.reserve(newView(jobA, rl("cpu", "1")))
		acc.reserve(newView(jobB, rl("cpu", "2")))
		acc.reserve(newView("job-other-user", rl("cpu", "4"), ownedBy(8)))
		acc.reserve(newView("job-other-queue", rl("cpu", "8"), inQueue("q-a2-u9")))

		So(cpuOf(acc.reservedExcluding(testUserID, publicQueue, jobA)), ShouldEqual, "2")
		So(cpuOf(acc.reservedExcluding(testUserID, publicQueue, "job-none")), ShouldEqual, "3")
		So(cpuOf(acc.reservedExcluding(8, publicQueue, "job-none")), ShouldEqual, "4")
	})
}
