package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

func TestUserFieldsAudiencePrivacy(t *testing.T) {
	mgr, db := newFirstLoginHarness(t, false, false)
	profile := model.UserAttribute{Nickname: "canonical", Group: ptr.To("NCG"), Phone: ptr.To("private"), Identity: ptr.To("intern")}
	user, err := mgr.createUser(t.Context(), "privacy-test", nil, &profile)
	if err != nil {
		t.Fatal(err)
	}
	config.GetConfig().UserFields = map[string]config.UserFieldConfig{"group": {ShowToAll: true}, "phone": {}, "createdAt": {}}
	for _, tc := range []struct {
		name    string
		role    model.Role
		id      uint
		private bool
	}{
		{"admin", model.RoleAdmin, 999, true}, {"self", model.RoleUser, user.ID, true}, {"other", model.RoleUser, 999, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				util.SetJWTContext(c, util.JWTMessage{UserID: tc.id, RolePlatform: tc.role})
				c.Next()
			})
			router.GET("/users/:name", (&UserMgr{}).GetUser)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users/"+user.Name, http.NoBody))
			var body struct{ Data map[string]any }
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Data["group"] != "NCG" {
				t.Fatal("public field omitted")
			}
			_, phone := body.Data["phone"]
			_, created := body.Data["createdAt"]
			_, identity := body.Data["identity"]
			if phone != tc.private || created != tc.private || identity {
				t.Fatalf("unexpected disclosure: %v", body.Data)
			}
		})
	}
	var stored model.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Attributes.Data().Identity == nil {
		t.Fatal("filtering modified stored data")
	}
}

func TestUserFieldsAtomicPatchAndSelfEdit(t *testing.T) {
	mgr, db := newFirstLoginHarness(t, false, false)
	profile := model.UserAttribute{Nickname: "original", Phone: ptr.To("old phone"), Identity: ptr.To("old identity")}
	user, err := mgr.createUser(t.Context(), "patch-test", nil, &profile)
	if err != nil {
		t.Fatal(err)
	}
	// Reproduce two writers that both loaded the old profile, then save disjoint keys.
	stale := *user
	if err := service.PatchUserProfile(t.Context(), user.Name, map[string]any{"phone": "new phone"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := mgr.updateUserIfNeeded(t.Context(), &stale, &model.UserAttribute{Identity: ptr.To("new identity")}); err != nil {
		t.Fatal(err)
	}
	if err := db.First(user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if *user.Attributes.Data().Phone != "new phone" || *user.Attributes.Data().Identity != "new identity" {
		t.Fatal("LDAP overwrote unrelated admin field")
	}
	exerciseSelfProfileRequests(t, user)
	if err := db.First(user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if *user.Attributes.Data().Identity != "new identity" || *user.Attributes.Data().Phone != "new phone" {
		t.Fatal("avatar edit lost profile fields")
	}
	if err := service.PatchUserProfile(t.Context(), user.Name, map[string]any{"phone": nil}, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.First(user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.Attributes.Data().Phone != nil || user.Attributes.Data().Identity == nil {
		t.Fatal("clear or omission semantics broken")
	}
}

func TestUserFieldsNicknameIsCanonical(t *testing.T) {
	mgr, db := newFirstLoginHarness(t, false, false)
	user, err := mgr.createUser(t.Context(), "nickname-test", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(user).UpdateColumn("attributes", datatypes.NewJSONType(model.UserAttribute{})).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.PatchUserProfile(t.Context(), user.Name, map[string]any{"nickname": "new name"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := query.User.WithContext(t.Context()).UnderlyingDB().First(user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.Nickname != "new name" || user.Attributes.Data().Nickname != "" {
		t.Fatal("empty extensions should remain empty")
	}
}

func exerciseSelfProfileRequests(t *testing.T, user *model.User) {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) { util.SetJWTContext(c, util.JWTMessage{UserID: user.ID}); c.Next() })
	router.PUT("/attributes", (&ContextMgr{}).UpdateUserAttributes)
	for _, tc := range []struct {
		body   string
		status int
	}{{`{"avatar":"https://example.invalid/avatar"}`, 200}, {`{"identity":"forged"}`, 400}, {`{"email":"unverified@example.invalid"}`, 400}} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPut, "/attributes", strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		if response.Code != tc.status {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	}
}

func TestUserFieldsVerifiedEmailUpdate(t *testing.T) {
	mgr, db := newFirstLoginHarness(t, false, false)
	user, err := mgr.createUser(t.Context(), "email-test", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	config.GetConfig().UserFields = map[string]config.UserFieldConfig{"email": {}}
	router := gin.New()
	router.Use(func(c *gin.Context) { util.SetJWTContext(c, util.JWTMessage{UserID: user.ID}); c.Next() })
	router.POST("/email", (&ContextMgr{}).UpdateUserEmail)
	verifyCodeCache.Lock()
	verifyCodeCache.Codes[user.ID] = emailVerification{Code: "123456", Email: "verified@example.invalid", Expires: time.Now().Add(time.Minute)}
	verifyCodeCache.Unlock()
	send := func(body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/email", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		return response
	}
	if response := send(`{"email":"wrong@example.invalid","code":"123456"}`); response.Code == 200 {
		t.Fatal("code was accepted for another email")
	}
	if response := send(`{"email":"verified@example.invalid","code":"123456"}`); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if err := db.First(user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.LastEmailVerifiedAt == nil || user.Attributes.Data().Email == nil || *user.Attributes.Data().Email != "verified@example.invalid" {
		t.Fatal("verified email was not saved")
	}
	if response := send(`{"email":"verified@example.invalid","code":"123456"}`); response.Code == 200 {
		t.Fatal("verification code was reused")
	}
	config.GetConfig().UserFields = map[string]config.UserFieldConfig{}
	if response := send(`{"email":"verified@example.invalid","code":"123456"}`); response.Code == 200 {
		t.Fatal("disabled email accepted verification")
	}
}
