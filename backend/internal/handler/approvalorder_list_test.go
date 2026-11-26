package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
)

func newApprovalOrderListTestContext(rawQuery string) *gin.Context {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/approvalorder?"+rawQuery, http.NoBody)
	return context
}

func TestBindApprovalOrderListQuery(t *testing.T) {
	request, types, statuses, err := bindApprovalOrderListQuery(
		newApprovalOrderListTestContext("page=2&page_size=20&search=++demo+&type=job&type=dataset&status=Pending&sort=name,-createdAt"),
	)
	if err != nil {
		t.Fatalf("bindApprovalOrderListQuery returned error: %v", err)
	}
	if request.Page != 2 || request.PageSize != 20 || request.Search != "demo" {
		t.Fatalf("unexpected pagination query: %#v", request)
	}
	if request.Sort != "name,-createdAt" {
		t.Fatalf("unexpected sort: %q", request.Sort)
	}
	if strings.Join(types, ",") != "job,dataset" || strings.Join(statuses, ",") != "Pending" {
		t.Fatalf("unexpected filters: types=%v statuses=%v", types, statuses)
	}
}

func TestBindApprovalOrderListQueryDefaults(t *testing.T) {
	request, types, statuses, err := bindApprovalOrderListQuery(newApprovalOrderListTestContext(""))
	if err != nil {
		t.Fatalf("bindApprovalOrderListQuery returned error: %v", err)
	}
	if request.Page != 1 || request.PageSize != 10 || request.Sort != approvalOrderDefaultSort {
		t.Fatalf("unexpected defaults: %#v", request)
	}
	if len(types) != 0 || len(statuses) != 0 {
		t.Fatalf("unexpected default filters: types=%v statuses=%v", types, statuses)
	}
}

func TestBindApprovalOrderListQueryRejectsInvalidValues(t *testing.T) {
	testCases := []string{
		"page=0",
		"page_size=201",
		"type=unknown",
		"status=unknown",
		"sort=creator",
		"sort=name,name,name,name",
		"search=" + strings.Repeat("a", approvalOrderMaxSearchRunes+1),
	}

	for _, rawQuery := range testCases {
		t.Run(rawQuery, func(t *testing.T) {
			if _, _, _, err := bindApprovalOrderListQuery(newApprovalOrderListTestContext(rawQuery)); err == nil {
				t.Fatal("expected query validation error")
			}
		})
	}
}

func TestApprovalOrderSortClausesAlwaysHaveTieBreaker(t *testing.T) {
	clauses := approvalOrderSortClauses("name,-createdAt")
	if len(clauses) != 3 || clauses[2] != "id DESC" {
		t.Fatalf("unexpected sort clauses: %v", clauses)
	}
}

func TestApprovalOrderListPreservesOptionalReviewer(t *testing.T) {
	reviewerID := uint(42)
	orders := []*model.ApprovalOrder{
		{ReviewerID: nil},
		{ReviewerID: &reviewerID, Reviewer: model.User{Name: "reviewer", Nickname: "Reviewer"}},
	}

	items := convertToApprovalOrderResps(orders)
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].ReviewerID != 0 || items[0].Reviewer.Username != "" {
		t.Fatalf("unexpected reviewer for unreviewed order: %#v", items[0])
	}
	if items[1].ReviewerID != reviewerID || items[1].Reviewer.Username != "reviewer" || items[1].Reviewer.Nickname != "Reviewer" {
		t.Fatalf("unexpected reviewer for reviewed order: %#v", items[1])
	}
}

func TestApprovalOrderPriorityBeforePagination(t *testing.T) {
	router := newLegacyDatabaseListsRouter(t)
	db := compatibilityFixtureDB(t)
	created := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	statuses := []model.ApprovalOrderStatus{
		model.ApprovalOrderStatusPending, model.ApprovalOrderStatusPending,
		model.ApprovalOrderStatusPending, model.ApprovalOrderStatusApproved,
		model.ApprovalOrderStatusRejected, model.ApprovalOrderStatusCancelled, "LegacyUnknown",
	}
	for index, status := range statuses {
		stamp := created.Add(time.Duration(index) * time.Hour)
		if index == 2 {
			stamp = created.Add(time.Hour)
		}
		if err := db.Create(&model.ApprovalOrder{
			Model: gorm.Model{ID: []uint{901, 902, 903, 904, 905, 906, 907}[index], CreatedAt: stamp},
			Name:  "priority-fixture", CreatorID: 1, Status: status,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.ApprovalOrder{
		Name: "priority-fixture", CreatorID: 2, Status: model.ApprovalOrderStatusPending,
	}).Error; err != nil {
		t.Fatal(err)
	}
	for _, sort := range []string{"", "&sort=status,-createdAt"} {
		var got []uint
		for page := 1; page <= 4; page++ {
			recorder := httptest.NewRecorder()
			path := fmt.Sprintf("/compat/orders?page=%d&page_size=2&search=priority-fixture%s", page, sort)
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, http.NoBody))
			var response resputil.Response[resputil.Page[ApprovalOrderResp]]
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || response.Data.Total != 7 {
				t.Fatalf("unexpected page %d: %s", page, recorder.Body.String())
			}
			for _, item := range response.Data.Items {
				got = append(got, item.ID)
			}
		}
		want := "[903 902 901 904 905 906 907]"
		if fmt.Sprint(got) != want {
			t.Fatalf("sort %q: got %v, want %s", sort, got, want)
		}
	}
	// An explicit chronological sort remains available to the user.
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/compat/orders?page=1&page_size=1&search=priority-fixture&sort=-createdAt", http.NoBody))
	var response resputil.Response[resputil.Page[ApprovalOrderResp]]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data.Items) != 1 || response.Data.Items[0].ID != 907 {
		t.Fatalf("explicit chronological order changed: %s", recorder.Body.String())
	}
}
