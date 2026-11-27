package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

func TestApprovalOrdersByNameRespectResourceAccess(t *testing.T) {
	const (
		name      = "shared-name"
		jobFilter = "?type=job"
	)
	seedApprovalOrderVisibility(t, name)

	for _, tt := range []struct {
		name      string
		userID    uint
		accountID uint
		role      model.Role
		filter    string
		status    int
		want      []uint
	}{
		{"job detail preserves legacy job orders", 11, 2, model.RoleUser, jobFilter, http.StatusOK, []uint{101}},
		{"job owner in another account", 11, 3, model.RoleUser, jobFilter, http.StatusOK, nil},
		{"another user in the same account", 12, 2, model.RoleUser, jobFilter, http.StatusOK, nil},
		{"CLI sees job and granted datasets", 11, 2, model.RoleUser, "", http.StatusOK, []uint{101, 102, 103, 104}},
		{"unrelated user sees only the public dataset", 13, 3, model.RoleUser, "", http.StatusOK, []uint{104}},
		{"dataset filter keeps only granted target IDs", 11, 2, model.RoleUser, "?type=dataset", http.StatusOK, []uint{102, 103, 104}},
		{"admin sees live targets across accounts", 12, 3, model.RoleAdmin, "", http.StatusOK, []uint{101, 102, 103, 104, 105, 107}},
		{"invalid filter", 11, 2, model.RoleUser, "?type=invalid", http.StatusBadRequest, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				util.SetJWTContext(c, util.JWTMessage{
					UserID: tt.userID, AccountID: tt.accountID, RolePlatform: tt.role,
				})
			})
			mgr := &ApprovalOrderMgr{}
			mgr.RegisterProtected(router.Group("/api/v1/approvalorder"))
			recorder := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/approvalorder/name/"+name+tt.filter, http.NoBody)
			router.ServeHTTP(recorder, req)
			if recorder.Code != tt.status {
				t.Fatalf("HTTP status = %d, want %d: %s", recorder.Code, tt.status, recorder.Body)
			}
			if tt.status != http.StatusOK {
				return
			}
			var response resputil.Response[[]ApprovalOrderResp]
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Data == nil {
				t.Fatal("expected a JSON array for the order list")
			}
			got := make([]uint, 0, len(response.Data))
			for _, order := range response.Data {
				got = append(got, order.ID)
				if order.ID == 101 && (order.ReviewedAt == nil || order.ReviewedAt.Sub(order.CreatedAt) != 10*time.Minute) {
					t.Errorf("reviewedAt = %v, want ten minutes after %v", order.ReviewedAt, order.CreatedAt)
				}
			}
			slices.Sort(got)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("order IDs = %v, want %v", got, tt.want)
			}
		})
	}
}

func seedApprovalOrderVisibility(t *testing.T, name string) {
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
	if err := db.AutoMigrate(&model.ApprovalOrder{}, &model.Dataset{}, &model.UserDataset{}, &model.AccountDataset{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE jobs (id INTEGER PRIMARY KEY, job_name TEXT UNIQUE, user_id INTEGER, account_id INTEGER, deleted_at DATETIME)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO jobs (id, job_name, user_id, account_id) VALUES (40, ?, 11, 2)`, name).Error; err != nil {
		t.Fatal(err)
	}
	query.SetDefault(db)
	insert := func(record any) {
		t.Helper()
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	for id := uint(20); id <= 25; id++ {
		insert(&model.Dataset{Model: gorm.Model{ID: id}, Name: name, URL: "dataset-path"})
	}
	insert(&model.UserDataset{UserID: 11, DatasetID: 20})
	insert(&model.AccountDataset{AccountID: 2, DatasetID: 21})
	insert(&model.AccountDataset{AccountID: model.DefaultAccountID, DatasetID: 22})
	insert(&model.UserDataset{UserID: 12, DatasetID: 23})
	insert(&model.UserDataset{UserID: 11, DatasetID: 24})
	revoked := &model.UserDataset{UserID: 11, DatasetID: 25}
	insert(revoked)
	if err := db.Delete(revoked).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&model.Dataset{}, 24).Error; err != nil {
		t.Fatal(err)
	}
	for _, order := range []struct {
		id       uint
		resource uint
		kind     model.ApprovalOrderType
	}{
		{101, 1, model.ApprovalOrderTypeJob},
		{102, 20, model.ApprovalOrderTypeDataset},
		{103, 21, model.ApprovalOrderTypeDataset},
		{104, 22, model.ApprovalOrderTypeDataset},
		{105, 23, model.ApprovalOrderTypeDataset},
		{106, 24, model.ApprovalOrderTypeDataset},
		{107, 25, model.ApprovalOrderTypeDataset},
		{108, 0, model.ApprovalOrderTypeDataset},
	} {
		record := &model.ApprovalOrder{
			Model: gorm.Model{ID: order.id}, Name: name, Type: order.kind, CreatorID: 99,
			Content: datatypes.NewJSONType(model.ApprovalOrderContent{ApprovalOrderTypeID: order.resource}),
		}
		if order.kind == model.ApprovalOrderTypeJob {
			record.Status = model.ApprovalOrderStatusApproved
			record.CreatedAt = time.Date(2026, time.November, 9, 1, 0, 0, 0, time.UTC)
			record.UpdatedAt = record.CreatedAt.Add(10 * time.Minute)
		}
		insert(record)
	}
	insert(&model.ApprovalOrder{
		Name: "unrelated-name", Type: model.ApprovalOrderTypeDataset, CreatorID: 99,
		Content: datatypes.NewJSONType(model.ApprovalOrderContent{ApprovalOrderTypeID: 22}),
	})
}
