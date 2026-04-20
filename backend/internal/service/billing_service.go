package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/cronjob"
	"github.com/raids-lab/crater/pkg/patrol"
)

const (
	defaultRunningSettlementIntervalMinutes = 5
	defaultBillingJobFreeMinutes            = 0
	defaultBillingIssueAmount               = 1000 * BillingPointScale
	defaultBillingIssuePeriodMinutes        = 43200
	minRunningSettlementIntervalMinutes     = 1
	billingCreateBlockedMsgPrefix           = "billing precheck blocked: account free balance (%d) and extra balance (%d) "
	billingCreateBlockedMsgSuffix           = "are both non-positive, please recharge or grant rewards first"
)

type BillingStatus struct {
	FeatureEnabled                    bool    `json:"featureEnabled"`
	Active                            bool    `json:"active"`
	RunningSettlementEnabled          bool    `json:"runningSettlementEnabled"`
	RunningSettlementIntervalMinutes  int     `json:"runningSettlementIntervalMinutes"`
	JobFreeMinutes                    int     `json:"jobFreeMinutes"`
	DefaultIssueAmount                float64 `json:"defaultIssueAmount"`
	DefaultIssuePeriodMinutes         int     `json:"defaultIssuePeriodMinutes"`
	AccountIssueAmountOverrideEnabled bool    `json:"accountIssueAmountOverrideEnabled"`
	AccountIssuePeriodOverrideEnabled bool    `json:"accountIssuePeriodOverrideEnabled"`
	BaseLoopCronStatus                string  `json:"baseLoopCronStatus"`
	BaseLoopCronEnabled               bool    `json:"baseLoopCronEnabled"`
}

type BillingUpdate struct {
	FeatureEnabled                    *bool  `json:"featureEnabled"`
	Active                            *bool  `json:"active"`
	RunningSettlementEnabled          *bool  `json:"runningSettlementEnabled"`
	RunningSettlementIntervalMinutes  *int   `json:"runningSettlementIntervalMinutes"`
	JobFreeMinutes                    *int   `json:"jobFreeMinutes"`
	DefaultIssueAmount                *int64 `json:"defaultIssueAmount"`
	DefaultIssuePeriodMinutes         *int   `json:"defaultIssuePeriodMinutes"`
	AccountIssueAmountOverrideEnabled *bool  `json:"accountIssueAmountOverrideEnabled"`
	AccountIssuePeriodOverrideEnabled *bool  `json:"accountIssuePeriodOverrideEnabled"`
}

type BillingResetAllResult struct {
	AccountsAffected     int       `json:"accountsAffected"`
	UserAccountsAffected int       `json:"userAccountsAffected"`
	IssuedAt             time.Time `json:"issuedAt"`
}

type billingStatusTargets struct {
	currentActive                   bool
	targetActive                    bool
	currentRunningSettlementEnabled bool
	targetRunningSettlementEnabled  bool
	currentFeatureEnabled           bool
	targetFeatureEnabled            bool
	currentBaseLoopCronEnabled      bool
}

type billingIssueConfig struct {
	defaultAmount         int64
	defaultPeriod         int
	amountOverrideEnabled bool
	periodOverrideEnabled bool
}

type BillingService struct {
	q                   *query.Query
	cronJobManager      *cronjob.CronJobManager
	lastTickMu          sync.Mutex
	lastRunningSettleAt time.Time
}

func NewBillingService(q *query.Query) *BillingService {
	return &BillingService{q: q}
}

func (s *BillingService) SetCronJobManager(cjm *cronjob.CronJobManager) {
	s.cronJobManager = cjm
}

func (s *BillingService) GetStatus(ctx context.Context) BillingStatus {
	cronStatus, cronEnabled := s.GetBaseLoopCronStatus(ctx)
	return BillingStatus{
		FeatureEnabled:                    s.IsFeatureEnabled(ctx),
		Active:                            s.IsActive(ctx),
		RunningSettlementEnabled:          s.IsRunningSettlementEnabled(ctx),
		RunningSettlementIntervalMinutes:  s.GetRunningSettlementIntervalMinutes(ctx),
		JobFreeMinutes:                    s.GetJobFreeMinutes(ctx),
		DefaultIssueAmount:                ToDisplayPoints(s.GetDefaultIssueAmount(ctx)),
		DefaultIssuePeriodMinutes:         s.GetDefaultIssuePeriodMinutes(ctx),
		AccountIssueAmountOverrideEnabled: s.IsAccountIssueAmountOverrideEnabled(ctx),
		AccountIssuePeriodOverrideEnabled: s.IsAccountIssuePeriodOverrideEnabled(ctx),
		BaseLoopCronStatus:                string(cronStatus),
		BaseLoopCronEnabled:               cronEnabled,
	}
}

func (s *BillingService) UpdateStatus(ctx context.Context, req BillingUpdate) error {
	if err := validateBillingUpdate(req); err != nil {
		return err
	}

	targets, err := s.resolveStatusTargets(ctx, req)
	if err != nil {
		return err
	}
	shouldIssueOnActivation := !targets.currentActive && targets.targetActive
	shouldPreSettleBeforeConfigUpdate := s.shouldPreSettleBeforeStatusUpdate(
		ctx,
		req,
		targets.currentFeatureEnabled,
		targets.currentActive,
		targets.targetFeatureEnabled,
		targets.targetActive,
	)
	settleAt := time.Now()
	updates := buildBillingStatusConfigUpdates(req, targets)
	err = query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.applyStatusUpdateTx(
			ctx,
			tx,
			req,
			targets.currentFeatureEnabled,
			targets.targetFeatureEnabled,
			shouldPreSettleBeforeConfigUpdate,
			shouldIssueOnActivation,
			settleAt,
			updates,
		)
	})
	if err != nil {
		return err
	}
	return s.syncBillingCronState(ctx, shouldSuspendBillingBaseLoopCron(targets))
}

func (s *BillingService) resolveStatusTargets(ctx context.Context, req BillingUpdate) (billingStatusTargets, error) {
	targets := billingStatusTargets{
		currentActive:                   s.IsActive(ctx),
		currentRunningSettlementEnabled: s.IsRunningSettlementEnabled(ctx),
	}
	if s.cronJobManager != nil {
		_, targets.currentBaseLoopCronEnabled = s.GetBaseLoopCronStatus(ctx)
	}
	targets.targetActive = targets.currentActive
	if req.Active != nil {
		targets.targetActive = *req.Active
	}
	targets.targetRunningSettlementEnabled = targets.currentRunningSettlementEnabled
	if req.RunningSettlementEnabled != nil {
		targets.targetRunningSettlementEnabled = *req.RunningSettlementEnabled
	}

	targets.currentFeatureEnabled, targets.targetFeatureEnabled = s.resolveBillingFeatureState(ctx, req)
	targets = normalizeBillingStatusTargets(targets)
	if targets.targetActive {
		if err := s.ValidateActivationReady(ctx); err != nil {
			return billingStatusTargets{}, err
		}
	}

	return targets, nil
}

func (s *BillingService) applyStatusUpdateTx(
	ctx context.Context,
	tx *gorm.DB,
	req BillingUpdate,
	currentFeatureEnabled bool,
	targetFeatureEnabled bool,
	shouldPreSettleBeforeConfigUpdate bool,
	shouldIssueOnActivation bool,
	settleAt time.Time,
	updates map[string]string,
) error {
	if shouldPreSettleBeforeConfigUpdate {
		if _, err := s.settleAllRunningJobsAtTx(ctx, tx, settleAt); err != nil {
			return err
		}
	}
	if err := applySystemConfigUpdates(ctx, tx, updates); err != nil {
		return err
	}
	if err := s.bootstrapOnFirstFeatureEnable(ctx, tx, req, currentFeatureEnabled); err != nil {
		return err
	}
	if shouldIssueOnActivation {
		if _, _, err := s.issueNeverIssuedAccountsNowTx(ctx, tx, settleAt); err != nil {
			return err
		}
	}
	return ensureBillingBaseLoopCronConfig(ctx, tx, targetFeatureEnabled)
}

func (s *BillingService) HandleBaseLoopCronSuspended(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.lastTickMu.Lock()
	s.lastRunningSettleAt = time.Time{}
	s.lastTickMu.Unlock()

	return query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return applySystemConfigUpdates(ctx, tx, map[string]string{
			model.ConfigKeyEnableRunningSettlement: strconv.FormatBool(false),
		})
	})
}

func (s *BillingService) IsFeatureEnabled(ctx context.Context) bool {
	return s.readBoolConfig(ctx, model.ConfigKeyEnableBillingFeature)
}

func (s *BillingService) IsActive(ctx context.Context) bool {
	return s.readBoolConfig(ctx, model.ConfigKeyEnableBillingActive)
}

func (s *BillingService) IsRunningSettlementEnabled(ctx context.Context) bool {
	return s.readBoolConfig(ctx, model.ConfigKeyEnableRunningSettlement)
}

func (s *BillingService) GetRunningSettlementIntervalMinutes(ctx context.Context) int {
	v := s.readIntConfig(ctx, model.ConfigKeyRunningSettlementIntervalMinute, defaultRunningSettlementIntervalMinutes)
	if v < minRunningSettlementIntervalMinutes {
		return defaultRunningSettlementIntervalMinutes
	}
	return v
}

func (s *BillingService) GetJobFreeMinutes(ctx context.Context) int {
	v := s.readIntConfig(ctx, model.ConfigKeyBillingJobFreeMinutes, defaultBillingJobFreeMinutes)
	if v < 0 {
		return defaultBillingJobFreeMinutes
	}
	return v
}

func (s *BillingService) GetDefaultIssueAmount(ctx context.Context) int64 {
	v := s.readAmountConfig(ctx, model.ConfigKeyBillingDefaultIssueAmount, defaultBillingIssueAmount)
	if v < 0 {
		return defaultBillingIssueAmount
	}
	return v
}

func (s *BillingService) GetDefaultIssuePeriodMinutes(ctx context.Context) int {
	v := s.readIntConfig(ctx, model.ConfigKeyBillingDefaultIssuePeriodMinute, defaultBillingIssuePeriodMinutes)
	if v <= 0 {
		return defaultBillingIssuePeriodMinutes
	}
	return v
}

func (s *BillingService) IsAccountIssueAmountOverrideEnabled(ctx context.Context) bool {
	return s.readBoolConfig(ctx, model.ConfigKeyBillingAccountIssueAmountOverrideEnabled)
}

func (s *BillingService) IsAccountIssuePeriodOverrideEnabled(ctx context.Context) bool {
	return s.readBoolConfig(ctx, model.ConfigKeyBillingAccountIssuePeriodOverrideEnabled)
}

func (s *BillingService) ResolveEffectiveIssueConfigForAccount(
	ctx context.Context,
	account *model.Account,
) (issueAmount int64, periodMinutes int) {
	return resolveEffectiveIssueConfig(
		account,
		s.IsAccountIssueAmountOverrideEnabled(ctx),
		s.IsAccountIssuePeriodOverrideEnabled(ctx),
		s.GetDefaultIssueAmount(ctx),
		s.GetDefaultIssuePeriodMinutes(ctx),
	)
}

func (s *BillingService) ResolveEffectiveIssueConfigForUserAccount(
	ctx context.Context,
	userAccount *model.UserAccount,
	account *model.Account,
) (issueAmount int64, periodMinutes int) {
	issueAmount, periodMinutes = s.ResolveEffectiveIssueConfigForAccount(ctx, account)
	issueAmount = resolveUserIssueAmount(issueAmount, userAccount, s.IsAccountIssueAmountOverrideEnabled(ctx))
	if periodMinutes <= 0 {
		periodMinutes = 0
	}
	return issueAmount, periodMinutes
}

func (s *BillingService) ComputeNextIssueAt(lastIssuedAt *time.Time, periodMinutes int, now time.Time) *time.Time {
	if periodMinutes <= 0 {
		return nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	if lastIssuedAt == nil || lastIssuedAt.IsZero() {
		next := now
		return &next
	}
	next := lastIssuedAt.Add(time.Duration(periodMinutes) * time.Minute)
	if !next.After(now) {
		v := now
		return &v
	}
	return &next
}

func (s *BillingService) GetBaseLoopCronStatus(ctx context.Context) (model.CronJobConfigStatus, bool) {
	if s.cronJobManager == nil {
		return model.CronJobConfigStatusUnknown, false
	}
	configs, err := s.cronJobManager.GetCronjobConfigs(ctx, []string{patrol.TRIGGER_BILLING_BASE_LOOP_JOB}, nil, nil, nil)
	if err != nil || len(configs) == 0 || configs[0] == nil {
		return model.CronJobConfigStatusUnknown, false
	}
	status := configs[0].Status
	return status, status != model.CronJobConfigStatusSuspended
}

func (s *BillingService) ValidateActivationReady(ctx context.Context) error {
	r := query.Resource
	resources, err := r.WithContext(ctx).Where(r.DeletedAt.IsNull()).Find()
	if err != nil {
		return fmt.Errorf("billing activation blocked: failed to validate resource pricing: %w", err)
	}
	if len(resources) == 0 {
		return errors.New("billing activation blocked: no resources found")
	}
	for _, r := range resources {
		if r.UnitPrice < 0 {
			return fmt.Errorf("billing activation blocked: resource %q has invalid unitPrice=%d", r.ResourceName, r.UnitPrice)
		}
	}
	return nil
}

func (s *BillingService) OnJobCreateCheck(ctx context.Context, userID, accountID uint) error {
	if !s.IsFeatureEnabled(ctx) || !s.IsActive(ctx) {
		return nil
	}
	var (
		user model.User
		ua   model.UserAccount
	)
	u := query.User
	uaQuery := query.UserAccount
	foundUser, err := u.WithContext(ctx).Where(u.ID.Eq(userID), u.DeletedAt.IsNull()).First()
	if err != nil {
		return fmt.Errorf("billing precheck failed: user not found: %w", err)
	}
	foundUA, err := uaQuery.WithContext(ctx).
		Where(
			uaQuery.UserID.Eq(userID),
			uaQuery.AccountID.Eq(accountID),
			uaQuery.DeletedAt.IsNull(),
		).
		First()
	if err != nil {
		return fmt.Errorf("billing precheck failed: user-account relation not found: %w", err)
	}
	user = *foundUser
	ua = *foundUA
	if shouldBlockJobCreateForBalance(ua.PeriodFreeBalance, user.ExtraBalance) {
		return fmt.Errorf(
			billingCreateBlockedMsgPrefix+billingCreateBlockedMsgSuffix,
			ua.PeriodFreeBalance,
			user.ExtraBalance,
		)
	}
	return nil
}

func shouldBlockJobCreateForBalance(periodFreeBalance, extraBalance int64) bool {
	return periodFreeBalance <= 0 && extraBalance <= 0
}

func (s *BillingService) OnJobFinishedSettlement(ctx context.Context, job *model.Job) error {
	if job == nil || !s.IsFeatureEnabled(ctx) || !s.IsActive(ctx) {
		return nil
	}
	settleAt := job.CompletedTimestamp
	if settleAt.IsZero() {
		settleAt = time.Now()
	}
	_, err := s.settleOneJobByIdentity(ctx, job.ID, job.JobName, settleAt)
	return err
}

func (s *BillingService) RunRunningSettlementTick(ctx context.Context) error {
	_, err := s.runRunningSettlementTickWithInterval(ctx, time.Now())
	return err
}

func (s *BillingService) runRunningSettlementTickWithInterval(
	ctx context.Context,
	now time.Time,
) (settled int, err error) {
	if !s.IsFeatureEnabled(ctx) || !s.IsActive(ctx) || !s.IsRunningSettlementEnabled(ctx) {
		return 0, nil
	}

	interval := time.Duration(s.GetRunningSettlementIntervalMinutes(ctx)) * time.Minute
	s.lastTickMu.Lock()
	defer s.lastTickMu.Unlock()
	if !s.lastRunningSettleAt.IsZero() && now.Sub(s.lastRunningSettleAt) < interval {
		return 0, nil
	}

	settled, err = s.runRunningSettlementTickOnce(ctx, now)
	if err != nil {
		return 0, err
	}
	s.lastRunningSettleAt = now
	return settled, nil
}

func (s *BillingService) RunBaseLoopOnce(ctx context.Context) (any, error) {
	featureEnabled := s.IsFeatureEnabled(ctx)
	if !featureEnabled {
		return map[string]any{
			"featureEnabled": false,
			"issuedAccounts": 0,
			"settledJobs":    0,
		}, nil
	}

	active := s.IsActive(ctx)
	issued := 0
	var err error
	if shouldIssueDueAccounts(featureEnabled, active) {
		issued, err = s.IssueDueAccounts(ctx)
		if err != nil {
			return nil, err
		}
	}

	settled := 0
	if active && s.IsRunningSettlementEnabled(ctx) {
		settled, err = s.runRunningSettlementTickWithInterval(ctx, time.Now())
		if err != nil {
			return nil, err
		}
	}

	return map[string]any{
		"featureEnabled": true,
		"issuedAccounts": issued,
		"settledJobs":    settled,
	}, nil
}

// ReconcileOnce keeps backward compatibility with old naming.
func (s *BillingService) ReconcileOnce(ctx context.Context) (any, error) {
	return s.RunBaseLoopOnce(ctx)
}

func (s *BillingService) IssueDueAccounts(ctx context.Context) (int, error) {
	if !shouldIssueDueAccounts(s.IsFeatureEnabled(ctx), s.IsActive(ctx)) {
		return 0, nil
	}

	now := time.Now()
	issued := 0
	err := query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		issueConfig := loadBillingIssueConfigTx(ctx, tx)
		accountQuery := query.Use(tx).Account
		accounts, err := accountQuery.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(accountQuery.DeletedAt.IsNull()).
			Find()
		if err != nil {
			return err
		}
		for i := range accounts {
			acc := accounts[i]
			issueAmount, periodMinutes := issueConfig.resolveForAccount(acc)
			if !isIssueConfigValid(issueAmount, periodMinutes) {
				continue
			}
			period := time.Duration(periodMinutes) * time.Minute
			if acc.BillingLastIssuedAt != nil && now.Sub(*acc.BillingLastIssuedAt) < period {
				continue
			}
			if _, err := s.issueAccountNowTx(ctx, tx, acc.ID, issueAmount, now); err != nil {
				return err
			}
			issued++
		}
		return nil
	})
	return issued, err
}

func (s *BillingService) IssueAccountNow(ctx context.Context, accountID uint) error {
	now := time.Now()
	return query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		issueConfig := loadBillingIssueConfigTx(ctx, tx)
		accountQuery := query.Use(tx).Account
		account, err := accountQuery.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(accountQuery.ID.Eq(accountID), accountQuery.DeletedAt.IsNull()).
			First()
		if err != nil {
			return err
		}
		issueAmount, periodMinutes := issueConfig.resolveForAccount(account)
		if !isIssueConfigValid(issueAmount, periodMinutes) {
			return nil
		}
		_, err = s.issueAccountNowTx(ctx, tx, accountID, issueAmount, now)
		return err
	})
}

func (s *BillingService) IssueUserAccountNow(ctx context.Context, userID, accountID uint) error {
	return query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.issueUserAccountNowTx(ctx, tx, userID, accountID)
	})
}

func (s *BillingService) bootstrapIssueConfigOnFeatureEnableTx(ctx context.Context, tx *gorm.DB) error {
	issueConfig := loadBillingIssueConfigTx(ctx, tx)
	var accounts []model.Account
	accountQuery := query.Use(tx).Account
	foundAccounts, err := accountQuery.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(accountQuery.DeletedAt.IsNull()).
		Find()
	if err != nil {
		return err
	}
	accounts = make([]model.Account, 0, len(foundAccounts))
	for i := range foundAccounts {
		accounts = append(accounts, *foundAccounts[i])
	}
	for i := range accounts {
		acc := &accounts[i]
		updates := map[string]any{}
		if acc.BillingIssueAmount == nil {
			v := issueConfig.defaultAmount
			updates["billing_issue_amount"] = v
			acc.BillingIssueAmount = &v
		}
		if acc.BillingIssuePeriodMinutes == nil {
			v := issueConfig.defaultPeriod
			updates["billing_issue_period_minutes"] = v
			acc.BillingIssuePeriodMinutes = &v
		}
		if len(updates) > 0 {
			if _, err := accountQuery.WithContext(ctx).
				Where(accountQuery.ID.Eq(acc.ID), accountQuery.DeletedAt.IsNull()).
				Updates(updates); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *BillingService) issueAccountNowTx(
	ctx context.Context,
	tx *gorm.DB,
	accountID uint,
	issueAmount int64,
	now time.Time,
) (int, error) {
	if getSystemBoolWithTx(ctx, tx, model.ConfigKeyEnableBillingActive) {
		if _, err := settleRunningJobsForAccountAtTx(ctx, tx, accountID, now); err != nil {
			return 0, err
		}
	}

	txQuery := query.Use(tx)
	uaQuery := txQuery.UserAccount
	amountOverrideEnabled := getSystemBoolWithTx(ctx, tx, model.ConfigKeyBillingAccountIssueAmountOverrideEnabled)
	foundUserAccounts, err := uaQuery.WithContext(tx.Statement.Context).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(uaQuery.AccountID.Eq(accountID), uaQuery.DeletedAt.IsNull()).
		Find()
	if err != nil {
		return 0, err
	}
	userAccounts := make([]model.UserAccount, 0, len(foundUserAccounts))
	for i := range foundUserAccounts {
		userAccounts = append(userAccounts, *foundUserAccounts[i])
	}

	for i := range userAccounts {
		ua := &userAccounts[i]
		issueAmountForUser := resolveUserIssueAmount(issueAmount, ua, amountOverrideEnabled)
		if _, err := uaQuery.WithContext(tx.Statement.Context).
			Where(uaQuery.UserID.Eq(ua.UserID), uaQuery.AccountID.Eq(ua.AccountID), uaQuery.DeletedAt.IsNull()).
			Update(uaQuery.PeriodFreeBalance, issueAmountForUser); err != nil {
			return 0, err
		}
	}

	accountQuery := txQuery.Account
	if _, err := accountQuery.WithContext(tx.Statement.Context).
		Where(accountQuery.ID.Eq(accountID), accountQuery.DeletedAt.IsNull()).
		Update(accountQuery.BillingLastIssuedAt, now); err != nil {
		return 0, err
	}
	return len(userAccounts), nil
}

func (s *BillingService) issueUserAccountNowTx(ctx context.Context, tx *gorm.DB, userID, accountID uint) error {
	txQuery := query.Use(tx)
	accountQuery := txQuery.Account
	account, err := accountQuery.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(accountQuery.ID.Eq(accountID), accountQuery.DeletedAt.IsNull()).
		First()
	if err != nil {
		return err
	}
	issueConfig := loadBillingIssueConfigTx(ctx, tx)
	issueAmount, periodMinutes := issueConfig.resolveForAccount(account)
	if !isIssueConfigValid(issueAmount, periodMinutes) {
		return nil
	}

	uaQuery := txQuery.UserAccount
	ua, err := uaQuery.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(uaQuery.UserID.Eq(userID), uaQuery.AccountID.Eq(accountID), uaQuery.DeletedAt.IsNull()).
		First()
	if err != nil {
		return err
	}

	issueAmount = resolveUserIssueAmount(issueAmount, ua, issueConfig.amountOverrideEnabled)
	_, err = uaQuery.WithContext(ctx).
		Where(uaQuery.UserID.Eq(userID), uaQuery.AccountID.Eq(accountID), uaQuery.DeletedAt.IsNull()).
		Update(uaQuery.PeriodFreeBalance, issueAmount)
	return err
}

func (s *BillingService) runRunningSettlementTickOnce(ctx context.Context, settleAt time.Time) (int, error) {
	settledJobs := 0
	err := query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		settledJobs, err = s.settleAllRunningJobsAtTx(ctx, tx, settleAt)
		return err
	})
	if err == nil {
		klog.Infof("[Billing] running settlement done, settledJobs=%d", settledJobs)
	}
	return settledJobs, err
}

func (s *BillingService) settleAllRunningJobsAtTx(ctx context.Context, tx *gorm.DB, settleAt time.Time) (int, error) {
	txQuery := query.Use(tx)
	priceMap, err := loadUnitPriceMapTx(ctx, tx)
	if err != nil {
		return 0, err
	}
	jobQuery := txQuery.Job
	foundJobs, err := jobQuery.WithContext(ctx).
		Where(jobQuery.DeletedAt.IsNull(), jobQuery.Status.Eq(string(batch.Running))).
		Find()
	if err != nil {
		return 0, err
	}
	runningJobs := make([]model.Job, 0, len(foundJobs))
	for i := range foundJobs {
		runningJobs = append(runningJobs, *foundJobs[i])
	}
	settledJobs := 0
	for i := range runningJobs {
		charged, err := settleOneJobTx(ctx, tx, runningJobs[i].ID, settleAt, priceMap)
		if err != nil {
			return 0, err
		}
		if charged > 0 {
			settledJobs++
		}
	}
	return settledJobs, nil
}

func (s *BillingService) UpdateResourceUnitPrice(ctx context.Context, resourceID uint, unitPrice int64) error {
	settleAt := time.Now()
	return query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txQuery := query.Use(tx)
		resourceQuery := txQuery.Resource
		resource, err := resourceQuery.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(resourceQuery.ID.Eq(resourceID), resourceQuery.DeletedAt.IsNull()).
			First()
		if err != nil {
			return err
		}
		if resource.UnitPrice == unitPrice {
			return nil
		}
		if s.IsFeatureEnabled(ctx) && s.IsActive(ctx) {
			if _, err := s.settleAllRunningJobsAtTx(ctx, tx, settleAt); err != nil {
				return err
			}
		}
		_, err = resourceQuery.WithContext(ctx).
			Where(resourceQuery.ID.Eq(resourceID), resourceQuery.DeletedAt.IsNull()).
			Update(resourceQuery.UnitPrice, unitPrice)
		return err
	})
}

func (s *BillingService) settleOneJobByIdentity(ctx context.Context, jobID uint, jobName string, settleAt time.Time) (int64, error) {
	var charged int64
	err := query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txQuery := query.Use(tx)
		priceMap, err := loadUnitPriceMapTx(ctx, tx)
		if err != nil {
			return err
		}
		var targetID uint
		if jobID > 0 {
			targetID = jobID
		} else {
			jobQuery := txQuery.Job
			job, err := jobQuery.WithContext(ctx).
				Where(jobQuery.JobName.Eq(jobName), jobQuery.DeletedAt.IsNull()).
				First()
			if err != nil {
				return err
			}
			targetID = job.ID
		}
		charged, err = settleOneJobTx(ctx, tx, targetID, settleAt, priceMap)
		return err
	})
	return charged, err
}

func settleOneJobTx(ctx context.Context, tx *gorm.DB, jobID uint, settleAt time.Time, priceMap map[string]int64) (int64, error) {
	job, err := loadJobForSettlementTx(tx, jobID)
	if err != nil {
		return 0, err
	}
	if job == nil || job.RunningTimestamp.IsZero() {
		return 0, nil
	}
	jobFreeMinutes := getSystemIntWithTx(ctx, tx, model.ConfigKeyBillingJobFreeMinutes, defaultBillingJobFreeMinutes)
	if jobFreeMinutes < 0 {
		jobFreeMinutes = defaultBillingJobFreeMinutes
	}
	billableDuration, billedUntil, ok := computeSettlementWindow(job, settleAt, jobFreeMinutes)
	if !ok {
		return 0, nil
	}
	newTotalMicro, jobCost := calcSettlementCharge(job, priceMap, billableDuration)
	freeDeduct, extraDeduct, freeDebt, err := deductSettlementCost(tx, job, jobCost)
	if err != nil {
		return 0, err
	}
	if err := persistJobSettlementState(tx, job.ID, billedUntil, newTotalMicro); err != nil {
		return 0, err
	}

	klog.Infof("[Billing] settled job=%s window=%s charged=%d free=%d extra=%d debt=%d",
		job.JobName, billableDuration, jobCost, freeDeduct, extraDeduct, freeDebt)
	return jobCost, nil
}

func validateBillingUpdate(req BillingUpdate) error {
	if req.RunningSettlementIntervalMinutes != nil && *req.RunningSettlementIntervalMinutes < minRunningSettlementIntervalMinutes {
		return fmt.Errorf("runningSettlementIntervalMinutes must be >= %d", minRunningSettlementIntervalMinutes)
	}
	if req.JobFreeMinutes != nil && *req.JobFreeMinutes < 0 {
		return errors.New("jobFreeMinutes must be >= 0")
	}
	if req.DefaultIssueAmount != nil && *req.DefaultIssueAmount < 0 {
		return errors.New("defaultIssueAmount must be >= 0")
	}
	if req.DefaultIssuePeriodMinutes != nil && *req.DefaultIssuePeriodMinutes <= 0 {
		return errors.New("defaultIssuePeriodMinutes must be > 0")
	}
	return nil
}

func (s *BillingService) resolveBillingFeatureState(
	ctx context.Context,
	req BillingUpdate,
) (currentFeatureEnabled, targetFeatureEnabled bool) {
	currentFeatureEnabled = s.IsFeatureEnabled(ctx)
	targetFeatureEnabled = currentFeatureEnabled
	if req.FeatureEnabled != nil {
		targetFeatureEnabled = *req.FeatureEnabled
	}
	return currentFeatureEnabled, targetFeatureEnabled
}

func buildBillingStatusConfigUpdates(req BillingUpdate, targets billingStatusTargets) map[string]string {
	updates := make(map[string]string)
	if req.FeatureEnabled != nil {
		updates[model.ConfigKeyEnableBillingFeature] = strconv.FormatBool(targets.targetFeatureEnabled)
	}
	if req.Active != nil || targets.currentActive != targets.targetActive {
		updates[model.ConfigKeyEnableBillingActive] = strconv.FormatBool(targets.targetActive)
	}
	if req.RunningSettlementEnabled != nil ||
		targets.currentRunningSettlementEnabled != targets.targetRunningSettlementEnabled {
		updates[model.ConfigKeyEnableRunningSettlement] = strconv.FormatBool(targets.targetRunningSettlementEnabled)
	}
	if req.RunningSettlementIntervalMinutes != nil {
		updates[model.ConfigKeyRunningSettlementIntervalMinute] = strconv.Itoa(*req.RunningSettlementIntervalMinutes)
	}
	if req.JobFreeMinutes != nil {
		updates[model.ConfigKeyBillingJobFreeMinutes] = strconv.Itoa(*req.JobFreeMinutes)
	}
	if req.DefaultIssueAmount != nil {
		updates[model.ConfigKeyBillingDefaultIssueAmount] = FormatBillingAmountConfigValue(*req.DefaultIssueAmount)
	}
	if req.DefaultIssuePeriodMinutes != nil {
		updates[model.ConfigKeyBillingDefaultIssuePeriodMinute] = strconv.Itoa(*req.DefaultIssuePeriodMinutes)
	}
	if req.AccountIssueAmountOverrideEnabled != nil {
		updates[model.ConfigKeyBillingAccountIssueAmountOverrideEnabled] = strconv.FormatBool(*req.AccountIssueAmountOverrideEnabled)
	}
	if req.AccountIssuePeriodOverrideEnabled != nil {
		updates[model.ConfigKeyBillingAccountIssuePeriodOverrideEnabled] = strconv.FormatBool(*req.AccountIssuePeriodOverrideEnabled)
	}
	return updates
}

func normalizeBillingStatusTargets(targets billingStatusTargets) billingStatusTargets {
	if !targets.targetFeatureEnabled {
		targets.targetActive = false
	}
	if !targets.currentBaseLoopCronEnabled {
		targets.targetRunningSettlementEnabled = false
	}
	if !targets.targetActive {
		targets.targetRunningSettlementEnabled = false
	}
	return targets
}

func shouldSuspendBillingBaseLoopCron(targets billingStatusTargets) bool {
	return !targets.targetFeatureEnabled || !targets.targetActive
}

func applySystemConfigUpdates(ctx context.Context, tx *gorm.DB, updates map[string]string) error {
	for k, v := range updates {
		if err := upsertSystemConfigTx(ctx, tx, k, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *BillingService) bootstrapOnFirstFeatureEnable(
	ctx context.Context,
	tx *gorm.DB,
	req BillingUpdate,
	currentFeatureEnabled bool,
) error {
	if req.FeatureEnabled != nil && *req.FeatureEnabled && !currentFeatureEnabled {
		return s.bootstrapIssueConfigOnFeatureEnableTx(ctx, tx)
	}
	return nil
}

func (s *BillingService) shouldPreSettleBeforeStatusUpdate(
	ctx context.Context,
	req BillingUpdate,
	currentFeatureEnabled bool,
	currentActive bool,
	targetFeatureEnabled bool,
	targetActive bool,
) bool {
	if !currentFeatureEnabled || !currentActive {
		return false
	}
	if !targetFeatureEnabled || !targetActive {
		return true
	}
	if req.JobFreeMinutes != nil && *req.JobFreeMinutes != s.GetJobFreeMinutes(ctx) {
		return true
	}
	return false
}

func (s *BillingService) issueAllConfiguredAccountsNowTx(
	ctx context.Context,
	tx *gorm.DB,
	now time.Time,
) (accountsAffected, userAccountsAffected int, err error) {
	return s.issueConfiguredAccountsNowTx(ctx, tx, now, false)
}

func (s *BillingService) issueNeverIssuedAccountsNowTx(
	ctx context.Context,
	tx *gorm.DB,
	now time.Time,
) (accountsAffected, userAccountsAffected int, err error) {
	return s.issueConfiguredAccountsNowTx(ctx, tx, now, true)
}

func (s *BillingService) issueConfiguredAccountsNowTx(
	ctx context.Context,
	tx *gorm.DB,
	now time.Time,
	onlyNeverIssued bool,
) (accountsAffected, userAccountsAffected int, err error) {
	issueConfig := loadBillingIssueConfigTx(ctx, tx)
	accountQuery := query.Use(tx).Account
	foundAccounts, err := accountQuery.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(accountQuery.DeletedAt.IsNull()).
		Find()
	if err != nil {
		return 0, 0, err
	}
	accounts := make([]model.Account, 0, len(foundAccounts))
	for i := range foundAccounts {
		accounts = append(accounts, *foundAccounts[i])
	}

	for i := range accounts {
		account := &accounts[i]
		if onlyNeverIssued && account.BillingLastIssuedAt != nil && !account.BillingLastIssuedAt.IsZero() {
			continue
		}
		issueAmount, periodMinutes := issueConfig.resolveForAccount(account)
		if !isIssueConfigValid(issueAmount, periodMinutes) {
			continue
		}
		issuedUserAccounts, err := s.issueAccountNowTx(ctx, tx, account.ID, issueAmount, now)
		if err != nil {
			return 0, 0, err
		}
		accountsAffected++
		userAccountsAffected += issuedUserAccounts
	}
	return accountsAffected, userAccountsAffected, nil
}

func (s *BillingService) ResetAllPeriodFreeBalances(ctx context.Context) (*BillingResetAllResult, error) {
	now := time.Now()
	result := &BillingResetAllResult{IssuedAt: now}
	err := query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		accountsAffected, userAccountsAffected, err := s.issueAllConfiguredAccountsNowTx(ctx, tx, now)
		if err != nil {
			return err
		}
		result.AccountsAffected = accountsAffected
		result.UserAccountsAffected = userAccountsAffected
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ensureBillingBaseLoopCronConfig(ctx context.Context, tx *gorm.DB, targetFeatureEnabled bool) error {
	jobName := patrol.TRIGGER_BILLING_BASE_LOOP_JOB
	cronQuery := query.Use(tx).CronJobConfig
	_, cronErr := cronQuery.WithContext(ctx).Where(cronQuery.Name.Eq(jobName)).First()
	if errors.Is(cronErr, gorm.ErrRecordNotFound) {
		if targetFeatureEnabled {
			newJob := &model.CronJobConfig{
				Name:    jobName,
				Type:    model.CronJobTypePatrolFunc,
				Spec:    "*/1 * * * *",
				Status:  model.CronJobConfigStatusSuspended,
				Config:  datatypes.JSON("{}"),
				EntryID: -1,
			}
			return tx.WithContext(ctx).Create(newJob).Error
		}
		return nil
	}
	return cronErr
}

func loadJobForSettlementTx(tx *gorm.DB, jobID uint) (*model.Job, error) {
	jobQuery := query.Use(tx).Job
	return jobQuery.WithContext(tx.Statement.Context).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(jobQuery.ID.Eq(jobID), jobQuery.DeletedAt.IsNull()).
		First()
}

func computeSettlementWindow(
	job *model.Job,
	settleAt time.Time,
	freeMinutes int,
) (billableDuration time.Duration, billedUntil time.Time, ok bool) {
	startAt := job.RunningTimestamp
	if startAt.IsZero() || !settleAt.After(startAt) {
		return 0, time.Time{}, false
	}

	lastSettledAt := startAt
	if job.LastSettledAt != nil && job.LastSettledAt.After(startAt) {
		lastSettledAt = *job.LastSettledAt
	}
	if !settleAt.After(lastSettledAt) {
		return 0, time.Time{}, false
	}

	elapsed := settleAt.Sub(lastSettledAt)
	freeBudget := time.Duration(freeMinutes) * time.Minute
	if freeBudget < 0 {
		freeBudget = 0
	}

	consumedFree := time.Duration(0)
	if job.LastSettledAt != nil && job.LastSettledAt.After(job.RunningTimestamp) {
		consumedFree = job.LastSettledAt.Sub(job.RunningTimestamp)
		if consumedFree > freeBudget {
			consumedFree = freeBudget
		}
	}
	remainingFree := freeBudget - consumedFree
	if remainingFree < 0 {
		remainingFree = 0
	}

	if elapsed <= remainingFree {
		return 0, settleAt, true
	}
	return elapsed - remainingFree, settleAt, true
}

func calcSettlementCharge(job *model.Job, priceMap map[string]int64, billableDuration time.Duration) (newTotalMicro, jobCost int64) {
	windowCostMicro := calcJobCostMicroPoints(job.Resources.Data(), priceMap, billableDuration)
	if windowCostMicro < 0 {
		windowCostMicro = 0
	}
	newTotalMicro = job.BilledPointsTotal + windowCostMicro
	if newTotalMicro < job.BilledPointsTotal {
		newTotalMicro = job.BilledPointsTotal
	}
	jobCost = newTotalMicro - job.BilledPointsTotal
	return newTotalMicro, jobCost
}

func deductSettlementCost(tx *gorm.DB, job *model.Job, jobCost int64) (freeDeduct, extraDeduct, freeDebt int64, err error) {
	txQuery := query.Use(tx)
	uaQuery := txQuery.UserAccount
	ua, err := uaQuery.WithContext(tx.Statement.Context).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(uaQuery.UserID.Eq(job.UserID), uaQuery.AccountID.Eq(job.AccountID), uaQuery.DeletedAt.IsNull()).
		First()
	if err != nil {
		return 0, 0, 0, err
	}
	userQuery := txQuery.User
	user, err := userQuery.WithContext(tx.Statement.Context).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(userQuery.ID.Eq(job.UserID), userQuery.DeletedAt.IsNull()).
		First()
	if err != nil {
		return 0, 0, 0, err
	}

	freeDeduct, extraDeduct, freeDebt = computeSettlementDeductions(ua.PeriodFreeBalance, user.ExtraBalance, jobCost)
	newPeriodFreeBalance := ua.PeriodFreeBalance - freeDeduct - freeDebt
	newExtraBalance := user.ExtraBalance - extraDeduct

	if _, err := uaQuery.WithContext(tx.Statement.Context).
		Where(uaQuery.UserID.Eq(ua.UserID), uaQuery.AccountID.Eq(ua.AccountID), uaQuery.DeletedAt.IsNull()).
		Update(uaQuery.PeriodFreeBalance, newPeriodFreeBalance); err != nil {
		return 0, 0, 0, err
	}
	if extraDeduct > 0 {
		if _, err := userQuery.WithContext(tx.Statement.Context).
			Where(userQuery.ID.Eq(user.ID), userQuery.DeletedAt.IsNull()).
			Update(userQuery.ExtraBalance, newExtraBalance); err != nil {
			return 0, 0, 0, err
		}
	}
	return freeDeduct, extraDeduct, freeDebt, nil
}

func computeSettlementDeductions(periodFreeBalance, extraBalance, jobCost int64) (freeDeduct, extraDeduct, freeDebt int64) {
	if jobCost <= 0 {
		return 0, 0, 0
	}

	freeAvailable := periodFreeBalance
	if freeAvailable < 0 {
		freeAvailable = 0
	}
	extraAvailable := extraBalance
	if extraAvailable < 0 {
		extraAvailable = 0
	}

	remaining := jobCost
	freeDeduct = minInt64(remaining, freeAvailable)
	remaining -= freeDeduct
	extraDeduct = minInt64(remaining, extraAvailable)
	remaining -= extraDeduct
	freeDebt = remaining

	return freeDeduct, extraDeduct, freeDebt
}

func settleRunningJobsForAccountAtTx(ctx context.Context, tx *gorm.DB, accountID uint, settleAt time.Time) (int, error) {
	txQuery := query.Use(tx)
	priceMap, err := loadUnitPriceMapTx(ctx, tx)
	if err != nil {
		return 0, err
	}

	jobQuery := txQuery.Job
	foundJobs, err := jobQuery.WithContext(ctx).
		Where(
			jobQuery.DeletedAt.IsNull(),
			jobQuery.AccountID.Eq(accountID),
			jobQuery.Status.Eq(string(batch.Running)),
		).
		Find()
	if err != nil {
		return 0, err
	}

	settledJobs := 0
	for i := range foundJobs {
		charged, settleErr := settleOneJobTx(ctx, tx, foundJobs[i].ID, settleAt, priceMap)
		if settleErr != nil {
			return 0, settleErr
		}
		if charged > 0 {
			settledJobs++
		}
	}

	return settledJobs, nil
}

func persistJobSettlementState(tx *gorm.DB, jobID uint, billedUntil time.Time, newTotalMicro int64) error {
	jobQuery := query.Use(tx).Job
	_, err := jobQuery.WithContext(tx.Statement.Context).
		Where(jobQuery.ID.Eq(jobID), jobQuery.DeletedAt.IsNull()).
		Updates(map[string]any{
			"last_settled_at":     billedUntil,
			"billed_points_total": newTotalMicro,
			"updated_at":          time.Now(),
		})
	return err
}

func loadUnitPriceMapTx(_ context.Context, tx *gorm.DB) (map[string]int64, error) {
	resourceQuery := query.Use(tx).Resource
	foundResources, err := resourceQuery.WithContext(tx.Statement.Context).
		Where(resourceQuery.DeletedAt.IsNull()).
		Find()
	if err != nil {
		return nil, err
	}
	resources := make([]model.Resource, 0, len(foundResources))
	for i := range foundResources {
		resources = append(resources, *foundResources[i])
	}
	priceMap := make(map[string]int64, len(resources))
	for i := range resources {
		priceMap[resources[i].ResourceName] = resources[i].UnitPrice
	}
	return priceMap, nil
}

func calcJobCostMicroPoints(resources v1.ResourceList, priceMap map[string]int64, billableDuration time.Duration) int64 {
	if billableDuration <= 0 || len(resources) == 0 || len(priceMap) == 0 {
		return 0
	}
	total := new(big.Rat)
	durationRat := big.NewRat(billableDuration.Nanoseconds(), int64(time.Hour))
	for name, qty := range resources {
		price := priceMap[string(name)]
		if price <= 0 {
			continue
		}
		amountRat, err := quantityToRat(qty)
		if err != nil || amountRat.Sign() <= 0 {
			continue
		}
		line := new(big.Rat).Mul(amountRat, big.NewRat(price, 1))
		line.Mul(line, durationRat)
		total.Add(total, line)
	}
	if total.Sign() <= 0 {
		return 0
	}
	return ratFloorToInt64(total)
}

func resolveEffectiveIssueConfig(
	account *model.Account,
	amountOverrideEnabled bool,
	periodOverrideEnabled bool,
	defaultAmount int64,
	defaultPeriod int,
) (issueAmount int64, periodMinutes int) {
	issueAmount = defaultAmount
	periodMinutes = defaultPeriod
	if amountOverrideEnabled && account != nil && account.BillingIssueAmount != nil {
		issueAmount = *account.BillingIssueAmount
	}
	if periodOverrideEnabled && account != nil && account.BillingIssuePeriodMinutes != nil {
		periodMinutes = *account.BillingIssuePeriodMinutes
	}
	if issueAmount < 0 {
		issueAmount = 0
	}
	if periodMinutes <= 0 {
		periodMinutes = 0
	}
	return issueAmount, periodMinutes
}

func resolveUserIssueAmount(issueAmount int64, userAccount *model.UserAccount, amountOverrideEnabled bool) int64 {
	if amountOverrideEnabled && userAccount != nil && userAccount.BillingIssueAmountOverride != nil {
		issueAmount = *userAccount.BillingIssueAmountOverride
	}
	if issueAmount < 0 {
		return 0
	}
	return issueAmount
}

func (cfg billingIssueConfig) resolveForAccount(account *model.Account) (issueAmount int64, periodMinutes int) {
	return resolveEffectiveIssueConfig(
		account,
		cfg.amountOverrideEnabled,
		cfg.periodOverrideEnabled,
		cfg.defaultAmount,
		cfg.defaultPeriod,
	)
}

func shouldIssueDueAccounts(featureEnabled, active bool) bool {
	return featureEnabled && active
}

func isIssueConfigValid(issueAmount int64, periodMinutes int) bool {
	if periodMinutes <= 0 {
		return false
	}
	return issueAmount > 0
}

func upsertSystemConfigTx(ctx context.Context, tx *gorm.DB, key, value string) error {
	cfgQuery := query.Use(tx).SystemConfig
	if _, err := cfgQuery.WithContext(ctx).
		Where(cfgQuery.Key.Eq(key)).
		Update(cfgQuery.Value, value); err != nil {
		return err
	}
	cnt, err := cfgQuery.WithContext(ctx).Where(cfgQuery.Key.Eq(key)).Count()
	if err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	return tx.WithContext(ctx).Create(&model.SystemConfig{Key: key, Value: value}).Error
}

func getSystemIntWithTx(ctx context.Context, tx *gorm.DB, key string, def int) int {
	cfgQuery := query.Use(tx).SystemConfig
	cfg, err := cfgQuery.WithContext(ctx).Where(cfgQuery.Key.Eq(key)).First()
	if err != nil {
		return def
	}
	v, err := strconv.Atoi(cfg.Value)
	if err != nil {
		return def
	}
	return v
}

func getDefaultIssueAmountWithTx(ctx context.Context, tx *gorm.DB) int64 {
	cfgQuery := query.Use(tx).SystemConfig
	cfg, err := cfgQuery.WithContext(ctx).Where(cfgQuery.Key.Eq(model.ConfigKeyBillingDefaultIssueAmount)).First()
	if err != nil {
		return defaultBillingIssueAmount
	}
	return ParseBillingAmountConfigValue(cfg.Value, defaultBillingIssueAmount)
}

func getSystemBoolWithTx(ctx context.Context, tx *gorm.DB, key string) bool {
	cfgQuery := query.Use(tx).SystemConfig
	cfg, err := cfgQuery.WithContext(ctx).Where(cfgQuery.Key.Eq(key)).First()
	if err != nil {
		return false
	}
	v, err := strconv.ParseBool(cfg.Value)
	if err != nil {
		return false
	}
	return v
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func loadBillingIssueConfigTx(ctx context.Context, tx *gorm.DB) billingIssueConfig {
	defaultAmount := getDefaultIssueAmountWithTx(ctx, tx)
	if defaultAmount < 0 {
		defaultAmount = defaultBillingIssueAmount
	}
	defaultPeriod := getSystemIntWithTx(ctx, tx, model.ConfigKeyBillingDefaultIssuePeriodMinute, defaultBillingIssuePeriodMinutes)
	if defaultPeriod <= 0 {
		defaultPeriod = defaultBillingIssuePeriodMinutes
	}
	return billingIssueConfig{
		defaultAmount:         defaultAmount,
		defaultPeriod:         defaultPeriod,
		amountOverrideEnabled: getSystemBoolWithTx(ctx, tx, model.ConfigKeyBillingAccountIssueAmountOverrideEnabled),
		periodOverrideEnabled: getSystemBoolWithTx(ctx, tx, model.ConfigKeyBillingAccountIssuePeriodOverrideEnabled),
	}
}

func (s *BillingService) syncBillingCronState(ctx context.Context, shouldSuspend bool) error {
	if s.cronJobManager == nil {
		return nil
	}
	if !shouldSuspend {
		return nil
	}
	configs, err := s.cronJobManager.GetCronjobConfigs(ctx, []string{patrol.TRIGGER_BILLING_BASE_LOOP_JOB}, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("sync billing cron state failed: %w", err)
	}
	if len(configs) == 0 {
		return nil
	}
	if configs[0] == nil || configs[0].Status == model.CronJobConfigStatusSuspended {
		return nil
	}
	desiredStatus := model.CronJobConfigStatusSuspended
	if err := s.cronJobManager.UpdateJobConfig(nil, patrol.TRIGGER_BILLING_BASE_LOOP_JOB, nil, nil, &desiredStatus, nil); err != nil {
		return fmt.Errorf("update billing cron state failed: %w", err)
	}
	return nil
}

func (s *BillingService) IsUserFacingEnabled(ctx context.Context) bool {
	return s.IsFeatureEnabled(ctx) && s.IsActive(ctx)
}

func (s *BillingService) readBoolConfig(ctx context.Context, key string) bool {
	val := s.readConfigValue(ctx, key)
	if val == "" {
		return false
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return false
	}
	return b
}

func (s *BillingService) readIntConfig(ctx context.Context, key string, def int) int {
	val := s.readConfigValue(ctx, key)
	if val == "" {
		return def
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return i
}

func (s *BillingService) readAmountConfig(ctx context.Context, key string, def int64) int64 {
	val := s.readConfigValue(ctx, key)
	if val == "" {
		return def
	}
	return ParseBillingAmountConfigValue(val, def)
}

func (s *BillingService) readConfigValue(ctx context.Context, key string) string {
	sc := s.q.SystemConfig
	cfg, err := sc.WithContext(ctx).Where(sc.Key.Eq(key)).First()
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			klog.Errorf("[Billing] read config %s failed: %v", key, err)
		}
		return ""
	}
	return cfg.Value
}
