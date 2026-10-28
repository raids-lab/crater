package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

func newJobTemplateListTestContext(rawQuery string) *gin.Context {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/jobtemplate/list?"+rawQuery, http.NoBody)
	return context
}

func TestBindJobTemplateListQueryDefaults(t *testing.T) {
	request, err := bindJobTemplateListQuery(newJobTemplateListTestContext(""))
	if err != nil {
		t.Fatalf("bindJobTemplateListQuery returned error: %v", err)
	}
	if request.Page != 1 || request.PageSize != 10 {
		t.Fatalf("unexpected pagination defaults: %#v", request)
	}
	if request.Owner != "all" || request.Sort != "-createdAt" {
		t.Fatalf("unexpected filter defaults: %#v", request)
	}
}

func TestBindJobTemplateListQueryNormalizesSearch(t *testing.T) {
	request, err := bindJobTemplateListQuery(
		newJobTemplateListTestContext("page=2&page_size=20&search=++demo+&owner=mine&sort=createdAt"),
	)
	if err != nil {
		t.Fatalf("bindJobTemplateListQuery returned error: %v", err)
	}
	if request.Page != 2 || request.PageSize != 20 {
		t.Fatalf("unexpected pagination: %#v", request)
	}
	if request.Search != "demo" || request.Owner != "mine" || request.Sort != "createdAt" {
		t.Fatalf("unexpected normalized query: %#v", request)
	}
	if request.offset() != 20 {
		t.Fatalf("unexpected offset: %d", request.offset())
	}
}

func TestBindJobTemplateListQueryRejectsInvalidValues(t *testing.T) {
	testCases := []string{
		"page=0",
		"page_size=201",
		"owner=unknown",
		"sort=name",
		"search=" + strings.Repeat("a", jobTemplateMaxSearchRunes+1),
	}

	for _, rawQuery := range testCases {
		t.Run(rawQuery, func(t *testing.T) {
			if _, err := bindJobTemplateListQuery(newJobTemplateListTestContext(rawQuery)); err == nil {
				t.Fatal("expected query validation error")
			}
		})
	}
}

func newJobTemplateRoutesTestRouter(t *testing.T) (router *gin.Engine, username string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get test database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}, &model.Jobtemplate{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	query.SetDefault(db)
	user := model.User{
		Model:  gorm.Model{ID: 1},
		Name:   "template-owner",
		Role:   model.RoleUser,
		Status: model.StatusActive,
		Space:  "/tmp/template-owner",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create template owner: %v", err)
	}
	createdAt := time.Date(2026, time.October, 13, 0, 0, 0, 0, time.UTC)
	for id := uint(1); id <= 3; id++ {
		template := model.Jobtemplate{
			Model:    gorm.Model{ID: id, CreatedAt: createdAt},
			Name:     fmt.Sprintf("template-%d", id),
			Template: "{}",
			UserID:   user.ID,
		}
		if err := db.Create(&template).Error; err != nil {
			t.Fatalf("create template: %v", err)
		}
	}

	router = gin.New()
	router.Use(func(c *gin.Context) {
		util.SetJWTContext(c, util.JWTMessage{
			UserID:       user.ID,
			Username:     user.Name,
			RolePlatform: model.RoleUser,
		})
		c.Next()
	})
	(&JobTemplateMgr{}).RegisterProtected(router.Group("/v1/jobtemplate"))
	return router, user.Name
}

func TestJobTemplateRoutesUseSinglePaginatedList(t *testing.T) {
	router, username := newJobTemplateRoutesTestRouter(t)
	t.Run("frontend list returns the requested page", func(t *testing.T) {
		response := requestJobTemplateRoute(t, router, "/v1/jobtemplate/list?page=2&page_size=1&sort=createdAt")
		var result resputil.Response[resputil.Page[JobTemplateresp]]
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode template page: %v", err)
		}
		page := result.Data
		if page.Page != 2 || page.PageSize != 1 || page.Total != 3 || len(page.Items) != 1 {
			t.Fatalf("unexpected template page: %+v", page)
		}
		if page.Items[0].Template != "{}" || page.Items[0].UserInfo.Username != username {
			t.Fatalf("template fields changed: %+v", page.Items[0])
		}
		if page.Items[0].ID != 2 {
			t.Fatalf("template page item = %d, want 2", page.Items[0].ID)
		}
	})
	t.Run("filters apply to the existing list path", func(t *testing.T) {
		response := requestJobTemplateRoute(t, router, "/v1/jobtemplate/list?page=1&owner=others")
		var result resputil.Response[resputil.Page[JobTemplateresp]]
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode filtered page: %v", err)
		}
		if result.Data.Total != 0 || len(result.Data.Items) != 0 {
			t.Fatalf("owner filter was ignored: %+v", result.Data)
		}
	})
	t.Run("detail route remains available", func(t *testing.T) {
		response := requestJobTemplateRoute(t, router, "/v1/jobtemplate/1")
		var result resputil.Response[JobTemplateresp]
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode template detail: %v", err)
		}
		if result.Data.ID != 1 || result.Data.Template != "{}" {
			t.Fatalf("unexpected template detail: %+v", result.Data)
		}
	})
}

func TestJobTemplateRoutesRejectInvalidRequests(t *testing.T) {
	router, _ := newJobTemplateRoutesTestRouter(t)
	for _, testCase := range []struct {
		path   string
		status int
	}{
		{path: "/v1/jobtemplate/list?page=0", status: http.StatusBadRequest},
		{path: "/v1/jobtemplate", status: http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, testCase.path, http.NoBody))
		if response.Code != testCase.status {
			t.Fatalf("GET %s returned HTTP %d: %s", testCase.path, response.Code, response.Body.String())
		}
	}
}

func TestJobTemplateEmptyListReturnsArray(t *testing.T) {
	router, _ := newJobTemplateRoutesTestRouter(t)
	j := query.Jobtemplate
	if _, err := j.WithContext(t.Context()).Where(j.ID.IsNotNull()).Delete(); err != nil {
		t.Fatalf("clear isolated template fixture: %v", err)
	}

	t.Run("empty page items remain an array", func(t *testing.T) {
		response := requestJobTemplateRoute(t, router, "/v1/jobtemplate/list?page=1&page_size=10")
		var result resputil.Response[struct {
			Items    json.RawMessage `json:"items"`
			Total    int64           `json:"total"`
			Page     int             `json:"page"`
			PageSize int             `json:"page_size"`
		}]
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode empty template page: %v", err)
		}
		page := result.Data
		if result.Code != resputil.OK || string(page.Items) != emptyJSONArray || page.Total != 0 ||
			page.Page != 1 || page.PageSize != 10 {
			t.Fatalf("unexpected empty page response: %s", response.Body.String())
		}
	})
}

func requestJobTemplateRoute(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s returned %d: %s", path, response.Code, response.Body.String())
	}
	return response
}
