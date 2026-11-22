// Copyright 2026 The Crater Project Team, RAIDS-Lab
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ber "github.com/go-asn1-ber/asn1-ber"
	ldap "github.com/go-ldap/ldap/v3"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/config"
)

const firstLoginUsername = "ldap-first-login-test"
const firstLoginPassword = "synthetic-test-password"

// This tiny LDAP fixture implements the bind/search messages used by Login.
// It listens only on loopback and contains exclusively synthetic credentials.
func startFirstLoginLDAP(t *testing.T, entry *ldap.Entry) string {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			serveFirstLoginLDAP(t, conn, entry)
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		<-done
	})
	return "ldap://" + listener.Addr().String()
}

func serveFirstLoginLDAP(t *testing.T, conn net.Conn, entry *ldap.Entry) {
	t.Helper()
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	for {
		packet, err := ber.ReadPacket(conn)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Errorf("read LDAP test request: %v", err)
			}
			return
		}
		if len(packet.Children) != 2 {
			t.Error("unexpected LDAP request envelope")
			return
		}
		messageID := packet.Children[0].Value.(int64)
		request := packet.Children[1]
		switch request.Tag {
		case ldap.ApplicationUnbindRequest:
			return
		case ldap.ApplicationBindRequest:
			if !respondFirstLoginBind(t, conn, entry, messageID, request) {
				return
			}
		case ldap.ApplicationSearchRequest:
			response := firstLoginSearchResponse(entry)
			if !writeFirstLoginLDAP(t, conn, messageID, response) ||
				!writeFirstLoginLDAP(t, conn, messageID, firstLoginLDAPResult(ldap.ApplicationSearchResultDone, ldap.LDAPResultSuccess)) {
				return
			}
		default:
			t.Errorf("unexpected LDAP operation %d", request.Tag)
			return
		}
	}
}

func firstLoginLDAPResult(tag ber.Tag, code int64) *ber.Packet {
	result := ber.Encode(ber.ClassApplication, ber.TypeConstructed, tag, nil, "")
	result.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagEnumerated, code, ""))
	result.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", ""))
	result.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", ""))
	return result
}

func writeFirstLoginLDAP(t *testing.T, conn net.Conn, id int64, response *ber.Packet) bool {
	t.Helper()
	envelope := ber.NewSequence("")
	envelope.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, id, ""))
	envelope.AppendChild(response)
	if _, err := conn.Write(envelope.Bytes()); err != nil {
		t.Errorf("write LDAP test response: %v", err)
		return false
	}
	return true
}

func newFirstLoginHarness(t *testing.T, billing, missingSID bool) (*AuthMgr, *gorm.DB) {
	t.Helper()
	attributes := map[string][]string{
		authTestUsernameAttribute: {firstLoginUsername}, "cn": {"First LDAP User"},
		"mail": {"first-login@example.invalid"}, "primaryGroupID": {"513"},
	}
	if !missingSID {
		attributes["objectSid"] = []string{string([]byte{1, 1, 0, 0, 0, 0, 0, 5, 42, 0, 0, 0})}
	}
	address := startFirstLoginLDAP(t, ldap.NewEntry("cn=test-user,dc=test", attributes))
	cfg := config.GetConfig()
	previousConfig := *cfg
	*cfg = config.Config{}
	t.Cleanup(func() { *cfg = previousConfig })
	cfg.Auth.LDAP.Enable = true
	cfg.Auth.LDAP.Server.Address = address
	cfg.Auth.LDAP.Server.BindDN = "cn=test-reader"
	cfg.Auth.LDAP.Server.BindPassword = "synthetic-reader-password"
	cfg.Auth.LDAP.Server.BaseDN = "dc=test"
	cfg.Auth.LDAP.AttributeMapping.Username = authTestUsernameAttribute
	cfg.Auth.LDAP.AttributeMapping.DisplayName = "cn"
	cfg.Auth.LDAP.AttributeMapping.Email = "mail"
	cfg.Auth.LDAP.UID.Source = config.UIDSourceRID
	cfg.Auth.LDAP.UID.RID.Offset = 10000
	cfg.Auth.Token.AccessTokenSecret = "synthetic-test-signing-secret"
	cfg.Auth.Token.RefreshTokenSecret = "synthetic-test-refresh-secret"
	// Local signup stays disabled: it must not prevent LDAP auto-registration.
	cfg.Auth.Normal.AllowRegister = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}, &model.Account{}, &model.UserAccount{}, &model.SystemConfig{}); err != nil {
		t.Fatal(err)
	}
	previousQuery := *query.Q
	query.SetDefault(db)
	t.Cleanup(func() { *query.Q = previousQuery })
	account := model.Account{
		Model: gorm.Model{ID: model.DefaultAccountID}, Name: authTestDefaultProject, Nickname: "Default", Space: authTestDefaultProject,
		BillingIssueAmount: ptr.To(int64(25) * service.BillingPointScale), BillingIssuePeriodMinutes: ptr.To(0),
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	if billing {
		createBillingSystemConfigs(t, db, 10080)
	} else if err := db.Create(&model.SystemConfig{Key: model.ConfigKeyEnableBillingFeature, Value: "false"}).Error; err != nil {
		t.Fatal(err)
	}
	mgr := NewAuthMgr(&RegisterConfig{
		BillingService: service.NewBillingService(query.Use(db)),
		UserBanService: service.NewUserBanService(query.Use(db)),
	}).(*AuthMgr)
	return mgr, db
}

func requestFirstLDAPLogin(t *testing.T, mgr *AuthMgr, password string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(LoginReq{AuthMethod: AuthMethodLDAP, Username: ptr.To(firstLoginUsername), Password: &password})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	mgr.RegisterPublic(router.Group("/api/auth"))
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertFirstLoginRowCounts(t *testing.T, db *gorm.DB, users, associations int64) {
	t.Helper()
	for _, check := range []struct {
		model any
		want  int64
	}{{&model.User{}, users}, {&model.UserAccount{}, associations}} {
		var count int64
		if err := db.Unscoped().Model(check.model).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != check.want {
			t.Fatalf("row count for %T = %d, want %d", check.model, count, check.want)
		}
	}
}

func TestAuthenticationFirstLDAPLoginCreatesAccountAndReturnsSession(t *testing.T) {
	for _, billing := range []bool{false, true} {
		t.Run("billing="+strconv.FormatBool(billing), func(t *testing.T) {
			mgr, db := newFirstLoginHarness(t, billing, false)
			assertFirstLoginRowCounts(t, db, 0, 0)
			var userID uint
			for attempt := 0; attempt < 2; attempt++ {
				recorder := requestFirstLDAPLogin(t, mgr, firstLoginPassword)
				var response resputil.Response[LoginResp]
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if recorder.Code != http.StatusOK || response.Code != 0 {
					t.Fatalf("login returned HTTP %d code %d: %s", recorder.Code, response.Code, response.Message)
				}
				assertFirstLoginContext(t, &response.Data)
				if attempt == 0 {
					userID = response.Data.User.ID
				} else if response.Data.User.ID != userID {
					t.Fatal("repeat login created a different user")
				}
				assertFirstLoginTokens(t, mgr, &response.Data, userID)
				assertFirstLoginRowCounts(t, db, 1, 1)
			}
			assertFirstLoginPersistence(t, db, userID, billing)
		})
	}
}

func TestAuthenticationFirstLDAPLoginFailureLeavesNoPartialAccount(t *testing.T) {
	for _, stage := range []string{"wrong password", "missing SID", "project association", authTestBillingFailure} {
		t.Run(stage, func(t *testing.T) {
			mgr, db := newFirstLoginHarness(t, stage == authTestBillingFailure, stage == "missing SID")
			password := firstLoginPassword
			wantStatus, wantCode := 500, bizerr.Internal.AccountInitializationFailed
			switch stage {
			case "wrong password":
				password = "incorrect-test-password"
				wantStatus, wantCode = 401, bizerr.Auth.InvalidCredentials
			case "missing SID":
				wantCode = bizerr.Internal.IdentityMappingFailed
			}
			writeFailed := injectFirstLoginFailure(t, db, stage)
			recorder := requestFirstLDAPLogin(t, mgr, password)
			var response resputil.Response[json.RawMessage]
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != wantStatus || response.Code != wantCode || string(response.Data) != authTestNullData {
				t.Fatalf("failure stage %s returned HTTP %d code %d; expected HTTP %d code %d and no session", stage, recorder.Code, response.Code, wantStatus, wantCode)
			}
			if strings.Contains(response.Message, "private-directory-detail") {
				t.Fatal("response exposed the underlying database failure")
			}
			if (stage == "project association" || stage == authTestBillingFailure) && !*writeFailed {
				t.Fatal("test did not reach the intended transactional failure")
			}
			assertFirstLoginRowCounts(t, db, 0, 0)
		})
	}
}

func TestAuthenticationPlatformAccountStates(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	permanent := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name         string
		status       model.Status
		bannedUntil  *time.Time
		restrictions model.UserBanRestrictions
		httpStatus   int
		code         bizerr.BizCode
	}{
		{"pending activation", model.StatusPending, nil, model.UserBanRestrictions{}, 401, bizerr.Auth.AccountNotActivated},
		{authTestDisabled, model.StatusInactive, nil, model.UserBanRestrictions{}, 403, bizerr.Forbidden.AccountDisabled},
		{"platform ban", model.StatusActive, &future, model.UserBanRestrictions{PlatformAccess: true}, 403, bizerr.Forbidden.UserBanned},
		{"permanent platform ban", model.StatusActive, &permanent, model.UserBanRestrictions{PlatformAccess: true}, 403, bizerr.Forbidden.UserBanned},
		{"expired platform ban", model.StatusActive, &past, model.UserBanRestrictions{PlatformAccess: true}, 200, 0},
		{"job submission restriction only", model.StatusActive, &future, model.UserBanRestrictions{JobSubmission: true}, 200, 0},
		{"active", model.StatusActive, nil, model.UserBanRestrictions{}, 200, 0},
		{"unknown status fails closed", model.Status(0), nil, model.UserBanRestrictions{}, 500, bizerr.Internal.ServiceError},
	} {
		t.Run(test.name, func(t *testing.T) {
			mgr, db := newFirstLoginHarness(t, false, false)
			initial := requestFirstLDAPLogin(t, mgr, firstLoginPassword)
			if initial.Code != http.StatusOK {
				t.Fatalf("test user could not sign in before status change: HTTP %d", initial.Code)
			}
			if err := db.Model(&model.User{}).Where("name = ?", firstLoginUsername).Updates(map[string]any{
				"status": test.status, "banned_timestamp": test.bannedUntil,
				"ban_restrictions": datatypes.NewJSONType(test.restrictions),
			}).Error; err != nil {
				t.Fatal(err)
			}
			recorder := requestFirstLDAPLogin(t, mgr, firstLoginPassword)
			var response resputil.Response[json.RawMessage]
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != test.httpStatus || response.Code != test.code {
				t.Fatalf("account state %s returned HTTP %d code %d, want HTTP %d code %d", test.name, recorder.Code, response.Code, test.httpStatus, test.code)
			}
			if test.httpStatus != http.StatusOK && string(response.Data) != authTestNullData {
				t.Fatal("restricted account received session data")
			}
			if test.httpStatus == http.StatusOK {
				var session *LoginResp
				if err := json.Unmarshal(response.Data, &session); err != nil {
					t.Fatal(err)
				}
				if session.AccessToken == "" || session.RefreshToken == "" {
					t.Fatal("account allowed to sign in did not receive a session")
				}
			}
			assertFirstLoginRowCounts(t, db, 1, 1)
		})
	}
}

func firstLoginSearchResponse(entry *ldap.Entry) *ber.Packet {
	response := ber.Encode(ber.ClassApplication, ber.TypeConstructed, ldap.ApplicationSearchResultEntry, nil, "")
	response.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, entry.DN, ""))
	attributes := ber.NewSequence("")
	for _, attribute := range entry.Attributes {
		item := ber.NewSequence("")
		item.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, attribute.Name, ""))
		values := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSet, nil, "")
		for _, value := range attribute.Values {
			values.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, value, ""))
		}
		item.AppendChild(values)
		attributes.AppendChild(item)
	}
	response.AppendChild(attributes)
	return response
}

func assertFirstLoginContext(t *testing.T, session *LoginResp) {
	t.Helper()
	if session.User.ID == 0 || session.User.Name != firstLoginUsername || session.Context.Queue != authTestDefaultProject ||
		session.Context.AccessQueue != model.AccessModeRO || session.Context.Space != firstLoginUsername {
		t.Fatal("login returned incomplete user or default-project context")
	}
}

func assertFirstLoginTokens(t *testing.T, mgr *AuthMgr, session *LoginResp, userID uint) {
	t.Helper()
	for _, token := range []string{session.AccessToken, session.RefreshToken} {
		if token == "" {
			t.Fatal("successful login returned an empty token")
		}
		check := mgr.tokenMgr.CheckToken
		if token == session.RefreshToken {
			check = mgr.tokenMgr.CheckRefreshToken
		}
		claims, err := check(token)
		if err != nil || claims.UserID != userID || claims.AccountID != model.DefaultAccountID {
			t.Fatal("session token did not identify the newly created user and default project")
		}
	}
}

func assertFirstLoginPersistence(t *testing.T, db *gorm.DB, userID uint, billing bool) {
	t.Helper()
	assertFirstLoginProfile(t, db, userID)

	var association model.UserAccount
	if err := db.Where("user_id = ?", userID).First(&association).Error; err != nil {
		t.Fatal(err)
	}
	if association.AccountID != model.DefaultAccountID || association.Role != model.RoleUser || association.AccessMode != model.AccessModeRO {
		t.Fatal("default-project association has unexpected permissions")
	}
	if billing && association.PeriodFreeBalance != int64(25)*service.BillingPointScale {
		t.Fatal("first login did not initialize the default-project billing balance")
	}
}

func assertFirstLoginProfile(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		t.Fatal(err)
	}
	attributes := user.Attributes.Data()
	if user.Password != nil || user.Status != model.StatusActive || user.Role != model.RoleUser ||
		user.Nickname != "First LDAP User" || attributes.ID != userID || attributes.UID == nil || *attributes.UID != "10042" ||
		attributes.GID == nil || *attributes.GID != "10513" || attributes.Email == nil || *attributes.Email != "first-login@example.invalid" {
		t.Fatal("LDAP account state, password policy, profile, or RID identity was not persisted correctly")
	}
}

func injectFirstLoginFailure(t *testing.T, db *gorm.DB, stage string) *bool {
	t.Helper()
	writeFailed := new(bool)
	fail := func(tx *gorm.DB) {
		if tx.Statement.Table == "user_accounts" {
			*writeFailed = true
			_ = tx.AddError(authTestError("private-directory-detail"))
		}
	}
	switch stage {
	case "project association":
		if err := db.Callback().Create().Before("gorm:create").Register("test:fail_project_association", fail); err != nil {
			t.Fatal(err)
		}
	case authTestBillingFailure:
		if err := db.Callback().Update().Before("gorm:update").Register("test:fail_billing_initialization", fail); err != nil {
			t.Fatal(err)
		}
	}
	return writeFailed
}

func respondFirstLoginBind(t *testing.T, conn net.Conn, entry *ldap.Entry, messageID int64, request *ber.Packet) bool {
	t.Helper()
	if len(request.Children) != 3 {
		t.Error("unexpected LDAP bind request")
		return false
	}
	dn := request.Children[1].Data.String()
	password := request.Children[2].Data.String()
	code := int64(ldap.LDAPResultInvalidCredentials)
	if (dn == "cn=test-reader" && password == "synthetic-reader-password") ||
		(dn == entry.DN && password == firstLoginPassword) {
		code = ldap.LDAPResultSuccess
	}
	return writeFirstLoginLDAP(t, conn, messageID, firstLoginLDAPResult(ldap.ApplicationBindResponse, code))
}
