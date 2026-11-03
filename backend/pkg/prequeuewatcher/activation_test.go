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

package prequeuewatcher

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/go-logr/logr"
	. "github.com/smartystreets/goconvey/convey"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/pkg/crclient"
)

const storedJobName = "stored-job"

// detachedWatcher builds gorm-gen statements over a dialector without a connection; every terminal
// call is stubbed, so no SQL ever executes.
func detachedWatcher(t *testing.T) *PrequeueWatcher {
	t.Helper()
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return &PrequeueWatcher{q: query.Use(db), logger: logr.Discard()}
}

func stubTransaction() {
	Mock((*query.Query).Transaction).To(func(q *query.Query, fc func(*query.Query) error, _ ...*sql.TxOptions) error {
		return fc(q)
	}).Build()
}

func stubClaim(rowsAffected int64, err error) {
	Mock((*gen.DO).Updates).Return(gen.ResultInfo{RowsAffected: rowsAffected}, err).Build()
}

func TestClaimAndActivatePrequeueJob(t *testing.T) {
	candidate := &model.Job{JobName: storedJobName, Status: model.Prequeue}

	PatchConvey("claim failure", t, func() {
		w := detachedWatcher(t)
		stubTransaction()
		stubClaim(0, errors.New("db down"))
		restore := Mock((*PrequeueWatcher).restoreJobForActivation).Return(&batch.Job{}, nil).Build()

		activated, err := w.claimAndActivatePrequeueJob(t.Context(), candidate)
		So(activated, ShouldBeFalse)
		So(err, ShouldNotBeNil)
		So(restore.MockTimes(), ShouldEqual, 0)
	})

	PatchConvey("lost the claim race", t, func() {
		w := detachedWatcher(t)
		stubTransaction()
		stubClaim(0, nil)
		restore := Mock((*PrequeueWatcher).restoreJobForActivation).Return(&batch.Job{}, nil).Build()
		activate := Mock(vcjobservice.ActivateJob).Return(nil).Build()

		activated, err := w.claimAndActivatePrequeueJob(t.Context(), candidate)
		So(activated, ShouldBeFalse)
		So(err, ShouldBeNil)
		So(restore.MockTimes(), ShouldEqual, 0)
		So(activate.MockTimes(), ShouldEqual, 0)
	})

	PatchConvey("restore failure rolls back", t, func() {
		w := detachedWatcher(t)
		stubTransaction()
		stubClaim(1, nil)
		Mock((*PrequeueWatcher).restoreJobForActivation).Return(nil, errors.New("template broken")).Build()
		activate := Mock(vcjobservice.ActivateJob).Return(nil).Build()

		activated, err := w.claimAndActivatePrequeueJob(t.Context(), candidate)
		So(activated, ShouldBeFalse)
		So(err, ShouldNotBeNil)
		So(activate.MockTimes(), ShouldEqual, 0)
	})

	PatchConvey("already existing vcjob counts as submitted", t, func() {
		w := detachedWatcher(t)
		stubTransaction()
		stubClaim(1, nil)
		Mock((*PrequeueWatcher).restoreJobForActivation).Return(&batch.Job{}, nil).Build()
		exists := apierrors.NewAlreadyExists(schema.GroupResource{Group: "batch.volcano.sh", Resource: "jobs"}, storedJobName)
		Mock(vcjobservice.ActivateJob).Return(exists).Build()

		activated, err := w.claimAndActivatePrequeueJob(t.Context(), candidate)
		So(activated, ShouldBeTrue)
		So(err, ShouldBeNil)
	})

	PatchConvey("other submit errors roll back", t, func() {
		w := detachedWatcher(t)
		stubTransaction()
		stubClaim(1, nil)
		Mock((*PrequeueWatcher).restoreJobForActivation).Return(&batch.Job{}, nil).Build()
		submitErr := errors.New("apiserver down")
		Mock(vcjobservice.ActivateJob).Return(submitErr).Build()

		activated, err := w.claimAndActivatePrequeueJob(t.Context(), candidate)
		So(activated, ShouldBeFalse)
		So(errors.Is(err, submitErr), ShouldBeTrue)
	})

	PatchConvey("submits under a deadline", t, func() {
		w := detachedWatcher(t)
		stubTransaction()
		stubClaim(1, nil)
		Mock((*PrequeueWatcher).restoreJobForActivation).Return(&batch.Job{}, nil).Build()
		hasDeadline := false
		Mock(vcjobservice.ActivateJob).To(
			func(ctx context.Context, _ client.Client, _ crclient.ServiceManagerInterface, _ *batch.Job) error {
				_, hasDeadline = ctx.Deadline()
				return nil
			}).Build()

		activated, err := w.claimAndActivatePrequeueJob(t.Context(), candidate)
		So(activated, ShouldBeTrue)
		So(err, ShouldBeNil)
		So(hasDeadline, ShouldBeTrue)
	})
}
