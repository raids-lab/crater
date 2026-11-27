package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/raids-lab/crater/dao/query"

	"gorm.io/gorm"
)

func TestRefreshLockAndCancellation(t *testing.T) {
	db := localDatabase(t)
	store := NewStatisticsService(query.Use(db))
	now := time.Now()
	must(t, db.Connection(func(conn *gorm.DB) error {
		conn = conn.Session(&gorm.Session{})
		if err := conn.Exec("SELECT pg_advisory_lock(?)", advisoryLockID).Error; err != nil {
			return err
		}
		defer conn.Exec("SELECT pg_advisory_unlock(?)", advisoryLockID)
		result, err := store.Refresh(t.Context(), now)
		if err == nil && !result.Skipped {
			t.Fatal("second instance was not excluded")
		}
		return err
	}))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	must(t, db.Callback().Query().After("gorm:query").Register("test:cancel-refresh", func(tx *gorm.DB) {
		if tx.Statement.Table == testSourceTable {
			cancel()
		}
	}))
	_, err := store.Refresh(ctx, now)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
	must(t, db.Callback().Query().Remove("test:cancel-refresh"))
	// The canceled run must release its session lock; another run can complete.
	result, err := store.Refresh(t.Context(), now)
	must(t, err)
	if result.Skipped {
		t.Fatal("canceled run leaked advisory lock")
	}
}

func TestRefreshDiscardsConnectionWhenUnlockFails(t *testing.T) {
	for _, mode := range []string{"error", "false"} {
		t.Run(mode, func(t *testing.T) {
			db := localDatabase(t)
			pool, err := db.DB()
			must(t, err)
			pool.SetMaxOpenConns(1)
			var lockedPID, nextPID int
			must(t, db.Raw("SELECT pg_backend_pid()").Scan(&lockedPID).Error)
			injected := false
			must(t, db.Callback().Row().Before("gorm:row").Register("test:fail-unlock", func(tx *gorm.DB) {
				if !strings.Contains(tx.Statement.SQL.String(), "pg_advisory_unlock") {
					return
				}
				injected = true
				tx.Statement.SQL.Reset()
				tx.Statement.Vars = nil
				if mode == "error" {
					tx.Statement.SQL.WriteString("SELECT invalid_unlock_function()")
				} else {
					tx.Statement.SQL.WriteString("SELECT false")
				}
			}))
			store := NewStatisticsService(query.Use(db))
			_, err = store.Refresh(t.Context(), time.Now())
			must(t, db.Callback().Row().Remove("test:fail-unlock"))
			if err == nil || !injected {
				t.Fatalf("unlock failure was not reported: %v", err)
			}
			must(t, db.Raw("SELECT pg_backend_pid()").Scan(&nextPID).Error)
			if nextPID == lockedPID {
				t.Fatal("locked physical connection returned to the pool")
			}
			result, err := store.Refresh(t.Context(), time.Now())
			must(t, err)
			if result.Skipped {
				t.Fatal("failed unlock left the advisory lock held")
			}
		})
	}
}
