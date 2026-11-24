package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

func TestUserFieldsAdminUpdatePreservesConcurrentLogin(t *testing.T) {
	mgr, db := newFirstLoginHarness(t, false, false)
	profile := model.UserAttribute{
		Nickname: "Original name",
		Identity: ptr.To("PhD student"),
		Email:    ptr.To("student@example.invalid"),
	}
	user, err := mgr.createUser(t.Context(), "profile-concurrency-test", nil, &profile)
	if err != nil {
		t.Fatal(err)
	}
	latestLogin := time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	const concurrentBalance int64 = 500
	concurrentChange := simulateConcurrentUserChanges(t, db, user.ID, latestLogin, concurrentBalance)
	previous := config.GetConfig().UserFields
	config.GetConfig().UserFields = map[string]config.UserFieldConfig{"nickname": {}}
	t.Cleanup(func() { config.GetConfig().UserFields = previous })
	profile.Nickname = "Edited name"
	body, err := json.Marshal(map[string]string{"nickname": profile.Nickname})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(func(c *gin.Context) { util.SetJWTContext(c, util.JWTMessage{RolePlatform: model.RoleAdmin}); c.Next() })
	router.PUT("/users/:name/attributes", (&UserMgr{}).AdminUpdateUserAttributes)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("PUT", "/users/"+user.Name+"/attributes", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	requireAuthResponse(t, recorder, 200, 0)
	var stored model.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !*concurrentChange || stored.LastLoginAt == nil || !stored.LastLoginAt.Equal(latestLogin) ||
		stored.Role != model.RoleAdmin || stored.ExtraBalance != concurrentBalance {
		t.Fatal("profile update overwrote concurrent login time, role, or balance")
	}
	requireSavedUserProfile(t, &stored, user, &profile)
}

func simulateConcurrentUserChanges(t *testing.T, db *gorm.DB, userID uint, latestLogin time.Time, concurrentBalance int64) *bool {
	t.Helper()
	concurrentChange := false
	// Reproduce independent login/role/balance changes immediately before the
	// profile update is written; it must update only the intended columns.
	const callback = "test:concurrent_user_changes"
	if err := db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "users" || concurrentChange {
			return
		}
		concurrentChange = true
		update := tx.Session(&gorm.Session{NewDB: true}).Model(&model.User{}).
			Where("id = ?", userID).UpdateColumns(map[string]any{
			"last_login_at": latestLogin,
			"role":          model.RoleAdmin,
			"extra_balance": concurrentBalance,
		})
		if update.Error != nil {
			t.Error(update.Error)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Callback().Update().Remove(callback); err != nil {
			t.Error(err)
		}
	})
	return &concurrentChange
}

func requireSavedUserProfile(t *testing.T, stored, user *model.User, profile *model.UserAttribute) {
	t.Helper()
	attributes := stored.Attributes.Data()
	if stored.Nickname != profile.Nickname || attributes.Nickname != profile.Nickname ||
		attributes.Identity == nil || *attributes.Identity != *profile.Identity ||
		attributes.Email == nil || *attributes.Email != *profile.Email {
		t.Fatal("edited profile or retained profile attributes were not saved")
	}
	if !stored.CreatedAt.Equal(user.CreatedAt) {
		t.Fatal("editing the profile changed account creation time")
	}
}
