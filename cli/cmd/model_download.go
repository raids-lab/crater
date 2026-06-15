package cmd

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

var modelDownloadCategories = []string{"model", "dataset"}

var modelDownloadCmd = &cobra.Command{
	Use:   "model-download",
	Short: "View model and dataset downloads",
	Long:  "View model and dataset download records, details, and logs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var modelDownloadLsCmd = &cobra.Command{Use: "ls", Short: "List model downloads", RunE: runModelDownloadLs}
var modelDownloadGetCmd = &cobra.Command{Use: "get <id>", Short: "Get a model download", Args: maxOneArg, RunE: runModelDownloadGet}
var modelDownloadLogsCmd = &cobra.Command{Use: "logs <id>", Short: "Show model download logs", Args: maxOneArg, RunE: runModelDownloadLogs}

func runModelDownloadLs(cmd *cobra.Command, _ []string) error {
	admin, _ := cmd.Flags().GetBool("admin")
	path := api.ModelDownloadListPath
	params := map[string]string{}
	category := getStringParam(cmd, "category")
	if category != "" {
		params["category"] = category
	}
	if admin {
		path = api.AdminModelDLPfx + "/models/downloads"
		params = nil
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "downloads", Path: path, Params: func(*cobra.Command) map[string]string { return params }, Table: printModelDownloadTable})
}

func runModelDownloadGet(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "download_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "download", Path: fmt.Sprintf("%s/%s", api.ModelDownloadListPath, api.UintPath(id)), Params: noParams, Table: printRawObject})
}

func runModelDownloadLogs(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "download_label_id", "id")
	if err != nil {
		return err
	}
	return runRawStringRead(cmd, fmt.Sprintf("%s/%s/logs", api.ModelDownloadListPath, api.UintPath(id)), nil, "logs")
}

func printModelDownloadTable(data interface{}) {
	fmt.Printf("%s %s %s %s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight(i18n.T("table_name"), 32), i18n.PadRight(i18n.T("table_type"), 10), i18n.PadRight(i18n.T("table_status"), 14), i18n.PadRight("PATH", 36), i18n.PadRight("UPDATED", 22))
	for _, row := range rawList(data) {
		fmt.Printf("%s %s %s %s %s %s\n", i18n.PadRight(rawString(row, "id"), 8), i18n.PadRight(rawString(row, "name"), 32), i18n.PadRight(rawString(row, "category"), 10), i18n.PadRight(rawString(row, "status"), 14), i18n.PadRight(rawString(row, "path"), 36), i18n.PadRight(rawString(row, "updatedAt"), 22))
	}
}

func init() {
	modelDownloadLsCmd.Flags().String("category", "", "Filter by model download category")
	modelDownloadLsCmd.Flags().Bool("admin", false, "Use admin model download list API")
	completion.RegisterFlagValue([]string{"model-download", "ls"}, "category", staticValueCompleter(modelDownloadCategories, nil))
	modelDownloadCmd.AddCommand(modelDownloadLsCmd, modelDownloadGetCmd, modelDownloadLogsCmd)
	rootCmd.AddCommand(modelDownloadCmd)
}
