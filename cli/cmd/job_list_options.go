package cmd

import (
	"fmt"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

const maxJobListPageSize = 200

func addJobListFlags(cmd *cobra.Command) {
	cmd.Flags().Int("page", 1, "Page number")
	cmd.Flags().Int("page-size", 50, "Page size")
	cmd.Flags().String("sort", "", "Sort fields")
	cmd.Flags().Bool("all-pages", false, "Fetch all pages")
}

func readJobPaginationOptions(cmd *cobra.Command) (api.ListOptions, error) {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	sortFields, _ := cmd.Flags().GetString("sort")
	allPages, _ := cmd.Flags().GetBool("all-pages")
	issues := make([]usageIssue, 0, 2)
	if page < 1 {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: "page must be at least 1",
			Field:   "page",
		})
	}
	if pageSize < 1 || pageSize > maxJobListPageSize {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: fmt.Sprintf("page-size must be between 1 and %d", maxJobListPageSize),
			Field:   "page-size",
		})
	}
	if len(issues) > 0 {
		return api.ListOptions{}, errUsageFromIssues(issues)
	}
	return api.ListOptions{
		Page:     page,
		PageSize: pageSize,
		Sort:     strings.TrimSpace(sortFields),
		AllPages: allPages,
	}, nil
}
