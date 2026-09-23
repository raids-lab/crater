package handler

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindAccountMemberPageQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/?page=2&page_size=20&search=%20jerry%20&role=2&role=3&accessmode=2&sort=-role,name",
		http.NoBody,
	)

	request, roles, accessModes, err := bindAccountMemberPageQuery(ctx)
	if err != nil {
		t.Fatalf("bindAccountMemberPageQuery() error = %v", err)
	}
	if request.Page != 2 || request.PageSize != 20 || request.Search != "jerry" {
		t.Fatalf("unexpected request: %+v", request)
	}
	if !reflect.DeepEqual(roles, []uint{2, 3}) {
		t.Fatalf("roles = %v", roles)
	}
	if !reflect.DeepEqual(accessModes, []uint{2}) {
		t.Fatalf("accessModes = %v", accessModes)
	}
}

func TestBindAccountMemberPageQueryRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"page=0",
		"page_size=201",
		"role=1",
		"accessmode=4",
		"sort=quota",
		"sort=name,name",
	}
	for _, rawQuery := range tests {
		t.Run(rawQuery, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+rawQuery, http.NoBody)
			if _, _, _, err := bindAccountMemberPageQuery(ctx); err == nil {
				t.Fatal("bindAccountMemberPageQuery() error = nil, want validation error")
			}
		})
	}
}

func TestAccountMemberSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	got := accountMemberSortClauses("-role,name")
	want := []string{"user_accounts.role DESC", "users.name ASC", "users.id ASC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("accountMemberSortClauses() = %v, want %v", got, want)
	}
}

func TestBindBillingMemberUserIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?user_id=3&user_id=7&user_id=3", http.NoBody)

	got, err := bindBillingMemberUserIDs(ctx)
	if err != nil {
		t.Fatalf("bindBillingMemberUserIDs() error = %v", err)
	}
	if !reflect.DeepEqual(got, []uint{3, 7}) {
		t.Fatalf("bindBillingMemberUserIDs() = %v", got)
	}
}

func TestBindBillingMemberUserIDsRejectsInvalidValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?user_id=0", http.NoBody)
	if _, err := bindBillingMemberUserIDs(ctx); err == nil {
		t.Fatal("bindBillingMemberUserIDs() error = nil, want validation error")
	}
}
