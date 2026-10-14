package image

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
)

func setupKanikoListTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Kaniko{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	query.SetDefault(db)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

func createKanikoListTestUsers(t *testing.T, db *gorm.DB, names ...string) {
	t.Helper()
	nextID := uint(1)
	for _, name := range names {
		user := model.User{
			Model:    gorm.Model{ID: nextID},
			Name:     name,
			Nickname: "nickname-" + name,
			Role:     model.RoleUser,
			Status:   model.StatusActive,
			Space:    "/tmp/" + name,
		}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create test user %q: %v", name, err)
		}
		nextID++
	}
}

func createKanikoListTestRecords(t *testing.T, db *gorm.DB, records ...model.Kaniko) {
	t.Helper()
	for index := range records {
		record := records[index]
		if record.ImagePackName == "" {
			record.ImagePackName = fmt.Sprintf("build-%d", record.ID)
		}
		if record.ImageLink == "" {
			record.ImageLink = fmt.Sprintf("registry.example/build-%d:latest", record.ID)
		}
		if record.NameSpace == "" {
			record.NameSpace = "test"
		}
		if record.Status == "" {
			record.Status = model.BuildJobInitial
		}
		if record.BuildSource == "" {
			record.BuildSource = model.Dockerfile
		}
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("create test image build %d: %v", record.ID, err)
		}
	}
}

func requestKanikoListPage(
	t *testing.T,
	db *gorm.DB,
	userID *uint,
	rawQuery string,
) resputil.Page[KanikoInfo] {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+rawQuery, http.NoBody)

	(&ImagePackMgr{}).listKanikoPageWithDB(ctx, db, userID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list image builds status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	var response resputil.Response[resputil.Page[KanikoInfo]]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode image build page: %v", err)
	}
	return response.Data
}

func kanikoInfoIDs(items []KanikoInfo) []uint {
	ids := make([]uint, len(items))
	for index := range items {
		ids[index] = items[index].ID
	}
	return ids
}

func TestBindKanikoListPageQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/?page=2&page_size=20&search=%20cuda%20&status=Running&status=Finished&sort=-createdAt,image",
		http.NoBody,
	)

	request, statuses, err := bindKanikoListPageQuery(ctx)
	if err != nil {
		t.Fatalf("bindKanikoListPageQuery() error = %v", err)
	}
	if request.Page != 2 || request.PageSize != 20 || request.Search != "cuda" {
		t.Fatalf("unexpected request: %+v", request)
	}
	if !reflect.DeepEqual(statuses, []string{"Running", "Finished"}) {
		t.Fatalf("statuses = %v", statuses)
	}
}

func TestBindKanikoListPageQueryRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"page=0",
		"page_size=201",
		"status=Unknown",
		"sort=description",
		"sort=archs",
		"sort=image,image",
	}
	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+query, http.NoBody)
			if _, _, err := bindKanikoListPageQuery(ctx); err == nil {
				t.Fatal("bindKanikoListPageQuery() error = nil, want validation error")
			}
		})
	}
}

func TestKanikoListSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	got := kanikoListSortClauses("-createdAt,image")
	want := []string{"kanikos.created_at DESC", "kanikos.image_link ASC", "kanikos.id DESC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("kanikoListSortClauses() = %v, want %v", got, want)
	}
}

func TestListKanikoPageAppliesSearchAndStatusBeforePagination(t *testing.T) {
	db := setupKanikoListTestDB(t)
	createKanikoListTestUsers(t, db, "alice", "bob")
	description := "CUDA target"
	createdAt := time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC)
	createKanikoListTestRecords(t, db,
		model.Kaniko{Model: gorm.Model{ID: 1, CreatedAt: createdAt}, UserID: 1, Status: model.BuildJobRunning, Description: &description},
		model.Kaniko{Model: gorm.Model{ID: 2, CreatedAt: createdAt}, UserID: 1, Status: model.BuildJobRunning, ImagePackName: "cuda-second"},
		model.Kaniko{Model: gorm.Model{ID: 3, CreatedAt: createdAt}, UserID: 1, Status: model.BuildJobFinished, Description: &description},
		model.Kaniko{Model: gorm.Model{ID: 4, CreatedAt: createdAt}, UserID: 1, Status: model.BuildJobRunning, ImagePackName: "unrelated"},
		model.Kaniko{Model: gorm.Model{ID: 5, CreatedAt: createdAt}, UserID: 2, Status: model.BuildJobRunning, Description: &description},
	)

	page := requestKanikoListPage(t, db, nil, "page=1&page_size=1&search=cuda&status=Running&sort=-createdAt")
	if page.Total != 3 || len(page.Items) != 1 {
		t.Fatalf("filtered page = total %d, items %d; want total 3, items 1", page.Total, len(page.Items))
	}
	if page.Items[0].Status != model.BuildJobRunning {
		t.Fatalf("filtered item status = %q, want Running", page.Items[0].Status)
	}

	creatorPage := requestKanikoListPage(t, db, nil, "page=1&page_size=10&search=nickname-bob&sort=-createdAt")
	if creatorPage.Total != 1 || !reflect.DeepEqual(kanikoInfoIDs(creatorPage.Items), []uint{5}) {
		t.Fatalf("creator search = total %d, ids %v; want total 1, ids [5]", creatorPage.Total, kanikoInfoIDs(creatorPage.Items))
	}
}

func TestListKanikoPageKeepsUserAndAdminScopesSeparate(t *testing.T) {
	db := setupKanikoListTestDB(t)
	createKanikoListTestUsers(t, db, "alice", "bob")
	createKanikoListTestRecords(t, db,
		model.Kaniko{Model: gorm.Model{ID: 1}, UserID: 1},
		model.Kaniko{Model: gorm.Model{ID: 2}, UserID: 2},
	)

	userID := uint(1)
	userPage := requestKanikoListPage(t, db, &userID, "page=1&page_size=10&sort=-createdAt")
	if userPage.Total != 1 || !reflect.DeepEqual(kanikoInfoIDs(userPage.Items), []uint{1}) {
		t.Fatalf("user page = total %d, ids %v; want total 1, ids [1]", userPage.Total, kanikoInfoIDs(userPage.Items))
	}
	adminPage := requestKanikoListPage(t, db, nil, "page=1&page_size=10&sort=-createdAt")
	if adminPage.Total != 2 {
		t.Fatalf("admin page total = %d, want 2", adminPage.Total)
	}
}

func TestListKanikoPageUsesStableSortAndReturnsEmptyOutOfRange(t *testing.T) {
	db := setupKanikoListTestDB(t)
	createKanikoListTestUsers(t, db, "alice")
	createdAt := time.Date(2026, time.September, 7, 13, 0, 0, 0, time.UTC)
	createKanikoListTestRecords(t, db,
		model.Kaniko{Model: gorm.Model{ID: 1, CreatedAt: createdAt}, UserID: 1},
		model.Kaniko{Model: gorm.Model{ID: 2, CreatedAt: createdAt}, UserID: 1},
		model.Kaniko{Model: gorm.Model{ID: 3, CreatedAt: createdAt}, UserID: 1},
	)

	first := requestKanikoListPage(t, db, nil, "page=1&page_size=2&sort=-createdAt")
	second := requestKanikoListPage(t, db, nil, "page=2&page_size=2&sort=-createdAt")
	outOfRange := requestKanikoListPage(t, db, nil, "page=3&page_size=2&sort=-createdAt")
	if !reflect.DeepEqual(kanikoInfoIDs(first.Items), []uint{3, 2}) {
		t.Fatalf("first page ids = %v, want [3 2]", kanikoInfoIDs(first.Items))
	}
	if !reflect.DeepEqual(kanikoInfoIDs(second.Items), []uint{1}) {
		t.Fatalf("second page ids = %v, want [1]", kanikoInfoIDs(second.Items))
	}
	if outOfRange.Total != 3 || len(outOfRange.Items) != 0 {
		t.Fatalf("out-of-range page = total %d, items %v; want total 3, empty items", outOfRange.Total, kanikoInfoIDs(outOfRange.Items))
	}
}
