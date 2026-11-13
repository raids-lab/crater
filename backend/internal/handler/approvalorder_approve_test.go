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
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/utils"
)

const testApprovalJob = "running-job"

func TestJobApprovalAndLock(t *testing.T) {
	db, router, initial := newJobApprovalFixture(t)
	checkJobApprovalSuccesses(t, db, router, initial)
	checkJobApprovalFailures(t, db, router, initial)
}

func TestConcurrentJobApprovalExtendsOnce(t *testing.T) {
	db, router, initial := newJobApprovalFixture(t)
	statuses := make(chan int, 2)
	for range 2 {
		go func() {
			w := sendJobOrderRequest(t, router, 1, "approve-and-lock", `{}`)
			statuses <- w.Code
		}()
	}
	count := make(map[int]int)
	for range 2 {
		count[<-statuses]++
	}
	if count[http.StatusOK] != 1 || count[http.StatusConflict] != 1 {
		t.Fatalf("status counts = %v, want one approval and one conflict", count)
	}
	checkStoredJobLock(t, db, testApprovalJob, initial.Add(8*time.Hour))
}

func TestAutoApprovalRecordsLockAndRollsBackOnInsertFailure(t *testing.T) {
	db, _, initial := newJobApprovalFixture(t)
	newOrder := func(id uint) *model.ApprovalOrder {
		return &model.ApprovalOrder{
			Model: gorm.Model{ID: id}, Name: testApprovalJob, Type: model.ApprovalOrderTypeJob,
			Content: datatypes.NewJSONType(model.ApprovalOrderContent{ApprovalOrderExtensionHours: 2}),
		}
	}
	if approved, err := createAutoApprovedJobOrder(t.Context(), newOrder(1)); approved || err == nil {
		t.Fatalf("duplicate insert: approved = %v, err = %v", approved, err)
	}
	checkStoredJobLock(t, db, testApprovalJob, initial)
	order := newOrder(10)
	if approved, err := createAutoApprovedJobOrder(t.Context(), order); !approved || err != nil {
		t.Fatalf("auto approval: approved = %v, err = %v", approved, err)
	}
	var stored model.ApprovalOrder
	if err := db.First(&stored, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if lock := stored.Content.Data().ApprovedLock; lock == nil || lock.DurationMinutes != 120 ||
		stored.Status != model.ApprovalOrderStatusApproved || stored.ReviewerID != nil {
		t.Fatalf("unexpected automatic approval: %+v, lock: %+v", stored, lock)
	}
	checkStoredJobLock(t, db, testApprovalJob, initial.Add(2*time.Hour))
}

func sendJobOrderRequest(t *testing.T, router *gin.Engine, id uint, suffix, body string) *httptest.ResponseRecorder {
	t.Helper()
	path := fmt.Sprintf("/api/v1/admin/approvalorder/%d/%s", id, suffix)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func checkJobApprovalSuccesses(t *testing.T, db *gorm.DB, router *gin.Engine, initial time.Time) {
	t.Helper()

	for _, tt := range []struct {
		name          string
		id            uint
		body          string
		minutes       int64
		permanent     bool
		wantRemaining time.Duration
	}{
		{"default requested duration", 1, `{}`, 480, false, 8 * time.Hour},
		{"web duration override", 2, `{"lock":{"days":0,"hours":1,"minutes":0}}`, 60, false, 9 * time.Hour},
		{"CLI duration override", 3, `{"lock":{"hours":1,"minutes":30},"reviewNotes":"needed"}`, 90, false, 10*time.Hour + 30*time.Minute},
		{"permanent lock", 6, `{"lock":{"isPermanent":true}}`, 0, true, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := sendJobOrderRequest(t, router, tt.id, "approve-and-lock", tt.body)
			if w.Code != http.StatusOK {
				t.Fatalf("HTTP status = %d, want 200: %s", w.Code, w.Body)
			}
			var reply resputil.Response[ApproveJobOrderResp]
			if err := json.Unmarshal(w.Body.Bytes(), &reply); err != nil {
				t.Fatal(err)
			}
			expected := initial.Add(tt.wantRemaining)
			jobName := testApprovalJob
			if tt.permanent {
				expected = utils.GetPermanentTime()
				jobName = "permanent-job"
			}
			if lock := reply.Data.ApprovedLock; lock.DurationMinutes != tt.minutes ||
				lock.IsPermanent != tt.permanent || !lock.LockedUntil.Equal(expected) {
				t.Fatalf("unexpected approved lock: %+v", lock)
			}
			order := checkStoredApproval(t, db, tt.id, expected)
			if tt.id == 3 && order.ReviewNotes != "needed" {
				t.Errorf("review notes = %q, want needed", order.ReviewNotes)
			}
			checkStoredJobLock(t, db, jobName, expected)
		})
	}
}

func checkStoredApproval(t *testing.T, db *gorm.DB, id uint, expected time.Time) model.ApprovalOrder {
	t.Helper()
	var order model.ApprovalOrder
	if err := db.First(&order, id).Error; err != nil {
		t.Fatal(err)
	}
	content := order.Content.Data()
	if order.Status != model.ApprovalOrderStatusApproved || ptr.Deref(order.ReviewerID, 0) != 9 ||
		content.ApprovalOrderExtensionHours != 8 || content.ApprovalOrderReason != "original reason" ||
		content.ApprovedLock == nil || !content.ApprovedLock.LockedUntil.Equal(expected) {
		t.Fatalf("unexpected recorded approval: %+v, content: %+v", order, content)
	}
	return order
}

func checkJobApprovalFailures(t *testing.T, db *gorm.DB, router *gin.Engine, initial time.Time) {
	t.Helper()

	if err := db.Exec(`CREATE TRIGGER reject_order_update BEFORE UPDATE ON approval_orders
		WHEN NEW.id = 4 BEGIN SELECT RAISE(ABORT, 'test write failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		id     uint
		suffix string
		body   string
		status int
	}{
		{"repeat does not extend again", 1, "approve-and-lock", `{}`, http.StatusConflict},
		{"order update failure rolls back the lock", 4, "approve-and-lock", `{}`, http.StatusInternalServerError},
		{"stopped job stays pending", 5, "approve-and-lock", `{}`, http.StatusConflict},
		{"old endpoint cannot approve job without locking", 5, "review", `{"status":"Approved"}`, http.StatusBadRequest},
		{"invalid permanent and finite duration", 7, "approve-and-lock", `{"lock":{"isPermanent":true,"hours":1}}`, http.StatusBadRequest},
		{"missing job stays pending", 8, "approve-and-lock", `{}`, http.StatusNotFound},
		{"dataset approval still uses review", 9, "review", `{"status":"Approved"}`, http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := sendJobOrderRequest(t, router, tt.id, tt.suffix, tt.body)
			if w.Code != tt.status {
				t.Fatalf("HTTP status = %d, want %d: %s", w.Code, tt.status, w.Body)
			}
			checkStoredJobLock(t, db, testApprovalJob, initial.Add(10*time.Hour+30*time.Minute))
			checkStoredJobLock(t, db, "stopped-job", initial)
			var order model.ApprovalOrder
			if err := db.First(&order, tt.id).Error; err != nil {
				t.Fatal(err)
			}
			if tt.id != 1 && tt.id != 9 && (order.Status != model.ApprovalOrderStatusPending || order.Content.Data().ApprovedLock != nil) {
				t.Fatalf("failed approval changed order state: %+v", order)
			}
		})
	}
}

func newJobApprovalFixture(t *testing.T) (*gorm.DB, *gin.Engine, time.Time) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&model.ApprovalOrder{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE jobs (id INTEGER PRIMARY KEY, job_name TEXT UNIQUE, status TEXT,
		locked_timestamp DATETIME, deleted_at DATETIME, updated_at DATETIME)`).Error; err != nil {
		t.Fatal(err)
	}
	initial := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	for name, status := range map[string]string{testApprovalJob: "Running", "stopped-job": "Completed", "permanent-job": "Running"} {
		if err := db.Exec(`INSERT INTO jobs (job_name, status, locked_timestamp) VALUES (?, ?, ?)`, name, status, initial).Error; err != nil {
			t.Fatal(err)
		}
	}
	id := uint(1)
	for _, name := range []string{testApprovalJob, testApprovalJob, testApprovalJob, testApprovalJob,
		"stopped-job", "permanent-job", testApprovalJob, "missing-job", "dataset-order"} {
		order := model.ApprovalOrder{
			Model: gorm.Model{ID: id}, Name: name, Type: model.ApprovalOrderTypeJob, CreatorID: 1,
			Content: datatypes.NewJSONType(model.ApprovalOrderContent{
				ApprovalOrderTypeID: 1, ApprovalOrderExtensionHours: 8, ApprovalOrderReason: "original reason",
			}),
		}
		if name == "dataset-order" {
			order.Type = model.ApprovalOrderTypeDataset
		}
		if err := db.Create(&order).Error; err != nil {
			t.Fatal(err)
		}
		id++
	}
	query.SetDefault(db)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		util.SetJWTContext(c, util.JWTMessage{UserID: 9, RolePlatform: model.RoleAdmin})
	})
	(&ApprovalOrderMgr{}).RegisterAdmin(router.Group("/api/v1/admin/approvalorder"))
	return db, router, initial
}

func checkStoredJobLock(t *testing.T, db *gorm.DB, jobName string, want time.Time) {
	t.Helper()
	var state struct{ LockedTimestamp time.Time }
	if err := db.Table("jobs").Select("locked_timestamp").Where("job_name = ?", jobName).Take(&state).Error; err != nil {
		t.Fatal(err)
	}
	if !state.LockedTimestamp.Equal(want) {
		t.Fatalf("%s lock = %v, want %v", jobName, state.LockedTimestamp, want)
	}
}
