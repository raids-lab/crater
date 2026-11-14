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
	"time"

	"gorm.io/gen/field"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
)

func (mgr *VolcanojobMgr) requestedInferenceWorkloads(
	ctx context.Context, scope jobListScope, request *jobListQuery, days int, enabled bool,
) ([]WorkloadResp, error) {
	if !enabled || !includesWorkloadKind(request.WorkloadKinds, workloadKindKthenaInference) ||
		!includesKthenaJobType(request.JobTypes) {
		return []WorkloadResp{}, nil
	}
	return mgr.findKthenaWorkloads(ctx, scope, request, days)
}

func (mgr *VolcanojobMgr) findWorkloadPage(
	ctx context.Context, scope jobListScope, request *jobListQuery, days int,
) ([]WorkloadResp, int64, error) {
	inference, err := mgr.requestedInferenceWorkloads(ctx, scope, request, days, mgr.isKthenaInferenceEnabled(ctx))
	if err != nil {
		return nil, 0, err
	}
	workloads := append([]WorkloadResp{}, inference...)
	var jobTotal int64
	jobOffset := 0
	if includesWorkloadKind(request.WorkloadKinds, workloadKindVolcanoJob) && includesVolcanoJobType(request.JobTypes) {
		jobRequest := *request
		jobRequest.JobTypes = regularJobTypes(request.JobTypes)
		q := applyJobFilters(ctx, scope, &jobRequest, days).Preload(query.Job.User).Preload(query.Job.Account)
		fields := workloadSortFields(q.UnderlyingDB().Name())
		for _, item := range request.sorts {
			if item.descending {
				q = q.Order(fields[item.field].Desc())
			} else {
				q = q.Order(fields[item.field].Asc())
			}
		}
		// At most len(inference) rows can displace jobs from the requested page.
		// All omitted jobs are guaranteed to precede that page in the merged order.
		jobOffset = max(0, request.offset()-len(inference))
		jobs, total, err := q.FindByPage(jobOffset, request.PageSize+len(inference))
		if err != nil {
			return nil, 0, err
		}
		jobTotal = total
		for _, job := range jobs {
			workloads = append(workloads, workloadFromJob(job))
		}
	}
	sortWorkloads(workloads, request.sorts)
	start := min(request.offset()-jobOffset, len(workloads))
	end := start + min(request.PageSize, len(workloads)-start)
	return workloads[start:end], jobTotal + int64(len(inference)), nil
}

func workloadJobFacets(
	ctx context.Context, scope jobListScope, request *jobListQuery, days int, facet string,
) ([]resputil.FacetItem, error) {
	if !includesWorkloadKind(request.WorkloadKinds, workloadKindVolcanoJob) || !includesVolcanoJobType(request.JobTypes) {
		return []resputil.FacetItem{}, nil
	}
	regular := *request
	regular.JobTypes = regularJobTypes(request.JobTypes)
	var group *field.String
	switch facet {
	case workloadFacetJobType:
		group = &query.Job.JobType
	case workloadFacetStatus:
		group = &query.Job.Status
	case workloadFacetOwner:
		group = &query.User.Nickname
	case workloadFacetGPUResource:
		return scanJobGPUResourceFacet(ctx, scope, &regular, days)
	default:
		total, err := applyJobFilters(ctx, scope, &regular, days).Count()
		return []resputil.FacetItem{{Value: workloadKindVolcanoJob, Count: total}}, err
	}
	return scanJobStringFacet(ctx, scope, &regular, days, group)
}

func mergeWorkloadFacets(left, right []resputil.FacetItem) []resputil.FacetItem {
	counts := make(map[string]int64, len(left)+len(right))
	for _, list := range [][]resputil.FacetItem{left, right} {
		for _, item := range list {
			if item.Value != "" && item.Count > 0 {
				counts[item.Value] += item.Count
			}
		}
	}
	return sortedFacetItems(counts)
}

// The SQL window and the in-memory merge must use the same ordering, including
// missing values and PostgreSQL's configurable text collation.
func workloadSortFields(dialect string) map[string]field.OrderExpr {
	fields := jobSortFields()
	textOrder := "COALESCE(?, '')"
	if dialect == "postgres" {
		textOrder = "COALESCE(?, '') COLLATE \"C\""
	}
	for _, name := range []string{
		workloadSortName, workloadSortJobName, workloadSortOwner, workloadSortQueue, workloadSortJobType, workloadFieldStatus,
	} {
		fields[name] = field.NewUnsafeFieldRaw(textOrder, fields[name])
	}
	for _, name := range []string{workloadSortStartedAt, workloadSortCompletedAt} {
		fields[name] = field.NewUnsafeFieldRaw("COALESCE(?, ?)", fields[name], time.Time{})
	}
	fields[workloadSortBilledPointsTotal] = field.NewUnsafeFieldRaw("COALESCE(?, 0)", fields[workloadSortBilledPointsTotal])
	return fields
}
