package operations

import (
	"encoding/json"
	"fmt"
	"math"
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
	"github.com/raids-lab/crater/internal/service"
)

func newOperationsPaginationRouter(t *testing.T, role model.Role) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.OperationLog{}, &model.CronJobRecord{}); err != nil {
		t.Fatal(err)
	}
	query.SetDefault(db)
	previous := service.OpLog
	service.OpLog = service.NewOperationLogService(db)
	t.Cleanup(func() { service.OpLog = previous })
	router := gin.New()
	admin := router.Group("/api/v1/admin", func(c *gin.Context) {
		if role != model.RoleAdmin {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	})
	(&OperationLogMgr{}).RegisterAdmin(admin.Group("/operation-logs"))
	(&OperationsMgr{}).RegisterAdmin(admin.Group("/operations"))
	return router, db
}

func TestOperationLogPaginationRoutes(t *testing.T) {
	router, db := newOperationsPaginationRouter(t, model.RoleAdmin)
	logs := make([]model.OperationLog, 531)
	for i := range logs {
		logs[i].Target = "full"
	}
	logs[0].Target = "single"
	if err := db.CreateInBatches(logs, 100).Error; err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		query                      string
		wantCount, wantTotal, code int
		pageSize                   bool
	}{
		{query: "?page_size=25", wantCount: 25, wantTotal: 531, code: 200, pageSize: true},
		{query: "?page_size=200", wantCount: 200, wantTotal: 531, code: 200, pageSize: true},
		{query: "?page_size=201", code: 400},
		{query: "?page_size=invalid", code: 400},
		{query: "?page_size=0", code: 400},
		{query: "?page=3&page_size=200", wantCount: 131, wantTotal: 531, code: 200, pageSize: true},
		{query: "?page=4&page_size=200", wantTotal: 531, code: 200, pageSize: true},
		{query: "?limit=500", wantCount: 500, wantTotal: 531, code: 200},
		{query: fmt.Sprintf("?limit=%d&target=single", math.MaxInt), wantCount: 1, wantTotal: 1, code: 200},
		{query: fmt.Sprintf("?limit=%d&target=absent", math.MaxInt), wantTotal: 0, code: 200},
	} {
		t.Run(tc.query, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/operation-logs"+tc.query, http.NoBody))
			if w.Code != tc.code {
				t.Fatalf("status %d: %s", w.Code, w.Body)
			}
			if tc.code != http.StatusOK {
				return
			}
			var body struct {
				Data struct {
					Items    []OperationLogResp
					Total    int
					PageSize *int `json:"page_size"`
				}
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Data.Items == nil || len(body.Data.Items) != tc.wantCount ||
				body.Data.Total != tc.wantTotal || (body.Data.PageSize != nil) != tc.pageSize {
				t.Fatalf("items=%d total=%d canonical=%v", len(body.Data.Items), body.Data.Total, body.Data.PageSize != nil)
			}
		})
	}
}

func TestCronJobRecordReadRoutes(t *testing.T) {
	router, db := newOperationsPaginationRouter(t, model.RoleAdmin)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	records := []model.CronJobRecord{
		{Name: "cleanup", ExecuteTime: now, Status: model.CronJobRecordStatusSuccess},
		{Name: "cleanup", ExecuteTime: now, Status: model.CronJobRecordStatusSuccess},
		{Name: "other", ExecuteTime: now, Status: model.CronJobRecordStatusSuccess},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatal(err)
	}

	const path = "/api/v1/admin/operations/cronjob/record/list"
	for _, tc := range []struct {
		name, method, params, body, listKey string
		wantCount                           int
		wantFirstID                         uint
		code                                int
	}{
		{"GET first page", http.MethodGet, "?scope_name=cleanup&status=success&page_size=1", "", "items", 1, 2, 200},
		{"GET second page", http.MethodGet, "?scope_name=cleanup&status=success&page=2&page_size=1", "", "items", 1, 1, 200},
		{"GET empty", http.MethodGet, "?scope_name=missing", "", "items", 0, 0, 200},
		{"GET invalid", http.MethodGet, "?page_size=201", "", "", 0, 0, 400},
		{"POST historical JSON", http.MethodPost, "", `{"name":["cleanup"],"status":"success","startTime":"2026-10-01T12:00:00Z","endTime":"2026-10-01T12:00:00Z"}`, "records", 2, 2, 200},
		{"POST empty", http.MethodPost, "", `{"name":["missing"]}`, "records", 0, 0, 200},
		{"POST invalid", http.MethodPost, "", `{"status":20}`, "", 0, 0, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tc.method, path+tc.params, strings.NewReader(tc.body)))
			if w.Code != tc.code {
				t.Fatalf("status %d: %s", w.Code, w.Body)
			}
			if tc.code != http.StatusOK {
				return
			}
			var body struct{ Data map[string]json.RawMessage }
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			var records []model.CronJobRecord
			if err := json.Unmarshal(body.Data[tc.listKey], &records); err != nil {
				t.Fatal(err)
			}
			if records == nil || len(records) != tc.wantCount ||
				(tc.wantFirstID != 0 && records[0].ID != tc.wantFirstID) {
				t.Fatalf("records = %+v, raw %s", records, w.Body)
			}
			if tc.method == http.MethodPost && len(body.Data) != 1 {
				t.Fatalf("legacy fields changed: %v", body.Data)
			}
			if tc.params == "?scope_name=cleanup&status=success&page_size=1" && string(body.Data["total"]) != "2" {
				t.Fatalf("wrong filtered total: %s", w.Body)
			}
		})
	}
}

func TestCronJobRecordRoutesRequireAdmin(t *testing.T) {
	const path = "/api/v1/admin/operations/cronjob/record/list"
	userRouter, _ := newOperationsPaginationRouter(t, model.RoleUser)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		w := httptest.NewRecorder()
		userRouter.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader("{}")))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s status %d: %s", method, w.Code, w.Body)
		}
	}
}
