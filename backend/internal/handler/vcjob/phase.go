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
	"context"

	"gorm.io/gen/field"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
)

const (
	//nolint:misspell // Preserve the historical API spelling.
	legacyCancelled     = "Cancelled"
	filterPhaseInit     = "Init" // Nonempty wire encoding of the existing empty job phase.
	filterPhaseStarting = "Starting"
)

func validRawJobStatus(value string) bool {
	switch value {
	case string(model.Prequeue), string(batch.Pending), string(batch.Aborting), string(batch.Aborted),
		string(batch.Running), string(batch.Restarting), string(batch.Completing), string(batch.Completed),
		string(batch.Terminating), string(batch.Terminated), string(batch.Failed), string(model.Deleted),
		string(model.Freed), legacyCancelled:
		return true
	default:
		return false
	}
}

func validJobFilterPhase(value string) bool {
	if value == filterPhaseInit || value == filterPhaseStarting || value == string(scheduling.PodGroupInqueue) {
		return true
	}
	return value != string(model.Prequeue) && value != legacyCancelled && validRawJobStatus(value)
}

// This is the sole database-stage expression, shared by filtering and facets.
// Fixed identifiers are never derived from input; every stage value is bound.
// Keep the existing getJobDisplayPhase semantics, merging legacy Prequeue into Pending.
func jobFilterPhaseExpr() field.Field {
	return field.NewUnsafeFieldRaw(`CASE
 WHEN jobs.status = ? THEN ?
 WHEN jobs.status NOT IN (?, ?) THEN jobs.status
 WHEN COALESCE(jobs.pod_group_phase, '') IN (?, ?) THEN ?
 WHEN COALESCE(jobs.pod_group_phase, '') NOT IN (?, ?) THEN ?
 WHEN jobs.status = ? THEN ?
 ELSE ? END`,
		model.Prequeue, batch.Pending,
		"", batch.Pending,
		scheduling.PodGroupRunning, scheduling.PodGroupCompleted, filterPhaseStarting,
		"", scheduling.PodGroupPending, scheduling.PodGroupInqueue,
		"", filterPhaseInit,
		batch.Pending)
}

func scanJobPhaseFacet(ctx context.Context, scope jobListScope, request *jobListQuery, defaultDays int) ([]resputil.FacetItem, error) {
	items := make([]resputil.FacetItem, 0)
	// Group by the output alias so PostgreSQL does not see two CASE expressions
	// with different bind parameter numbers as different grouping expressions.
	err := applyJobFilters(ctx, scope, request, defaultDays).
		Select(jobFilterPhaseExpr().As("value"), query.Job.ID.Count().As("count")).
		Group(field.NewString("", "value")).
		Order(query.Job.ID.Count().Desc(), field.NewString("", "value").Asc()).
		Scan(&items)
	return items, err
}
