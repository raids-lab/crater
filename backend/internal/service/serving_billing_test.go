package service

import (
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func servingBillingFixture(t *testing.T) (*BillingService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err = db.AutoMigrate(&model.ServingUsage{}, &model.User{}, &model.Account{}, &model.UserAccount{}, &model.SystemConfig{}, &model.Resource{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{
		&model.User{Model: gorm.Model{ID: 1}, ExtraBalance: 10 * BillingPointScale},
		&model.UserAccount{UserID: 1, AccountID: 2, PeriodFreeBalance: 100 * BillingPointScale},
		&model.SystemConfig{Key: model.ConfigKeyEnableBillingFeature, Value: "true"},
		&model.SystemConfig{Key: model.ConfigKeyEnableBillingActive, Value: "true"},
		&model.SystemConfig{Key: model.ConfigKeyBillingJobFreeMinutes, Value: "0"},
		&model.Resource{ResourceName: "cpu", UnitPrice: 60 * BillingPointScale},
	} {
		if err = db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	return NewBillingService(query.Use(db)), db
}

func TestServingBillingReplayAndFinalSettlement(t *testing.T) {
	svc, db := servingBillingFixture(t)
	start := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	observed := &model.ServingUsage{PodUID: "pod-1", DeploymentID: "deployment", Namespace: "jobs", ServiceName: "qwen", UserID: 1, AccountID: 2,
		Resources: datatypes.NewJSONType(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")}), StartedAt: start, ObservedUntil: start.Add(30 * time.Minute)}
	if err := svc.ObserveServingUsage(t.Context(), observed); err != nil {
		t.Fatal(err)
	}
	settle := func() {
		t.Helper()
		if err := db.Transaction(func(tx *gorm.DB) error {
			_, err := settleServingUsagesTx(t.Context(), tx, 0, map[string]int64{"cpu": 60 * BillingPointScale})
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	settle()
	settle()
	end := start.Add(45 * time.Minute)
	observed.ObservedUntil = end
	observed.EndedAt = &end
	if err := svc.ObserveServingUsage(t.Context(), observed); err != nil {
		t.Fatal(err)
	}
	if err := svc.ObserveServingUsage(t.Context(), observed); err != nil {
		t.Fatal(err)
	}
	var usage model.ServingUsage
	if err := db.First(&usage, "pod_uid = ?", observed.PodUID).Error; err != nil {
		t.Fatal(err)
	}
	if usage.BilledPointsTotal != 90*BillingPointScale {
		t.Fatalf("cost = %d", usage.BilledPointsTotal)
	}
	var account model.UserAccount
	if err := db.First(&account).Error; err != nil {
		t.Fatal(err)
	}
	if account.PeriodFreeBalance != 10*BillingPointScale {
		t.Fatalf("balance = %d", account.PeriodFreeBalance)
	}
	summary, err := svc.ServingBilling(t.Context(), "deployment", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if summary.BilledPoints != 90 || summary.MeteredPods != 1 {
		t.Fatalf("summary = %+v", summary)
	}
	foreign, err := svc.ServingBilling(t.Context(), "deployment", 1, 3)
	if err != nil || foreign.BilledPoints != 0 {
		t.Fatalf("foreign billing %+v %v", foreign, err)
	}
}

func TestServingBillingDisabledPeriodDoesNotBecomeDebt(t *testing.T) {
	svc, db := servingBillingFixture(t)
	start := time.Now().UTC().Truncate(time.Second)
	usage := &model.ServingUsage{PodUID: "new-pod", DeploymentID: "deployment", UserID: 1, AccountID: 2,
		Resources: datatypes.NewJSONType(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")}), StartedAt: start, ObservedUntil: start.Add(time.Hour)}
	if err := db.Model(&model.SystemConfig{}).Where("key = ?", model.ConfigKeyEnableBillingActive).Update("value", "false").Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.ObserveServingUsage(t.Context(), usage); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.SystemConfig{}).Where("key = ?", model.ConfigKeyEnableBillingActive).Update("value", "true").Error; err != nil {
		t.Fatal(err)
	}
	end := start.Add(2 * time.Hour)
	usage.ObservedUntil = end
	usage.EndedAt = &end
	if err := svc.ObserveServingUsage(t.Context(), usage); err != nil {
		t.Fatal(err)
	}
	var saved model.ServingUsage
	if err := db.First(&saved, "pod_uid = ?", usage.PodUID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.BilledPointsTotal != 60*BillingPointScale {
		t.Fatalf("disabled hour was charged: %d", saved.BilledPointsTotal)
	}
}

func TestServingBillingActivationEpochAfterCollectorDowntime(t *testing.T) {
	svc, db := servingBillingFixture(t)
	start := time.Now().UTC().Truncate(time.Second)
	activated := start.Add(time.Hour)
	if err := db.Create(&model.SystemConfig{Key: model.ConfigKeyServingBillingEpoch, Value: activated.Format(time.RFC3339Nano)}).Error; err != nil {
		t.Fatal(err)
	}
	end := start.Add(2 * time.Hour)
	usage := &model.ServingUsage{PodUID: "epoch-pod", DeploymentID: "deployment", UserID: 1, AccountID: 2,
		Resources: datatypes.NewJSONType(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")}), StartedAt: start, ObservedUntil: end, EndedAt: &end}
	if err := svc.ObserveServingUsage(t.Context(), usage); err != nil {
		t.Fatal(err)
	}
	var saved model.ServingUsage
	if err := db.First(&saved, "pod_uid = ?", usage.PodUID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.BilledPointsTotal != 60*BillingPointScale {
		t.Fatalf("pre-activation interval charged: %d", saved.BilledPointsTotal)
	}
}

func TestServingBillingFailureRollsBackFinalCheckpoint(t *testing.T) {
	svc, db := servingBillingFixture(t)
	start := time.Now().UTC().Truncate(time.Second)
	end := start.Add(time.Hour)
	usage := &model.ServingUsage{PodUID: "rollback-pod", DeploymentID: "deployment", UserID: 999, AccountID: 2,
		Resources: datatypes.NewJSONType(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")}), StartedAt: start, ObservedUntil: end, EndedAt: &end}
	if err := svc.ObserveServingUsage(t.Context(), usage); err == nil {
		t.Fatal("settlement with missing payer succeeded")
	}
	var count int64
	if err := db.Model(&model.ServingUsage{}).Where("pod_uid = ?", usage.PodUID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed settlement committed its final checkpoint")
	}
}
