package cmd

import (
	"fmt"
	"os"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "View accounts",
	Long:  "View account lists, account details, members, quotas, and billing settings.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var accountLsCmd = &cobra.Command{Use: "ls", Short: "List accounts", RunE: runAccountLs}
var accountGetCmd = &cobra.Command{
	Use:   "get <id-or-name>",
	Short: "Get an account",
	Args:  maxOneArg,
	RunE:  runAccountGet,
}
var accountMembersCmd = &cobra.Command{
	Use:   "members <id>",
	Short: "List account members",
	Args:  maxOneArg,
	RunE:  runAccountMembers,
}
var accountUsersOutCmd = &cobra.Command{
	Use:   "users-out <id>",
	Short: "List users outside an account",
	Args:  maxOneArg,
	RunE:  runAccountUsersOut,
}
var accountQuotaCmd = &cobra.Command{
	Use:   "quota <id>",
	Short: "Get account quota",
	Args:  maxOneArg,
	RunE:  runAccountQuota,
}
var accountBillingCmd = &cobra.Command{Use: "billing", Short: "View account billing"}
var accountBillingConfigCmd = &cobra.Command{
	Use:   "config <id>",
	Short: "Get account billing config",
	Args:  maxOneArg,
	RunE:  runAccountBillingConfig,
}
var accountBillingMembersCmd = &cobra.Command{
	Use:   "members <id>",
	Short: "List account billing members",
	Args:  maxOneArg,
	RunE:  runAccountBillingMembers,
}

func maxOneArg(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errTooManyArgs(cmd, len(args), 1)
	}
	return nil
}

func runAccountLs(cmd *cobra.Command, _ []string) error {
	admin, _ := cmd.Flags().GetBool("admin")
	path := api.AccountsPrefix
	if admin {
		path = api.AdminAccountsPrefix
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "accounts", Path: path, Params: noParams, Table: printAccountTable})
}

func runAccountGet(cmd *cobra.Command, args []string) error {
	idOrName, err := requiredArg(args, "account_label_id_or_name", "account")
	if err != nil {
		return err
	}
	admin, _ := cmd.Flags().GetBool("admin")
	path := api.AccountsPrefix + "/by-name/" + idOrName
	if admin {
		id, err := requiredUintArg(args, "account_label_id", "id")
		if err != nil {
			return err
		}
		path = api.AdminAccountsPrefix + "/" + api.UintPath(id)
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "account", Path: path, Params: noParams, Table: printRawObject})
}

func runAccountMembers(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	admin, _ := cmd.Flags().GetBool("admin")
	path := fmt.Sprintf("%s/%s/users", api.AccountsPrefix, api.UintPath(id))
	if admin {
		path = fmt.Sprintf("%s/userIn/%s", api.AdminAccountsPrefix, api.UintPath(id))
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "members", Path: path, Params: noParams, Table: printAccountMemberTable})
}

func runAccountUsersOut(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	admin, _ := cmd.Flags().GetBool("admin")
	path := fmt.Sprintf("%s/%s/users/out", api.AccountsPrefix, api.UintPath(id))
	if admin {
		path = fmt.Sprintf("%s/userOutOf/%s", api.AdminAccountsPrefix, api.UintPath(id))
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "users", Path: path, Params: noParams, Table: printAccountMemberTable})
}

func runAccountQuota(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "quota", Path: fmt.Sprintf("%s/%s/quota", api.AdminAccountsPrefix, api.UintPath(id)), Params: noParams, Table: printRawObject})
}

func runAccountBillingConfig(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	admin, _ := cmd.Flags().GetBool("admin")
	prefix := api.AccountsPrefix
	if admin {
		prefix = api.AdminAccountsPrefix
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "billing_config", Path: fmt.Sprintf("%s/%s/billing/config", prefix, api.UintPath(id)), Params: noParams, Table: printRawObject})
}

func runAccountBillingMembers(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	admin, _ := cmd.Flags().GetBool("admin")
	prefix := api.AccountsPrefix
	if admin {
		prefix = api.AdminAccountsPrefix
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "members", Path: fmt.Sprintf("%s/%s/billing/members", prefix, api.UintPath(id)), Params: noParams, Table: printAccountBillingMemberTable})
}

func printAccountTable(data interface{}) {
	fmt.Printf("%s %s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight(i18n.T("table_name"), 24), i18n.PadRight("NICKNAME", 24), i18n.PadRight(i18n.T("table_role"), 14))
	for _, row := range rawList(data) {
		fmt.Printf("%s %s %s %s\n", i18n.PadRight(rawString(row, "id"), 8), i18n.PadRight(rawString(row, "name"), 24), i18n.PadRight(rawString(row, "nickname"), 24), i18n.PadRight(rawString(row, "role"), 14))
	}
}

func printAccountMemberTable(data interface{}) {
	fmt.Printf("%s %s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight(i18n.T("table_name"), 24), i18n.PadRight(i18n.T("table_role"), 14), i18n.PadRight("ACCESS", 12))
	for _, row := range rawList(data) {
		name := rawString(row, "name")
		if name == "" {
			name = rawNestedString(row, "userInfo", "name")
		}
		fmt.Printf("%s %s %s %s\n", i18n.PadRight(rawString(row, "id"), 8), i18n.PadRight(name, 24), i18n.PadRight(rawString(row, "role"), 14), i18n.PadRight(rawString(row, "accessmode"), 12))
	}
}

func printAccountBillingMemberTable(data interface{}) {
	fmt.Printf("%s %s %s %s\n", i18n.PadRight("USER_ID", 10), i18n.PadRight("USERNAME", 24), i18n.PadRight("NICKNAME", 24), i18n.PadRight("AVAILABLE", 14))
	for _, row := range rawList(data) {
		fmt.Printf("%s %s %s %s\n", i18n.PadRight(rawString(row, "userId"), 10), i18n.PadRight(rawString(row, "username"), 24), i18n.PadRight(rawString(row, "nickname"), 24), i18n.PadRight(rawString(row, "totalAvailable"), 14))
	}
}

func printRawObject(data interface{}) {
	b, err := output.MarshalJSONPretty(data)
	if err != nil {
		fmt.Printf("%v\n", data)
		return
	}
	_, _ = os.Stdout.Write(b)
}

func init() {
	accountLsCmd.Flags().Bool("admin", false, "Use admin account list API")
	accountGetCmd.Flags().Bool("admin", false, "Use admin account detail API")
	accountMembersCmd.Flags().Bool("admin", false, "Use admin account members API")
	accountUsersOutCmd.Flags().Bool("admin", false, "Use admin users-out API")
	accountBillingConfigCmd.Flags().Bool("admin", false, "Use admin billing config API")
	accountBillingMembersCmd.Flags().Bool("admin", false, "Use admin billing members API")
	accountBillingCmd.AddCommand(accountBillingConfigCmd, accountBillingMembersCmd)
	accountCmd.AddCommand(accountLsCmd, accountGetCmd, accountMembersCmd, accountUsersOutCmd, accountQuotaCmd, accountBillingCmd)
	rootCmd.AddCommand(accountCmd)
}
