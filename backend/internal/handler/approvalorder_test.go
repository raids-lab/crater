package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/util"
)

func TestCreateApprovalOrderJobStatus(t *testing.T) {
	for _, tc := range []struct {
		status  batch.JobPhase
		allowed bool
	}{
		{batch.Running, true}, {batch.Pending, true}, {model.Prequeue, true},
		{batch.Failed, false}, {batch.Completed, false}, {batch.Aborted, false},
		{batch.Terminated, false}, {model.Deleted, false}, {model.Freed, false},
	} {
		t.Run(string(tc.status), func(t *testing.T) {
			db := newApprovalOrderTestDB(t, tc.status)
			mgr := &ApprovalOrderMgr{}
			order := &model.ApprovalOrder{Name: "test-job", Type: model.ApprovalOrderTypeJob, Status: model.ApprovalOrderStatusPending}
			err := mgr.createApprovalOrder(t.Context(), query.Use(db), order, false)
			if tc.allowed && err != nil {
				t.Fatal(err)
			}
			if !tc.allowed && !errors.Is(err, errApprovalOrderJobInactive) {
				t.Fatalf("error = %v, want inactive job rejection", err)
			}
			var count int64
			if err := db.Model(&model.ApprovalOrder{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			want := int64(0)
			if tc.allowed {
				want = 1
			}
			if count != want {
				t.Fatalf("persisted orders = %d, want %d", count, want)
			}
			originalQuery := *query.Q
			query.SetDefault(db)
			t.Cleanup(func() { *query.Q = originalQuery })
			router := gin.New()
			router.Use(func(c *gin.Context) { util.SetJWTContext(c, util.JWTMessage{UserID: 1}) })
			router.POST("/orders", mgr.CreateApprovalOrder)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"name":"test-job","type":"job","approvalOrderExtensionHours":24}`))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)
			wantStatus := http.StatusBadRequest
			if tc.allowed {
				wantStatus = http.StatusOK
			}
			if recorder.Code != wantStatus {
				t.Fatalf("HTTP status = %d, want %d: %s", recorder.Code, wantStatus, recorder.Body.String())
			}
			if err := db.Model(&model.ApprovalOrder{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 2*want {
				t.Fatalf("persisted orders after request = %d, want %d", count, 2*want)
			}
		})
	}
}

func newApprovalOrderTestDB(t *testing.T, status batch.JobPhase) *gorm.DB {
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
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("CREATE TABLE jobs (id integer PRIMARY KEY, job_name text, status text, deleted_at datetime, updated_at datetime, locked_timestamp datetime)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO jobs (id, job_name, status) VALUES (1, 'test-job', ?)", status).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE approval_orders (id integer PRIMARY KEY, created_at datetime, updated_at datetime, deleted_at datetime, name text, type text, status text, content text, review_notes text, creator_id integer, reviewer_id integer)").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreateApprovalOrderAutoApprovalBoundary(t *testing.T) {
	for _, tc := range []struct {
		name           string
		hours          uint
		recentApproval bool
		want           model.ApprovalOrderStatus
	}{
		{"below limit", 11, false, model.ApprovalOrderStatusApproved},
		{"at limit", 12, false, model.ApprovalOrderStatusApproved},
		{"above limit", 13, false, model.ApprovalOrderStatusPending},
		{"at limit with recent approval", 12, true, model.ApprovalOrderStatusPending},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newApprovalOrderTestDB(t, batch.Running)
			originalQuery := *query.Q
			query.SetDefault(db)
			t.Cleanup(func() { *query.Q = originalQuery })
			if tc.recentApproval {
				previous := model.ApprovalOrder{Name: "previous-job", Type: model.ApprovalOrderTypeJob, Status: model.ApprovalOrderStatusApproved, CreatorID: 1}
				if err := db.Create(&previous).Error; err != nil {
					t.Fatal(err)
				}
			}
			mgr := &ApprovalOrderMgr{}
			router := gin.New()
			router.Use(func(c *gin.Context) { util.SetJWTContext(c, util.JWTMessage{UserID: 1}) })
			router.POST("/orders", mgr.CreateApprovalOrder)
			recorder := httptest.NewRecorder()
			body := fmt.Sprintf(`{"name":"test-job","type":"job","approvalOrderExtensionHours":%d}`, tc.hours)
			request := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("HTTP status = %d: %s", recorder.Code, recorder.Body.String())
			}
			var order model.ApprovalOrder
			if err := db.Where("name = ?", "test-job").First(&order).Error; err != nil {
				t.Fatal(err)
			}
			if order.Status != tc.want {
				t.Fatalf("order status = %s, want %s", order.Status, tc.want)
			}
			var job model.Job
			if err := db.First(&job).Error; err != nil {
				t.Fatal(err)
			}
			if locked := !job.LockedTimestamp.IsZero(); locked != (tc.want == model.ApprovalOrderStatusApproved) {
				t.Fatalf("job locked = %t, order status = %s", locked, tc.want)
			}
		})
	}
}

func TestCreateApprovalOrderRejectsStatusChangeWithoutExtendingLock(t *testing.T) {
	for _, tc := range []struct {
		hours     uint
		extension time.Duration
	}{
		{11, 11 * time.Hour},
		{12, 12 * time.Hour},
	} {
		t.Run(fmt.Sprintf("%d hours", tc.hours), func(t *testing.T) {
			db := newApprovalOrderTestDB(t, batch.Running)
			originalQuery := *query.Q
			query.SetDefault(db)
			t.Cleanup(func() { *query.Q = originalQuery })
			lockedUntil := time.Now().UTC().Add(6 * time.Hour).Truncate(time.Second)
			if err := db.Model(&model.Job{}).Where("id = ?", 1).Update("locked_timestamp", lockedUntil).Error; err != nil {
				t.Fatal(err)
			}

			// Change status after the initial check and eligibility lookup, before creation.
			statusChanged := false
			if err := db.Callback().Query().After("gorm:query").Register("test:restart_before_creation", func(tx *gorm.DB) {
				if tx.Statement.Table == "approval_orders" && !statusChanged {
					statusChanged = true
					if err := db.Model(&model.Job{}).Where("id = ?", 1).Update("status", batch.Restarting).Error; err != nil {
						t.Fatal(err)
					}
				}
			}); err != nil {
				t.Fatal(err)
			}

			sendAutoApprovalRequest(t, tc.hours, http.StatusBadRequest)
			if !statusChanged {
				t.Fatal("job status change was not injected")
			}
			assertApprovalOrderLockState(t, db, lockedUntil, 0)

			if err := db.Model(&model.Job{}).Where("id = ?", 1).Update("status", batch.Running).Error; err != nil {
				t.Fatal(err)
			}
			sendAutoApprovalRequest(t, tc.hours, http.StatusOK)
			assertApprovalOrderLockState(t, db, lockedUntil.Add(tc.extension), 1)
		})
	}
}

func TestCreateApprovalOrderInsertFailureRollsBackLock(t *testing.T) {
	db := newApprovalOrderTestDB(t, batch.Running)
	originalQuery := *query.Q
	query.SetDefault(db)
	t.Cleanup(func() { *query.Q = originalQuery })
	lockedUntil := time.Now().UTC().Add(6 * time.Hour).Truncate(time.Second)
	if err := db.Model(&model.Job{}).Where("id = ?", 1).Update("locked_timestamp", lockedUntil).Error; err != nil {
		t.Fatal(err)
	}

	insertAttempted := false
	if err := db.Callback().Create().Before("gorm:create").Register("test:fail_order_insert", func(tx *gorm.DB) {
		if tx.Statement.Table == "approval_orders" {
			insertAttempted = true
			_ = tx.AddError(bizerr.Internal.DatabaseError.New("injected approval order insert failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}

	sendAutoApprovalRequest(t, 12, http.StatusInternalServerError)
	if !insertAttempted {
		t.Fatal("approval order insert failure was not injected")
	}
	assertApprovalOrderLockState(t, db, lockedUntil, 0)

	if err := db.Callback().Create().Remove("test:fail_order_insert"); err != nil {
		t.Fatal(err)
	}
	sendAutoApprovalRequest(t, 12, http.StatusOK)
	assertApprovalOrderLockState(t, db, lockedUntil.Add(12*time.Hour), 1)
}

func sendAutoApprovalRequest(t *testing.T, hours uint, wantStatus int) {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) { util.SetJWTContext(c, util.JWTMessage{UserID: 1}) })
	router.POST("/orders", (&ApprovalOrderMgr{}).CreateApprovalOrder)
	recorder := httptest.NewRecorder()
	body := fmt.Sprintf(`{"name":"test-job","type":"job","approvalOrderExtensionHours":%d}`, hours)
	request := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != wantStatus {
		t.Fatalf("HTTP status = %d, want %d: %s", recorder.Code, wantStatus, recorder.Body.String())
	}
}

func assertApprovalOrderLockState(t *testing.T, db *gorm.DB, wantLock time.Time, wantOrders int64) {
	t.Helper()
	var job model.Job
	if err := db.First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if !job.LockedTimestamp.Equal(wantLock) {
		t.Errorf("locked until = %s, want %s", job.LockedTimestamp, wantLock)
	}
	var count int64
	if err := db.Model(&model.ApprovalOrder{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != wantOrders {
		t.Errorf("persisted orders = %d, want %d", count, wantOrders)
	}
}
