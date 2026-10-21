package handler

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindModelDownloadPageQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/?page=2&page_size=20&search=%20llama%20&category=model&status=Ready&sort=-updatedAt,name",
		http.NoBody,
	)

	request, statuses, err := bindModelDownloadPageQuery(ctx)
	if err != nil {
		t.Fatalf("bindModelDownloadPageQuery() error = %v", err)
	}
	if request.Page != 2 || request.PageSize != 20 || request.Search != "llama" || request.Category != "model" {
		t.Fatalf("unexpected request: %+v", request)
	}
	if !reflect.DeepEqual(statuses, []string{"Ready"}) {
		t.Fatalf("statuses = %v", statuses)
	}
}

func TestBindModelDownloadPageQueryRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"page=0",
		"page_size=201",
		"category=other",
		"status=Unknown",
		"sort=source",
		"sort=name,name",
	}
	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+query, http.NoBody)
			if _, _, err := bindModelDownloadPageQuery(ctx); err == nil {
				t.Fatal("bindModelDownloadPageQuery() error = nil, want validation error")
			}
		})
	}
}

func TestModelDownloadSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	got := modelDownloadSortClauses("-updatedAt,name")
	want := []string{"model_downloads.updated_at DESC", "model_downloads.name ASC", "model_downloads.id DESC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("modelDownloadSortClauses() = %v, want %v", got, want)
	}
}
