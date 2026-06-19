package vcjob

import (
	"context"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gen"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
)

// jobListReq is the shared request shape for paginated job-list endpoints.
//
// `Days` is kept for backward compatibility with old `?days=` consumers.
// New consumers should send explicit `start_time` / `end_time` instead.
//
// Filters are applied in addition to the per-handler scoping (self/admin/user).
// `Status`, `JobType` and `Queue` accept multiple comma-separated values.
type jobListReq struct {
	payload.ListPageQuery
	Days      int    `form:"days"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	Status    string `form:"status"`
	JobType   string `form:"job_type"`
	Queue     string `form:"queue"`
	Search    string `form:"search"`
}

// activeFilter is a single column filter passed in from the request.
//
// It records the dimension key (matching the Facets map key the frontend reads)
// and the predicate. A facet is computed by applying every filter except its
// own dimension, which is the standard faceted-search semantic.
type activeFilter struct {
	dimension string
	cond      gen.Condition
}

// jobFilterPlan is the result of parsing jobListReq into a set of predicates.
//
// `scope` contains predicates that always apply (e.g. user-id / account-id /
// time window). `dimensions` contains per-dimension predicates that we may need
// to omit when computing the corresponding facet count.
type jobFilterPlan struct {
	scope      []gen.Condition
	dimensions []activeFilter
}

// parseJobFilters converts the request into a jobFilterPlan.
//
// Time-window logic mirrors the previous handlers:
//   - explicit start_time / end_time (RFC3339) take precedence
//   - days == -1 disables the time filter
//   - days <= 0 falls back to defaultDays
//   - otherwise CreatedAt >= now - days
func parseJobFilters(req *jobListReq, defaultDays int) (*jobFilterPlan, error) {
	j := query.Job
	plan := &jobFilterPlan{}

	// time window
	if req.StartTime != "" {
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid start_time")
		}
		plan.scope = append(plan.scope, j.CreatedAt.Gte(t))
	}
	if req.EndTime != "" {
		t, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid end_time")
		}
		plan.scope = append(plan.scope, j.CreatedAt.Lte(t))
	}
	if req.StartTime == "" && req.EndTime == "" && req.Days != -1 {
		days := defaultDays
		if req.Days > 0 {
			days = req.Days
		}
		plan.scope = append(plan.scope, j.CreatedAt.Gte(time.Now().AddDate(0, 0, -days)))
	}

	// search by job_name (use Name column which is the human-readable display name)
	if s := strings.TrimSpace(req.Search); s != "" {
		plan.scope = append(plan.scope, j.Name.Like("%"+s+"%"))
	}

	// faceted dimensions: status / job_type / queue
	if statuses := splitMulti(req.Status); len(statuses) > 0 {
		plan.dimensions = append(plan.dimensions, activeFilter{
			dimension: "status",
			cond:      j.Status.In(statuses...),
		})
	}
	if jobTypes := splitMulti(req.JobType); len(jobTypes) > 0 {
		plan.dimensions = append(plan.dimensions, activeFilter{
			dimension: "jobType",
			cond:      j.JobType.In(jobTypes...),
		})
	}
	if queues := splitMulti(req.Queue); len(queues) > 0 {
		plan.dimensions = append(plan.dimensions, activeFilter{
			dimension: "queue",
			cond:      j.Queue.In(queues...),
		})
	}

	return plan, nil
}

func splitMulti(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := parts[:0]
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// allConditions returns scope + every dimension filter.
func (p *jobFilterPlan) allConditions() []gen.Condition {
	all := make([]gen.Condition, 0, len(p.scope)+len(p.dimensions))
	all = append(all, p.scope...)
	for i := range p.dimensions {
		all = append(all, p.dimensions[i].cond)
	}
	return all
}

// conditionsExcept returns scope + every dimension filter except `omit`.
//
// Used when computing facet counts for a single dimension.
func (p *jobFilterPlan) conditionsExcept(omit string) []gen.Condition {
	all := make([]gen.Condition, 0, len(p.scope)+len(p.dimensions))
	all = append(all, p.scope...)
	for i := range p.dimensions {
		if p.dimensions[i].dimension == omit {
			continue
		}
		all = append(all, p.dimensions[i].cond)
	}
	return all
}

// applyJobOrder maps the optional sort_by/order pair to a query order clause.
// Defaults to created_at desc when sort_by is empty or unrecognized.
func applyJobOrder(qu query.IJobDo, sortBy string, order payload.Order) query.IJobDo {
	j := query.Job
	desc := order != payload.Asc

	switch sortBy {
	case "name":
		if desc {
			return qu.Order(j.Name.Desc())
		}
		return qu.Order(j.Name)
	case "status":
		if desc {
			return qu.Order(j.Status.Desc())
		}
		return qu.Order(j.Status)
	case "job_type":
		if desc {
			return qu.Order(j.JobType.Desc())
		}
		return qu.Order(j.JobType)
	case "queue":
		if desc {
			return qu.Order(j.Queue.Desc())
		}
		return qu.Order(j.Queue)
	case "created_at", "":
		fallthrough
	default:
		if desc || sortBy == "" {
			return qu.Order(j.CreatedAt.Desc())
		}
		return qu.Order(j.CreatedAt)
	}
}

// fetchJobsPaged runs the paginated rows query, the total count and three
// facet GROUP BY queries concurrently using errgroup.
//
// The returned facets always contain status / jobType / queue keys (possibly
// empty maps) so that the frontend toolbar can render badges deterministically.
//
// `withPreload` controls whether Account / User relations are preloaded for
// the rows query. Billing endpoints set this to false to skip joins.
func fetchJobsPaged(
	ctx context.Context,
	plan *jobFilterPlan,
	pq payload.ListPageQuery,
	withPreload bool,
	defaultDaysCovered bool,
) ([]*model.Job, int64, map[string]map[string]int64, error) {
	_ = defaultDaysCovered // reserved for future tuning hooks

	offset, limit, order := pq.Normalize()
	paging := pq.IsPagingRequested()
	j := query.Job

	var (
		rows   []*model.Job
		total  int64
		facets = map[string]map[string]int64{
			"status":  {},
			"jobType": {},
			"queue":   {},
		}
	)

	g, gctx := errgroup.WithContext(ctx)

	// rows
	g.Go(func() error {
		qu := j.WithContext(gctx).Where(plan.allConditions()...)
		if withPreload {
			qu = qu.Preload(j.Account).Preload(j.User)
		}
		qu = applyJobOrder(qu, pq.SortBy, order)
		if paging {
			qu = qu.Offset(offset).Limit(limit)
		}
		out, err := qu.Find()
		if err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "list jobs")
		}
		rows = out
		return nil
	})

	// total — only when paging requested; otherwise we use len(rows) below
	if paging {
		g.Go(func() error {
			count, err := j.WithContext(gctx).Where(plan.allConditions()...).Count()
			if err != nil {
				return bizerr.Internal.DatabaseError.Wrap(err, "count jobs")
			}
			total = count
			return nil
		})
	}

	// facets — only when caller explicitly opted into pagination
	if paging {
		dims := []struct {
			key string
			col string
		}{
			{"status", "status"},
			{"jobType", "job_type"},
			{"queue", "queue"},
		}
		for _, d := range dims {
			d := d
			g.Go(func() error {
				conds := plan.conditionsExcept(d.key)
				result, err := groupCount(gctx, conds, d.col)
				if err != nil {
					return bizerr.Internal.DatabaseError.Wrap(err, "facet "+d.key)
				}
				facets[d.key] = result
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return nil, 0, nil, err
	}
	if !paging {
		total = int64(len(rows))
		facets = nil
	}
	return rows, total, facets, nil
}

// groupCount runs a `SELECT col, COUNT(*) FROM jobs WHERE <conds> GROUP BY col`
// using the underlying gorm.DB so we can scan into a typed slice.
func groupCount(ctx context.Context, conds []gen.Condition, column string) (map[string]int64, error) {
	j := query.Job

	type row struct {
		Key   string `gorm:"column:key"`
		Count int64  `gorm:"column:count"`
	}

	var rows []row
	db := j.WithContext(ctx).Where(conds...).UnderlyingDB().
		Select(column + " AS key, COUNT(*) AS count").
		Group(column)
	if err := db.Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		if r.Key == "" {
			continue
		}
		out[r.Key] = r.Count
	}
	return out, nil
}
