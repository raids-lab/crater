package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

func TestStatisticsAuthorization(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, IgnoreRelationshipsWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err = db.AutoMigrate(&model.UserAccount{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.UserAccount{UserID: 1, AccountID: 2, Role: model.RoleAdmin}).Error; err != nil {
		t.Fatal(err)
	}
	local := query.Use(db)
	previous := query.UserAccount
	query.UserAccount = &local.UserAccount
	t.Cleanup(func() { query.UserAccount = previous })
	for _, tc := range []struct {
		name       string
		scope      payload.StatisticsScope
		target     uint
		role       model.Role
		adminRoute bool
		want       bool
	}{
		{"platform denied", payload.ScopeCluster, 0, model.RoleUser, false, false},
		{"admin on protected route", payload.ScopeCluster, 0, model.RoleAdmin, false, false},
		{"platform admin", payload.ScopeCluster, 0, model.RoleAdmin, true, true},
		{"account admin", payload.ScopeAccount, 2, model.RoleUser, false, true},
		{"other account", payload.ScopeAccount, 3, model.RoleUser, false, false},
		{"missing account", payload.ScopeAccount, 0, model.RoleAdmin, true, false},
		{"user detail", payload.ScopeUser, 2, model.RoleUser, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/", http.NoBody)
			util.SetJWTContext(c, util.JWTMessage{UserID: 1, RolePlatform: tc.role})
			if got := authorizeStatistics(c, &payload.StatisticsReq{Scope: tc.scope, TargetID: tc.target}, tc.adminRoute); got != tc.want {
				t.Fatalf("authorized=%v", got)
			}
		})
	}
}
func TestStatisticsRejectsNonUTCDayRange(t *testing.T) {
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		want       bool
	}{{"UTC", day, day.Add(24 * time.Hour), true}, {"local midnight", day.Add(-8 * time.Hour), day.Add(16 * time.Hour), false}, {"reversed", day.Add(24 * time.Hour), day, false}, {"same", day, day, false}} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if got := validateStatisticsRange(c, &payload.StatisticsReq{StartTime: tc.start, EndTime: tc.end}); got != tc.want {
				t.Fatalf("valid=%v", got)
			}
		})
	}
}

func TestStatisticsClusterScopeRequiresAdminRoute(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		util.SetJWTContext(c, util.JWTMessage{UserID: 1, RolePlatform: model.RoleAdmin})
	})
	manager := NewStatisticsMgr(&RegisterConfig{})
	manager.RegisterProtected(router.Group("/v1/statistics"))
	manager.RegisterAdmin(router.Group("/v1/admin/statistics"))
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/v1/statistics", http.StatusForbidden},
		{"/v1/statistics/rankings", http.StatusForbidden},
		{"/v1/admin/statistics", http.StatusBadRequest},
		{"/v1/admin/statistics/rankings", http.StatusBadRequest},
	} {
		t.Run(tc.path, func(t *testing.T) {
			// A non-midnight range fails validation only after authorization succeeds.
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tc.path+"?scope=cluster&step=day&dimension=user&metric=usage&order=desc&resourceName=cpu&startTime=2026-11-01T01:00:00Z&endTime=2026-11-02T00:00:00Z", http.NoBody)
			router.ServeHTTP(recorder, request)
			if recorder.Code != tc.status {
				t.Fatalf("status=%d, body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestStatisticsPersonalRouteControlsProfileVisibility(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, IgnoreRelationshipsWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err = db.AutoMigrate(&model.Resource{}, &model.ResourceUsageDaily{}); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	daily := []model.ResourceUsageDaily{
		{StatDate: start, CoveredUntil: end, IsComplete: true},
		{StatDate: start, CoveredUntil: end, IsComplete: true, UserID: 2, AccountID: 3,
			ResourceName: "cpu", UsageSeconds: 7200, ProfiledUsageSeconds: 7200},
	}
	if err = db.Create(&daily).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, path string
		role       model.Role
		userID     uint
		full       bool
	}{
		{"ordinary user querying self", "/v1/statistics", model.RoleUser, 2, true},
		{"ordinary user querying another user", "/v1/statistics", model.RoleUser, 1, false},
		{"admin on ordinary route", "/v1/statistics", model.RoleAdmin, 1, false},
		{"platform admin", "/v1/admin/statistics", model.RoleAdmin, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				util.SetJWTContext(c, util.JWTMessage{UserID: tc.userID, RolePlatform: tc.role})
			})
			mgr := NewStatisticsMgr(&RegisterConfig{StatisticsService: service.NewStatisticsService(query.Use(db))})
			mgr.RegisterProtected(router.Group("/v1/statistics"))
			mgr.RegisterAdmin(router.Group("/v1/admin/statistics"))
			recorder := httptest.NewRecorder()
			url := tc.path + "?scope=user&targetID=2&startTime=2026-11-01T00:00:00Z&endTime=2026-11-02T00:00:00Z"
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, url, http.NoBody))
			assertStatisticsOverview(t, recorder, tc.full)
		})
	}
}

func assertStatisticsOverview(t *testing.T, response *httptest.ResponseRecorder, full bool) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", response.Code, response.Body.String())
	}
	var got struct {
		Data struct {
			TotalUsage map[string]map[string]json.RawMessage
			Series     []map[string]json.RawMessage
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	cpu := got.Data.TotalUsage["cpu"]
	if string(cpu["usage"]) != "2" || len(got.Data.Series) != 1 {
		t.Fatalf("missing allocated usage: %s", response.Body.String())
	}
	for _, field := range []string{"effectiveUsage", "profiledUsage", "utilization", "coverage"} {
		if _, present := cpu[field]; present != full {
			t.Fatalf("CPU %q presence=%v, want %v", field, present, full)
		}
	}
	if _, present := got.Data.Series[0]["effectiveUsage"]; present != full {
		t.Fatalf("series effectiveUsage presence=%v, want %v", present, full)
	}
	if full && string(cpu["effectiveUsage"]) != "0" {
		t.Fatalf("response lost a measured zero: %s", cpu["effectiveUsage"])
	}
}

func TestStatisticsRejectsExcessiveRange(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		util.SetJWTContext(c, util.JWTMessage{UserID: 1, RolePlatform: model.RoleAdmin})
	})
	manager := NewStatisticsMgr(&RegisterConfig{})
	manager.RegisterAdmin(router.Group("/v1/admin/statistics"))
	for _, path := range []string{"/v1/admin/statistics", "/v1/admin/statistics/rankings"} {
		recorder := httptest.NewRecorder()
		url := path + "?scope=cluster&dimension=user&metric=usage&order=desc&resourceName=cpu" +
			"&startTime=2024-01-01T00:00:00Z&endTime=2025-01-02T00:00:00Z"
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, url, http.NoBody))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d, body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}
