package handler

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
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
