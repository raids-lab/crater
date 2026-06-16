package cmd

import (
	"fmt"
	"os"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
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

var accountLsCmd = &cobra.Command{Use: "ls", Short: "List accounts", Args: noArgs, RunE: runAccountLs}
var accountGetCmd = &cobra.Command{
	Use:   "get <id-or-name>",
	Short: "Get an account",
	Args:  exactArgs(1, "account"),
	RunE:  runAccountGet,
}
var accountMembersCmd = &cobra.Command{
	Use:   "members <id>",
	Short: "List account members",
	Args:  exactArgs(1, "id"),
	RunE:  runAccountMembers,
}
var accountUsersOutCmd = &cobra.Command{
	Use:   "users-out <id>",
	Short: "List users outside an account",
	Args:  exactArgs(1, "id"),
	RunE:  runAccountUsersOut,
}
var accountQuotaCmd = &cobra.Command{
	Use:   "quota <id>",
	Short: "Get account quota",
	Args:  exactArgs(1, "id"),
	RunE:  runAccountQuota,
}
var accountBillingCmd = &cobra.Command{Use: "billing", Short: "View account billing"}
var accountBillingConfigCmd = &cobra.Command{
	Use:   "config <id>",
	Short: "Get account billing config",
	Args:  exactArgs(1, "id"),
	RunE:  runAccountBillingConfig,
}
var accountBillingMembersCmd = &cobra.Command{
	Use:   "members <id>",
	Short: "List account billing members",
	Args:  exactArgs(1, "id"),
	RunE:  runAccountBillingMembers,
}

var adminAccountCmd = &cobra.Command{Use: "account", Short: "View admin account resources"}
var adminAccountLsCmd = &cobra.Command{Use: "ls", Short: "List accounts", Args: noArgs, RunE: runAdminAccountLs}
var adminAccountGetCmd = &cobra.Command{Use: "get <id>", Short: "Get an account", Args: exactArgs(1, "id"), RunE: runAdminAccountGet}
var adminAccountMembersCmd = &cobra.Command{Use: "members <id>", Short: "List account members", Args: exactArgs(1, "id"), RunE: runAdminAccountMembers}
var adminAccountUsersOutCmd = &cobra.Command{Use: "users-out <id>", Short: "List users outside an account", Args: exactArgs(1, "id"), RunE: runAdminAccountUsersOut}
var adminAccountQuotaCmd = &cobra.Command{Use: "quota <id>", Short: "Get account quota", Args: exactArgs(1, "id"), RunE: runAccountQuota}
var adminAccountBillingCmd = &cobra.Command{Use: "billing", Short: "View account billing"}
var adminAccountBillingConfigCmd = &cobra.Command{Use: "config <id>", Short: "Get account billing config", Args: exactArgs(1, "id"), RunE: runAdminAccountBillingConfig}
var adminAccountBillingMembersCmd = &cobra.Command{Use: "members <id>", Short: "List account billing members", Args: exactArgs(1, "id"), RunE: runAdminAccountBillingMembers}

func maxOneArg(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errTooManyArgs(cmd, len(args), 1)
	}
	return nil
}

func exactArgs(n int, fields ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			return errTooManyArgs(cmd, len(args), n)
		}
		if len(args) < n {
			field := "arg"
			if len(fields) > len(args) {
				field = fields[len(args)]
			}
			argName := positionalArgName(field)
			return errUsageFromIssues([]usageIssue{{
				Code:    errorcodes.ErrMissingRequiredFlag,
				Message: i18n.T("err_missing_required_arg", positionalLabel(field), argName),
				Field:   argName,
			}})
		}
		return nil
	}
}

func positionalArgName(field string) string {
	switch field {
	case "node-name", "job-name", "order-name":
		return "name"
	default:
		return field
	}
}

func positionalLabel(field string) string {
	switch field {
	case "node-name":
		return i18n.T("node_label_name")
	case "job-name":
		return i18n.T("job_label_name")
	case "order-name":
		return i18n.T("order_label_name")
	case "id":
		return "id"
	case "account":
		return i18n.T("account_label_id_or_name")
	case "namespace":
		return i18n.T("pod_label_namespace")
	case "pod":
		return i18n.T("pod_label_name")
	case "container":
		return i18n.T("container_label_name")
	case "username":
		return i18n.T("user_label_name")
	default:
		return field
	}
}

func runAccountLs(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "accounts", Path: api.AccountsPrefix, Params: noParams, Table: printAccountTable})
}

func runAdminAccountLs(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "accounts", Path: api.AdminAccountsPrefix, Params: noParams, Table: printAccountTable})
}

func runAccountGet(cmd *cobra.Command, args []string) error {
	idOrName, err := requiredArg(args, "account_label_id_or_name", "account")
	if err != nil {
		return err
	}
	path := api.AccountsPrefix + "/by-name/" + idOrName
	return runRawRead(cmd, rawReadSpec{PayloadKey: "account", Path: path, Params: noParams, Table: printRawObject})
}

func runAdminAccountGet(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "account", Path: api.AdminAccountsPrefix + "/" + api.UintPath(id), Params: noParams, Table: printRawObject})
}

func runAccountMembers(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	path := fmt.Sprintf("%s/%s/users", api.AccountsPrefix, api.UintPath(id))
	return runRawRead(cmd, rawReadSpec{PayloadKey: "members", Path: path, Params: noParams, Table: printAccountMemberTable})
}

func runAdminAccountMembers(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "members", Path: fmt.Sprintf("%s/userIn/%s", api.AdminAccountsPrefix, api.UintPath(id)), Params: noParams, Table: printAccountMemberTable})
}

func runAccountUsersOut(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	path := fmt.Sprintf("%s/%s/users/out", api.AccountsPrefix, api.UintPath(id))
	return runRawRead(cmd, rawReadSpec{PayloadKey: "users", Path: path, Params: noParams, Table: printAccountMemberTable})
}

func runAdminAccountUsersOut(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "users", Path: fmt.Sprintf("%s/userOutOf/%s", api.AdminAccountsPrefix, api.UintPath(id)), Params: noParams, Table: printAccountMemberTable})
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
	return runRawRead(cmd, rawReadSpec{PayloadKey: "billing_config", Path: fmt.Sprintf("%s/%s/billing/config", api.AccountsPrefix, api.UintPath(id)), Params: noParams, Table: printRawObject})
}

func runAdminAccountBillingConfig(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "billing_config", Path: fmt.Sprintf("%s/%s/billing/config", api.AdminAccountsPrefix, api.UintPath(id)), Params: noParams, Table: printRawObject})
}

func runAccountBillingMembers(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "members", Path: fmt.Sprintf("%s/%s/billing/members", api.AccountsPrefix, api.UintPath(id)), Params: noParams, Table: printAccountBillingMemberTable})
}

func runAdminAccountBillingMembers(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "account_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "members", Path: fmt.Sprintf("%s/%s/billing/members", api.AdminAccountsPrefix, api.UintPath(id)), Params: noParams, Table: printAccountBillingMemberTable})
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
	accountBillingCmd.AddCommand(accountBillingConfigCmd, accountBillingMembersCmd)
	accountCmd.AddCommand(accountLsCmd, accountGetCmd, accountMembersCmd, accountUsersOutCmd, accountBillingCmd)
	rootCmd.AddCommand(accountCmd)
	adminAccountBillingCmd.AddCommand(adminAccountBillingConfigCmd, adminAccountBillingMembersCmd)
	adminAccountCmd.AddCommand(adminAccountLsCmd, adminAccountGetCmd, adminAccountMembersCmd, adminAccountUsersOutCmd, adminAccountQuotaCmd, adminAccountBillingCmd)
	adminCmd.AddCommand(adminAccountCmd)
}
