package service

import (
	"testing"
	"time"

	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"

	"github.com/raids-lab/crater/dao/model"
)

const boolFalseString = "false"

func TestComputeSettlementDeductions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		periodFreeBalance int64
		extraBalance      int64
		jobCost           int64
		wantFreeDeduct    int64
		wantExtraDeduct   int64
		wantFreeDebt      int64
	}{
		{
			name:              "uses free balance first when sufficient",
			periodFreeBalance: 12,
			extraBalance:      8,
			jobCost:           7,
			wantFreeDeduct:    7,
			wantExtraDeduct:   0,
			wantFreeDebt:      0,
		},
		{
			name:              "consumes extra balance before creating debt",
			periodFreeBalance: 10,
			extraBalance:      5,
			jobCost:           20,
			wantFreeDeduct:    10,
			wantExtraDeduct:   5,
			wantFreeDebt:      5,
		},
		{
			name:              "skips negative free balance and spends extra",
			periodFreeBalance: -3,
			extraBalance:      6,
			jobCost:           4,
			wantFreeDeduct:    0,
			wantExtraDeduct:   4,
			wantFreeDebt:      0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			freeDeduct, extraDeduct, freeDebt := computeSettlementDeductions(
				tc.periodFreeBalance,
				tc.extraBalance,
				tc.jobCost,
			)

			if freeDeduct != tc.wantFreeDeduct {
				t.Fatalf("freeDeduct = %d, want %d", freeDeduct, tc.wantFreeDeduct)
			}
			if extraDeduct != tc.wantExtraDeduct {
				t.Fatalf("extraDeduct = %d, want %d", extraDeduct, tc.wantExtraDeduct)
			}
			if freeDebt != tc.wantFreeDebt {
				t.Fatalf("freeDebt = %d, want %d", freeDebt, tc.wantFreeDebt)
			}
		})
	}
}

func TestResolveUserIssueAmount(t *testing.T) {
	t.Parallel()

	override := int64(25)
	tests := []struct {
		name                  string
		baseAmount            int64
		userAccount           *model.UserAccount
		amountOverrideEnabled bool
		want                  int64
	}{
		{
			name:                  "uses override when switch is enabled",
			baseAmount:            10,
			userAccount:           &model.UserAccount{BillingIssueAmountOverride: &override},
			amountOverrideEnabled: true,
			want:                  25,
		},
		{
			name:                  "ignores override when switch is disabled",
			baseAmount:            10,
			userAccount:           &model.UserAccount{BillingIssueAmountOverride: &override},
			amountOverrideEnabled: false,
			want:                  10,
		},
		{
			name:                  "clamps negative result to zero",
			baseAmount:            -3,
			userAccount:           nil,
			amountOverrideEnabled: false,
			want:                  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveUserIssueAmount(tc.baseAmount, tc.userAccount, tc.amountOverrideEnabled)
			if got != tc.want {
				t.Fatalf("resolveUserIssueAmount() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestShouldIssueDueAccounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		featureEnabled bool
		active         bool
		want           bool
	}{
		{
			name:           "requires feature enabled and active",
			featureEnabled: true,
			active:         true,
			want:           true,
		},
		{
			name:           "skips when feature disabled",
			featureEnabled: false,
			active:         true,
			want:           false,
		},
		{
			name:           "skips when inactive",
			featureEnabled: true,
			active:         false,
			want:           false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldIssueDueAccounts(tc.featureEnabled, tc.active)
			if got != tc.want {
				t.Fatalf("shouldIssueDueAccounts() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestShouldBlockJobCreateForBalance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		periodFreeBalance int64
		extraBalance      int64
		want              bool
	}{
		{
			name:              "blocks when both balances are zero",
			periodFreeBalance: 0,
			extraBalance:      0,
			want:              true,
		},
		{
			name:              "blocks when free balance is negative and extra balance is zero",
			periodFreeBalance: -1,
			extraBalance:      0,
			want:              true,
		},
		{
			name:              "allows when free balance is positive",
			periodFreeBalance: 1,
			extraBalance:      0,
			want:              false,
		},
		{
			name:              "allows when extra balance is positive",
			periodFreeBalance: 0,
			extraBalance:      1,
			want:              false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldBlockJobCreateForBalance(tc.periodFreeBalance, tc.extraBalance)
			if got != tc.want {
				t.Fatalf("shouldBlockJobCreateForBalance() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCalcSettlementChargeUsesScheduleTypeBillingMultiplier(t *testing.T) {
	t.Parallel()

	normal := model.ScheduleTypeNormal
	backfill := model.ScheduleTypeBackfill
	resources := v1.ResourceList{
		v1.ResourceCPU: apiresource.MustParse("2"),
	}
	priceMap := map[string]int64{
		string(v1.ResourceCPU): 3 * BillingPointScale,
	}

	normalJob := &model.Job{
		ScheduleType:      &normal,
		Resources:         datatypes.NewJSONType(resources),
		BilledPointsTotal: BillingPointScale,
	}
	newTotalMicro, jobCost := calcSettlementCharge(normalJob, priceMap, 30*time.Minute)
	if jobCost != 3*BillingPointScale {
		t.Fatalf("normal jobCost = %d, want %d", jobCost, 3*BillingPointScale)
	}
	if newTotalMicro != 4*BillingPointScale {
		t.Fatalf("normal newTotalMicro = %d, want %d", newTotalMicro, 4*BillingPointScale)
	}

	backfillJob := &model.Job{
		ScheduleType:      &backfill,
		Resources:         datatypes.NewJSONType(resources),
		BilledPointsTotal: BillingPointScale,
	}
	newTotalMicro, jobCost = calcSettlementCharge(backfillJob, priceMap, 30*time.Minute)
	if jobCost != 0 {
		t.Fatalf("backfill jobCost = %d, want 0", jobCost)
	}
	if newTotalMicro != BillingPointScale {
		t.Fatalf("backfill newTotalMicro = %d, want %d", newTotalMicro, BillingPointScale)
	}
}

func TestDeductSettlementCostSkipsNonPositiveCost(t *testing.T) {
	t.Parallel()

	freeDeduct, extraDeduct, freeDebt, err := deductSettlementCost(nil, nil, 0)
	if err != nil {
		t.Fatalf("deductSettlementCost() err = %v, want nil", err)
	}
	if freeDeduct != 0 || extraDeduct != 0 || freeDebt != 0 {
		t.Fatalf("deductions = (%d, %d, %d), want (0, 0, 0)", freeDeduct, extraDeduct, freeDebt)
	}
}

func TestShouldSkipJobCreateBillingCheck(t *testing.T) {
	t.Parallel()

	backfill := model.ScheduleTypeBackfill
	normal := model.ScheduleTypeNormal
	if !shouldSkipJobCreateBillingCheck(&backfill) {
		t.Fatal("expected backfill jobs to skip billing create check")
	}
	if shouldSkipJobCreateBillingCheck(&normal) {
		t.Fatal("expected normal jobs to keep billing create check")
	}
	if shouldSkipJobCreateBillingCheck(nil) {
		t.Fatal("expected missing schedule type to keep billing create check")
	}
}

func TestNormalizeBillingStatusTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   billingStatusTargets
		want    billingStatusTargets
		suspend bool
	}{
		{
			name: "disabling feature forces active and running settlement off",
			input: billingStatusTargets{
				targetFeatureEnabled:           false,
				targetActive:                   true,
				targetRunningSettlementEnabled: true,
			},
			want: billingStatusTargets{
				targetFeatureEnabled:           false,
				targetActive:                   false,
				targetRunningSettlementEnabled: false,
			},
			suspend: true,
		},
		{
			name: "disabling active forces running settlement off",
			input: billingStatusTargets{
				targetFeatureEnabled:           true,
				targetActive:                   false,
				targetRunningSettlementEnabled: true,
			},
			want: billingStatusTargets{
				targetFeatureEnabled:           true,
				targetActive:                   false,
				targetRunningSettlementEnabled: false,
			},
			suspend: true,
		},
		{
			name: "disabled base loop cron forces running settlement off only",
			input: billingStatusTargets{
				targetFeatureEnabled:           true,
				targetActive:                   true,
				targetRunningSettlementEnabled: true,
				currentBaseLoopCronEnabled:     false,
			},
			want: billingStatusTargets{
				targetFeatureEnabled:           true,
				targetActive:                   true,
				targetRunningSettlementEnabled: false,
				currentBaseLoopCronEnabled:     false,
			},
			suspend: false,
		},
		{
			name: "feature and active on keep base loop eligible",
			input: billingStatusTargets{
				targetFeatureEnabled:           true,
				targetActive:                   true,
				targetRunningSettlementEnabled: true,
				currentBaseLoopCronEnabled:     true,
			},
			want: billingStatusTargets{
				targetFeatureEnabled:           true,
				targetActive:                   true,
				targetRunningSettlementEnabled: true,
				currentBaseLoopCronEnabled:     true,
			},
			suspend: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeBillingStatusTargets(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeBillingStatusTargets() = %#v, want %#v", got, tc.want)
			}
			if shouldSuspendBillingBaseLoopCron(got) != tc.suspend {
				t.Fatalf(
					"shouldSuspendBillingBaseLoopCron() = %v, want %v",
					shouldSuspendBillingBaseLoopCron(got),
					tc.suspend,
				)
			}
		})
	}
}

func TestBuildBillingStatusConfigUpdates(t *testing.T) {
	t.Parallel()

	req := BillingUpdate{FeatureEnabled: boolPtr(false)}
	targets := billingStatusTargets{
		currentActive:                   true,
		targetActive:                    false,
		currentRunningSettlementEnabled: true,
		targetRunningSettlementEnabled:  false,
		currentFeatureEnabled:           true,
		targetFeatureEnabled:            false,
	}

	updates := buildBillingStatusConfigUpdates(req, targets)

	if updates[model.ConfigKeyEnableBillingFeature] != boolFalseString {
		t.Fatalf("featureEnabled update = %q, want false", updates[model.ConfigKeyEnableBillingFeature])
	}
	if updates[model.ConfigKeyEnableBillingActive] != boolFalseString {
		t.Fatalf("active update = %q, want false", updates[model.ConfigKeyEnableBillingActive])
	}
	if updates[model.ConfigKeyEnableRunningSettlement] != boolFalseString {
		t.Fatalf(
			"runningSettlement update = %q, want false",
			updates[model.ConfigKeyEnableRunningSettlement],
		)
	}
}

func TestBuildBillingStatusConfigUpdatesWritesNormalizedFalseWhenRequestAskedTrue(t *testing.T) {
	t.Parallel()

	req := BillingUpdate{
		Active:                   boolPtr(true),
		RunningSettlementEnabled: boolPtr(true),
	}
	targets := billingStatusTargets{
		currentActive:                   false,
		targetActive:                    true,
		currentRunningSettlementEnabled: false,
		targetRunningSettlementEnabled:  true,
		currentFeatureEnabled:           true,
		targetFeatureEnabled:            true,
	}

	updates := buildBillingStatusConfigUpdates(req, targets)

	if updates[model.ConfigKeyEnableBillingActive] != "true" {
		t.Fatalf("active update = %q, want true", updates[model.ConfigKeyEnableBillingActive])
	}
	if updates[model.ConfigKeyEnableRunningSettlement] != "true" {
		t.Fatalf(
			"runningSettlement update = %q, want true",
			updates[model.ConfigKeyEnableRunningSettlement],
		)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
