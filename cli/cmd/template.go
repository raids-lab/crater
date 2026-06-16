package cmd

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "View job templates",
	Long:  "View saved job templates from the active Crater platform.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var templateLsCmd = &cobra.Command{Use: "ls", Short: "List job templates", Args: noArgs, RunE: runTemplateLs}
var templateGetCmd = &cobra.Command{Use: "get <id>", Short: "Get a job template", Args: exactArgs(1, "id"), RunE: runTemplateGet}

func runTemplateLs(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "templates", Path: api.JobTemplatePrefix + "/list", Params: noParams, Table: printTemplateTable})
}

func runTemplateGet(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "template_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "template", Path: fmt.Sprintf("%s/%s", api.JobTemplatePrefix, api.UintPath(id)), Params: noParams, Table: printRawObject})
}

func printTemplateTable(data interface{}) {
	fmt.Printf("%s %s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight(i18n.T("table_name"), 28), i18n.PadRight(i18n.T("table_owner"), 18), i18n.PadRight("CREATED", 22))
	for _, row := range rawList(data) {
		fmt.Printf("%s %s %s %s\n", i18n.PadRight(rawString(row, "id"), 8), i18n.PadRight(rawString(row, "name"), 28), i18n.PadRight(rawNestedString(row, "userInfo", "nickname"), 18), i18n.PadRight(rawString(row, "createdAt"), 22))
	}
}

func init() {
	templateCmd.AddCommand(templateLsCmd, templateGetCmd)
	rootCmd.AddCommand(templateCmd)
}
