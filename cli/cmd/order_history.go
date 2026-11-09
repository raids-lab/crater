package cmd

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
)

type approvalOrderHistoryEntry struct {
	Attempt        int                  `json:"attempt"`
	ID             uint                 `json:"id"`
	Status         string               `json:"status"`
	RequestedHours uint                 `json:"requestedHours"`
	ApprovedLock   *api.ApprovedJobLock `json:"approvedLock,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	ReviewedAt     *time.Time           `json:"reviewedAt,omitempty"`
	ReviewSeconds  *int64               `json:"reviewSeconds,omitempty"`
}

type approvalOrderHistory struct {
	CurrentAttempt int                         `json:"currentAttempt"`
	ApprovedCount  int                         `json:"approvedCount"`
	Entries        []approvalOrderHistoryEntry `json:"entries"`
}

func readJobApprovalHistory(client api.ApprovalOrderClient, current *api.ApprovalOrder) (*approvalOrderHistory, string) {
	orders, err := client.ListApprovalOrdersByName(current.Name, "job")
	if err != nil {
		return nil, cliErrFromAPI(err).Message
	}
	slices.SortFunc(orders, func(a, b api.ApprovalOrder) int {
		if n := a.CreatedAt.Compare(b.CreatedAt); n != 0 {
			return n
		}
		return cmp.Compare(a.ID, b.ID)
	})
	history := &approvalOrderHistory{Entries: make([]approvalOrderHistoryEntry, 0, len(orders))}
	for _, order := range orders {
		if order.Type != "job" || order.Name != current.Name {
			continue
		}
		entry := approvalOrderHistoryEntry{
			Attempt:        len(history.Entries) + 1,
			ID:             order.ID,
			Status:         order.Status,
			RequestedHours: order.Content.ApprovalOrderExtensionHours,
			ApprovedLock:   order.Content.ApprovedLock,
			CreatedAt:      order.CreatedAt,
			ReviewedAt:     order.ReviewedAt,
		}
		if order.ReviewedAt != nil && !order.CreatedAt.IsZero() && !order.ReviewedAt.Before(order.CreatedAt) {
			seconds := int64(order.ReviewedAt.Sub(order.CreatedAt).Round(time.Second) / time.Second)
			entry.ReviewSeconds = &seconds
		}
		if order.ID == current.ID {
			history.CurrentAttempt = entry.Attempt
		}
		if order.Status == "Approved" {
			history.ApprovedCount++
		}
		history.Entries = append(history.Entries, entry)
	}
	if history.CurrentAttempt == 0 {
		return nil, i18n.T("order_history_current_missing")
	}
	return history, ""
}

func printJobApprovalHistory(history *approvalOrderHistory) {
	fmt.Printf("\n%s\n", i18n.T("order_history_heading", history.CurrentAttempt, history.ApprovedCount))
	fmt.Printf("%s %s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("order_history_attempt"), 8),
		i18n.PadRight(i18n.T("table_id"), 8),
		i18n.PadRight(i18n.T("table_status"), 12),
		i18n.PadRight(i18n.T("order_history_hours"), 18),
		i18n.PadRight(i18n.T("order_history_approved_duration"), 18),
		i18n.PadRight(i18n.T("table_created_at"), 25),
		i18n.T("order_history_review_time"))
	for _, entry := range history.Entries {
		reviewTime := "-"
		if entry.ReviewSeconds != nil {
			reviewTime = (time.Duration(*entry.ReviewSeconds) * time.Second).String()
		}
		fmt.Printf("%s %s %s %s %s %s %s\n",
			i18n.PadRight(fmt.Sprint(entry.Attempt), 8),
			i18n.PadRight(fmt.Sprint(entry.ID), 8),
			i18n.PadRight(entry.Status, 12),
			i18n.PadRight(fmt.Sprint(entry.RequestedHours), 18),
			i18n.PadRight(approvedLockLabel(entry.ApprovedLock), 18),
			i18n.PadRight(formatAPITime(entry.CreatedAt), 25),
			reviewTime)
	}
}

func approvedLockLabel(lock *api.ApprovedJobLock) string {
	if lock == nil {
		return "-"
	}
	if lock.IsPermanent {
		return i18n.T("order_approved_permanent_lock")
	}
	return (time.Duration(lock.DurationMinutes) * time.Minute).String()
}
