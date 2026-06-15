package cmd

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "View current account context",
	Long:  "View current account feature status, quota, job resource usage, and billing summary.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var contextPrequeueCmd = &cobra.Command{Use: "prequeue", Short: "Get prequeue feature status", RunE: runContextPrequeue}
var contextQuotaCmd = &cobra.Command{Use: "quota", Short: "Get current account quota", RunE: runContextQuota}
var contextResourcesCmd = &cobra.Command{Use: "resources", Short: "Get current job resource summary", RunE: runContextResources}
var contextBillingCmd = &cobra.Command{Use: "billing", Short: "Get current billing summary", RunE: runContextBilling}

func runContextPrequeue(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "prequeue", Path: api.ContextPrefix + "/prequeue", Params: noParams, Table: printRawObject})
}

func runContextQuota(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "quota", Path: api.ContextPrefix + "/quota", Params: noParams, Table: printRawObject})
}

func runContextResources(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "resources", Path: api.ContextPrefix + "/job-resource-summary", Params: noParams, Table: printContextResources})
}

func runContextBilling(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "billing", Path: api.ContextPrefix + "/billing/summary", Params: noParams, Table: printRawObject})
}

func printContextResources(data interface{}) {
	m := rawMap(data)
	fmt.Printf("%s: %s\n", "RunningJobs", emptyDash(rawString(m, "runningJobs")))
	fmt.Printf("%s: %s\n", "PendingJobs", emptyDash(rawString(m, "pendingJobs")))
	fmt.Printf("%s: %s\n", "CPU", rawResourceSummary(m, "cpu"))
	fmt.Printf("%s: %s\n", "Memory", rawResourceSummary(m, "memory"))
	accelerators, _ := m["accelerators"].([]interface{})
	if len(accelerators) > 0 {
		fmt.Printf("%s:\n", i18n.T("table_resources"))
		for _, item := range accelerators {
			row, _ := item.(map[string]interface{})
			fmt.Printf("- %s %s\n", emptyDash(rawString(row, "resource")), rawResourceSummary(map[string]interface{}{"x": row}, "x"))
		}
	}
}

func init() {
	contextCmd.AddCommand(contextPrequeueCmd, contextQuotaCmd, contextResourcesCmd, contextBillingCmd)
	rootCmd.AddCommand(contextCmd)
}
