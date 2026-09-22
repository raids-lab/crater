package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newJobTemplateListTestContext(rawQuery string) *gin.Context {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/jobtemplate?"+rawQuery, http.NoBody)
	return context
}

func TestBindJobTemplateListQueryDefaults(t *testing.T) {
	request, err := bindJobTemplateListQuery(newJobTemplateListTestContext(""))
	if err != nil {
		t.Fatalf("bindJobTemplateListQuery returned error: %v", err)
	}
	if request.Page != 1 || request.PageSize != 10 {
		t.Fatalf("unexpected pagination defaults: %#v", request)
	}
	if request.Owner != "all" || request.Sort != "-createdAt" {
		t.Fatalf("unexpected filter defaults: %#v", request)
	}
}

func TestBindJobTemplateListQueryNormalizesSearch(t *testing.T) {
	request, err := bindJobTemplateListQuery(
		newJobTemplateListTestContext("page=2&page_size=20&search=++demo+&owner=mine&sort=createdAt"),
	)
	if err != nil {
		t.Fatalf("bindJobTemplateListQuery returned error: %v", err)
	}
	if request.Page != 2 || request.PageSize != 20 {
		t.Fatalf("unexpected pagination: %#v", request)
	}
	if request.Search != "demo" || request.Owner != "mine" || request.Sort != "createdAt" {
		t.Fatalf("unexpected normalized query: %#v", request)
	}
	if request.offset() != 20 {
		t.Fatalf("unexpected offset: %d", request.offset())
	}
}

func TestBindJobTemplateListQueryRejectsInvalidValues(t *testing.T) {
	testCases := []string{
		"page=0",
		"page_size=201",
		"owner=unknown",
		"sort=name",
		"search=" + strings.Repeat("a", jobTemplateMaxSearchRunes+1),
	}

	for _, rawQuery := range testCases {
		t.Run(rawQuery, func(t *testing.T) {
			if _, err := bindJobTemplateListQuery(newJobTemplateListTestContext(rawQuery)); err == nil {
				t.Fatal("expected query validation error")
			}
		})
	}
}
