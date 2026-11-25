package cmd

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

var datasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "View datasets and models",
	Long:  "View dataset/model records and sharing relationships.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var datasetLsCmd = &cobra.Command{Use: "ls", Short: "List datasets and models", Args: noArgs, RunE: runDatasetLs}
var datasetGetCmd = &cobra.Command{Use: "get <id>", Short: "Get a dataset or model", Args: exactArgs(1, "id"), RunE: runDatasetGet}
var datasetUsersCmd = &cobra.Command{Use: "users <id>", Short: "List users shared with a dataset", Args: exactArgs(1, "id"), RunE: runDatasetUsers}
var datasetQueuesCmd = &cobra.Command{Use: "queues <id>", Short: "List queues shared with a dataset", Args: exactArgs(1, "id"), RunE: runDatasetQueues}
var datasetUsersOutCmd = &cobra.Command{Use: "users-out <id>", Short: "List users not shared with a dataset", Args: exactArgs(1, "id"), RunE: runDatasetUsersOut}
var datasetQueuesOutCmd = &cobra.Command{Use: "queues-out <id>", Short: "List queues not shared with a dataset", Args: exactArgs(1, "id"), RunE: runDatasetQueuesOut}
var adminDatasetCmd = &cobra.Command{Use: "dataset", Short: "View admin datasets and models"}
var adminDatasetLsCmd = &cobra.Command{Use: "ls", Short: "List datasets and models", Args: noArgs, RunE: runAdminDatasetLs}

const (
	datasetUsersInSuffix   = "usersIn"
	datasetQueuesInSuffix  = "queuesIn"
	datasetUsersOutSuffix  = "usersNotIn"
	datasetQueuesOutSuffix = "queuesNotIn"
)

func runDatasetLs(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{AllPages: true, PayloadKey: "datasets", Path: api.DatasetPrefix + "/mydataset", Params: noParams, Table: printDatasetTable})
}

func runAdminDatasetLs(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{AllPages: true, PayloadKey: "datasets", Path: api.AdminDatasetPrefix + "/alldataset", Params: noParams, Table: printDatasetTable})
}

func runDatasetGet(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "dataset_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "dataset", Path: fmt.Sprintf("%s/detail/%s", api.DatasetPrefix, api.UintPath(id)), Params: noParams, Table: printDatasetTable})
}

func runDatasetUsers(cmd *cobra.Command, args []string) error {
	return runDatasetShareList(cmd, args, "users", datasetUsersInSuffix)
}

func runDatasetQueues(cmd *cobra.Command, args []string) error {
	return runDatasetShareList(cmd, args, "queues", datasetQueuesInSuffix)
}

func runDatasetUsersOut(cmd *cobra.Command, args []string) error {
	return runDatasetShareList(cmd, args, "users", datasetUsersOutSuffix)
}

func runDatasetQueuesOut(cmd *cobra.Command, args []string) error {
	return runDatasetShareList(cmd, args, "queues", datasetQueuesOutSuffix)
}

func runDatasetShareList(cmd *cobra.Command, args []string, payloadKey string, suffix string) error {
	id, err := requiredUintArg(args, "dataset_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{
		PayloadKey: payloadKey,
		Path:       datasetShareListPath(id, suffix),
		Params:     noParams,
		Table:      printSimpleTableWrapper("id", "name", "nickname", "isowner"),
	})
}

func datasetShareListPath(id uint, suffix string) string {
	return fmt.Sprintf("%s/%s/%s", api.DatasetPrefix, api.UintPath(id), suffix)
}

func printDatasetTable(data interface{}) {
	fmt.Printf("%s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_id"), 8),
		i18n.PadRight(i18n.T("table_name"), 28),
		i18n.PadRight(i18n.T("table_type"), 12),
		i18n.PadRight("MOUNTS", 10),
		i18n.PadRight(i18n.T("table_owner"), 18),
		i18n.PadRight("URL", 36))
	for _, row := range rawList(data) {
		fmt.Printf("%s %s %s %s %s %s\n",
			i18n.PadRight(rawString(row, "id"), 8),
			i18n.PadRight(rawString(row, "name"), 28),
			i18n.PadRight(rawString(row, "type"), 12),
			i18n.PadRight(rawString(row, "mountCount"), 10),
			i18n.PadRight(rawNestedString(row, "userInfo", "nickname"), 18),
			i18n.PadRight(rawString(row, "url"), 36))
	}
}

func init() {
	datasetCmd.AddCommand(datasetLsCmd, datasetGetCmd, datasetUsersCmd, datasetQueuesCmd, datasetUsersOutCmd, datasetQueuesOutCmd)
	rootCmd.AddCommand(datasetCmd)
	adminDatasetCmd.AddCommand(adminDatasetLsCmd)
	adminCmd.AddCommand(adminDatasetCmd)
}
