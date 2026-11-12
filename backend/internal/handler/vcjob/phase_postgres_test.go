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

import "testing"

func TestJobPhasePostgresOverviewFacets(t *testing.T) {
	db := phaseTestDB(t)
	if db.Name() != phaseTestPostgresDialect {
		t.Fatal("run this test with make test-job-phases-postgres")
	}
	for _, statement := range []string{
		"INSERT INTO users (id,name,nickname) VALUES (1,'one','User One'),(2,'two','User Two')",
		"INSERT INTO accounts (id,name,nickname) VALUES (1,'first','First'),(2,'second','Second')",
		`INSERT INTO jobs (id,status,pod_group_phase,user_id,account_id,job_type,name,resources) VALUES
		 (1,'Prequeue',NULL,1,1,'custom','legacy','{}'),
		 (2,'Pending','Inqueue',1,1,'custom','queued','{}'),
		 (3,'Pending','Running',1,1,'custom','starting','{}'),
		 (4,'Running','Running',1,1,'custom','running','{"nvidia.com/a100":"2"}'),
		 (5,'Running','Running',2,1,'custom','other-user','{"nvidia.com/a100":"8"}'),
		 (6,'Running','Running',1,2,'custom','other-account','{"nvidia.com/a100":"4"}'),
		 (7,'Completed',NULL,1,1,'custom','completed','{}'),
		 (8,'FuturePhase',NULL,1,1,'custom','unknown','{}')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	days, user, account := -1, uint(1), uint(1)
	request := jobListQuery{Days: &days}
	scope := jobListScope{UserID: &user, AccountID: &account}
	facets, err := findJobFacets(t.Context(), scope, &request, 30, true)
	if err != nil {
		t.Fatal(err)
	}
	assertJobPhaseFacet(t, facets["phase"], map[string]int64{
		"Pending": 1, "Inqueue": 1, "Starting": 1, "Running": 1, "Completed": 1, "FuturePhase": 1,
	})
	assertJobPhaseFacet(t, facets["status"], map[string]int64{
		"Prequeue": 1, "Pending": 2, "Running": 1, "Completed": 1, "FuturePhase": 1,
	})
	assertJobPhaseFacet(t, facets["job_type"], map[string]int64{"custom": 6})
	assertJobPhaseFacet(t, facets["owner"], map[string]int64{"User One": 1})
	assertJobPhaseFacet(t, facets["gpu_resource"], map[string]int64{"a100": 2})

	request.Search = "no-matching-job"
	facets, err = findJobFacets(t.Context(), scope, &request, 30, true)
	if err != nil {
		t.Fatal(err)
	}
	for name, items := range facets {
		if items == nil || len(items) != 0 {
			t.Fatalf("empty %s facet=%+v", name, items)
		}
	}
}
