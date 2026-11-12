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
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

type phaseFixture struct {
	Status        string  `json:"status"`
	PodGroupPhase *string `json:"podGroupPhase"`
	Phase         string  `json:"phase"`
}

const phaseTestPostgresDialect = "postgres"

func phaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dialector := sqlite.Open(":memory:")
	timestampType := "datetime"
	if os.Getenv("CRATER_PHASE_TEST_DATABASE") == phaseTestPostgresDialect {
		dsn := os.Getenv("CRATER_TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Fatal("CRATER_TEST_POSTGRES_DSN must point to an isolated PostgreSQL 16 test database")
		}
		dialector = postgres.Open(dsn)
		timestampType = "timestamptz"
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
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
	if db.Name() == phaseTestPostgresDialect {
		var version int
		if err := db.Raw("SHOW server_version_num").Scan(&version).Error; err != nil {
			t.Fatal(err)
		}
		if version/10000 != 16 {
			t.Fatalf("PostgreSQL 16 is required, server_version_num=%d", version)
		}
		db = db.Begin()
		if db.Error != nil {
			t.Fatal(db.Error)
		}
		t.Cleanup(func() { _ = db.Rollback().Error })
	}
	// PostgreSQL fixtures shadow application tables on this transaction's pinned
	// connection. Rollback removes them without changing persistent schemas.
	for _, statement := range []string{
		"CREATE TEMP TABLE jobs (id bigint primary key, status text, pod_group_phase text, user_id bigint, account_id bigint, job_type text, name text, job_name text, queue text, nodes jsonb DEFAULT '[]', resources jsonb DEFAULT '{}', creation_timestamp " + timestampType + ", deleted_at " + timestampType + ")",
		"CREATE TEMP TABLE users (id bigint primary key, name text, nickname text, deleted_at " + timestampType + ")",
		"CREATE TEMP TABLE accounts (id bigint primary key, name text, nickname text, deleted_at " + timestampType + ")",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	query.SetDefault(db)
	return db
}

func assertJobPhaseFacet(t *testing.T, items []resputil.FacetItem, want map[string]int64) {
	t.Helper()
	got := make(map[string]int64, len(items))
	for _, item := range items {
		if _, duplicate := got[item.Value]; duplicate {
			t.Fatalf("duplicate facet value %q: %+v", item.Value, items)
		}
		got[item.Value] = item.Count
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("facet=%v want=%v", got, want)
	}
}

func TestJobPhaseFacetsRetainAllOtherFilters(t *testing.T) {
	db := phaseTestDB(t)
	now := time.Now().UTC()
	for _, row := range []struct {
		id, userID, accountID int
		status, group, kind   string
		name, node            string
		createdAt             time.Time
		deletedAt             *time.Time
	}{
		{1, 1, 1, "Pending", "Pending", "custom", "needle", "node-one", now, nil},
		{2, 1, 1, "Pending", "Inqueue", "custom", "needle", "node-one", now, nil},
		{3, 1, 1, "Pending", "Running", "custom", "needle", "node-one", now, nil},
		{4, 1, 1, "Prequeue", "Pending", "custom", "needle", "node-one", now, nil},
		{5, 1, 1, "Running", "Running", "custom", "needle", "node-one", now, nil},
		{6, 1, 1, "Completed", "Running", "custom", "needle", "node-one", now, nil},
		{7, 1, 1, "Failed", "Running", "custom", "needle", "node-one", now, nil},
		{8, 2, 1, "Pending", "Inqueue", "custom", "needle", "node-one", now, nil},
		{9, 1, 2, "Pending", "Inqueue", "custom", "needle", "node-one", now, nil},
		{10, 1, 1, "Pending", "Inqueue", "jupyter", "needle", "node-one", now, nil},
		{11, 1, 1, "Pending", "Inqueue", "custom", "haystack", "node-one", now, nil},
		{12, 1, 1, "Pending", "Inqueue", "custom", "needle", "node-other", now, nil},
		{13, 1, 1, "Pending", "Inqueue", "custom", "needle", "node-one", now.AddDate(0, 0, -30), nil},
		{14, 1, 1, "Pending", "Inqueue", "custom", "needle", "node-one", now, &now},
		{15, 1, 1, "Pending", "Inqueue", "tensorboard", "needle", "node-one", now, nil},
	} {
		nodes, err := json.Marshal([]string{row.node})
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Exec("INSERT INTO jobs (id,user_id,account_id,status,pod_group_phase,job_type,name,nodes,creation_timestamp,deleted_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
			row.id, row.userID, row.accountID, row.status, row.group, row.kind, row.name, string(nodes), row.createdAt, row.deletedAt).Error; err != nil {
			t.Fatal(err)
		}
	}
	days, user, account, node := 7, uint(1), uint(1), "node-one"
	request := jobListQuery{
		Page: 1, PageSize: 1, Days: &days, Search: "needle", Node: &node,
		JobTypes: []string{"custom"}, Statuses: []string{"Pending"}, Phases: []string{"Inqueue"},
		sorts: []jobSort{{field: "id"}},
	}
	scope := jobListScope{UserID: &user, AccountID: &account}
	rows, total, err := findJobs(t.Context(), scope, &request, 30)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].ID != 2 {
		t.Fatalf("rows=%+v total=%d err=%v", rows, total, err)
	}
	facets, err := findJobFacets(t.Context(), scope, &request, 30, false)
	if err != nil {
		t.Fatal(err)
	}
	assertJobPhaseFacet(t, facets["phase"], map[string]int64{"Pending": 1, "Inqueue": 1, "Starting": 1})
	assertJobPhaseFacet(t, facets["status"], map[string]int64{"Pending": 1})
	assertJobPhaseFacet(t, facets["job_type"], map[string]int64{"custom": 1, "jupyter": 1})
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

func TestJobPhaseNonWaitingMatrix(t *testing.T) {
	db := phaseTestDB(t)
	states := []string{"Prequeue", "Aborting", "Aborted", "Running", "Restarting", "Completing", "Completed", "Terminating", "Terminated", "Failed", "Deleted", "Freed", legacyCancelled, "FuturePhase"}
	groups := []string{"", "Pending", "Inqueue", "Running", "Completed", "Unknown", "FuturePhase"}
	for _, status := range states {
		want := status
		if status == "Prequeue" {
			want = "Pending"
		}
		for _, group := range append(groups, "NULL") {
			var podGroup *string
			if group != "NULL" {
				podGroup = &group
			}
			if err := db.Exec("DELETE FROM jobs").Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Exec("INSERT INTO jobs (id,status,pod_group_phase) VALUES (1,?,?)", status, podGroup).Error; err != nil {
				t.Fatal(err)
			}
			var got string
			if err := query.Job.WithContext(t.Context()).Select(jobFilterPhaseExpr()).Scan(&got); err != nil || got != want {
				t.Fatalf("status=%q podGroup=%v phase=%q want=%q err=%v", status, podGroup, got, want, err)
			}
		}
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
	if err != nil || !reflect.DeepEqual(all["phase"], selected["phase"]) {
		t.Fatalf("facets changed under own selection: %+v err=%v", selected, err)
	}
	request.Search = "missing"
	empty, err := scanJobPhaseFacet(t.Context(), jobListScope{}, &request, 30)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty=%v err=%v", empty, err)
	}
}

func TestJobFacetRetainsOtherStatusDimension(t *testing.T) {
	db := phaseTestDB(t)
	if err := db.Exec("INSERT INTO jobs (id,status,pod_group_phase,user_id,account_id,job_type,name) VALUES (1,'Pending','Running',1,1,'custom','target'),(2,'Prequeue',NULL,1,1,'custom','target'),(3,'Running',NULL,1,1,'custom','target'),(4,'Pending','Running',2,1,'custom','target'),(5,'Pending','Running',1,2,'custom','target')").Error; err != nil {
		t.Fatal(err)
	}
	days, user, account := -1, uint(1), uint(1)
	request := jobListQuery{Days: &days, Statuses: []string{"Pending"}, Phases: []string{"Starting"}}
	facets, err := findJobFacets(t.Context(), jobListScope{UserID: &user, AccountID: &account}, &request, 30, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"status", "phase"} {
		if len(facets[key]) != 1 || facets[key][0].Count != 1 {
			t.Fatalf("%s facet ignored the other dimension or scope: %+v", key, facets[key])
		}
	}
	if facets["status"][0].Value != "Pending" || facets["phase"][0].Value != "Starting" {
		t.Fatalf("unexpected facets: %+v", facets)
	}
}

func TestExplicitJobSortTieBreakers(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want []jobSort
	}{
		{"phase,id", []jobSort{{field: "phase"}, {field: "id"}, {field: "createdAt", descending: true}}},
		{"phase,createdAt,id", []jobSort{{field: "phase"}, {field: "createdAt"}, {field: "id"}}},
		{"phase,-createdAt,-id", []jobSort{{field: "phase"}, {field: "createdAt", descending: true}, {field: "id", descending: true}}},
	} {
		sorts, err := parseJobSort(tc.raw)
		if err != nil || !reflect.DeepEqual(sorts, tc.want) {
			t.Errorf("sort=%q got=%+v want=%+v err=%v", tc.raw, sorts, tc.want, err)
		}
	}
}

func TestJobSelfPhaseScope(t *testing.T) {
	db := phaseTestDB(t)
	if err := db.Exec("INSERT INTO jobs (id,status,pod_group_phase,user_id,account_id,job_type,name) VALUES (1,'Pending','Running',1,1,'custom','one'),(2,'Pending','Running',2,1,'custom','two'),(3,'Pending','Running',1,2,'custom','other-account')").Error; err != nil {
		t.Fatal(err)
	}
	for _, userID := range []uint{1, 2} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		util.SetJWTContext(c, util.JWTMessage{UserID: userID, AccountID: 1, RolePlatform: model.RoleUser, RoleAccount: model.RoleUser})
		scope := selfJobScope(c)
		days := -1
		sorts, err := parseJobSort("phase,id")
		if err != nil {
			t.Fatal(err)
		}
		request := jobListQuery{Page: 1, PageSize: 1, Days: &days, Phases: []string{"Starting"}, sorts: sorts}
		rows, total, err := findJobs(t.Context(), scope, &request, 30)
		if err != nil || total != 1 || len(rows) != 1 || rows[0].UserID != userID || rows[0].AccountID != 1 {
			t.Fatalf("user=%d rows=%+v total=%d err=%v", userID, rows, total, err)
		}
		facets, err := findJobFacets(t.Context(), scope, &request, 30, false)
		if err != nil || len(facets["phase"]) != 1 || facets["phase"][0].Count != 1 {
			t.Fatalf("user=%d facets=%+v err=%v", userID, facets, err)
		}
	}
}

func TestBindJobFilterPhase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		query   string
		invalid bool
	}{
		{"phase=Pending&phase=Starting", false},
		{"phase=Pending,Inqueue", false},
		{"phase=Pending&phase=Inqueue,Starting", false},
		{"phase=Init", false},
		{"status=Prequeue&phase=Pending", false},
		{"status=" + legacyCancelled, false},
		{"status=Inqueue", true},
		{"phase=Prequeue", true},
		{"phase=" + legacyCancelled, true},
		{"phase=invalid", true},
		{"phase=Pending%27%3BSELECT+1", true},
		{"sort=phase,-phase", true},
		{"sort=id,-id", true},
		{"sort=phase,id", false},
		{"sort=phase%3BDROP+TABLE+jobs", true},
		{"sort=" + strings.Repeat("x", 4096), true},
		{"search=" + strings.Repeat("x", 129), true},
		{"phase=", true},
		{"phase=Pending,", true},
		{"phase=,Pending", true},
		{"phase=Pending,,Inqueue", true},
		{"phase=" + strings.TrimSuffix(strings.Repeat("Pending,", 21), ","), true},
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

func TestJobPhaseQueryEncodings(t *testing.T) {
	db := phaseTestDB(t)
	if err := db.Exec(`INSERT INTO jobs (id,status,pod_group_phase,user_id,account_id,job_type,name) VALUES
	 (1,'Pending','Pending',1,1,'custom','pending'),
	 (2,'Pending','Inqueue',1,1,'custom','queued'),
	 (3,'Pending','Running',1,1,'custom','starting'),
	 (4,'Prequeue',NULL,1,1,'custom','legacy')`).Error; err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		query string
		want  []uint
	}{
		{"phase=Pending,Inqueue", []uint{1, 2, 4}},
		{"phase=Pending&phase=Inqueue", []uint{1, 2, 4}},
		{"status=Pending&phase=Pending,Inqueue", []uint{1, 2}},
		{"status=Pending&phase=Pending&phase=Inqueue", []uint{1, 2}},
		{"status=Pending&phase=Pending&phase=Inqueue,Starting", []uint{1, 2, 3}},
	} {
		t.Run(testCase.query, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/?days=-1&sort=id&"+testCase.query, http.NoBody)
			request, err := bindJobListQuery(ctx, true)
			if err != nil {
				t.Fatal(err)
			}
			rows, total, err := findJobs(t.Context(), jobListScope{}, &request, 30)
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]uint, len(rows))
			for index, row := range rows {
				ids[index] = row.ID
			}
			if total != int64(len(testCase.want)) || !reflect.DeepEqual(ids, testCase.want) {
				t.Fatalf("total=%d ids=%v want=%v", total, ids, testCase.want)
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
		{"explicit ID ascending", "phase,id", nil, []uint{1, 2, 3, 9, 10, 11, 17, 4, 5, 8, 12, 13, 16, 6, 7, 14, 15, 18, 21, 19, 20, 22, 23}},
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
