package service

import (
	"testing"
	"time"

	"github.com/raids-lab/crater/dao/query"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
)

func TestRankingsRejectInvalidSQLSelectors(t *testing.T) {
	for _, selector := range []string{"dimension", "metric", "order"} {
		t.Run(selector, func(t *testing.T) {
			req := payload.StatisticsRankingReq{Dimension: "user", Metric: "usage", Order: statisticsOrderDesc}
			switch selector {
			case "dimension":
				req.Dimension = "user; SELECT 1"
			case "metric":
				req.Metric = "sum(usage_seconds)"
			case "order":
				req.Order = "desc; SELECT 1"
			}
			_, err := NewStatisticsService(nil).GetRankings(t.Context(), &req)
			businessError, ok := bizerr.FromError(err)
			if !ok || businessError.Code != bizerr.BadRequest.ParameterError {
				t.Fatalf("invalid %s must be rejected before querying: %v", selector, err)
			}
		})
	}
}

//nolint:gocyclo // Check weighting, scopes and ranks against one fixed dataset.
func TestOverviewAndRankings(t *testing.T) {
	db := localDatabase(t)
	insertResources(t, db)
	day := StatisticsDay(time.Now()).AddDate(0, 0, -1)
	// Unequal durations prove ratios are weighted by evaluated allocation.
	for i := 0; i < 3; i++ {
		j := testJob(string(rune('a'+i)), day, day.Add(time.Duration(i+1)*time.Hour))
		j.UserID = []uint{1, 2, 3}[i]
		j.AccountID = []uint{1, 1, 2}[i]
		if i == 1 {
			j.ProfileData.Data().CPUUsageAvg = number(0)
		}
		if i == 2 {
			j.ProfileData = nil
		}
		must(t, db.Create(&j).Error)
	}
	must(t, db.Create(&[]model.User{
		{Model: gorm.Model{ID: 1}, Name: "one", Space: "/one", Nickname: "One", Role: model.RoleUser},
		{Model: gorm.Model{ID: 2}, Name: "two", Space: "/two", Nickname: "Two", Role: model.RoleUser},
		{Model: gorm.Model{ID: 3}, Name: "three", Space: "/three", Nickname: "Three", Role: model.RoleUser},
	}).Error)
	must(t, db.Create(&[]model.Account{
		{Model: gorm.Model{ID: 1}, Name: "team1", Space: "/team1", Nickname: "Team 1"},
		{Model: gorm.Model{ID: 2}, Name: "team2", Space: "/team2", Nickname: "Team 2"},
	}).Error)
	store := NewStatisticsService(query.Use(db))
	_, err := store.Refresh(t.Context(), day.Add(25*time.Hour))
	must(t, err)
	f := Filter{Start: day, End: day.Add(24 * time.Hour), Scope: "cluster", Step: "day"}
	overview, err := store.overview(t.Context(), &f)
	must(t, err)
	cpu := overview.TotalUsage["cpu"]
	closeTo(t, cpu.Usage, 12)
	closeTo(t, cpu.EffectiveUsage, 1)
	closeTo(t, cpu.ProfiledUsage, 6)
	closeTo(t, *cpu.Utilization, 1.0/6)
	closeTo(t, *cpu.Coverage, .5)
	if overview.TotalUsage["looks-like-gpu"].Utilization != nil {
		t.Fatal("unsupported resource evaluated")
	}
	if len(overview.Series) != 1 {
		t.Fatal("incorrect daily buckets")
	}
	closeTo(t, overview.Series[0].Usage["cpu"], 12)
	if _, exists := overview.Series[0].EffectiveUsage["looks-like-gpu"]; exists {
		t.Fatal("unknown effective usage must be omitted from series")
	}
	for _, scope := range []string{ScopeUser, ScopeAccount} {
		scoped := f
		scoped.Scope = scope
		scoped.TargetID = 2
		got, err := store.overview(t.Context(), &scoped)
		must(t, err)
		want := 4.0
		if scope == ScopeAccount {
			want = 6
		}
		closeTo(t, got.TotalUsage["cpu"].Usage, want)
	}
	f.Step = "week"
	f.End = day.AddDate(0, 0, 9)
	weekly, err := store.overview(t.Context(), &f)
	must(t, err)
	if len(weekly.Series) != 2 {
		t.Fatal("incorrect weekly buckets")
	}
	closeTo(t, weekly.Series[0].Usage["cpu"], 12)
	f.End = day.AddDate(0, 0, 1)
	r := RankingFilter{Filter: f, Dimension: ScopeUser, Metric: "utilization", Order: "asc", ResourceName: "cpu", Page: 1, PageSize: 1}
	ranked, err := store.rankings(t.Context(), &r)
	must(t, err)
	if ranked.Total != 2 || len(ranked.Rows) != 1 || ranked.Rows[0].ID != 2 || ranked.Rows[0].Rank != 1 {
		t.Fatalf("zero-vs-missing low utilization rank: %+v", ranked)
	}
	r.Page = 2
	ranked, err = store.rankings(t.Context(), &r)
	must(t, err)
	if ranked.Rows[0].ID != 1 || ranked.Rows[0].Rank != 2 {
		t.Fatal("global rank lost across pages")
	}
	r.Page = 1
	r.PageSize = 20
	r.Metric = MetricUsage
	r.Order = statisticsOrderDesc
	r.Dimension = ScopeAccount
	ranked, err = store.rankings(t.Context(), &r)
	must(t, err)
	if ranked.Total != 2 || ranked.Rows[0].Rank != 1 || ranked.Rows[1].Rank != 1 || ranked.Rows[0].ID != 1 {
		t.Fatalf("ties/order: %+v", ranked)
	}
	r.Dimension = ScopeUser
	r.Metric = "effectiveUsage"
	r.ResourceName = ""
	r.ResourceType = "gpu"
	ranked, err = store.rankings(t.Context(), &r)
	must(t, err)
	if ranked.Total != 2 {
		t.Fatal("GPU type dictionary filter failed")
	}
	closeTo(t, ranked.Rows[0].EffectiveUsage, 2)
}

func TestResourceStatisticsExcludeTensorboardIncludingDeletedPanels(t *testing.T) {
	db := localDatabase(t)
	day := StatisticsDay(time.Now()).AddDate(0, 0, -1)
	for i, name := range []string{"training-a", "training-b", "tb-active", "tb-deleted"} {
		j := testJob(name, day, day.Add(2*time.Hour))
		j.UserID = []uint{1, 2, 1, 2}[i]
		j.AccountID = j.UserID
		if i >= 2 {
			j.JobType = model.JobTypeTensorboard
		}
		if i == 0 || i == 3 {
			j.DeletedAt = gorm.DeletedAt{Time: day.Add(2 * time.Hour), Valid: true}
		}
		if i == 2 {
			j.CompletedTimestamp = time.Time{}
		}
		must(t, db.Create(&j).Error)
	}
	store := NewStatisticsService(query.Use(db))
	_, err := store.Refresh(t.Context(), day.Add(25*time.Hour))
	must(t, err)
	for _, tc := range []struct {
		scope string
		id    uint
		want  float64
	}{{"cluster", 0, 8}, {ScopeUser, 1, 4}, {ScopeAccount, 2, 4}} {
		got, err := store.overview(t.Context(), &Filter{Start: day, End: day.AddDate(0, 0, 1), Scope: tc.scope, TargetID: tc.id, Step: "day"})
		must(t, err)
		closeTo(t, got.TotalUsage["cpu"].Usage, tc.want)
	}
}

func TestAllGPURankingIncludesRetiredResources(t *testing.T) {
	db := localDatabase(t)
	insertResources(t, db)
	today := StatisticsDay(time.Now())
	day := today.AddDate(0, 0, -1)
	job := testJob("retired-gpu", day, day.Add(time.Hour))
	must(t, db.Create(&job).Error)
	store := NewStatisticsService(query.Use(db))
	_, err := store.Refresh(t.Context(), today)
	must(t, err)

	must(t, db.Where("resource_name = ?", "vendor.example/accelerator").Delete(&model.Resource{}).Error)
	must(t, db.Model(&job).Update("profile_data", `{"gpu_util_avg":0.5}`).Error)
	_, err = store.Refresh(t.Context(), today)
	must(t, err)
	ranking, err := store.rankings(t.Context(), &RankingFilter{
		Filter:    Filter{Start: day, End: today, Scope: "cluster"},
		Dimension: ScopeUser, Metric: MetricEffectiveUsage, Order: statisticsOrderDesc,
		ResourceType: "gpu", Page: 1, PageSize: 20,
	})
	must(t, err)
	if ranking.Total != 1 {
		t.Fatalf("retired GPU absent from all-GPU ranking: %+v", ranking)
	}
	closeTo(t, ranking.Rows[0].Usage, 4)
	closeTo(t, ranking.Rows[0].EffectiveUsage, 2)
}

func TestCalendarDayDiffBeyondDurationLimit(t *testing.T) {
	start := time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	const millenniumDays = 365242
	if got := calendarDayDiff(start, end); got != millenniumDays {
		t.Fatalf("calendar day difference=%d, want %d", got, millenniumDays)
	}
	leapDay := time.Date(2024, 2, 29, 23, 59, 59, 0, time.UTC)
	if got := calendarDayDiff(leapDay, leapDay.Add(time.Second)); got != 1 {
		t.Fatalf("leap day rollover difference=%d, want 1", got)
	}
}

func TestStatisticsSeriesDistinguishesMissingPartialAndEmptyDays(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	day := func(offset int) time.Time { return start.AddDate(0, 0, offset) }
	markers := []*model.ResourceUsageDaily{
		{StatDate: day(1), IsComplete: true, CoveredUntil: day(2)},
		{StatDate: day(2), IsComplete: true, CoveredUntil: day(2).Add(time.Hour)},
	}
	rows := []model.ResourceUsageDaily{{StatDate: day(2), ResourceName: "cpu", UsageSeconds: 7200}}
	filter := Filter{Start: start, End: day(3)}
	var daily payload.StatisticsResp
	buildStatisticsSeries(&daily, &filter, 1, markers, rows)
	want := []string{bucketPending, bucketComplete, bucketPartial}
	for i, point := range daily.Series {
		if point.Status != want[i] {
			t.Fatalf("day %d status=%s, want %s", i, point.Status, want[i])
		}
	}
	closeTo(t, daily.Series[1].Usage["cpu"], 0)
	closeTo(t, daily.Series[2].Usage["cpu"], 2)

	var weekly payload.StatisticsResp
	buildStatisticsSeries(&weekly, &filter, 7, markers, rows)
	if weekly.Series[0].Status != bucketPartial {
		t.Fatalf("week with missing or unfinished days marked %s", weekly.Series[0].Status)
	}
}
