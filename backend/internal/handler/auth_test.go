// Copyright 2026 The Crater Project Team, RAIDS-Lab
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ldap "github.com/go-ldap/ldap/v3"
	imrocreq "github.com/imroc/req/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/config"
)

const (
	authTestUsernameAttribute   = "uid"
	authTestDefaultProject      = "default"
	authTestBillingFailure      = "billing initialization"
	authTestNullData            = "null"
	authTestDatabaseUnavailable = "database unavailable"
	authTestDisabled            = "disabled"
)

type authTestError string

func (e authTestError) Error() string { return string(e) }

type authTestLDAP struct {
	bindErrors  []error
	binds       int
	searchError error
	entries     []*ldap.Entry
	filter      string
}

func (l *authTestLDAP) Bind(_, _ string) error {
	index := l.binds
	l.binds++
	if index < len(l.bindErrors) {
		return l.bindErrors[index]
	}
	return nil
}

func (l *authTestLDAP) Search(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
	l.filter = req.Filter
	return &ldap.SearchResult{Entries: l.entries}, l.searchError
}

func assertAuthError(t *testing.T, err error, status int, code bizerr.BizCode, message string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an authentication error")
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	resputil.HandleError(ctx, err)
	var response resputil.Response[any]
	if decodeErr := json.Unmarshal(recorder.Body.Bytes(), &response); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if recorder.Code != status || response.Code != code {
		t.Fatalf("response = HTTP %d code %d: %s; want HTTP %d code %d", recorder.Code, response.Code, response.Message, status, code)
	}
	if !strings.Contains(response.Message, message) {
		t.Fatalf("message %q does not contain %q", response.Message, message)
	}
	if strings.Contains(response.Message, "private-directory-detail") {
		t.Fatalf("response exposed the underlying error: %s", response.Message)
	}
}

func TestAuthenticationLDAPFailureStages(t *testing.T) {
	credentialsError := ldap.NewError(ldap.LDAPResultInvalidCredentials, authTestError("private-directory-detail"))
	serviceError := authTestError("private-directory-detail")
	entry := ldap.NewEntry("cn=test", map[string][]string{})
	sidEntry := ldap.NewEntry("cn=test", map[string][]string{"objectSid": {string([]byte{1, 1, 0, 0, 0, 0, 0, 5, 42, 0, 0, 0})}})
	tests := []struct {
		name       string
		connection authTestLDAP
		source     string
		status     int
		code       bizerr.BizCode
		message    string
		binds      int
	}{
		{"admin credentials", authTestLDAP{bindErrors: []error{credentialsError}}, authTestDefaultProject, 500, bizerr.Internal.ThirdPartyApiError, "service account", 1},
		{"search unavailable", authTestLDAP{searchError: serviceError}, authTestDefaultProject, 500, bizerr.Internal.ThirdPartyApiError, "could not be searched", 1},
		{"search timeout", authTestLDAP{searchError: context.DeadlineExceeded}, authTestDefaultProject, 504, bizerr.GatewayTimeout.GatewayTimeout, "timed out", 1},
		{"no entry", authTestLDAP{}, authTestDefaultProject, 401, bizerr.Auth.RegisterNotFound, "No matching LDAP account", 1},
		{"multiple entries", authTestLDAP{entries: []*ldap.Entry{entry, entry}}, authTestDefaultProject, 500, bizerr.Internal.ThirdPartyApiError, "multiple directory entries", 1},
		{"user credentials", authTestLDAP{entries: []*ldap.Entry{entry}, bindErrors: []error{nil, credentialsError}}, "rid", 401, bizerr.Auth.InvalidCredentials, "username or password", 2},
		{"user bind unavailable", authTestLDAP{entries: []*ldap.Entry{entry}, bindErrors: []error{nil, serviceError}}, authTestDefaultProject, 500, bizerr.Internal.ThirdPartyApiError, "could not verify", 2},
		{"missing SID after successful bind", authTestLDAP{entries: []*ldap.Entry{entry}}, "rid", 500, bizerr.Internal.IdentityMappingFailed, "authentication succeeded", 2},
		{"missing primary group after successful bind", authTestLDAP{entries: []*ldap.Entry{sidEntry}}, "rid", 500, bizerr.Internal.IdentityMappingFailed, "primary group could not be mapped", 2},
		{"missing POSIX identity", authTestLDAP{entries: []*ldap.Entry{entry}}, "ldap", 500, bizerr.Internal.IdentityMappingFailed, "UID or GID attributes are missing", 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Auth.LDAP.UID.Source = test.source
			cfg.Auth.LDAP.AttributeMapping.Username = authTestUsernameAttribute
			mgr := &AuthMgr{}
			attr := &model.UserAttribute{}
			err := mgr.authenticateLDAP(&test.connection, cfg, "test", "password", attr)
			assertAuthError(t, err, test.status, test.code, test.message)
			if test.connection.binds != test.binds {
				t.Fatalf("bind calls = %d, want %d", test.connection.binds, test.binds)
			}
			if test.name == "user credentials" && attr.Name != "" {
				t.Fatal("password rejection populated identity attributes")
			}
			if test.name == "admin credentials" && !errors.Is(err, credentialsError) {
				t.Fatal("admin failure lost the underlying cause")
			}
		})
	}
}

func TestAuthenticationLDAPSuccessAndFilterEscaping(t *testing.T) {
	tests := []struct {
		name, source     string
		attributes       map[string][]string
		wantUID, wantGID string
	}{
		{authTestDefaultProject, authTestDefaultProject, map[string][]string{}, "", ""},
		{"POSIX", "ldap", map[string][]string{"uidNumber": {"10042"}, "gidNumber": {"10513"}}, "10042", "10513"},
		{"RID", "rid", map[string][]string{"objectSid": {string([]byte{1, 1, 0, 0, 0, 0, 0, 5, 42, 0, 0, 0})}, "primaryGroupID": {"513"}}, "10042", "10513"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Auth.LDAP.AttributeMapping.Username = authTestUsernameAttribute
			cfg.Auth.LDAP.UID.Source = test.source
			cfg.Auth.LDAP.UID.RID.Offset = 10000
			cfg.Auth.LDAP.UID.LDAPAttribute.UID = "uidNumber"
			cfg.Auth.LDAP.UID.LDAPAttribute.GID = "gidNumber"
			conn := &authTestLDAP{entries: []*ldap.Entry{ldap.NewEntry("cn=test", test.attributes)}}
			attr := &model.UserAttribute{}
			err := (&AuthMgr{}).authenticateLDAP(conn, cfg, "a*)(uid=*)", "password", attr)
			if err != nil {
				t.Fatal(err)
			}
			if conn.filter != `(uid=a\2a\29\28uid=\2a\29)` {
				t.Fatalf("unescaped LDAP filter: %s", conn.filter)
			}
			if conn.binds != 2 {
				t.Fatalf("bind calls = %d", conn.binds)
			}
			if test.wantUID != "" && (attr.UID == nil || attr.GID == nil || *attr.UID != test.wantUID || *attr.GID != test.wantGID) {
				t.Fatalf("unexpected UID/GID: %+v", attr)
			}
		})
	}
}

func TestAuthenticationExternalUIDResponses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		success bool
	}{
		{"success", 200, `{"uid":"10042","gid":"10513"}`, true},
		{"service error without error field", 503, `{}`, false},
		{"missing GID", 200, `{"uid":"10042"}`, false},
		{"invalid JSON", 200, `{`, false},
		{"error result", 404, `{"error":"private-directory-detail"}`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("username") != "test" {
					t.Error("username was not passed to the UID service")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			mgr := &AuthMgr{req: imrocreq.C()}
			result, err := mgr.fetchExternalUID(context.Background(), "test", server.URL)
			if test.success {
				if err != nil || result.UID != "10042" || result.GID != "10513" {
					t.Fatalf("UID lookup = %+v, %v", result, err)
				}
			} else {
				assertAuthError(t, err, 502, bizerr.BadGateway.BadGateway, "LDAP authentication succeeded")
			}
		})
	}
	t.Run("canceled request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := (&AuthMgr{req: imrocreq.C()}).fetchExternalUID(ctx, "test", "http://example.invalid")
		assertAuthError(t, err, 502, bizerr.BadGateway.BadGateway, "UID service")
	})
}

func TestAuthenticationLocalDatabaseFailureIsNotBadCredentials(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, password TEXT, deleted_at DATETIME)").Error; err != nil {
		t.Fatal(err)
	}
	query.SetDefault(db)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	mgr := &AuthMgr{}
	assertAuthError(t, mgr.normalAuth(ctx, "missing", "password"), 401, bizerr.Auth.RegisterNotFound, "No matching local account")
	hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO users (id, name, password) VALUES (1, ?, ?)", "test", string(hash)).Error; err != nil {
		t.Fatal(err)
	}
	assertAuthError(t, mgr.normalAuth(ctx, "test", "wrong"), 401, bizerr.Auth.InvalidCredentials, "incorrect")
	if err := mgr.normalAuth(ctx, "test", "correct"); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("DROP TABLE users").Error; err != nil {
		t.Fatal(err)
	}
	assertAuthError(t, mgr.normalAuth(ctx, "test", "correct"), 500, bizerr.Internal.DatabaseError, "Account lookup")
}

func TestAuthenticationRejectsMalformedOrEmptyCredentials(t *testing.T) {
	tests := []struct {
		body string
		code bizerr.BizCode
	}{
		{`{`, bizerr.BadRequest.InvalidRequest},
		{`{"auth":"ldap"}`, bizerr.BadRequest.MissingParameter},
		{`{"auth":"ldap","username":"test","password":""}`, bizerr.BadRequest.ParameterError},
		{`{"auth":"ldap","username":"  ","password":"password"}`, bizerr.BadRequest.ParameterError},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(test.body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		(&AuthMgr{}).Login(ctx)
		var response resputil.Response[any]
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if recorder.Code != 400 || response.Code != test.code {
			t.Fatalf("body %s: HTTP %d, response %+v", test.body, recorder.Code, response)
		}
	}
}

func TestAuthenticationProjectLookupFailureClassification(t *testing.T) {
	assertAuthError(t, loginQueueError(gorm.ErrRecordNotFound), 403, bizerr.Forbidden.NoProjectAccess, "No accessible project")
	assertAuthError(t, loginQueueError(authTestError("private-directory-detail")), 500, bizerr.Internal.DatabaseError, "Project lookup")
}

func TestAuthenticationAccountCreationErrors(t *testing.T) {
	assertAuthError(t, accountCreationError(gorm.ErrDuplicatedKey), 409, bizerr.Conflict.ResourceAlreadyExists, "already registered")
	assertAuthError(t, accountCreationError(testSQLStateError{state: "23505"}), 409, bizerr.Conflict.ResourceAlreadyExists, "already registered")
	assertAuthError(t, accountCreationError(authTestError("private-directory-detail")), 500, bizerr.Internal.AccountInitializationFailed, "initialization")
	err := bizerr.Internal.ServiceError.New("Billing initialization is unavailable. Contact the platform administrator.")
	if !errors.Is(accountCreationError(err), err) {
		t.Fatal("account creation replaced a specific business error")
	}
}

func TestAuthenticationIdentityRangeValidation(t *testing.T) {
	for _, value := range []string{"", "-1", "4294967296", "not-a-number"} {
		if validUnixID(value) {
			t.Fatalf("invalid Unix ID accepted: %q", value)
		}
	}
	if !validUnixID("1001") {
		t.Fatal("valid identity rejected")
	}
	for _, test := range []struct {
		rid    uint64
		offset int
	}{{42, -100}, {4294967295, 100}} {
		if _, err := mappedUnixID(test.rid, test.offset); err == nil {
			t.Fatal("RID offset outside Unix identity range accepted")
		}
	}
	if _, err := parseRIDFromSID([]byte{2, 1, 0, 0, 0, 0, 0, 5, 42, 0, 0, 0}); err == nil {
		t.Fatal("invalid SID revision accepted")
	}
	cfg := &config.Config{}
	cfg.Auth.LDAP.UID.Source = "unsupported"
	conn := &authTestLDAP{entries: []*ldap.Entry{ldap.NewEntry("cn=test", nil)}}
	assertAuthError(t, (&AuthMgr{}).authenticateLDAP(conn, cfg, "test", "password", &model.UserAttribute{}), 500, bizerr.Internal.IdentityMappingFailed, "source is unsupported")
}
