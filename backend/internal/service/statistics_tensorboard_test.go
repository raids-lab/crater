/*
Copyright 2026 The Crater Project Team, RAIDS-Lab

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

package service

import (
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
)

func setupTensorboardStatisticsDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Statistics uses package-level query handles, so do not run this test in parallel.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.Job{}, &model.Resource{}); err != nil {
		t.Fatal(err)
	}
	localQuery := query.Use(db)
	previousJob, previousResource := query.Job, query.Resource
	query.Job, query.Resource = &localQuery.Job, &localQuery.Resource
	t.Cleanup(func() { query.Job, query.Resource = previousJob, previousResource })
	return db
}

func seedTensorboardStatistics(t *testing.T, db *gorm.DB, start, end time.Time) {
	t.Helper()
	jobs := []model.Job{
		{JobName: "training-a", UserID: 1, AccountID: 1, JobType: model.JobTypeCustom},
		{JobName: "training-b", UserID: 2, AccountID: 2, JobType: model.JobTypeCustom},
		{JobName: "tb-active", UserID: 1, AccountID: 1, JobType: model.JobTypeTensorboard},
		{JobName: "tb-deleted", UserID: 2, AccountID: 2, JobType: model.JobTypeTensorboard},
	}
	for index := range jobs {
		job := &jobs[index]
		job.Name = job.JobName
		job.CreationTimestamp, job.RunningTimestamp, job.CompletedTimestamp = start, start, end
		job.Status = batch.Completed
		cpu := "100m"
		switch index {
		case 0:
			cpu = "1"
		case 1:
			cpu = "2"
		}
		job.Resources = datatypes.NewJSONType(corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse(cpu),
		})
		if index == 0 || index == 3 {
			job.DeletedAt = gorm.DeletedAt{Time: end, Valid: true}
		}
		if index == 2 {
			job.CompletedTimestamp = time.Time{}
			job.Status = batch.Running
		}
		if err := db.Create(job).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestResourceStatisticsExcludeTensorboardIncludingDeletedPanels(t *testing.T) {
	db := setupTensorboardStatisticsDB(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	seedTensorboardStatistics(t, db, start, end)
	tests := []struct {
		name      string
		scope     payload.StatisticsScope
		targetID  uint
		wantUsage float64
	}{
		{"cluster", payload.ScopeCluster, 0, 6},
		{"user", payload.ScopeUser, 1, 2},
		{"account", payload.ScopeAccount, 2, 4},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := Statistics.GetResourceStatistics(t.Context(), &payload.StatisticsReq{
				StartTime: start, EndTime: end, Step: payload.StepDay, Scope: test.scope, TargetID: test.targetID,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := response.TotalUsage[ResNameCPU].Usage; got != test.wantUsage {
				t.Fatalf("CPU core-hours = %v, want %v", got, test.wantUsage)
			}
			if len(response.Series) != 1 || response.Series[0].Usage[ResNameCPU] != test.wantUsage {
				t.Fatalf("incorrect time series: %#v", response.Series)
			}
		})
	}
}
