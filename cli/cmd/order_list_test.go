package cmd

import (
	"encoding/json"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

func TestOrderListFlagsExposeCreatorOnlyForAdmin(t *testing.T) {
	userCommand := &cobra.Command{}
	addOrderListFlags(userCommand, false)
	if userCommand.Flags().Lookup("creator") != nil {
		t.Fatal("user order list must not expose --creator")
	}

	adminCommand := &cobra.Command{}
	addOrderListFlags(adminCommand, true)
	if adminCommand.Flags().Lookup("creator") == nil {
		t.Fatal("admin order list must expose --creator")
	}
}

func TestReadOrderListOptionsAggregatesPaginationAndDomainIssues(t *testing.T) {
	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	command := &cobra.Command{}
	addOrderListFlags(command, false)
	for flag, value := range map[string]string{
		"page":      "0",
		"page-size": "201",
		"status":    "Unknown",
		"type":      "model",
	} {
		if err := command.Flags().Set(flag, value); err != nil {
			t.Fatalf("set --%s: %v", flag, err)
		}
	}

	_, err := readOrderListOptions(command, false)
	if err == nil {
		t.Fatal("expected invalid list options to fail")
	}
	for _, message := range []string{
		"page must be at least 1",
		"page-size must be between 1 and 200",
		"invalid approval order status: Unknown",
		"invalid approval order type: model",
	} {
		if !strings.Contains(err.Error(), message) {
			t.Fatalf("error %q does not include %q", err, message)
		}
	}
}

func TestReadOrderListOptionsNormalizesStatus(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "pending", want: "Pending"},
		{input: "APPROVED", want: "Approved"},
		{input: "rEjEcTeD", want: "Rejected"},
		{input: " canceled ", want: "Canceled"},
		{input: "Pending", want: "Pending"},
	} {
		t.Run(test.input, func(t *testing.T) {
			command := &cobra.Command{}
			addOrderListFlags(command, true)
			if err := command.Flags().Set("status", test.input); err != nil {
				t.Fatalf("set --status: %v", err)
			}
			options, err := readOrderListOptions(command, true)
			if err != nil {
				t.Fatalf("read list options: %v", err)
			}
			if options.Status != test.want {
				t.Errorf("status = %q, want %q", options.Status, test.want)
			}
		})
	}
}

func TestFilterApprovalOrdersUsesTypedFields(t *testing.T) {
	older := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	newer := older.Add(24 * time.Hour)
	orders := []api.ApprovalOrder{
		{
			ID: 1, Name: "old-pending", Type: "job", Status: "Pending", CreatedAt: older,
			Creator: api.ApprovalUserInfo{Username: "alice", Nickname: "Alice"},
		},
		{
			ID: 4, Name: "new-approved", Type: "job", Status: "Approved", CreatedAt: newer,
			Creator: api.ApprovalUserInfo{Username: "bob", Nickname: "Bob"},
		},
		{
			ID: 2, Name: "new-pending-low-id", Type: "dataset", Status: "Pending", CreatedAt: newer,
			Creator: api.ApprovalUserInfo{Username: "alice", Nickname: "Alice"},
		},
		{
			ID: 3, Name: "new-pending-high-id", Type: "job", Status: "Pending", CreatedAt: newer,
			Creator: api.ApprovalUserInfo{Username: "alice", Nickname: "Alice"},
		},
	}

	filtered := filterApprovalOrders(orders, orderListOptions{
		Admin: true, Status: "Pending", Type: "job", Creator: "ALI", Search: "high",
	})
	if len(filtered) != 1 || filtered[0].ID != 3 {
		t.Fatalf("typed filters returned %#v, want order 3", filtered)
	}
}

func TestSortApprovalOrdersBeforePagination(t *testing.T) {
	older := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	newer := older.Add(24 * time.Hour)
	orders := []api.ApprovalOrder{
		{ID: 1, Name: "old-pending", Type: "job", Status: "Pending", CreatedAt: older},
		{ID: 4, Name: "new-approved", Type: "job", Status: "Approved", CreatedAt: newer},
		{ID: 2, Name: "new-pending-low-id", Type: "job", Status: "Pending", CreatedAt: newer},
		{ID: 3, Name: "new-pending-high-id", Type: "job", Status: "Pending", CreatedAt: newer},
		{ID: 5, Name: "filtered-dataset", Type: "dataset", Status: "Pending", CreatedAt: newer.Add(time.Hour)},
	}

	filtered := filterApprovalOrders(orders, orderListOptions{Type: "job"})
	sortApprovalOrders(filtered)
	page := paginateList(filtered, api.ListOptions{Page: 1, PageSize: 2})

	if page.Total != 4 {
		t.Fatalf("filtered total = %d, want 4", page.Total)
	}
	got := []uint{page.Items[0].ID, page.Items[1].ID}
	if !reflect.DeepEqual(got, []uint{3, 2}) {
		t.Fatalf("first page IDs = %v, want [3 2]", got)
	}
}

func TestUserOrderListIgnoresCreatorFilter(t *testing.T) {
	command := &cobra.Command{}
	addOrderListFlags(command, false)
	options, err := readOrderListOptions(command, false)
	if err != nil {
		t.Fatalf("read user order options: %v", err)
	}
	options.Creator = "nobody"

	orders := []api.ApprovalOrder{{
		ID: 1, Name: "mine", Creator: api.ApprovalUserInfo{Username: "alice"},
	}}
	if got := filterApprovalOrders(orders, options); len(got) != 1 {
		t.Fatalf("user-side creator filter unexpectedly removed orders: %#v", got)
	}
}

func TestWriteOrderListPageJSONPaginationAndAllPages(t *testing.T) {
	orders := []api.ApprovalOrder{{
		ID: 7, Name: "train", Type: "job", Status: "Pending",
	}}
	page := api.Page[api.ApprovalOrder]{
		Items: orders, Total: 21, Page: 2, PageSize: 15,
	}

	paged := captureOrderListStdout(t, func() {
		if err := writeListPage("orders", page, false, printOrderTable); err != nil {
			t.Fatalf("write paged JSON: %v", err)
		}
	})
	var pagedEnvelope map[string]interface{}
	if err := json.Unmarshal([]byte(paged), &pagedEnvelope); err != nil {
		t.Fatalf("decode paged JSON: %v", err)
	}
	pagedData := pagedEnvelope["data"].(map[string]interface{})
	if _, ok := pagedData["pagination"]; !ok {
		t.Fatalf("paged JSON has no pagination metadata: %s", paged)
	}

	all := captureOrderListStdout(t, func() {
		if err := writeListPage("orders", page, true, printOrderTable); err != nil {
			t.Fatalf("write all-pages JSON: %v", err)
		}
	})
	var allEnvelope map[string]interface{}
	if err := json.Unmarshal([]byte(all), &allEnvelope); err != nil {
		t.Fatalf("decode all-pages JSON: %v", err)
	}
	allData := allEnvelope["data"].(map[string]interface{})
	if _, ok := allData["pagination"]; ok {
		t.Fatalf("all-pages JSON unexpectedly has pagination metadata: %s", all)
	}
	allOrders, ok := allData["orders"].([]interface{})
	if !ok || len(allOrders) != 1 {
		t.Fatalf("all-pages JSON has no typed orders payload: %s", all)
	}
	firstOrder, ok := allOrders[0].(map[string]interface{})
	if !ok || firstOrder["id"] != float64(7) {
		t.Fatalf("all-pages JSON has no typed orders payload: %s", all)
	}
}

func TestPrintOrderTableUsesTypedFieldsAndI18nHeaders(t *testing.T) {
	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	createdAt := time.Date(2026, 7, 25, 8, 30, 0, 0, time.UTC)
	got := captureOrderListStdout(t, func() {
		outputJSON = false
		printOrderTable([]api.ApprovalOrder{{
			ID: 9, Name: "typed-order", Type: "dataset", Status: "Approved",
			CreatedAt: createdAt,
			Creator:   api.ApprovalUserInfo{Username: "alice", Nickname: "Alice"},
		}})
	})
	for _, value := range []string{
		i18n.T("table_id"),
		i18n.T("table_name"),
		i18n.T("table_type"),
		i18n.T("table_status"),
		i18n.T("table_owner"),
		i18n.T("table_created_at"),
		"typed-order",
		"Alice",
		createdAt.Format(time.RFC3339),
	} {
		if !strings.Contains(got, value) {
			t.Fatalf("table output %q does not include %q", got, value)
		}
	}
}

func captureOrderListStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	oldOutputJSON := outputJSON
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer
	outputJSON = true
	t.Cleanup(func() {
		os.Stdout = oldStdout
		outputJSON = oldOutputJSON
	})

	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout pipe: %v", err)
	}
	os.Stdout = oldStdout
	outputJSON = oldOutputJSON
	return string(got)
}
