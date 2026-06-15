package cmd

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

var billingCmd = &cobra.Command{
	Use:   "billing",
	Short: "View billing information",
	Long:  "View billing status, prices, summaries, and job billing records.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var billingStatusCmd = &cobra.Command{Use: "status", Short: "Get billing feature status", RunE: runBillingStatus}
var billingSummaryCmd = &cobra.Command{Use: "summary", Short: "Get current billing summary", RunE: runContextBilling}
var billingPricesCmd = &cobra.Command{Use: "prices", Short: "List billing prices", RunE: runResourcePrices}
var billingJobsCmd = &cobra.Command{Use: "jobs", Short: "List job billing records", RunE: runBillingJobs}
var billingJobCmd = &cobra.Command{Use: "job <name>", Short: "Get job billing detail", Args: maxOneArg, RunE: runBillingJob}

func runBillingStatus(cmd *cobra.Command, _ []string) error {
	admin, _ := cmd.Flags().GetBool("admin")
	path := api.SystemConfigPrefix + "/billing"
	if admin {
		path = api.AdminSysConfigPfx + "/billing"
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "billing", Path: path, Params: noParams, Table: printRawObject})
}

func runBillingJobs(cmd *cobra.Command, _ []string) error {
	admin, _ := cmd.Flags().GetBool("admin")
	all, _ := cmd.Flags().GetBool("all")
	user := getStringParam(cmd, "user")
	path := api.VCJobsPrefix + "/billing"
	params := map[string]string{}
	if user != "" {
		path = api.VCJobsPrefix + "/billing/user/" + user
		params["days"] = getIntParam(cmd, "days")
	} else if all {
		path = api.VCJobsPrefix + "/billing/all"
		params["days"] = getIntParam(cmd, "days")
	}
	if admin {
		path = api.VCJobsPrefix + "/billing"
		if user != "" {
			path = api.VCJobsPrefix + "/billing/user/" + user
		}
		path = "/api/v1/admin" + path[len("/api/v1"):]
		params["days"] = getIntParam(cmd, "days")
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "billing", Path: path, Params: func(*cobra.Command) map[string]string { return params }, Table: printJobBillingTable})
}

func runBillingJob(cmd *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "billing", Path: fmt.Sprintf("%s/%s/billing", api.VCJobsPrefix, name), Params: noParams, Table: printRawObject})
}

func printJobBillingTable(data interface{}) {
	fmt.Printf("%s %s %s\n", i18n.PadRight(i18n.T("table_name"), 28), i18n.PadRight("JOB_NAME", 34), i18n.PadRight("POINTS", 14))
	for _, row := range rawList(data) {
		fmt.Printf("%s %s %s\n", i18n.PadRight(rawString(row, "name"), 28), i18n.PadRight(rawString(row, "jobName"), 34), i18n.PadRight(rawString(row, "billedPointsTotal"), 14))
	}
}

func init() {
	billingStatusCmd.Flags().Bool("admin", false, "Use admin billing status API")
	billingJobsCmd.Flags().Bool("all", false, "List all visible job billing records")
	billingJobsCmd.Flags().Bool("admin", false, "Use admin job billing API")
	billingJobsCmd.Flags().String("user", "", "Filter job billing by username")
	billingJobsCmd.Flags().Int("days", 30, "Look back days")
	billingCmd.AddCommand(billingStatusCmd, billingSummaryCmd, billingPricesCmd, billingJobsCmd, billingJobCmd)
	rootCmd.AddCommand(billingCmd)
}
