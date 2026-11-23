package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/config"
)

func TestAuthenticationLastLoginTime(t *testing.T) {
	mgr, db := newFirstLoginHarness(t, false, false)
	session := firstSession(t, mgr)
	var user model.User
	if err := db.First(&user, session.User.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.LastLoginAt == nil || user.LastLoginAt.Before(user.CreatedAt) {
		t.Fatal("first LDAP login did not record a login time")
	}
	profileUpdatedAt := user.UpdatedAt
	old := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	setHistoricalLoginTime(t, db, &user, old)
	requireAuthResponse(t, authRequest(t, mgr, "check", "", session.AccessToken), 200, 0)
	requireLoginTime(t, db, &user, profileUpdatedAt, old, false)
	requireAuthResponse(t, authRequest(t, mgr, "refresh", fmt.Sprintf(`{"refreshToken":%q}`, session.RefreshToken), ""), 200, 0)
	requireLoginTime(t, db, &user, profileUpdatedAt, old, false)
	requireAuthResponse(t, authRequest(t, mgr, "switch", `{"queue":"default"}`, session.AccessToken), 200, 0)
	requireLoginTime(t, db, &user, profileUpdatedAt, old, false)
	failed := requestFirstLDAPLogin(t, mgr, "incorrect-password")
	if failed.Code == 200 {
		t.Fatal("wrong password unexpectedly accepted")
	}
	requireLoginTime(t, db, &user, profileUpdatedAt, old, false)
	if err := db.Model(&user).UpdateColumn("status", model.StatusInactive).Error; err != nil {
		t.Fatal(err)
	}
	if result := requestFirstLDAPLogin(t, mgr, firstLoginPassword); result.Code == 200 {
		t.Fatal("disabled user signed in")
	}
	requireLoginTime(t, db, &user, profileUpdatedAt, old, false)
	if err := db.Model(&user).UpdateColumn("status", model.StatusActive).Error; err != nil {
		t.Fatal(err)
	}
	requireAuthResponse(t, requestFirstLDAPLogin(t, mgr, firstLoginPassword), 200, 0)
	requireLoginTime(t, db, &user, profileUpdatedAt, old, true)
	// The same shared successful-login path must record local logins as well.
	hash, err := bcrypt.GenerateFromPassword([]byte("local-test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).UpdateColumn("password", string(hash)).Error; err != nil {
		t.Fatal(err)
	}
	config.GetConfig().Auth.Normal.AllowLogin = true
	setHistoricalLoginTime(t, db, &user, old)
	requireAuthResponse(t, authRequest(t, mgr, "login", fmt.Sprintf(`{"auth":"normal","username":%q,"password":"local-test-password"}`, user.Name), ""), 200, 0)
	requireLoginTime(t, db, &user, profileUpdatedAt, old, true)
	requireListedUserTimes(t, mgr, &user)
	// Signup also establishes a session; it records a time without LDAP attributes.
	config.GetConfig().Auth.Normal.AllowRegister = true
	requireAuthResponse(t, authRequest(t, mgr, "signup", `{"userName":"local-new-user","password":"local-test-password"}`, ""), 200, 0)
	var local model.User
	if err := db.Where("name = ?", "local-new-user").First(&local).Error; err != nil {
		t.Fatal(err)
	}
	if local.LastLoginAt == nil || local.Password == nil || *local.Password == "" {
		t.Fatal("signup session missing login time")
	}
}

func setHistoricalLoginTime(t *testing.T, db *gorm.DB, user *model.User, old time.Time) {
	t.Helper()
	if err := db.Model(user).UpdateColumn("last_login_at", old).Error; err != nil {
		t.Fatal(err)
	}
}

func requireLoginTime(t *testing.T, db *gorm.DB, user *model.User, profileUpdatedAt, old time.Time, changed bool) {
	t.Helper()
	if err := db.First(user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !user.UpdatedAt.Equal(profileUpdatedAt) {
		t.Fatal("recording a login changed profile updated_at")
	}
	if user.LastLoginAt == nil || (user.LastLoginAt.Equal(old) == changed) {
		t.Fatalf("login time = %v; changed=%v", user.LastLoginAt, changed)
	}
}

func requireListedUserTimes(t *testing.T, mgr *AuthMgr, user *model.User) {
	t.Helper()
	// The admin list returns the actual creation time and current login time.
	listRecorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(listRecorder)
	ctx.Request = httptest.NewRequest("GET", "/v1/admin/users", http.NoBody)
	(&UserMgr{billingService: mgr.billingService}).ListUser(ctx)
	recorder := listRecorder.Body.Bytes()
	var list resputil.Response[[]UserResp]
	if err := json.Unmarshal(recorder, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 1 || !list.Data[0].CreatedAt.Equal(user.CreatedAt) || list.Data[0].LastLoginAt == nil || !list.Data[0].LastLoginAt.Equal(*user.LastLoginAt) {
		t.Fatalf("user list did not return account times: %s", recorder)
	}
}
