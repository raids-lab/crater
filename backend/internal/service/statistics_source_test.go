package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func TestIndependentSourceWritesDoNotContendOnStatistics(t *testing.T) {
	db := localDatabase(t)
	day := StatisticsDay(time.Now()).AddDate(0, 0, -1)
	first := testJob("first-writer", day, day.Add(time.Hour))
	second := testJob("second-writer", day, day.Add(time.Hour))
	must(t, db.Create(&first).Error)
	must(t, db.Create(&second).Error)
	_, err := NewStatisticsService(query.Use(db)).Refresh(t.Context(), day.AddDate(0, 0, 1).Add(time.Hour))
	must(t, err)
	must(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&first).Update("profile_data", `{"cpu_usage_avg":0.25}`).Error; err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		return db.WithContext(ctx).Model(&second).Update("profile_data", `{"cpu_usage_avg":0.75}`).Error
	}))
}

func TestRefreshInvalidatesLateCommittingProfile(t *testing.T) {
	db := localDatabase(t)
	today := StatisticsDay(time.Now())
	day := today.AddDate(0, 0, -1)
	now := today.Add(time.Hour)
	job := testJob("late-commit", day, day.Add(time.Hour))
	must(t, db.Create(&job).Error)

	source := db.Begin()
	must(t, source.Error)
	t.Cleanup(func() { _ = source.Rollback().Error })
	must(t, source.Model(&job).Updates(map[string]any{
		"profile_data": `{"cpu_usage_avg":0.5}`,
	}).Error)

	store := NewStatisticsService(query.Use(db))
	_, err := store.Refresh(t.Context(), now)
	must(t, err)
	closeTo(t, dailyCPU(t, db, day).EffectiveUsageSeconds, 3600)

	must(t, source.Commit().Error)
	_, err = store.Refresh(t.Context(), now)
	must(t, err)
	closeTo(t, dailyCPU(t, db, day).EffectiveUsageSeconds, 1800)
}

func TestRefreshWatermarkCoversLateCommittingNewJob(t *testing.T) {
	db := localDatabase(t)
	today := StatisticsDay(time.Now())
	day := today.AddDate(0, 0, -1)
	job := testJob("visible", day, day.Add(time.Hour))
	must(t, db.Create(&job).Error)

	source := db.Begin()
	must(t, source.Error)
	t.Cleanup(func() { _ = source.Rollback().Error })
	var writerStart time.Time
	must(t, source.Raw("SELECT transaction_timestamp()").Scan(&writerStart).Error)
	late := testJob("not-yet-visible", day, day.Add(time.Hour))
	late.UserID = 2
	late.UpdatedAt = writerStart
	must(t, source.Create(&late).Error)

	store := NewStatisticsService(query.Use(db))
	_, err := store.Refresh(t.Context(), today)
	must(t, err)
	marker := readDayMarker(t, db, day)
	if !marker.UpdatedAt.Before(writerStart) {
		t.Fatalf("watermark %s must precede writer transaction start %s", marker.UpdatedAt, writerStart)
	}
	must(t, source.Commit().Error)
	result, err := store.Refresh(t.Context(), today)
	must(t, err)
	if result.Days != 1 {
		t.Fatalf("expected only the affected day to be rebuilt, got %d", result.Days)
	}
	var usage model.ResourceUsageDaily
	must(t, db.First(&usage, "stat_date = ? AND user_id = ? AND resource_name = ?", day, late.UserID, "cpu").Error)
	closeTo(t, usage.UsageSeconds, 7200)
	result, err = store.Refresh(t.Context(), today)
	must(t, err)
	if result.Days != 0 {
		t.Fatalf("unchanged days were rebuilt: %d", result.Days)
	}
}

func TestRefreshInvalidatesOldAndNewJobIntervals(t *testing.T) {
	db := localDatabase(t)
	today := StatisticsDay(time.Now())
	day := today.AddDate(0, 0, -4)
	job := testJob("moved", day.Add(time.Hour), day.Add(25*time.Hour))
	must(t, db.Create(&job).Error)
	store := NewStatisticsService(query.Use(db))
	_, err := store.Refresh(t.Context(), today)
	must(t, err)
	closeTo(t, dailyCPU(t, db, day).UsageSeconds, 23*7200)
	closeTo(t, dailyCPU(t, db, day.AddDate(0, 0, 1)).UsageSeconds, 7200)

	must(t, db.Model(&job).Updates(map[string]any{
		"running_timestamp":   job.RunningTimestamp.AddDate(0, 0, 2),
		"completed_timestamp": job.CompletedTimestamp.AddDate(0, 0, 2),
	}).Error)
	_, err = store.Refresh(t.Context(), today)
	must(t, err)
	var stale int64
	must(t, db.Model(&model.ResourceUsageDaily{}).Where("stat_date < ? AND resource_name <> ''", day.AddDate(0, 0, 2)).Count(&stale).Error)
	if stale != 0 {
		t.Fatalf("old job interval retained %d resource rows", stale)
	}
	moved := dailyCPU(t, db, day.AddDate(0, 0, 2))
	closeTo(t, moved.UsageSeconds, 23*7200)
	closeTo(t, dailyCPU(t, db, day.AddDate(0, 0, 3)).UsageSeconds, 7200)
}

func TestStatisticsNewJobCompletionLeavesHistoricalDayPublished(t *testing.T) {
	db := localDatabase(t)
	today := StatisticsDay(time.Now())
	yesterday, now := today.AddDate(0, 0, -1), today.Add(time.Hour)
	oldJob := testJob("yesterday", yesterday, yesterday.Add(time.Hour))
	must(t, db.Create(&oldJob).Error)
	store := NewStatisticsService(query.Use(db))
	_, err := store.Refresh(t.Context(), now)
	must(t, err)
	before := readDayMarker(t, db, yesterday)

	newJob := testJob("today", today.Add(10*time.Minute), time.Time{})
	newJob.CreationTimestamp = today
	newJob.Status = batch.Running
	must(t, db.Create(&newJob).Error)
	must(t, db.Model(&newJob).Updates(map[string]any{"completed_timestamp": now, "status": batch.Completed}).Error)
	stale, err := daySourcesChanged(db.WithContext(t.Context()), yesterday, &before)
	must(t, err)
	if stale {
		t.Fatal("today's new job invalidated yesterday for the same owner")
	}

	result, err := store.Refresh(t.Context(), now)
	must(t, err)
	after := readDayMarker(t, db, yesterday)
	if result.Days != 1 || !after.IsComplete || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("expected only today to be rebuilt, got %d days; yesterday's marker before=%+v after=%+v",
			result.Days, before, after)
	}
	closeTo(t, dailyCPU(t, db, yesterday).UsageSeconds, 7200)
	closeTo(t, dailyCPU(t, db, today).EffectiveUsageSeconds, 3000)
}

func TestRefreshInvalidatesEarlyUTCHoursInAnySessionTimeZone(t *testing.T) {
	for _, zone := range []string{"UTC", "PST8PDT", "Asia/Shanghai"} {
		t.Run(zone, func(t *testing.T) {
			db := localDatabaseWithTimeZone(t, zone)
			var actual string
			must(t, db.Raw("SHOW TimeZone").Scan(&actual).Error)
			if actual != zone {
				t.Fatalf("session time zone = %q, want %q", actual, zone)
			}
			today := StatisticsDay(time.Now())
			day := today.AddDate(0, 0, -1)
			job := testJob("early-utc", day.Add(10*time.Minute), day.Add(30*time.Minute))
			must(t, db.Create(&job).Error)
			store := NewStatisticsService(query.Use(db))
			_, err := store.Refresh(t.Context(), today)
			must(t, err)
			closeTo(t, dailyCPU(t, db, day).EffectiveUsageSeconds, 1200)

			must(t, db.Model(&job).Update("profile_data", `{"cpu_usage_avg":0.5}`).Error)
			_, err = store.Refresh(t.Context(), today)
			must(t, err)
			closeTo(t, dailyCPU(t, db, day).EffectiveUsageSeconds, 600)
		})
	}
}

func TestRefreshCapsLegacySoftDeletedJobAtDeletion(t *testing.T) {
	db := localDatabase(t)
	today := StatisticsDay(time.Now())
	day := today.AddDate(0, 0, -2)
	job := testJob("deleted-without-completion", day, time.Time{})
	job.DeletedAt = gorm.DeletedAt{Time: day.Add(2 * time.Hour), Valid: true}
	must(t, db.Create(&job).Error)
	_, err := NewStatisticsService(query.Use(db)).Refresh(t.Context(), today)
	must(t, err)
	closeTo(t, dailyCPU(t, db, day).UsageSeconds, 14400)
	var stale int64
	must(t, db.Model(&model.ResourceUsageDaily{}).Where("stat_date > ? AND resource_name <> ''", day).Count(&stale).Error)
	if stale != 0 {
		t.Fatalf("deleted job accumulated %d rows after deletion day", stale)
	}
}
