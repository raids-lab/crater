package image

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"gorm.io/gorm"
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
	got := imageListSortClauses("-createdAt,image")
	want := []string{"images.created_at DESC", "images.image_link ASC", "images.id DESC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("imageListSortClauses() = %v, want %v", got, want)
	}
}

func TestImageBuildIDIsResolvedByImagePackName(t *testing.T) {
	kanikos := []*model.Kaniko{
		{Model: gorm.Model{ID: 5814}, ImagePackName: "test-build-001"},
	}
	got := imageBuildIDsByImagePackName(kanikos)
	if got["test-build-001"] != 5814 {
		t.Fatalf("build ID = %d, want 5814", got["test-build-001"])
	}
}

func TestImageVisibilityExpressionKeepsExpectedPrecedence(t *testing.T) {
	db := setupKanikoListTestDB(t)
	expression := imageVisibilityExpression(context.Background(), 23, 7, false)
	statement := db.Session(&gorm.Session{DryRun: true}).Model(&model.Image{}).
		Select("images.id, ? AS image_share_status", expression).Find(&[]visibleImageRow{}).Statement
	want := []any{true, uint(model.DefaultAccountID), "Public", uint(7), "AccountShare", uint(23), "Private", uint(23), "UserShare"}
	if !reflect.DeepEqual(statement.Vars, want) {
		t.Fatalf("visibility bindings = %#v, want %#v", statement.Vars, want)
	}
	if strings.Contains(statement.SQL.String(), "account_id = 7") || strings.Contains(statement.SQL.String(), "user_id = 23") {
		t.Fatalf("visibility IDs must be bound, SQL = %s", statement.SQL.String())
	}
}
