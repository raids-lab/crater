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

package vcjob

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

type regularJobTestOwner struct {
	gorm.Model
	Name     string
	Nickname string
	Space    string
}

func setupRegularJobListDB(t *testing.T) *gorm.DB {
	t.Helper()
	// These handlers use package-level query handles, so do not run this test in parallel.
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
	if err := db.AutoMigrate(&model.Job{}); err != nil {
		t.Fatal(err)
	}
	// Only owner display fields are used; avoid shared model index names in SQLite.
	for _, table := range []string{"users", "accounts"} {
		if err := db.Table(table).AutoMigrate(&regularJobTestOwner{}); err != nil {
			t.Fatal(err)
		}
	}
	localQuery := query.Use(db)
	previousJob, previousUser, previousAccount := query.Job, query.User, query.Account
	query.Job, query.User, query.Account = &localQuery.Job, &localQuery.User, &localQuery.Account
	t.Cleanup(func() {
		query.Job, query.User, query.Account = previousJob, previousUser, previousAccount
	})
	return db
}

func seedRegularJobList(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, name := range []string{"alice", "bob", "panel-only"} {
		if err := db.Table("users").Create(&regularJobTestOwner{
			Name: name, Nickname: name, Space: name}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"queue-a", "queue-b"} {
		if err := db.Table("accounts").Create(&regularJobTestOwner{
			Name: name, Nickname: name, Space: name}).Error; err != nil {
			t.Fatal(err)
		}
	}
	jobs := []model.Job{
		{JobName: "training-a", UserID: 1, AccountID: 1, JobType: model.JobTypeCustom, Status: batch.Running},
		{JobName: "notebook-a", UserID: 1, AccountID: 1, JobType: model.JobTypeJupyter, Status: batch.Pending},
		{JobName: "editor-b", UserID: 2, AccountID: 2, JobType: model.JobTypeWebIDE, Status: batch.Running},
		{JobName: "tb-running", UserID: 1, AccountID: 1, JobType: model.JobTypeTensorboard, Status: batch.Running},
		{JobName: "tb-completed", UserID: 1, AccountID: 1, JobType: model.JobTypeTensorboard, Status: batch.Completed},
		{JobName: "tb-only-owner", UserID: 3, AccountID: 2, JobType: model.JobTypeTensorboard, Status: batch.Running},
	}
	for index := range jobs {
		jobs[index].Name = jobs[index].JobName
		jobs[index].CreationTimestamp = time.Now()
		if err := db.Create(&jobs[index]).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestRegularJobListsAndFacetsExcludeTensorboard(t *testing.T) {
	db := setupRegularJobListDB(t)
	seedRegularJobList(t, db)
	userID, accountID := uint(1), uint(2)
	tests := []struct {
		name      string
		scope     jobListScope
		request   jobListQuery
		wantRows  int
		wantTotal int64
	}{
		{"overview", jobListScope{}, jobListQuery{Page: 1, PageSize: 100}, 3, 3},
		{"self", jobListScope{UserID: &userID}, jobListQuery{Page: 1, PageSize: 100}, 2, 2},
		{"account", jobListScope{AccountID: &accountID}, jobListQuery{Page: 1, PageSize: 100}, 1, 1},
		{"search", jobListScope{}, jobListQuery{Page: 1, PageSize: 100, Search: "tb-"}, 0, 0},
		{"job type", jobListScope{}, jobListQuery{Page: 1, PageSize: 100, JobTypes: []string{"custom"}}, 1, 1},
		{"pagination", jobListScope{}, jobListQuery{Page: 2, PageSize: 2}, 1, 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rows, total, err := findJobs(t.Context(), test.scope, &test.request, allJobsDefaultDays)
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != test.wantRows || total != test.wantTotal {
				t.Fatalf("rows/total = %d/%d, want %d/%d", len(rows), total, test.wantRows, test.wantTotal)
			}
			for _, row := range rows {
				if row.JobType == model.JobTypeTensorboard {
					t.Fatal("auxiliary panel leaked into a regular job list")
				}
			}
		})
	}
	t.Run("facets", testRegularJobFacetsExcludeTensorboard)
}

func testRegularJobFacetsExcludeTensorboard(t *testing.T) {
	t.Helper()
	request := jobListQuery{}
	facets, err := findJobFacets(t.Context(), jobListScope{}, &request, allJobsDefaultDays, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"job_type", "status"} {
		var total int64
		for _, item := range facets[name] {
			if item.Value == string(model.JobTypeTensorboard) || item.Value == string(batch.Completed) {
				t.Fatalf("auxiliary-only facet leaked into %s: %s", name, item.Value)
			}
			total += item.Count
		}
		if total != 3 {
			t.Fatalf("%s facet total = %d, want 3", name, total)
		}
	}
	request.Statuses = []string{string(batch.Running)}
	owners, err := scanJobStringFacet(t.Context(), jobListScope{}, &request,
		allJobsDefaultDays, &query.User.Nickname)
	if err != nil {
		t.Fatal(err)
	}
	if len(owners) != 2 {
		t.Fatalf("running owner facets = %d, want 2", len(owners))
	}
	for _, owner := range owners {
		if owner.Value == "panel-only" || owner.Count != 1 {
			t.Fatalf("incorrect running owner facet: %#v", owner)
		}
	}
}
