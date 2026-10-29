package handler

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"gorm.io/gorm"
)

func TestBindAdminAccountPageQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?page=2&page_size=25&search=%20lab%20", http.NoBody)

	request, err := bindAdminAccountPageQuery(ctx)
	if err != nil {
		t.Fatalf("bindAdminAccountPageQuery() error = %v", err)
	}
	if request.Page != 2 || request.PageSize != 25 || request.Search != "lab" || request.Sort != adminAccountDefaultSort {
		t.Fatalf("unexpected request: %+v", request)
	}
}

func TestValidateAdminAccountSort(t *testing.T) {
	if err := validateAdminAccountSort("-nickname,createdAt"); err != nil {
		t.Fatalf("validateAdminAccountSort() error = %v", err)
	}
	if err := validateAdminAccountSort("quota"); err == nil {
		t.Fatal("validateAdminAccountSort() error = nil, want unsupported sort error")
	}
}

func TestAdminAccountSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	got := adminAccountSortClauses("-nickname,createdAt")
	want := []string{"nickname DESC", "created_at ASC", "id DESC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("adminAccountSortClauses() = %v, want %v", got, want)
	}
}

func TestAccountMemberPageSortsBeforePagingAndFilters(t *testing.T) {
	newDatasetPageTestRouter(t, nil)
	db := query.User.WithContext(t.Context()).UnderlyingDB().Session(&gorm.Session{NewDB: true})
	if err := db.AutoMigrate(&model.UserAccount{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]model.UserAccount{
		{UserID: 1, AccountID: 1, Role: model.RoleUser, AccessMode: model.AccessMode(2)},
		{UserID: 2, AccountID: 1, Role: model.RoleAdmin, AccessMode: model.AccessMode(3)},
	}).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query string
		want  uint
		total int64
	}{
		{"/?sort=-name&page_size=1", 2, 2},
		{"/?sort=-name&page=2&page_size=1", 1, 2},
		{"/?role=2", 1, 1},
		{"/?accessmode=3", 2, 1},
		{"/?search=ALICE", 1, 1},
	} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodGet, tc.query, http.NoBody)
		request, err := bindAccountMembersPageQuery(ctx)
		if err != nil {
			t.Fatal(err)
		}
		items, total, err := (&AccountMgr{}).getUsersInAccountPage(ctx, 1, request)
		if err != nil || total != tc.total || len(items) != 1 || items[0].ID != tc.want {
			t.Fatalf("query=%s items=%+v total=%d err=%v", tc.query, items, total, err)
		}
	}
}
