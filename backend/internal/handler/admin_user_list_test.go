package handler

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindAdminUserPageQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?page=3&page_size=20&search=%20alice%20", http.NoBody)

	request, err := bindAdminUserPageQuery(ctx)
	if err != nil {
		t.Fatalf("bindAdminUserPageQuery() error = %v", err)
	}
	if request.Page != 3 || request.PageSize != 20 || request.Search != "alice" || request.Sort != adminUserDefaultSort {
		t.Fatalf("unexpected request: %+v", request)
	}
}

func TestParseAdminUserFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?role=1&role=3&status=2&banned=true", http.NoBody)

	roles, err := parseAdminUserIntFilters(ctx, "role")
	if err != nil || !reflect.DeepEqual(roles, []uint8{1, 3}) {
		t.Fatalf("roles = %v, error = %v", roles, err)
	}
	statuses, err := parseAdminUserIntFilters(ctx, "status")
	if err != nil || !reflect.DeepEqual(statuses, []uint8{2}) {
		t.Fatalf("statuses = %v, error = %v", statuses, err)
	}
	banned, err := parseAdminUserBannedFilters(ctx)
	if err != nil || !reflect.DeepEqual(banned, []bool{true}) {
		t.Fatalf("banned = %v, error = %v", banned, err)
	}
}

func TestAdminUserSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	got := adminUserSortClauses("-createdAt,name")
	want := []string{"created_at DESC", "name ASC", "id DESC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("adminUserSortClauses() = %v, want %v", got, want)
	}
}
