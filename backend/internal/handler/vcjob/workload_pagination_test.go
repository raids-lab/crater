package vcjob

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
)

func TestBatchWorkloadsPageInDatabase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	connection, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = connection.Close() })
	for _, statement := range []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, nickname TEXT, deleted_at DATETIME)",
		"CREATE TABLE accounts (id INTEGER PRIMARY KEY, nickname TEXT, deleted_at DATETIME)",
		"CREATE TABLE jobs (id INTEGER PRIMARY KEY, job_name TEXT, creation_timestamp DATETIME, user_id INTEGER, account_id INTEGER, deleted_at DATETIME)",
		"INSERT INTO jobs (id, job_name, creation_timestamp) VALUES (1, 'first', '2026-01-01'), (2, 'second', '2026-01-02'), (3, 'third', '2026-01-03')",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	query.SetDefault(db)
	var selected string
	if err := db.Callback().Query().After("gorm:query").Register("check-page", func(tx *gorm.DB) {
		sql := tx.Statement.SQL.String()
		if tx.Statement.Table == "jobs" && !strings.Contains(sql, "count(") {
			selected = sql
		}
	}); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/?days=-1&page=2&page_size=1&workload_kind=volcano-job", http.NoBody)
	manager := &VolcanojobMgr{}
	manager.listWorkloads(ctx, -1, jobListScope{})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data resputil.Page[WorkloadResp] `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Total != 3 || len(response.Data.Items) != 1 || response.Data.Items[0].JobName != "second" {
		t.Fatalf("unexpected page: %s", recorder.Body.String())
	}
	if !strings.Contains(selected, "LIMIT 1 OFFSET 1") {
		t.Fatalf("query loaded unbounded jobs: %s", selected)
	}
}
