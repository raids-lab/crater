package cmd

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

var orderCmd = &cobra.Command{
	Use:   "order",
	Short: "View approval orders",
	Long:  "View approval order lists and details.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var orderLsCmd = &cobra.Command{Use: "ls", Short: "List approval orders", RunE: runOrderLs}
var orderGetCmd = &cobra.Command{Use: "get <id>", Short: "Get an approval order", Args: maxOneArg, RunE: runOrderGet}
var orderByNameCmd = &cobra.Command{Use: "by-name <name>", Short: "List approval orders by name", Args: maxOneArg, RunE: runOrderByName}

func runOrderLs(cmd *cobra.Command, _ []string) error {
	admin, _ := cmd.Flags().GetBool("admin")
	path := api.ApprovalOrderPrefix
	if admin {
		path = api.AdminApprovalPrefix
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "orders", Path: path, Params: noParams, Table: printOrderTable})
}

func runOrderGet(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "order_label_id", "id")
	if err != nil {
		return err
	}
	admin, _ := cmd.Flags().GetBool("admin")
	prefix := api.ApprovalOrderPrefix
	if admin {
		prefix = api.AdminApprovalPrefix
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "order", Path: fmt.Sprintf("%s/%s", prefix, api.UintPath(id)), Params: noParams, Table: printRawObject})
}

func runOrderByName(cmd *cobra.Command, args []string) error {
	name, err := requiredArg(args, "order_label_name", "name")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "orders", Path: api.ApprovalOrderPrefix + "/name/" + name, Params: noParams, Table: printOrderTable})
}

func printOrderTable(data interface{}) {
	fmt.Printf("%s %s %s %s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight(i18n.T("table_name"), 28), i18n.PadRight(i18n.T("table_type"), 16), i18n.PadRight(i18n.T("table_status"), 14), i18n.PadRight("CREATOR", 18), i18n.PadRight("CREATED", 22))
	for _, row := range rawList(data) {
		fmt.Printf("%s %s %s %s %s %s\n", i18n.PadRight(rawString(row, "id"), 8), i18n.PadRight(rawString(row, "name"), 28), i18n.PadRight(rawString(row, "type"), 16), i18n.PadRight(rawString(row, "status"), 14), i18n.PadRight(rawNestedString(row, "creator", "nickname"), 18), i18n.PadRight(rawString(row, "createdAt"), 22))
	}
}

func init() {
	orderLsCmd.Flags().Bool("admin", false, "Use admin approval order list API")
	orderGetCmd.Flags().Bool("admin", false, "Use admin approval order detail API")
	orderCmd.AddCommand(orderLsCmd, orderGetCmd, orderByNameCmd)
	rootCmd.AddCommand(orderCmd)
}
