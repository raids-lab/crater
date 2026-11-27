package service

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/raids-lab/crater/dao/query"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

func TestDayInitializationIsAtomicAcrossInsertBatches(t *testing.T) {
	db := localDatabase(t)
	day := StatisticsDay(time.Now())
	now := day.Add(time.Hour)
	jobs := make([]model.Job, 51)
	for i := uint(0); i < 51; i++ {
		jobs[i] = testJob(fmt.Sprintf("atomic-%d", i), day, now)
		jobs[i].UserID = i + 1
	}
	must(t, db.CreateInBatches(jobs, 100).Error)
	injected := bizerr.Internal.DatabaseError.New("injected second insert batch failure")
	batches := 0
	must(t, db.Callback().Create().After("gorm:create").Register("test:fail-insert-batch", func(tx *gorm.DB) {
		if tx.Statement.Table == testDailyTable {
			batches++
			if batches == 2 {
				_ = tx.AddError(injected)
			}
		}
	}))
	err := prepareDay(db, day, now)
	must(t, db.Callback().Create().Remove("test:fail-insert-batch"))
	if !errors.Is(err, injected) || batches != 2 {
		t.Fatalf("second batch failure was not exercised: batches=%d, error=%v", batches, err)
	}
	var count int64
	must(t, db.Model(&model.ResourceUsageDaily{}).Count(&count).Error)
	if count != 0 {
		t.Fatalf("partial initialization committed %d entries", count)
	}
	must(t, prepareDay(db, day, now))
	must(t, db.Model(&model.ResourceUsageDaily{}).Count(&count).Error)
	if count != 205 {
		t.Fatalf("expected all 204 resource entries and the day marker, got %d", count)
	}
	must(t, db.Model(&model.ResourceUsageDaily{}).Where("is_complete").Count(&count).Error)
	if count != 0 {
		t.Fatal("initialization marked uncalculated entries as complete")
	}
	// A failed replacement also preserves the previous complete day.
	must(t, completeDay(db, day))
	before := dailyCPU(t, db, day)
	batches = 0
	must(t, db.Callback().Create().After("gorm:create").Register("test:fail-insert-batch", func(tx *gorm.DB) {
		if tx.Statement.Table == testDailyTable {
			batches++
			if batches == 2 {
				_ = tx.AddError(injected)
			}
		}
	}))
	err = prepareDay(db, day, now)
	must(t, db.Callback().Create().Remove("test:fail-insert-batch"))
	if !errors.Is(err, injected) {
		t.Fatalf("expected failed replacement: %v", err)
	}
	after := dailyCPU(t, db, day)
	closeTo(t, after.UsageSeconds, before.UsageSeconds)
	if !after.IsComplete || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatal("failed replacement changed the previous publication")
	}
}

func TestDayResumesOnlyUnfinishedEntries(t *testing.T) {
	db := localDatabase(t)
	day := StatisticsDay(time.Now())
	now := day.Add(time.Hour)
	job := testJob("resume", day, time.Time{})
	must(t, db.Create(&job).Error)
	must(t, prepareDay(db, day, now))
	service := NewStatisticsService(query.Use(db))
	filter := &Filter{Start: day, End: day.AddDate(0, 0, 1), Scope: "cluster", Step: "day"}
	before, err := service.overview(t.Context(), filter)
	must(t, err)
	if len(before.TotalUsage) != 0 {
		t.Fatal("placeholders leaked into statistics")
	}
	injected := bizerr.Internal.DatabaseError.New("injected entry update failure")
	writes := 0
	must(t, db.Callback().Update().Before("gorm:update").Register("test:stop-calculation", func(tx *gorm.DB) {
		if tx.Statement.Table == testDailyTable {
			writes++
			if writes == 2 {
				_ = tx.AddError(injected)
			}
		}
	}))
	err = completeDay(db, day)
	must(t, db.Callback().Update().Remove("test:stop-calculation"))
	if !errors.Is(err, injected) {
		t.Fatalf("expected interrupted calculation: %v", err)
	}
	var count int64
	must(t, db.Model(&model.ResourceUsageDaily{}).Where("is_complete AND resource_name <> ?", "").Count(&count).Error)
	if count != 1 {
		t.Fatalf("expected exactly one committed entry, got %d", count)
	}
	marker := readDayMarker(t, db, day)
	if marker.IsComplete {
		t.Fatal("an interrupted day was marked complete")
	}
	partial, err := service.overview(t.Context(), filter)
	must(t, err)
	if len(partial.TotalUsage) != 1 || partial.Progress.PendingDays != 1 {
		t.Fatalf("unexpected partial result: %+v", partial)
	}
	writes = 0
	must(t, db.Callback().Update().Before("gorm:update").Register("test:count-resumed", func(tx *gorm.DB) {
		if tx.Statement.Table == testDailyTable {
			writes++
		}
	}))
	nextTick := now.Add(time.Hour)
	result, err := service.Refresh(t.Context(), nextTick)
	must(t, db.Callback().Update().Remove("test:count-resumed"))
	must(t, err)
	if writes != 4 {
		t.Fatalf("expected three unfinished entries plus the day marker, got %d writes", writes)
	}
	finished := readDayMarker(t, db, day)
	if !finished.IsComplete || !finished.CoveredUntil.Equal(now) || result.Progress.PendingDays != 0 {
		t.Fatalf("resume did not complete the day: %+v", result)
	}
	closeTo(t, dailyCPU(t, db, day).UsageSeconds, 7200)
	_, err = service.Refresh(t.Context(), nextTick)
	must(t, err)
	closeTo(t, dailyCPU(t, db, day).UsageSeconds, 14400)
}

func TestConcurrentSourceCorrectionAndNewEntriesAreRefreshed(t *testing.T) {
	db := localDatabase(t)
	today := StatisticsDay(time.Now())
	day, now := today.AddDate(0, 0, -1), today.Add(time.Hour)
	job := testJob("concurrent-source", day, day.Add(time.Hour))
	must(t, db.Create(&job).Error)
	must(t, prepareDay(db, day, now))
	var once atomic.Bool
	must(t, db.Callback().Query().After("gorm:query").Register("test:correct-source", func(tx *gorm.DB) {
		if tx.Statement.Table == testSourceTable && once.CompareAndSwap(false, true) {
			must(t, db.Model(&job).Update("profile_data", `{"cpu_usage_avg":0.25}`).Error)
			added := testJob("new-owner", day, day.Add(time.Hour))
			added.UserID = 2
			must(t, db.Create(&added).Error)
		}
	}))
	must(t, completeDay(db, day))
	must(t, db.Callback().Query().Remove("test:correct-source"))
	if !once.Load() {
		t.Fatal("source correction was not exercised")
	}
	closeTo(t, dailyCPU(t, db, day).EffectiveUsageSeconds, 3600)
	_, err := NewStatisticsService(query.Use(db)).Refresh(t.Context(), now)
	must(t, err)
	var corrected model.ResourceUsageDaily
	must(t, db.First(&corrected, "stat_date = ? AND user_id = 1 AND resource_name = ?", day, "cpu").Error)
	closeTo(t, corrected.EffectiveUsageSeconds, 900)
	var added model.ResourceUsageDaily
	must(t, db.First(&added, "stat_date = ? AND user_id = 2 AND resource_name = ?", day, "cpu").Error)
	closeTo(t, added.UsageSeconds, 7200)
	if !added.IsComplete {
		t.Fatal("new source entry remained incomplete")
	}
}

func TestOwnerJobsAreReadOnceForAllResources(t *testing.T) {
	db := localDatabase(t)
	day := StatisticsDay(time.Now())
	job := testJob("one-owner", day, day.Add(time.Hour))
	must(t, db.Create(&job).Error)
	must(t, prepareDay(db, day, day.Add(time.Hour)))
	scans := 0
	must(t, db.Callback().Query().After("gorm:query").Register("test:count-source-scans", func(tx *gorm.DB) {
		if tx.Statement.Table == testSourceTable {
			scans++
		}
	}))
	must(t, completeDay(db, day))
	must(t, db.Callback().Query().Remove("test:count-source-scans"))
	if scans != 1 {
		t.Fatalf("expected one source scan shared by four resources, got %d", scans)
	}
}

func readDayMarker(t *testing.T, db *gorm.DB, day time.Time) model.ResourceUsageDaily {
	t.Helper()
	var row model.ResourceUsageDaily
	must(t, db.First(&row, "stat_date = ? AND resource_name = ?", day, "").Error)
	return row
}

func TestHistoricalReplacementPublishesOnlyAfterCalculationSucceeds(t *testing.T) {
	db := localDatabase(t)
	today := StatisticsDay(time.Now())
	day := today.AddDate(0, 0, -1)
	now := today.Add(time.Hour)
	job := testJob("replace-history", day, day.Add(time.Hour))
	must(t, db.Create(&job).Error)
	store := NewStatisticsService(query.Use(db))
	_, err := store.Refresh(t.Context(), now)
	must(t, err)
	must(t, db.Model(&job).Update("profile_data", `{"cpu_usage_avg":0.25}`).Error)

	injected := bizerr.Internal.DatabaseError.New("injected calculation failure")
	must(t, db.Callback().Update().Before("gorm:update").Register("test:fail-replacement", func(tx *gorm.DB) {
		if tx.Statement.Table == testDailyTable {
			closeTo(t, dailyCPU(t, db, day).EffectiveUsageSeconds, 3600)
			_ = tx.AddError(injected)
		}
	}))
	_, err = store.Refresh(t.Context(), now)
	must(t, db.Callback().Update().Remove("test:fail-replacement"))
	if !errors.Is(err, injected) || !readDayMarker(t, db, day).IsComplete {
		t.Fatalf("interrupted replacement lost previous publication: %v", err)
	}
	closeTo(t, dailyCPU(t, db, day).EffectiveUsageSeconds, 3600)
	_, err = store.Refresh(t.Context(), now.Add(time.Minute))
	must(t, err)
	closeTo(t, dailyCPU(t, db, day).EffectiveUsageSeconds, 900)
}

func TestStatisticsTodayReplacementPreservesPublishedBatchOnFailure(t *testing.T) {
	db := localDatabase(t)
	day := StatisticsDay(time.Now())
	now := day.Add(time.Hour)
	job := testJob("replace-today", day, time.Time{})
	must(t, db.Create(&job).Error)
	store := NewStatisticsService(query.Use(db))
	_, err := store.Refresh(t.Context(), now)
	must(t, err)
	previous := readDayMarker(t, db, day)

	assertPublished := func() {
		t.Helper()
		marker, cpu := readDayMarker(t, db, day), dailyCPU(t, db, day)
		if !marker.IsComplete || !cpu.IsComplete || !marker.UpdatedAt.Equal(previous.UpdatedAt) || !marker.CoveredUntil.Equal(now) {
			t.Errorf("previous batch is no longer published: marker=%+v, cpu=%+v", marker, cpu)
		}
		closeTo(t, cpu.UsageSeconds, 7200)
	}
	injected := bizerr.Internal.DatabaseError.New("injected today's calculation failure")
	must(t, db.Callback().Update().Before("gorm:update").Register("test:fail-today-replacement", func(tx *gorm.DB) {
		if tx.Statement.Table == testDailyTable {
			assertPublished()
			_ = tx.AddError(injected)
		}
	}))
	next := now.Add(time.Hour)
	_, err = store.Refresh(t.Context(), next)
	must(t, db.Callback().Update().Remove("test:fail-today-replacement"))
	if !errors.Is(err, injected) {
		t.Fatalf("expected replacement to fail: %v", err)
	}
	assertPublished()

	_, err = store.Refresh(t.Context(), next)
	must(t, err)
	if marker := readDayMarker(t, db, day); !marker.IsComplete || !marker.CoveredUntil.Equal(next) {
		t.Fatalf("successful replacement did not publish the new cutoff: %+v", marker)
	}
	closeTo(t, dailyCPU(t, db, day).UsageSeconds, 14400)
}
