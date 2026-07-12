package vcjob

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gen"
	"gorm.io/gen/field"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

const (
	allJobsDefaultDays  = 7
	userJobsDefaultDays = 30
	jobMaxSearchRunes   = 128
)

type jobListScope struct {
	UserID    *uint
	AccountID *uint
}

type jobListQuery struct {
	Page          int      `form:"page,default=1" binding:"min=1"`
	PageSize      int      `form:"page_size,default=10" binding:"min=1,max=200"`
	Sort          string   `form:"sort"`
	Search        string   `form:"search"`
	Days          *int     `form:"days" binding:"omitempty,eq=-1|gt=0"`
	JobTypes      []string `form:"job_type" binding:"max=20,dive,required"`
	ScheduleTypes []int    `form:"schedule_type" binding:"max=20,dive,oneof=0 1"`
	Statuses      []string `form:"status" binding:"max=20,dive,required"`
	Node          *string  `form:"node"`
	sorts         []jobSort
}

type jobSort struct {
	field      string
	descending bool
}

func bindJobListQuery(c *gin.Context, withSort bool) (jobListQuery, error) {
	var request jobListQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		return jobListQuery{}, bizerr.BadRequest.ParameterError.Wrap(err, "invalid job list query")
	}
	request.Search = strings.TrimSpace(request.Search)
	if utf8.RuneCountInString(request.Search) > jobMaxSearchRunes {
		return jobListQuery{}, bizerr.BadRequest.ParameterError.New("search accepts at most 128 characters")
	}
	if request.Node != nil {
		*request.Node = strings.TrimSpace(*request.Node)
		if *request.Node == "" {
			return jobListQuery{}, bizerr.BadRequest.ParameterError.New("node must not be empty")
		}
	}
	if err := validateJobListEnums(&request); err != nil {
		return jobListQuery{}, err
	}
	if request.Page > int(^uint(0)>>1)/request.PageSize {
		return jobListQuery{}, bizerr.BadRequest.ParameterError.New("page is too large for page_size")
	}
	if withSort {
		sorts, err := parseJobSort(request.Sort)
		if err != nil {
			return jobListQuery{}, err
		}
		request.sorts = sorts
	}
	return request, nil
}

func (request *jobListQuery) offset() int {
	return (request.Page - 1) * request.PageSize
}

func (request *jobListQuery) days(defaultDays int) int {
	if request.Days == nil {
		return defaultDays
	}
	return *request.Days
}

func parseJobSort(raw string) ([]jobSort, error) {
	if raw == "" {
		return []jobSort{{field: "createdAt", descending: true}, {field: "id", descending: true}}, nil
	}
	allowed := map[string]struct{}{
		"name": {}, "jobName": {}, "owner": {}, "queue": {}, "jobType": {},
		"scheduleType": {}, "status": {}, "billedPointsTotal": {}, "createdAt": {},
		"startedAt": {}, "completedAt": {},
	}
	parts := strings.Split(raw, ",")
	if len(parts) > 3 {
		return nil, bizerr.BadRequest.ParameterError.New("sort accepts at most 3 fields")
	}
	seen := make(map[string]struct{}, len(parts)+1)
	sorts := make([]jobSort, 0, len(parts)+1)
	for _, part := range parts {
		descending := strings.HasPrefix(part, "-")
		name := strings.TrimPrefix(part, "-")
		if _, ok := allowed[name]; !ok {
			return nil, bizerr.BadRequest.ParameterError.New("unsupported sort field " + strconv.Quote(name))
		}
		if _, ok := seen[name]; ok {
			return nil, bizerr.BadRequest.ParameterError.New("duplicate sort field " + strconv.Quote(name))
		}
		seen[name] = struct{}{}
		sorts = append(sorts, jobSort{field: name, descending: descending})
	}
	sorts = append(sorts, jobSort{field: "id", descending: true})
	return sorts, nil
}

//nolint:misspell // Volcano exposes this phase as "Cancelled".
func validateJobListEnums(request *jobListQuery) error {
	for _, value := range request.JobTypes {
		switch value {
		case "jupyter", "webide", "pytorch", "tensorflow", "kuberay", "deepspeed", "openmpi", "custom":
		default:
			return bizerr.BadRequest.ParameterError.New("unsupported job_type " + strconv.Quote(value))
		}
	}
	for _, value := range request.Statuses {
		switch value {
		case "Prequeue", "Pending", "Aborting", "Aborted", "Running", "Restarting", "Completing",
			"Completed", "Terminating", "Terminated", "Failed", "Deleted", "Freed", "Cancelled":
		default:
			return bizerr.BadRequest.ParameterError.New("unsupported status " + strconv.Quote(value))
		}
	}
	return nil
}

func findJobs(
	ctx context.Context,
	scope jobListScope,
	request *jobListQuery,
	defaultDays int,
) ([]*model.Job, int64, error) {
	q := applyJobFilters(ctx, scope, request, defaultDays).
		Preload(query.Job.User).
		Preload(query.Job.Account)
	fields := jobSortFields()
	for _, sort := range request.sorts {
		expression := fields[sort.field]
		if sort.descending {
			q = q.Order(expression.Desc())
		} else {
			q = q.Order(expression.Asc())
		}
	}
	return q.FindByPage(request.offset(), request.PageSize)
}

func applyJobFilters(
	ctx context.Context,
	scope jobListScope,
	request *jobListQuery,
	defaultDays int,
) query.IJobDo {
	j := query.Job
	u := query.User
	a := query.Account
	q := j.WithContext(ctx).
		Select(j.ALL).
		LeftJoin(u, j.UserID.EqCol(u.ID)).
		LeftJoin(a, j.AccountID.EqCol(a.ID))

	if scope.UserID != nil {
		q = q.Where(j.UserID.Eq(*scope.UserID))
	}
	if scope.AccountID != nil {
		q = q.Where(j.AccountID.Eq(*scope.AccountID))
	}
	if days := request.days(defaultDays); days != -1 {
		q = q.Where(j.CreationTimestamp.Gte(time.Now().AddDate(0, 0, -days)))
	}
	if len(request.JobTypes) > 0 {
		q = q.Where(j.JobType.In(request.JobTypes...))
	}
	if len(request.ScheduleTypes) > 0 {
		q = q.Where(j.ScheduleType.In(request.ScheduleTypes...))
	}
	if len(request.Statuses) > 0 {
		q = q.Where(j.Status.In(request.Statuses...))
	}
	if request.Node != nil {
		q = q.Where(gen.Cond(datatypes.JSONArrayQuery("nodes").Contains(*request.Node))...)
	}
	if request.Search != "" {
		pattern := util.ContainsPattern(request.Search)
		q = q.Where(field.Or(
			j.Name.Lower().Like(pattern),
			j.JobName.Lower().Like(pattern),
			j.Queue.Lower().Like(pattern),
			u.Name.Lower().Like(pattern),
			u.Nickname.Lower().Like(pattern),
			a.Name.Lower().Like(pattern),
			a.Nickname.Lower().Like(pattern),
		))
	}
	return q
}

func jobSortFields() map[string]field.OrderExpr {
	j := query.Job
	return map[string]field.OrderExpr{
		"id":                j.ID,
		"name":              j.Name,
		"jobName":           j.JobName,
		"owner":             query.User.Nickname,
		"queue":             query.Account.Nickname,
		"jobType":           j.JobType,
		"scheduleType":      j.ScheduleType,
		"status":            j.Status,
		"billedPointsTotal": j.BilledPointsTotal,
		"createdAt":         j.CreationTimestamp,
		"startedAt":         j.RunningTimestamp,
		"completedAt":       j.CompletedTimestamp,
	}
}

func findJobFacets(
	ctx context.Context,
	scope jobListScope,
	request *jobListQuery,
	defaultDays int,
	includeOverview bool,
) (map[string][]resputil.FacetItem, error) {
	typeRequest := jobFacetQuery(request, "job_type")
	types, err := scanJobStringFacet(ctx, scope, &typeRequest, defaultDays, &query.Job.JobType)
	if err != nil {
		return nil, err
	}
	scheduleRequest := jobFacetQuery(request, "schedule_type")
	schedules, err := scanJobIntFacet(ctx, scope, &scheduleRequest, defaultDays, &query.Job.ScheduleType)
	if err != nil {
		return nil, err
	}
	statusRequest := jobFacetQuery(request, "status")
	statuses, err := scanJobStringFacet(ctx, scope, &statusRequest, defaultDays, &query.Job.Status)
	if err != nil {
		return nil, err
	}
	result := map[string][]resputil.FacetItem{
		"job_type":      types,
		"schedule_type": schedules,
		"status":        statuses,
	}
	if !includeOverview {
		return result, nil
	}
	runningRequest := *request
	runningRequest.Statuses = []string{"Running"}
	result["owner"], err = scanJobStringFacet(
		ctx, scope, &runningRequest, defaultDays, &query.User.Nickname,
	)
	if err != nil {
		return nil, err
	}
	result["gpu_resource"], err = scanJobGPUResourceFacet(ctx, scope, &runningRequest, defaultDays)
	return result, err
}

func jobFacetQuery(source *jobListQuery, facet string) jobListQuery {
	request := *source
	switch facet {
	case "job_type":
		request.JobTypes = nil
	case "schedule_type":
		request.ScheduleTypes = nil
	case "status":
		request.Statuses = nil
	}
	return request
}

func scanJobStringFacet(
	ctx context.Context,
	scope jobListScope,
	request *jobListQuery,
	defaultDays int,
	group *field.String,
) ([]resputil.FacetItem, error) {
	rows := make([]struct {
		Value string
		Count int64
	}, 0)
	err := applyJobFilters(ctx, scope, request, defaultDays).
		Select(group.As("value"), query.Job.ID.Count().As("count")).
		Group(group).
		Order(query.Job.ID.Count().Desc(), group.Asc()).
		Scan(&rows)
	if err != nil {
		return nil, err
	}
	items := make([]resputil.FacetItem, len(rows))
	for index, row := range rows {
		items[index] = resputil.FacetItem{Value: row.Value, Count: row.Count}
	}
	return items, nil
}

func scanJobIntFacet(
	ctx context.Context,
	scope jobListScope,
	request *jobListQuery,
	defaultDays int,
	group *field.Int,
) ([]resputil.FacetItem, error) {
	rows := make([]struct {
		Value int
		Count int64
	}, 0)
	err := applyJobFilters(ctx, scope, request, defaultDays).
		Select(group.As("value"), query.Job.ID.Count().As("count")).
		Group(group).
		Order(query.Job.ID.Count().Desc(), group.Asc()).
		Scan(&rows)
	if err != nil {
		return nil, err
	}
	items := make([]resputil.FacetItem, len(rows))
	for index, row := range rows {
		items[index] = resputil.FacetItem{Value: strconv.Itoa(row.Value), Count: row.Count}
	}
	return items, nil
}

func scanJobGPUResourceFacet(
	ctx context.Context,
	scope jobListScope,
	request *jobListQuery,
	defaultDays int,
) ([]resputil.FacetItem, error) {
	items := make([]resputil.FacetItem, 0)
	err := applyJobFilters(ctx, scope, request, defaultDays).
		UnderlyingDB().
		Joins("CROSS JOIN LATERAL jsonb_each_text(jobs.resources) AS resource(key, value)").
		Where("resource.key LIKE ?", "nvidia.com/%").
		Select("regexp_replace(resource.key, '^nvidia.com/', '') AS value, " +
			"SUM(CASE WHEN resource.value ~ '^[0-9]+$' THEN resource.value::bigint ELSE 0 END) AS count").
		Group("resource.key").
		Order("count DESC, resource.key ASC").
		Scan(&items).Error
	return items, err
}
