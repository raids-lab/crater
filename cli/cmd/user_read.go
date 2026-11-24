package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

const (
	adminUserRoleGuest      uint8 = 1
	adminUserRoleUser       uint8 = 2
	adminUserRoleAdmin      uint8 = 3
	adminUserStatusPending  uint8 = 1
	adminUserStatusActive   uint8 = 2
	adminUserStatusInactive uint8 = 3
)

var (
	adminUserRoles = map[string]uint8{
		"Guest": adminUserRoleGuest,
		"User":  adminUserRoleUser,
		"Admin": adminUserRoleAdmin,
	}
	adminUserStatuses = map[string]uint8{
		"Pending":  adminUserStatusPending,
		"Active":   adminUserStatusActive,
		"Inactive": adminUserStatusInactive,
	}
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "View users",
	Long:  "View user details and admin user lists.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var userGetCmd = &cobra.Command{Use: "get <username>", Short: "Get a user", Args: exactArgs(1, "username"), RunE: runUserGet}
var userEmailCmd = &cobra.Command{Use: "email-verified", Short: "Check current user's email verification status", Args: noArgs, RunE: runUserEmail}
var userBillingCmd = &cobra.Command{Use: "billing", Short: "View user billing"}
var userBillingSummaryCmd = &cobra.Command{Use: "summary", Short: "List user billing summaries", Args: noArgs, RunE: runUserBillingSummary}
var userBillingAccountsCmd = &cobra.Command{Use: "accounts <username>", Short: "List user billing accounts", Args: exactArgs(1, "username"), RunE: runUserBillingAccounts}
var adminUserCmd = &cobra.Command{Use: "user", Short: "View admin users"}
var adminUserLsCmd = &cobra.Command{Use: "ls", Short: "List users", Args: noArgs, RunE: runUserLs}
var adminUserBillingCmd = &cobra.Command{Use: "billing", Short: "View user billing"}
var adminUserBillingSummaryCmd = &cobra.Command{Use: "summary", Short: "List user billing summaries", Args: noArgs, RunE: runUserBillingSummary}
var adminUserBillingAccountsCmd = &cobra.Command{Use: "accounts <username>", Short: "List user billing accounts", Args: exactArgs(1, "username"), RunE: runUserBillingAccounts}

type adminUserListOptions struct {
	api.ListOptions
	Base   bool
	Search string
	Role   string
	Status string
}

func runUserLs(cmd *cobra.Command, _ []string) error {
	options, err := readAdminUserListOptions(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	users, err := client.ListAdminUsers(options.Base)
	if err != nil {
		return cliErrFromAPI(err)
	}
	users = filterAdminUsers(users, options.Search, options.Role, options.Status)
	sortAdminUsers(users)
	page := paginateList(users, options.ListOptions)
	if outputJSON || options.Base {
		return writeListPage("users", page, options.AllPages, printUserTable)
	}
	fields, err := client.GetRaw("/v1/system-config/user-fields", nil)
	if err != nil {
		var requestError *api.RequestError
		if errors.As(err, &requestError) && requestError.HTTPStatus == http.StatusNotFound {
			return writeListPage("users", page, options.AllPages, printUserTable)
		}
		return cliErrFromAPI(err)
	}
	return writeListPage("users", page, options.AllPages, func(rows []api.AdminUser) { printUserFieldsTable(rows, rawMap(fields)) })
}

func readAdminUserListOptions(cmd *cobra.Command) (adminUserListOptions, error) {
	listOptions, issues := listPaginationOptions(cmd, maxCLIPageSize)
	base, _ := cmd.Flags().GetBool("base")
	search, _ := cmd.Flags().GetString("search")
	role, _ := cmd.Flags().GetString("role")
	status, _ := cmd.Flags().GetString("status")
	search = strings.TrimSpace(search)
	role = strings.TrimSpace(role)
	status = strings.TrimSpace(status)
	if role != "" {
		if _, ok := adminUserRoles[role]; !ok {
			issues = append(issues, invalidIssue("role", i18n.T("err_invalid_user_role", role)))
		}
	}
	if status != "" {
		if _, ok := adminUserStatuses[status]; !ok {
			issues = append(issues, invalidIssue("status", i18n.T("err_invalid_user_status", status)))
		}
	}
	if base && (role != "" || status != "") {
		issues = append(issues, invalidIssue("base", i18n.T("err_user_base_filter_conflict")))
	}
	if len(issues) > 0 {
		return adminUserListOptions{}, errUsageFromIssues(issues)
	}
	return adminUserListOptions{
		ListOptions: listOptions,
		Base:        base,
		Search:      search,
		Role:        role,
		Status:      status,
	}, nil
}

func filterAdminUsers(
	users []api.AdminUser,
	search string,
	role string,
	status string,
) []api.AdminUser {
	search = strings.ToLower(strings.TrimSpace(search))
	filtered := users[:0]
	for _, user := range users {
		name, nickname := adminUserNames(user)
		if search != "" &&
			!strings.Contains(strings.ToLower(name), search) &&
			!strings.Contains(strings.ToLower(nickname), search) {
			continue
		}
		if role != "" && user.Role != adminUserRoles[role] {
			continue
		}
		if status != "" && user.Status != adminUserStatuses[status] {
			continue
		}
		filtered = append(filtered, user)
	}
	return filtered
}

func adminUserNames(user api.AdminUser) (string, string) {
	name := user.Name
	nickname := user.Nickname
	if name == "" && user.Attributes != nil {
		name = user.Attributes.Name
	}
	if nickname == "" && user.Attributes != nil {
		nickname = user.Attributes.Nickname
	}
	return name, nickname
}

func sortAdminUsers(users []api.AdminUser) {
	sort.SliceStable(users, func(left, right int) bool {
		if users[left].ID != users[right].ID {
			return users[left].ID > users[right].ID
		}
		leftName, _ := adminUserNames(users[left])
		rightName, _ := adminUserNames(users[right])
		return strings.ToLower(leftName) < strings.ToLower(rightName)
	})
}

func runUserGet(cmd *cobra.Command, args []string) error {
	username, err := requiredArg(args, "user_label_name", "username")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "user", Path: api.UsersPrefix + "/" + username, Params: noParams, Table: printRawObject})
}

func runUserEmail(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "email", Path: api.UsersPrefix + "/email/verified", Params: noParams, Table: printRawObject})
}

func runUserBillingSummary(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "summaries", Path: api.AdminUsersPrefix + "/billing/summary", Params: noParams, Table: printSimpleTableWrapper("userId", "username", "totalAvailable", "periodFreeTotal", "extraBalance")})
}

func runUserBillingAccounts(cmd *cobra.Command, args []string) error {
	username, err := requiredArg(args, "user_label_name", "username")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "accounts", Path: fmt.Sprintf("%s/%s/billing/accounts", api.AdminUsersPrefix, username), Params: noParams, Table: printSimpleTableWrapper("accountId", "accountName", "accountNickname", "totalAvailable")})
}

func printUserTable(users []api.AdminUser) {
	fmt.Printf("%s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_id"), 8),
		i18n.PadRight(i18n.T("table_name"), 24),
		i18n.PadRight(i18n.T("table_nickname"), 24),
		i18n.PadRight(i18n.T("table_role"), 10),
		i18n.PadRight(i18n.T("table_status"), 10))
	for _, user := range users {
		name, nickname := adminUserNames(user)
		id := "-"
		if user.ID != 0 {
			id = fmt.Sprint(user.ID)
		}
		fmt.Printf("%s %s %s %s %s\n",
			i18n.PadRight(id, 8),
			i18n.PadRight(name, 24),
			i18n.PadRight(nickname, 24),
			i18n.PadRight(userRoleLabel(user.Role), 10),
			i18n.PadRight(userStatusLabel(user.Status), 10))
	}
}

func userRoleLabel(role uint8) string {
	switch role {
	case adminUserRoleGuest:
		return i18n.T("user_role_guest")
	case adminUserRoleUser:
		return i18n.T("user_role_user")
	case adminUserRoleAdmin:
		return i18n.T("user_role_admin")
	}
	return "-"
}

func userStatusLabel(status uint8) string {
	switch status {
	case adminUserStatusPending:
		return i18n.T("user_status_pending")
	case adminUserStatusActive:
		return i18n.T("user_status_active")
	case adminUserStatusInactive:
		return i18n.T("user_status_inactive")
	}
	return "-"
}

func init() {
	userCmd.AddCommand(userGetCmd, userEmailCmd, userFieldsCmd)
	rootCmd.AddCommand(userCmd)
	adminUserLsCmd.Flags().Bool("base", false, "List base user information")
	adminUserLsCmd.Flags().String("search", "", i18n.T("flag_search"))
	adminUserLsCmd.Flags().String("role", "", i18n.T("flag_role"))
	adminUserLsCmd.Flags().String("status", "", i18n.T("flag_status"))
	addListPaginationFlags(adminUserLsCmd)
	completion.RegisterFlagValue([]string{"admin", "user", "ls"}, "role", staticValueCompleter([]string{"Guest", "User", "Admin"}, nil))
	completion.RegisterFlagValue([]string{"admin", "user", "ls"}, "status", staticValueCompleter([]string{"Pending", "Active", "Inactive"}, nil))
	adminUserBillingCmd.AddCommand(adminUserBillingSummaryCmd, adminUserBillingAccountsCmd)
	adminUserCmd.AddCommand(adminUserLsCmd, adminUserBillingCmd)
	adminCmd.AddCommand(adminUserCmd)
}
