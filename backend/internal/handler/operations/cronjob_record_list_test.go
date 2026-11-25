package operations

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindCronJobRecordPageQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/?page=2&page_size=20&search=%20cleanup%20&status=success&sort=-executeTime,name",
		http.NoBody,
	)

	request, statuses, err := bindCronJobRecordPageQuery(ctx)
	if err != nil {
		t.Fatalf("bindCronJobRecordPageQuery() error = %v", err)
	}
	if request.Page != 2 || request.PageSize != 20 || request.Search != "cleanup" {
		t.Fatalf("unexpected request: %+v", request)
	}
	if !reflect.DeepEqual(statuses, []string{"success"}) {
		t.Fatalf("statuses = %v", statuses)
	}
}

func TestBindCronJobRecordPageQueryRejectsStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?status=pending", http.NoBody)

	if _, _, err := bindCronJobRecordPageQuery(ctx); err == nil {
		t.Fatal("bindCronJobRecordPageQuery() error = nil, want invalid status error")
	}
}

func TestCronJobRecordSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	got := cronJobRecordSortClauses("-executeTime,name")
	want := []string{"execute_time DESC", "name ASC", "id DESC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cronJobRecordSortClauses() = %v, want %v", got, want)
	}
}
