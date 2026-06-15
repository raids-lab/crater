package cmd

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
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

var userLsCmd = &cobra.Command{Use: "ls", Short: "List users", RunE: runUserLs}
var userGetCmd = &cobra.Command{Use: "get <username>", Short: "Get a user", Args: maxOneArg, RunE: runUserGet}
var userEmailCmd = &cobra.Command{Use: "email-verified", Short: "Check current user's email verification status", RunE: runUserEmail}
var userBillingCmd = &cobra.Command{Use: "billing", Short: "View user billing"}
var userBillingSummaryCmd = &cobra.Command{Use: "summary", Short: "List user billing summaries", RunE: runUserBillingSummary}
var userBillingAccountsCmd = &cobra.Command{Use: "accounts <username>", Short: "List user billing accounts", Args: maxOneArg, RunE: runUserBillingAccounts}

func runUserLs(cmd *cobra.Command, _ []string) error {
	base, _ := cmd.Flags().GetBool("base")
	path := api.AdminUsersPrefix
	if base {
		path = api.AdminUsersPrefix + "/baseinfo"
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "users", Path: path, Params: noParams, Table: printUserTable})
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

func printUserTable(data interface{}) {
	fmt.Printf("%s %s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight(i18n.T("table_name"), 24), i18n.PadRight("NICKNAME", 24), i18n.PadRight(i18n.T("table_role"), 14))
	for _, row := range rawList(data) {
		name := rawString(row, "name")
		nickname := rawString(row, "nickname")
		if attrs, ok := row["attributes"].(map[string]interface{}); ok {
			if name == "" {
				name = rawString(attrs, "name")
			}
			if nickname == "" {
				nickname = rawString(attrs, "nickname")
			}
		}
		fmt.Printf("%s %s %s %s\n", i18n.PadRight(rawString(row, "id"), 8), i18n.PadRight(name, 24), i18n.PadRight(nickname, 24), i18n.PadRight(rawString(row, "role"), 14))
	}
}

func init() {
	userLsCmd.Flags().Bool("base", false, "List base user information")
	userBillingCmd.AddCommand(userBillingSummaryCmd, userBillingAccountsCmd)
	userCmd.AddCommand(userLsCmd, userGetCmd, userEmailCmd, userBillingCmd)
	rootCmd.AddCommand(userCmd)
}
