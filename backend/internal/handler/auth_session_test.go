package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/middleware"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

func authRequest(t *testing.T, mgr *AuthMgr, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	mgr.RegisterPublic(router.Group("/api/auth"))
	protected := router.Group("/api/v1/auth", middleware.AuthProtected(mgr.userBanService))
	mgr.RegisterProtected(protected)
	protected.GET("/probe", func(c *gin.Context) { resputil.Success(c, util.GetToken(c)) })
	method := http.MethodPost
	if path == "check" || path == "probe" {
		method = http.MethodGet
	}
	prefix := "/api/auth/"
	if path == "probe" || path == "switch" {
		prefix = "/api/v1/auth/"
	}
	req := httptest.NewRequest(method, prefix+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}
func requireAuthResponse(t *testing.T, result *httptest.ResponseRecorder, status int, code bizerr.BizCode) {
	t.Helper()
	var response resputil.Response[json.RawMessage]
	if err := json.Unmarshal(result.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if result.Code != status || response.Code != code {
		t.Fatalf("HTTP %d code %d: %s; want %d/%d", result.Code, response.Code, result.Body.String(), status, code)
	}
	if status >= 400 && string(response.Data) != authTestNullData {
		t.Fatal("failed authentication returned session data")
	}
}
func firstSession(t *testing.T, mgr *AuthMgr) LoginResp {
	t.Helper()
	result := requestFirstLDAPLogin(t, mgr, firstLoginPassword)
	requireAuthResponse(t, result, 200, 0)
	var response resputil.Response[LoginResp]
	if err := json.Unmarshal(result.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response.Data
}
func TestAuthenticationSessionRevocationAndCurrentPermissions(t *testing.T) {
	for _, state := range []string{"pending", authTestDisabled, "banned", "deleted", "no project", authTestDatabaseUnavailable, "roles changed"} {
		t.Run(state, func(t *testing.T) {
			mgr, db := newFirstLoginHarness(t, false, false)
			session := firstSession(t, mgr)
			status, code := 200, bizerr.BizCode(0)
			user := db.Model(&model.User{}).Where("id = ?", session.User.ID)
			var err error
			switch state {
			case "pending":
				err = user.Update("status", model.StatusPending).Error
				status, code = 401, bizerr.Auth.AccountNotActivated
			case authTestDisabled:
				err = user.Update("status", model.StatusInactive).Error
				status, code = 403, bizerr.Forbidden.AccountDisabled
			case "banned":
				err = user.Updates(map[string]any{"banned_timestamp": time.Now().Add(time.Hour), "ban_restrictions": datatypes.NewJSONType(model.UserBanRestrictions{PlatformAccess: true})}).Error
				status, code = 403, bizerr.Forbidden.UserBanned
			case "deleted":
				err = db.Delete(&model.User{}, session.User.ID).Error
				status, code = 401, bizerr.Auth.TokenInvalid
			case "no project":
				err = db.Where("user_id = ?", session.User.ID).Delete(&model.UserAccount{}).Error
				status, code = 403, bizerr.Forbidden.NoProjectAccess
			case authTestDatabaseUnavailable:
				conn, _ := db.DB()
				err = conn.Close()
				status, code = 500, bizerr.Internal.DatabaseError
			case "roles changed":
				err = user.Update("role", model.RoleGuest).Error
			}
			if err != nil {
				t.Fatal(err)
			}
			requireAuthResponse(t, authRequest(t, mgr, "check", "", session.AccessToken), status, code)
			requireAuthResponse(t, authRequest(t, mgr, "probe", "", session.AccessToken), status, code)
			refreshed := authRequest(t, mgr, "refresh", `{"refreshToken":"`+session.RefreshToken+`"}`, "")
			requireAuthResponse(t, refreshed, status, code)
			if state == "roles changed" {
				var response resputil.Response[RefreshResp]
				_ = json.Unmarshal(refreshed.Body.Bytes(), &response)
				claims, err := mgr.tokenMgr.CheckToken(response.Data.AccessToken)
				if err != nil || claims.RolePlatform != model.RoleGuest {
					t.Fatal("refresh retained a revoked role")
				}
			}
		})
	}
}
func TestAuthenticationSignupPoliciesAndRollback(t *testing.T) {
	for _, test := range []string{"success", authTestDisabled, "duplicate", "bad name", "long password", "missing default project"} {
		t.Run(test, func(t *testing.T) {
			mgr, db := newFirstLoginHarness(t, false, false)
			config.GetConfig().Auth.Normal.AllowRegister = true
			name, password := "normal-user", "valid-password"
			status, code := 200, bizerr.BizCode(0)
			switch test {
			case authTestDisabled:
				config.GetConfig().Auth.Normal.AllowRegister = false
				status, code = 403, bizerr.Forbidden.RegistrationDisabled
			case "duplicate":
				name = firstLoginUsername
				firstSession(t, mgr)
				status, code = 409, bizerr.Conflict.ResourceAlreadyExists
			case "bad name":
				name = "../escape"
				status, code = 400, bizerr.BadRequest.ParameterError
			case "long password":
				password = strings.Repeat("界", 25)
				status, code = 400, bizerr.BadRequest.ParameterError
			case "missing default project":
				if err := db.Delete(&model.Account{}, model.DefaultAccountID).Error; err != nil {
					t.Fatal(err)
				}
				status, code = 500, bizerr.Internal.AccountInitializationFailed
			}
			body, _ := json.Marshal(SignupReq{Username: name, Password: password})
			requireAuthResponse(t, authRequest(t, mgr, "signup", string(body), ""), status, code)
			if test == "missing default project" {
				assertFirstLoginRowCounts(t, db, 0, 0)
			}
			if test == "success" {
				config.GetConfig().Auth.Normal.AllowLogin = true
				body, _ = json.Marshal(map[string]string{"auth": "normal", "username": name, "password": password})
				requireAuthResponse(t, authRequest(t, mgr, "login", string(body), ""), 200, 0)
				assertFirstLoginRowCounts(t, db, 1, 1)
			}
		})
	}
}
func TestAuthenticationLocalCredentialsAndProfile(t *testing.T) {
	mgr, db := newFirstLoginHarness(t, false, false)
	session := firstSession(t, mgr)
	config.GetConfig().Auth.Normal.AllowLogin = true
	body := `{"auth":"normal","username":"` + firstLoginUsername + `","password":"correct"}`
	requireAuthResponse(t, authRequest(t, mgr, "login", body, ""), 401, bizerr.Auth.PasswordLoginUnavailable)
	if err := db.Model(&model.User{}).Where("id = ?", session.User.ID).Update("password", "bad-hash").Error; err != nil {
		t.Fatal(err)
	}
	requireAuthResponse(t, authRequest(t, mgr, "login", body, ""), 500, bizerr.Internal.ServiceError)
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	if err := db.Model(&model.User{}).Where("id = ?", session.User.ID).Updates(map[string]any{"password": string(hash), "nickname": "Chosen name"}).Error; err != nil {
		t.Fatal(err)
	}
	result := authRequest(t, mgr, "login", body, "")
	requireAuthResponse(t, result, 200, 0)
	var response resputil.Response[LoginResp]
	_ = json.Unmarshal(result.Body.Bytes(), &response)
	if response.Data.User.Nickname != "Chosen name" {
		t.Fatal("local login overwrote nickname")
	}
	user := model.User{Model: gorm.Model{ID: 1}, Name: "test", Nickname: "chosen"}
	attributes := model.UserAttribute{UID: ptr.To("1001"), GID: ptr.To("1001")}
	mgr.syncUserAttributes(&user, &attributes, &model.UserAttribute{GID: ptr.To("2002")})
	if *attributes.GID != "2002" || *attributes.UID != "1001" {
		t.Fatal("independent group update lost")
	}
	config.GetConfig().Auth.Normal.AllowLogin = false
	requireAuthResponse(t, authRequest(t, mgr, "login", body, ""), 403, bizerr.Forbidden.LoginMethodDisabled)
	config.GetConfig().Auth.LDAP.Enable = false
	requireAuthResponse(t, requestFirstLDAPLogin(t, mgr, firstLoginPassword), 403, bizerr.Forbidden.LoginMethodDisabled)
}
func TestAuthenticationSessionInputAndPurpose(t *testing.T) {
	mgr, _ := newFirstLoginHarness(t, false, false)
	session := firstSession(t, mgr)
	requireAuthResponse(t, authRequest(t, mgr, "check", "", ""), 200, 0)
	requireAuthResponse(t, authRequest(t, mgr, "check", "", session.RefreshToken), 401, bizerr.Auth.TokenInvalid)
	requireAuthResponse(t, authRequest(t, mgr, "refresh", `{"refreshToken":"`+session.AccessToken+`"}`, ""), 401, bizerr.Auth.TokenInvalid)
	requireAuthResponse(t, authRequest(t, mgr, "refresh", `{}`, ""), 400, bizerr.BadRequest.InvalidRequest)
	requireAuthResponse(t, authRequest(t, mgr, "switch", `{"queue":"default"}`, session.AccessToken), 200, 0)
	requireAuthResponse(t, authRequest(t, mgr, "switch", `{"queue":"missing"}`, session.AccessToken), 403, bizerr.Forbidden.NoProjectAccess)
}

func TestAuthenticationFreshLoginSelectsAccessibleProjectAndSwitchRecovers(t *testing.T) {
	mgr, db := newFirstLoginHarness(t, false, false)
	session := firstSession(t, mgr)
	project := model.Account{Name: "second", Space: "second", Nickname: "Second"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	association := model.UserAccount{UserID: session.User.ID, AccountID: project.ID, Role: model.RoleUser, AccessMode: model.AccessModeNA}
	if err := db.Create(&association).Error; err != nil {
		t.Fatal(err)
	}
	// The latest association is disabled; fresh login should still select default.
	repeated := firstSession(t, mgr)
	if repeated.Context.Queue != authTestDefaultProject {
		t.Fatal("fresh login selected an inaccessible project")
	}
	requireAuthResponse(t, authRequest(t, mgr, "switch", `{"queue":"second"}`, session.AccessToken), 403, bizerr.Forbidden.NoProjectAccess)
	if err := db.Model(&association).Update("access_mode", model.AccessModeRO).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("user_id = ? AND account_id = ?", session.User.ID, model.DefaultAccountID).Delete(&model.UserAccount{}).Error; err != nil {
		t.Fatal(err)
	}
	// Switch intentionally works after the old project's membership is revoked.
	requireAuthResponse(t, authRequest(t, mgr, "switch", `{"queue":"second"}`, session.AccessToken), 200, 0)
	if err := db.Delete(&project).Error; err != nil {
		t.Fatal(err)
	}
	requireAuthResponse(t, requestFirstLDAPLogin(t, mgr, firstLoginPassword), 403, bizerr.Forbidden.NoProjectAccess)
}

// Local login must enforce the same platform restrictions without consulting a
// directory or creating an account as a side effect of a failed login.
func TestAuthenticationLocalLoginFailureMatrix(t *testing.T) {
	for _, state := range []string{"missing", "wrong password", "pending", authTestDisabled, "platform ban", "expired ban", "job-only ban", "no project", "no password", "corrupt hash", authTestDatabaseUnavailable, "profile write failure", "password over bcrypt limit"} {
		t.Run(state, func(t *testing.T) {
			mgr, db := newFirstLoginHarness(t, false, false)
			config.GetConfig().Auth.LDAP.Enable = false
			config.GetConfig().Auth.Normal.AllowLogin = true
			hash, err := bcrypt.GenerateFromPassword([]byte(strings.Repeat("x", 72)), bcrypt.MinCost)
			if err != nil {
				t.Fatal(err)
			}
			user, err := mgr.createUser(t.Context(), "local-test", nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err = db.Model(user).Update("password", string(hash)).Error; err != nil {
				t.Fatal(err)
			}
			username, password, status, code := configureLocalLoginFailure(t, db, user, state)
			body, _ := json.Marshal(map[string]string{"auth": "normal", "username": username, "password": password})
			result := authRequest(t, mgr, "login", string(body), "")
			requireAuthResponse(t, result, status, code)
			if strings.Contains(result.Body.String(), "private-profile-detail") {
				t.Fatal("profile failure exposed internal diagnostics")
			}
			if state != authTestDatabaseUnavailable {
				// Count all rows, including the intentionally soft-deleted association,
				// to detect unintended account creation or project recreation.
				assertFirstLoginRowCounts(t, db, 1, 1)
			}
		})
	}
}

func configureLocalLoginFailure(t *testing.T, db *gorm.DB, user *model.User, state string) (username, password string, status int, code bizerr.BizCode) {
	t.Helper()
	username, password = "local-test", strings.Repeat("x", 72)
	status, code = 200, bizerr.BizCode(0)

	switch state {
	case "missing":
		username = "missing-test"
		status, code = 401, bizerr.Auth.RegisterNotFound
	case "wrong password":
		password = "incorrect"
		status, code = 401, bizerr.Auth.InvalidCredentials
	case "password over bcrypt limit":
		password += "suffix"
		status, code = 400, bizerr.BadRequest.ParameterError
	}
	var err error
	switch state {
	case "pending":
		err = db.Model(user).Update("status", model.StatusPending).Error
		status, code = 401, bizerr.Auth.AccountNotActivated
	case authTestDisabled:
		err = db.Model(user).Update("status", model.StatusInactive).Error
		status, code = 403, bizerr.Forbidden.AccountDisabled
	case "platform ban", "expired ban", "job-only ban":
		err = configureLocalLoginBan(db, user, state)
		if state == "platform ban" {
			status, code = 403, bizerr.Forbidden.UserBanned
		}
	case "no project":
		err = db.Where("user_id = ?", user.ID).Delete(&model.UserAccount{}).Error
		status, code = 403, bizerr.Forbidden.NoProjectAccess
	case "no password":
		err = db.Model(user).Update("password", nil).Error
		status, code = 401, bizerr.Auth.PasswordLoginUnavailable
	case "corrupt hash":
		err = db.Model(user).Update("password", "invalid-hash").Error
		status, code = 500, bizerr.Internal.ServiceError
	case authTestDatabaseUnavailable:
		conn, _ := db.DB()
		err = conn.Close()
		status, code = 500, bizerr.Internal.DatabaseError
	case "profile write failure":
		// Older profiles can be missing their mirrored ID/name; a failed repair is
		// a database error, not a credential rejection.
		err = db.Callback().Update().Before("gorm:update").Register("test:profile_failure", func(tx *gorm.DB) {
			if tx.Statement.Table == "users" {
				_ = tx.AddError(authTestError("private-profile-detail"))
			}
		})
		status, code = 500, bizerr.Internal.DatabaseError
	}
	if err != nil {
		t.Fatal(err)
	}
	return username, password, status, code
}

func configureLocalLoginBan(db *gorm.DB, user *model.User, state string) error {
	until := time.Now().Add(time.Hour)
	restrictions := model.UserBanRestrictions{PlatformAccess: true}
	if state == "expired ban" {
		until = time.Now().Add(-time.Hour)
	}
	if state == "job-only ban" {
		restrictions = model.UserBanRestrictions{JobSubmission: true}
	}
	return db.Model(user).Updates(map[string]any{"banned_timestamp": until, "ban_restrictions": datatypes.NewJSONType(restrictions)}).Error
}
