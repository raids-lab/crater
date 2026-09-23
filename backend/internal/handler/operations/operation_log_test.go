package operations

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOperationLogPageQueryDefaultsAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/?page=2&page_size=25&search=%20node%20&sort=-createdAt,operationType&operation_type=DrainNode",
		http.NoBody,
	)

	request, err := (&OperationLogMgr{}).bindPageQuery(ctx)
	if err != nil {
		t.Fatalf("bindPageQuery() error = %v", err)
	}

	if request.Page != 2 || request.PageSize != 25 || request.Search != "node" {
		t.Fatalf("unexpected request: %+v", request)
	}
	if request.Sort != "-createdAt,operationType" {
		t.Fatalf("unexpected sort: %q", request.Sort)
	}
	if got := firstOperationLogQueryValue(ctx.QueryArray("operation_type")); got != "DrainNode" {
		t.Fatalf("unexpected operation type: %q", got)
	}
}

func TestOperationLogPageQueryRejectsUnsupportedSort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?sort=message", http.NoBody)

	if _, err := (&OperationLogMgr{}).bindPageQuery(ctx); err == nil {
		t.Fatal("bindPageQuery() error = nil, want unsupported sort error")
	}
}

func TestOperationLogSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	got := operationLogSortClauses("-createdAt,status")
	want := []string{"created_at DESC", "status ASC", "id DESC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("operationLogSortClauses() = %v, want %v", got, want)
	}
}
