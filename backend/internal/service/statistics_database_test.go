package service

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/raids-lab/crater/dao/query"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/raids-lab/crater/dao/model"
)

const (
	testSourceTable = "jobs"
	testDailyTable  = "resource_usage_daily"
)

// Integration tests write only to a unique schema on an explicitly selected loopback database.
func localDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return localDatabaseWithTimeZone(t, "UTC")
}

func localDatabaseWithTimeZone(t *testing.T, zone string) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("CRATER_STATISTICS_TEST_DSN")
	if dsn == "" {
		t.Skip("set CRATER_STATISTICS_TEST_DSN to run local PostgreSQL tests")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if (u.Scheme != "postgres" && u.Scheme != "postgresql") || (u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") {
		t.Fatal("statistics tests require an explicit loopback PostgreSQL URL")
	}
	config := &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, IgnoreRelationshipsWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)}
	admin, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("statistics_test_%d", time.Now().UnixNano())
	if err = admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		pool, _ := admin.DB()
		_ = pool.Close()
	})
	q := u.Query()
	q.Set("search_path", schema)
	q.Set("options", "-c TimeZone="+zone)
	u.RawQuery = q.Encode()
	db, err := gorm.Open(postgres.Open(u.String()), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool, _ := db.DB(); _ = pool.Close() })
	if err = db.AutoMigrate(&model.Job{}, &model.Resource{}, &model.User{}, &model.Account{}, &model.ResourceUsageDaily{}); err != nil {
		t.Fatal(err)
	}
	return db
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func insertResources(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, r := range testResources() {
		must(t, db.Create(r).Error)
	}
}
func dailyCPU(t *testing.T, db *gorm.DB, day time.Time) model.ResourceUsageDaily {
	t.Helper()
	var row model.ResourceUsageDaily
	must(t, db.First(&row, "stat_date = ? AND resource_name = ?", day, "cpu").Error)
	return row
}
func TestRefreshRecoveryAndInvalidation(t *testing.T) {
	db := localDatabase(t)
	insertResources(t, db)
	today := StatisticsDay(time.Now())
	start := today.AddDate(0, 0, -3)
	now := today.Add(time.Hour)
	j := testJob("finished", start.Add(23*time.Hour), start.Add(25*time.Hour))
	must(t, db.Create(&j).Error)
	// Retain deleted training jobs, exclude deleted and running TensorBoards.
	must(t, db.Delete(&j).Error)
	tb := testJob("tb", start, now)
	tb.JobType = model.JobTypeTensorboard
	must(t, db.Create(&tb).Error)
	never := testJob("never", time.Time{}, now)
	must(t, db.Create(&never).Error)
	store := NewStatisticsService(query.Use(db))
	first, err := store.Refresh(t.Context(), now)
	must(t, err)
	if first.Progress.CompleteBefore != today.Format(time.DateOnly) {
		t.Fatalf("cursor failed: %+v", first)
	}
	closeTo(t, dailyCPU(t, db, start).UsageSeconds, 7200)
	closeTo(t, dailyCPU(t, db, start.AddDate(0, 0, 1)).EffectiveUsageSeconds, 3600)
	// Day 3 is empty but cursor must advance over it. Repeated refresh cannot duplicate rows.
	_, err = store.Refresh(t.Context(), now)
	must(t, err)
	closeTo(t, dailyCPU(t, db, start).UsageSeconds, 7200)
	must(t, db.Unscoped().Model(&j).Update("profile_data", `{"cpu_usage_avg":0.5}`).Error)
	_, err = store.Refresh(t.Context(), now)
	must(t, err)
	closeTo(t, dailyCPU(t, db, start).EffectiveUsageSeconds, 1800)
	// A late completion truncates a previously open interval and evaluates its past dates.
	running := testJob("running", start.AddDate(0, 0, 2), time.Time{})
	must(t, db.Create(&running).Error)
	_, err = store.Refresh(t.Context(), now)
	must(t, err)
	closeTo(t, dailyCPU(t, db, today).ProfiledUsageSeconds, 0)
	must(t, db.Model(&running).Updates(map[string]any{
		"completed_timestamp": today.Add(-time.Hour),
	}).Error)
	_, err = store.Refresh(t.Context(), now)
	must(t, err)
	var count int64
	must(t, db.Model(&model.ResourceUsageDaily{}).Where("stat_date = ? AND resource_name <> ?", today, "").Count(&count).Error)
	if count != 0 {
		t.Fatal("late completion left stale today rows")
	}
	closeTo(t, dailyCPU(t, db, start.AddDate(0, 0, 2)).EffectiveUsageSeconds, 23*3600)
	// Resource reclassification invalidates prior GPU estimates.
	must(t, db.Model(&model.Resource{}).Where("resource_name = ?", "vendor.example/accelerator").Update("type", model.ResourceTypeRDMA).Error)
	_, err = store.Refresh(t.Context(), now)
	must(t, err)
	var gpu model.ResourceUsageDaily
	must(t, db.First(&gpu, "stat_date = ? AND resource_name = ?", start.AddDate(0, 0, 2), "vendor.example/accelerator").Error)
	closeTo(t, gpu.ProfiledUsageSeconds, 0)
	// A new, older job rewinds the durable backfill cursor.
	older := testJob("older", start.AddDate(0, 0, -2), start.AddDate(0, 0, -2).Add(time.Hour))
	must(t, db.Create(&older).Error)
	res, err := store.Refresh(t.Context(), now)
	must(t, err)
	if res.Progress.FirstDate != StatisticsDay(older.RunningTimestamp).Format(time.DateOnly) {
		t.Fatal("did not recover newly inserted history")
	}
}

func TestRefreshBatchesAndInitialProgress(t *testing.T) {
	db := localDatabase(t)
	today := StatisticsDay(time.Now())
	now := today.Add(time.Hour)
	p, err := progress(db, now)
	must(t, err)
	if p.Ready {
		t.Fatal("unpublished state must not be ready")
	}
	jobs := make([]model.Job, 501)
	for i := range jobs {
		jobs[i] = testJob(fmt.Sprintf("batch-%d", i), today, today.Add(time.Hour))
	}
	must(t, db.CreateInBatches(jobs, 100).Error)
	_, err = NewStatisticsService(query.Use(db)).Refresh(t.Context(), now)
	must(t, err)
	closeTo(t, dailyCPU(t, db, today).UsageSeconds, 501*7200)
}
