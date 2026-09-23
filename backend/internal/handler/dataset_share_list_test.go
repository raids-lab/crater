package handler

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindDatasetSharePageQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/?page=2&page_size=20&search=%20jerry%20&sort=-name",
		http.NoBody,
	)
	request, err := bindDatasetSharePageQuery(ctx)
	if err != nil {
		t.Fatalf("bindDatasetSharePageQuery() error = %v", err)
	}
	if request.Page != 2 || request.PageSize != 20 || request.Search != "jerry" || request.Sort != "-name" {
		t.Fatalf("unexpected request: %+v", request)
	}
}

func TestBindDatasetSharePageQueryRejectsInvalidValues(t *testing.T) {
	tests := []string{"page=0", "page_size=201", "sort=createdAt", "sort=name,-name"}
	for _, rawQuery := range tests {
		t.Run(rawQuery, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+rawQuery, http.NoBody)
			if _, err := bindDatasetSharePageQuery(ctx); err == nil {
				t.Fatal("bindDatasetSharePageQuery() error = nil, want validation error")
			}
		})
	}
}

func TestDatasetShareSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	got := datasetShareSortClauses("-name", "users.name", "users.id")
	want := []string{"users.name DESC", "users.id ASC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("datasetShareSortClauses() = %v, want %v", got, want)
	}
}
