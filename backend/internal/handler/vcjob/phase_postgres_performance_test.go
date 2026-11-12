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
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/raids-lab/crater/dao/query"
)

type phasePlanLogger struct {
	logger.Interface
	statements []string
}

func (log *phasePlanLogger) Trace(_ context.Context, _ time.Time, statement func() (string, int64), _ error) {
	sql, _ := statement()
	log.statements = append(log.statements, sql)
}

type phasePerformanceCase struct {
	name, sort string
	scope      jobListScope
	phases     []string
	facet      bool
}

func TestJobPhasePostgresPerformance(t *testing.T) {
	for _, size := range []int{10000, 100000, 1000000} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			db := phasePerformanceDB(t, size)
			user, account := uint(1), uint(1)
			scope := jobListScope{UserID: &user, AccountID: &account}
			for _, testCase := range []phasePerformanceCase{
				{name: "default", sort: "-createdAt"},
				{name: "raw-status", sort: "status"},
				{name: "phase-asc", sort: "phase"},
				{name: "phase-desc", sort: "-phase"},
				{name: "phase-scoped", sort: "phase", scope: scope, phases: []string{"Pending", "Starting"}},
				{name: "facet-all", facet: true},
				{name: "facet-scoped", scope: scope, facet: true},
			} {
				t.Run(testCase.name, func(t *testing.T) {
					plan := phasePerformancePlan(t, db, &testCase)
					t.Logf("PHASE_PLAN rows=%d case=%s plan=%s", size, testCase.name, plan)
				})
			}
		})
	}
}

func phasePerformanceDB(t *testing.T, size int) *gorm.DB {
	t.Helper()
	db := phaseTestDB(t)
	if db.Name() != phaseTestPostgresDialect {
		t.Fatal("run this test with make test-job-phases-postgres-performance")
	}
	for _, statement := range []string{
		"SET LOCAL jit = off",
		"CREATE INDEX phase_test_creation ON jobs (creation_timestamp DESC, id DESC)",
		"CREATE INDEX phase_test_scope ON jobs (user_id, account_id, creation_timestamp DESC)",
		"CREATE INDEX phase_test_status ON jobs (status, creation_timestamp DESC)",
		"INSERT INTO users (id,name,nickname) SELECT n,'user-'||n,'User '||n FROM generate_series(1,20) n",
		"INSERT INTO accounts (id,name,nickname) SELECT n,'account-'||n,'Account '||n FROM generate_series(1,4) n",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec(`INSERT INTO jobs
     (id,status,pod_group_phase,user_id,account_id,job_type,name,creation_timestamp,nodes)
     SELECT n,
     (ARRAY['','Prequeue','Pending','Pending','Pending','Running','Completed','Failed','Deleted','Freed','Cancelled','FuturePhase'])[1+n%12],
     (ARRAY[NULL,'Pending','Inqueue','Running','Completed','Unknown'])[1+n%6],
     1+n%20,1+n%4,'custom','target-'||n,
     now()-(n%365)*interval '1 day','["node-one"]'::jsonb
     FROM generate_series(1,?) n`, size).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("ANALYZE jobs").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func phasePerformancePlan(t *testing.T, db *gorm.DB, testCase *phasePerformanceCase) string {
	t.Helper()
	log := &phasePlanLogger{Interface: logger.Default}
	db = db.Session(&gorm.Session{Logger: log})
	query.SetDefault(db)
	sorts, err := parseJobSort(testCase.sort)
	if err != nil {
		t.Fatal(err)
	}
	days := -1
	request := jobListQuery{Page: 1, PageSize: 100, Days: &days, Phases: testCase.phases, sorts: sorts}
	if testCase.facet {
		items, err := scanJobPhaseFacet(t.Context(), testCase.scope, &request, 30)
		if err != nil || len(items) == 0 {
			t.Fatalf("facet items=%v err=%v", items, err)
		}
	} else {
		rows, total, err := findJobs(t.Context(), testCase.scope, &request, 30)
		if err != nil || len(rows) != 100 || total < 100 {
			t.Fatalf("rows=%d total=%d err=%v", len(rows), total, err)
		}
	}
	var statement string
	for _, sql := range log.statements {
		if strings.Contains(sql, `FROM "jobs"`) && !strings.HasPrefix(strings.ToLower(sql), "select count(") {
			statement = sql
			break
		}
	}
	if statement == "" {
		t.Fatal("production query was not captured")
	}
	var rawPlan []byte
	// The captured statement contains this test's fixed synthetic values.
	if err := db.Raw("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) " + statement).Row().Scan(&rawPlan); err != nil {
		t.Fatal(err)
	}
	var plan bytes.Buffer
	if err := json.Compact(&plan, rawPlan); err != nil {
		t.Fatal(err)
	}
	return plan.String()
}
