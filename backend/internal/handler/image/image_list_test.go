package image

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindImageListPageQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/?page=2&page_size=20&search=%20cuda%20&imageShareStatus=Public&imageShareStatus=AccountShare&sort=-createdAt,image",
		http.NoBody,
	)

	request, statuses, err := bindImageListPageQuery(ctx)
	if err != nil {
		t.Fatalf("bindImageListPageQuery() error = %v", err)
	}
	if request.Page != 2 || request.PageSize != 20 || request.Search != "cuda" {
		t.Fatalf("unexpected request: %+v", request)
	}
	if !reflect.DeepEqual(statuses, []string{"Public", "AccountShare"}) {
		t.Fatalf("statuses = %v", statuses)
	}
}

func TestBindImageListPageQueryRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"page=0",
		"page_size=201",
		"imageShareStatus=Unknown",
		"sort=archs",
		"sort=image,image",
	}
	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+query, http.NoBody)
			if _, _, err := bindImageListPageQuery(ctx); err == nil {
				t.Fatal("bindImageListPageQuery() error = nil, want validation error")
			}
		})
	}
}

func TestImageListSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	got := imageListSortClauses("-createdAt,image", "visibility_case")
	want := []string{"images.created_at DESC", "images.image_link ASC", "images.id DESC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("imageListSortClauses() = %v, want %v", got, want)
	}
}

func TestImageVisibilityExpressionKeepsExpectedPrecedence(t *testing.T) {
	expression := imageVisibilityExpression(23, 7, false)
	parts := []string{"THEN 'Public'", "account_id = 7", "THEN 'AccountShare'", "images.user_id = 23", "THEN 'Private'", "user_id = 23", "THEN 'UserShare'"}
	position := -1
	for _, part := range parts {
		next := strings.Index(expression[position+1:], part)
		if next < 0 {
			t.Fatalf("visibility expression missing %q: %s", part, expression)
		}
		position += next + 1
	}
}
