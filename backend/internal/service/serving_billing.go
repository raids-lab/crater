package service

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/raids-lab/crater/dao/model"
)

// ObserveServingUsage persists the observed resource-occupancy interval before
// a Pod's billing finalizer can be released. No Kubernetes I/O runs in this transaction.
func (s *BillingService) ObserveServingUsage(ctx context.Context, observed *model.ServingUsage) error {
	return s.q.User.WithContext(ctx).UnderlyingDB().Session(&gorm.Session{NewDB: true, Context: ctx}).Transaction(func(tx *gorm.DB) error {
		initial := *observed
		initial.EndedAt = nil
		initial.ObservedUntil = initial.StartedAt
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&initial).Error; err != nil {
			return err
		}
		var current model.ServingUsage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "pod_uid = ?", observed.PodUID).Error; err != nil {
			return err
		}
		if current.EndedAt != nil {
			return nil
		}
		if observed.ObservedUntil.After(current.ObservedUntil) {
			current.ObservedUntil = observed.ObservedUntil
		}
		if observed.EndedAt != nil {
			current.EndedAt = observed.EndedAt
			current.ObservedUntil = *observed.EndedAt
		}
		if err := tx.Model(&current).Updates(map[string]any{
			"observed_until": current.ObservedUntil, "ended_at": current.EndedAt,
		}).Error; err != nil {
			return err
		}
		enabled := getSystemBoolWithTx(ctx, tx, model.ConfigKeyEnableBillingFeature) &&
			getSystemBoolWithTx(ctx, tx, model.ConfigKeyEnableBillingActive)
		if !enabled {
			// Disabled billing does not accumulate debt to be charged on reactivation.
			return tx.Model(&current).Update("last_settled_at", current.ObservedUntil).Error
		}
		if current.EndedAt == nil {
			return nil
		}
		prices, err := loadUnitPriceMapTx(ctx, tx)
		if err != nil {
			return err
		}
		_, err = settleServingUsageTx(ctx, tx, &current, prices)
		return err
	})
}

// Called by the same periodic, price-change and account-credit transactions as jobs.
func settleServingUsagesTx(ctx context.Context, tx *gorm.DB, accountID uint, prices map[string]int64) (int, error) {
	var usages []model.ServingUsage
	q := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Order("pod_uid")
	if accountID != 0 {
		q = q.Where("account_id = ?", accountID)
	}
	if err := q.Where("last_settled_at IS NULL OR last_settled_at < observed_until").Find(&usages).Error; err != nil {
		return 0, err
	}
	count := 0
	for i := range usages {
		cost, err := settleServingUsageTx(ctx, tx, &usages[i], prices)
		if err != nil {
			return 0, err
		}
		if cost > 0 {
			count++
		}
	}
	return count, nil
}

func settleServingUsageTx(ctx context.Context, tx *gorm.DB, usage *model.ServingUsage, prices map[string]int64) (int64, error) {
	// Reuse arithmetic and balance order without storing a synthetic Job row.
	job := &model.Job{UserID: usage.UserID, AccountID: usage.AccountID, Resources: usage.Resources,
		RunningTimestamp: usage.StartedAt, LastSettledAt: usage.LastSettledAt, BilledPointsTotal: usage.BilledPointsTotal}
	var epoch model.SystemConfig
	if err := tx.WithContext(ctx).Where("key = ?", model.ConfigKeyServingBillingEpoch).Limit(1).Find(&epoch).Error; err != nil {
		return 0, err
	}
	if epoch.Value != "" {
		activated, err := time.Parse(time.RFC3339Nano, epoch.Value)
		if err != nil {
			return 0, err
		}
		if job.LastSettledAt == nil || job.LastSettledAt.Before(activated) {
			job.LastSettledAt = &activated
		}
	}
	freeMinutes := getSystemIntWithTx(ctx, tx, model.ConfigKeyBillingJobFreeMinutes, defaultBillingJobFreeMinutes)
	duration, until, ok := computeSettlementWindow(job, usage.ObservedUntil, freeMinutes)
	if !ok {
		return 0, nil
	}
	total, cost := calcSettlementCharge(job, prices, duration)
	if _, _, _, err := deductSettlementCost(tx, job, cost); err != nil {
		return 0, err
	}
	err := tx.Model(usage).Updates(map[string]any{"last_settled_at": until, "billed_points_total": total, "updated_at": time.Now()}).Error
	return cost, err
}

// ServingBillingSummary is scoped by immutable deployment identity and account.
type ServingBillingSummary struct {
	BilledPoints  float64    `json:"billedPoints"`
	LastSettledAt *time.Time `json:"lastSettledAt"`
	MeteredPods   int64      `json:"meteredPods"`
}

func (s *BillingService) ServingBilling(
	ctx context.Context, deploymentID string, userID, accountID uint,
) (ServingBillingSummary, error) {
	var aggregate struct {
		Total       int64
		MeteredPods int64
	}
	scope := s.q.User.WithContext(ctx).UnderlyingDB().Session(&gorm.Session{NewDB: true, Context: ctx}).
		Model(&model.ServingUsage{}).
		Where("deployment_id = ? AND user_id = ? AND account_id = ?", deploymentID, userID, accountID)
	if err := scope.Session(&gorm.Session{}).
		Select("COALESCE(SUM(billed_points_total), 0) AS total, COUNT(*) AS metered_pods").Scan(&aggregate).Error; err != nil {
		return ServingBillingSummary{}, err
	}
	var latest model.ServingUsage
	err := scope.Session(&gorm.Session{}).Select("last_settled_at").
		Where("last_settled_at IS NOT NULL").Order("last_settled_at DESC").Limit(1).Find(&latest).Error
	return ServingBillingSummary{BilledPoints: ToDisplayPoints(aggregate.Total),
		LastSettledAt: latest.LastSettledAt, MeteredPods: aggregate.MeteredPods}, err
}
