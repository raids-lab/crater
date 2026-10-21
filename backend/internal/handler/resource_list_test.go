package handler

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

func setupResourcePageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.Resource{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	query.SetDefault(db)
	return db
}

func createResourcePageTestRecords(t *testing.T, db *gorm.DB, records ...model.Resource) {
	t.Helper()
	for index := range records {
		record := records[index]
		if record.ResourceName == "" {
			record.ResourceName = fmt.Sprintf("resource-%d", record.ID)
		}
		if record.ResourceType == "" {
			record.ResourceType = "nvidia.com/gpu"
		}
		if record.Format == "" {
			record.Format = "count"
		}
		if record.Type == nil {
			resourceType := model.ResourceTypeGPU
			record.Type = &resourceType
		}
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("create test resource %d: %v", record.ID, err)
		}
	}
}

func requestResourcePage(
	t *testing.T,
	db *gorm.DB,
	rawQuery string,
) resputil.Page[ResourceResp] {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+rawQuery, http.NoBody)

	(&ResourceMgr{}).listResourcePageWithDB(ctx, db)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list resources status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	var response resputil.Response[resputil.Page[ResourceResp]]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode resource page: %v", err)
	}
	return response.Data
}

func resourcePageIDs(items []ResourceResp) []uint {
	ids := make([]uint, len(items))
	for index := range items {
		ids[index] = items[index].ID
	}
	return ids
}

func TestBindResourcePageQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/?page=2&page_size=20&search=%20gpu%20&type=gpu&type=vgpu&sort=-amount,name",
		http.NoBody,
	)

	request, resourceTypes, err := bindResourcePageQuery(ctx)
	if err != nil {
		t.Fatalf("bindResourcePageQuery() error = %v", err)
	}
	if request.Page != 2 || request.PageSize != 20 || request.Search != "gpu" {
		t.Fatalf("unexpected request: %+v", request)
	}
	if !reflect.DeepEqual(resourceTypes, []string{"gpu", "vgpu"}) {
		t.Fatalf("resourceTypes = %v", resourceTypes)
	}
}

func TestBindResourcePageQueryRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"page=0",
		"page_size=201",
		"type=unknown",
		"sort=unitPrice",
		"sort=name,name",
	}
	for _, rawQuery := range tests {
		t.Run(rawQuery, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+rawQuery, http.NoBody)
			if _, _, err := bindResourcePageQuery(ctx); err == nil {
				t.Fatal("bindResourcePageQuery() error = nil, want validation error")
			}
		})
	}
}

func TestResourcePageSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	got := resourcePageSortClauses("-amount,name")
	want := []string{"amount DESC", "resource_name ASC", "id ASC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resourcePageSortClauses() = %v, want %v", got, want)
	}
}

func TestBindResourcePriceIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/?resource_id=2&resource_id=5&resource_id=2",
		http.NoBody,
	)
	got, err := bindResourcePriceIDs(ctx)
	if err != nil {
		t.Fatalf("bindResourcePriceIDs() error = %v", err)
	}
	if !reflect.DeepEqual(got, []uint{2, 5}) {
		t.Fatalf("bindResourcePriceIDs() = %v", got)
	}
}

func TestListResourcePageAppliesSearchAndTypeBeforePagination(t *testing.T) {
	db := setupResourcePageTestDB(t)
	rdma := model.ResourceTypeRDMA
	createdAt := time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC)
	createResourcePageTestRecords(t, db,
		model.Resource{Model: gorm.Model{ID: 1, CreatedAt: createdAt}, ResourceName: "gpu-a", ResourceType: "nvidia.com/a"},
		model.Resource{Model: gorm.Model{ID: 2, CreatedAt: createdAt}, ResourceName: "gpu-b", ResourceType: "nvidia.com/b"},
		model.Resource{Model: gorm.Model{ID: 3, CreatedAt: createdAt}, ResourceName: "rdma-a", ResourceType: "rdma", Type: &rdma},
	)

	page := requestResourcePage(t, db, "page=1&page_size=1&search=gpu&type=gpu&sort=-name")
	if page.Total != 2 || len(page.Items) != 1 {
		t.Fatalf("filtered page = total %d, items %d; want total 2, items 1", page.Total, len(page.Items))
	}
	if page.Items[0].Name != "gpu-b" {
		t.Fatalf("filtered item = %q, want gpu-b", page.Items[0].Name)
	}
}

func TestListResourcePageUsesStableSortAndEmptyOutOfRange(t *testing.T) {
	db := setupResourcePageTestDB(t)
	createdAt := time.Date(2026, time.September, 7, 13, 0, 0, 0, time.UTC)
	createResourcePageTestRecords(t, db,
		model.Resource{Model: gorm.Model{ID: 1, CreatedAt: createdAt}, ResourceName: "resource-a", Label: "same"},
		model.Resource{Model: gorm.Model{ID: 2, CreatedAt: createdAt}, ResourceName: "resource-b", Label: "same"},
		model.Resource{Model: gorm.Model{ID: 3, CreatedAt: createdAt}, ResourceName: "resource-c", Label: "same"},
	)

	first := requestResourcePage(t, db, "page=1&page_size=2&sort=label")
	second := requestResourcePage(t, db, "page=2&page_size=2&sort=label")
	outOfRange := requestResourcePage(t, db, "page=3&page_size=2&sort=label")
	if !reflect.DeepEqual(resourcePageIDs(first.Items), []uint{1, 2}) {
		t.Fatalf("first page ids = %v, want [1 2]", resourcePageIDs(first.Items))
	}
	if !reflect.DeepEqual(resourcePageIDs(second.Items), []uint{3}) {
		t.Fatalf("second page ids = %v, want [3]", resourcePageIDs(second.Items))
	}
	if outOfRange.Total != 3 || len(outOfRange.Items) != 0 {
		t.Fatalf("out-of-range page = total %d, items %v; want total 3, empty items", outOfRange.Total, resourcePageIDs(outOfRange.Items))
	}
}

func TestListResourcePageReturnsEmptySearchResult(t *testing.T) {
	db := setupResourcePageTestDB(t)
	createResourcePageTestRecords(t, db, model.Resource{Model: gorm.Model{ID: 1}, ResourceName: "gpu-a"})

	page := requestResourcePage(t, db, "page=1&page_size=10&search=does-not-exist")
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("empty search = total %d, items %d; want 0, 0", page.Total, len(page.Items))
	}
}
