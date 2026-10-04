package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/raids-lab/crater/dao/query"
	"gorm.io/gorm"
)

var localAdmissionLocks [256]sync.Mutex

type admissionLockBusy struct{}

func (admissionLockBusy) Error() string { return "admission lock busy" }

// WithWorkloadAdmission serializes quota observation and reservation for both
// training jobs and serving deployments across API and controller replicas.
// The dedicated transaction holds only the advisory lock; reservation writes
// complete before it releases the lock and are visible to the next observer.
func WithWorkloadAdmission(ctx context.Context, userID, accountID uint, fn func() error) error {
	db := query.GetDB()
	if db.Name() != "postgres" {
		lock := &localAdmissionLocks[(userID*31+accountID)%uint(len(localAdmissionLocks))]
		lock.Lock()
		defer lock.Unlock()
		return fn()
	}
	// Do not leave waiting transactions occupying the pool while the lock holder
	// needs other DB connections for policy checks and reservation writes.
	busy := admissionLockBusy{}
	const retryDelay = 50 * time.Millisecond
	for {
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			scope := fmt.Sprintf("crater-admission:%d:%d", accountID, userID)
			var locked bool
			if err := tx.Raw("SELECT pg_try_advisory_xact_lock(hashtextextended(?, 0))", scope).Scan(&locked).Error; err != nil {
				return err
			}
			if !locked {
				return busy
			}
			return fn()
		})
		if !errors.Is(err, busy) {
			return err
		}
		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
