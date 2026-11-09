package vcjob

import (
	"context"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/pkg/utils"
)

// JobLockDuration is a finite extension or a permanent lock.
type JobLockDuration struct {
	IsPermanent bool  `json:"isPermanent"`
	Days        int64 `json:"days"`
	Hours       int64 `json:"hours"`
	Minutes     int64 `json:"minutes"`
}

func (d JobLockDuration) durationMinutes() (int64, error) {
	const maxMinutes = math.MaxInt64 / int64(time.Minute)
	if d.Days < 0 || d.Hours < 0 || d.Minutes < 0 {
		return 0, bizerr.BadRequest.ParameterError.New("lock duration values must be non-negative")
	}
	if d.IsPermanent {
		if d.Days != 0 || d.Hours != 0 || d.Minutes != 0 {
			return 0, bizerr.BadRequest.ParameterError.New("permanent lock cannot include a finite duration")
		}
		return 0, nil
	}
	if d.Days > maxMinutes/1440 || d.Hours > maxMinutes/60 || d.Minutes > maxMinutes {
		return 0, bizerr.BadRequest.ParameterError.New("lock duration is too large")
	}
	minutes := d.Days*1440 + d.Hours*60 + d.Minutes
	if minutes == 0 || minutes > maxMinutes {
		return 0, bizerr.BadRequest.ParameterError.New("lock duration must be positive and within range")
	}
	return minutes, nil
}

// ExtendJobLock locks the job row and extends its deadline inside the caller's transaction.
func ExtendJobLock(
	ctx context.Context, tx *query.Query, jobName string, duration JobLockDuration, requireRunning bool,
) (*model.ApprovedJobLock, error) {
	minutes, err := duration.durationMinutes()
	if err != nil {
		return nil, err
	}
	j := tx.Job
	jobs := j.WithContext(ctx).Where(j.JobName.Eq(jobName))
	job, err := jobs.Select(j.Status, j.LockedTimestamp).Clauses(clause.Locking{Strength: "UPDATE"}).Take()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerr.NotFound.DataBaseNotFound.New("job not found")
	}
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to load job lock")
	}
	if requireRunning && job.Status != batch.Running {
		return nil, bizerr.Conflict.ResourceStatusError.New("only running jobs can be approved and locked")
	}
	permanent := utils.GetPermanentTime()
	if job.LockedTimestamp.Equal(permanent) {
		return nil, bizerr.Conflict.ResourceStatusError.New("job is already permanently locked")
	}
	lockUntil := permanent
	if !duration.IsPermanent {
		base := utils.GetLocalTime()
		if job.LockedTimestamp.After(base) {
			base = job.LockedTimestamp
		}
		lockUntil = base.Add(time.Duration(minutes) * time.Minute)
		if !lockUntil.Before(permanent) {
			return nil, bizerr.BadRequest.ParameterError.New("lock expiration is out of range")
		}
	}
	if _, err := j.WithContext(ctx).Where(j.JobName.Eq(jobName)).Update(j.LockedTimestamp, lockUntil); err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to update job lock")
	}
	return &model.ApprovedJobLock{
		IsPermanent: duration.IsPermanent, DurationMinutes: minutes, LockedUntil: lockUntil,
	}, nil
}
