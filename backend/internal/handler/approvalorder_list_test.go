package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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
