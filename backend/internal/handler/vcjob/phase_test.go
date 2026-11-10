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

package vcjob

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/query"
)

type phaseFixture struct {
	Status        string  `json:"status"`
	PodGroupPhase *string `json:"podGroupPhase"`
	Phase         string  `json:"phase"`
}

func phaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if query.Job != nil {
		previous := query.Job.WithContext(t.Context()).UnderlyingDB()
		t.Cleanup(func() { query.SetDefault(previous) })
	}
	for _, statement := range []string{
		"CREATE TABLE jobs (id integer primary key, status text, pod_group_phase text, user_id integer, account_id integer, job_type text, name text, job_name text, queue text, creation_timestamp datetime, deleted_at datetime)",
		"CREATE TABLE users (id integer primary key, name text, nickname text, deleted_at datetime)",
		"CREATE TABLE accounts (id integer primary key, name text, nickname text, deleted_at datetime)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	query.SetDefault(db)
	return db
}

func phaseFixtures(t *testing.T) []phaseFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/filter_phases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []phaseFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	return fixtures
}

func TestJobPhaseSQLMatchesSharedFixtures(t *testing.T) {
	db := phaseTestDB(t)
	fixtures := phaseFixtures(t)
	want := map[string][]uint{}
	var id uint
	for _, fixture := range fixtures {
		id++
		if err := db.Exec("INSERT INTO jobs (id,status,pod_group_phase,user_id,account_id,job_type,name) VALUES (?,?,?,?,?,?,?)", id, fixture.Status, fixture.PodGroupPhase, 1, 1, "custom", "target").Error; err != nil {
			t.Fatal(err)
		}
		want[fixture.Phase] = append(want[fixture.Phase], id)
	}
	// Same stages outside the user's scope must stay out of both results and counts.
	if err := db.Exec("INSERT INTO jobs (id,status,pod_group_phase,user_id,account_id,job_type,name,deleted_at) VALUES (100,'Pending','Inqueue',2,1,'custom','target',NULL),(101,'Pending','Inqueue',1,2,'custom','target',NULL),(102,'Pending','Inqueue',1,1,'custom','target','2026-01-01')").Error; err != nil {
		t.Fatal(err)
	}
	days := -1
	user, account := uint(1), uint(1)
	scope := jobListScope{UserID: &user, AccountID: &account}
	request := jobListQuery{Page: 1, PageSize: 1, Days: &days, Search: "target", sorts: []jobSort{{field: "id"}}}
	facets, err := scanJobPhaseFacet(t.Context(), scope, &request, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(facets) != len(want) {
		t.Fatalf("facets=%+v want stages=%v", facets, want)
	}
	for _, facet := range facets {
		if facet.Count != int64(len(want[facet.Value])) {
			t.Fatalf("facet=%+v want=%v", facet, want[facet.Value])
		}
	}
	for phase, ids := range want {
		t.Run(phase, func(t *testing.T) {
			filtered := request
			filtered.Phases = []string{phase}
			rows, total, err := findJobs(t.Context(), scope, &filtered, 30)
			if err != nil || total != int64(len(ids)) || len(rows) != 1 || rows[0].ID != ids[0] {
				t.Fatalf("rows=%+v total=%d err=%v want=%v", rows, total, err, ids)
			}
		})
	}
}

func TestJobPhaseCompositionAndLegacyQueries(t *testing.T) {
	db := phaseTestDB(t)
	for i, f := range phaseFixtures(t) {
		if err := db.Exec("INSERT INTO jobs (id,status,pod_group_phase,user_id,account_id,job_type,name) VALUES (?,?,?,?,?,?,?)", i+1, f.Status, f.PodGroupPhase, 1, 1, "custom", "target").Error; err != nil {
			t.Fatal(err)
		}
	}
	days := -1
	request := jobListQuery{Days: &days}
	for _, tc := range []struct {
		statuses, phases []string
		want             int64
	}{
		{[]string{"Pending"}, nil, 8},
		{[]string{"Prequeue"}, nil, 1},
		{nil, []string{"Pending"}, 4},
		{[]string{"Pending"}, []string{"Pending"}, 3},
		{nil, []string{"Pending", "Starting"}, 8},
	} {
		request.Statuses, request.Phases = tc.statuses, tc.phases
		got, err := applyJobFilters(t.Context(), jobListScope{}, &request, 30).Count()
		if err != nil || got != tc.want {
			t.Fatalf("status=%v phase=%v count=%d want=%d err=%v", tc.statuses, tc.phases, got, tc.want, err)
		}
	}
	request.Statuses = nil
	request.Phases = nil
	all, err := findJobFacets(t.Context(), jobListScope{}, &request, 30, false)
	if err != nil {
		t.Fatal(err)
	}
	request.Phases = []string{"Starting"}
	selected, err := findJobFacets(t.Context(), jobListScope{}, &request, 30, false)
	if err != nil || !reflect.DeepEqual(all["phase"], selected["phase"]) || !reflect.DeepEqual(all["status"], selected["status"]) {
		t.Fatalf("facets changed under own selection: %+v err=%v", selected, err)
	}
	request.Search = "missing"
	empty, err := scanJobPhaseFacet(t.Context(), jobListScope{}, &request, 30)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty=%v err=%v", empty, err)
	}
}

func TestBindJobFilterPhase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		query   string
		invalid bool
	}{
		{"phase=Pending&phase=Starting", false},
		{"phase=Init", false},
		{"status=Prequeue&phase=Pending", false},
		{"status=Inqueue", true},
		{"phase=Prequeue", true},
		{"phase=" + legacyCancelled, true},
		{"phase=invalid", true},
		{"phase=", true},
		{strings.TrimSuffix(strings.Repeat("phase=Pending&", 21), "&"), true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+tc.query, http.NoBody)
			_, err := bindJobListQuery(ctx, true)
			if (err != nil) != tc.invalid {
				t.Fatalf("err=%v invalid=%v", err, tc.invalid)
			}
		})
	}
}

func TestJobPhaseSortingAcrossPages(t *testing.T) {
	db := phaseTestDB(t)
	for i, fixture := range phaseFixtures(t) {
		if err := db.Exec("INSERT INTO jobs (id,status,pod_group_phase,user_id,account_id,job_type,name,creation_timestamp) VALUES (?,?,?,?,?,?,?,?)",
			i+1, fixture.Status, fixture.PodGroupPhase, 1, 1, "custom", "target", "2026-01-01").Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec("INSERT INTO jobs (id,status,user_id,account_id,job_type,name) VALUES (23,'FuturePhase',1,1,'custom','target'),(24,'Pending',2,1,'custom','target')").Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, sort string
		phases     []string
		want       []uint
	}{
		{"ascending", "phase", nil, []uint{3, 2, 1, 17, 11, 10, 9, 16, 13, 12, 8, 5, 4, 15, 14, 7, 6, 18, 21, 19, 20, 22, 23}},
		{"descending", "-phase", nil, []uint{22, 20, 19, 21, 18, 15, 14, 7, 6, 16, 13, 12, 8, 5, 4, 17, 11, 10, 9, 3, 2, 1, 23}},
		{"multiple phases", "phase", []string{"Pending", "Inqueue"}, []uint{17, 11, 10, 9, 16, 13, 12, 8, 5, 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sorts, err := parseJobSort(tc.sort)
			if err != nil {
				t.Fatal(err)
			}
			days, user, account := -1, uint(1), uint(1)
			request := jobListQuery{PageSize: 3, Days: &days, Phases: tc.phases, sorts: sorts}
			scope := jobListScope{UserID: &user, AccountID: &account}
			var got []uint
			for request.Page = 1; len(got) < len(tc.want); request.Page++ {
				rows, total, err := findJobs(t.Context(), scope, &request, 30)
				if err != nil || total != int64(len(tc.want)) || len(rows) == 0 {
					t.Fatalf("page=%d total=%d err=%v", request.Page, total, err)
				}
				for _, row := range rows {
					got = append(got, row.ID)
				}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("IDs=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestParseJobPhaseSorting(t *testing.T) {
	for _, raw := range []string{"phase", "-phase", "name,phase,-createdAt"} {
		if _, err := parseJobSort(raw); err != nil {
			t.Errorf("sort %q: %v", raw, err)
		}
	}
	for _, raw := range []string{"phase,-phase", "phase,name,owner,queue", "phase;DROP TABLE jobs"} {
		if _, err := parseJobSort(raw); err == nil {
			t.Errorf("sort %q must be rejected", raw)
		}
	}
	sorts, err := parseJobSort("phase")
	want := []jobSort{{field: "phase"}, {field: "createdAt", descending: true}, {field: "id", descending: true}}
	if err != nil || !reflect.DeepEqual(sorts, want) {
		t.Fatalf("sorts=%+v err=%v", sorts, err)
	}
}
