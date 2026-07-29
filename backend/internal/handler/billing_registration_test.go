// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"strconv"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
)

func TestCreateUserIssuesDefaultAccountBalanceWithZeroPeriod(t *testing.T) {
	db := newBillingHandlerTestDB(t, "create_user_initial_billing")
	zeroPeriod := 0
	issueAmount := int64(25) * service.BillingPointScale
	account := model.Account{
		Model:                     gorm.Model{ID: model.DefaultAccountID},
		Name:                      "default",
		Nickname:                  "Default",
		Space:                     "default",
		BillingIssueAmount:        &issueAmount,
		BillingIssuePeriodMinutes: &zeroPeriod,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	createBillingSystemConfigs(t, db, 10080)

	billingService := service.NewBillingService(query.Use(db))
	mgr := NewAuthMgr(&RegisterConfig{BillingService: billingService}).(*AuthMgr)
	user, err := mgr.createUser(t.Context(), "new-user", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	var userAccount model.UserAccount
	if err := db.Where("user_id = ? AND account_id = ?", user.ID, model.DefaultAccountID).
		First(&userAccount).Error; err != nil {
		t.Fatal(err)
	}
	if userAccount.PeriodFreeBalance != issueAmount {
		t.Fatalf("period free balance = %d, want %d", userAccount.PeriodFreeBalance, issueAmount)
	}
}

func newBillingHandlerTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Account{},
		&model.UserAccount{},
		&model.SystemConfig{},
	); err != nil {
		t.Fatal(err)
	}
	query.SetDefault(db)
	return db
}

func createBillingSystemConfigs(t *testing.T, db *gorm.DB, defaultPeriod int) {
	t.Helper()
	configs := []model.SystemConfig{
		{Key: model.ConfigKeyEnableBillingFeature, Value: "true"},
		{Key: model.ConfigKeyBillingDefaultIssuePeriodMinute, Value: strconv.Itoa(defaultPeriod)},
		{Key: model.ConfigKeyBillingAccountIssueAmountOverrideEnabled, Value: "true"},
		{Key: model.ConfigKeyBillingAccountIssuePeriodOverrideEnabled, Value: "true"},
	}
	if err := db.Create(&configs).Error; err != nil {
		t.Fatal(err)
	}
}
