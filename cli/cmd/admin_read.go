package cmd

import (
	"math"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "View admin-only resources",
	Long:  "View admin-only read resources such as system configuration, queue quotas, operation logs, and GPU analyses.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var adminSystemConfigCmd = &cobra.Command{Use: "system-config", Short: "View system configuration"}
var adminSystemConfigLLMCmd = &cobra.Command{Use: "llm", Short: "Get LLM configuration", Args: noArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "llm", Path: api.AdminSysConfigPfx + "/llm", Params: noParams, Table: printRawObject})
}}
var adminSystemConfigGPUCmd = &cobra.Command{Use: "gpu-analysis", Short: "Get GPU analysis status", Args: noArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "gpu_analysis", Path: api.AdminSysConfigPfx + "/gpu-analysis", Params: noParams, Table: printRawObject})
}}
var adminSystemConfigSchedulerExtenderCmd = &cobra.Command{Use: "scheduler-extender", Short: "Get scheduler extender configuration", Args: noArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "scheduler_extender", Path: api.AdminSysConfigPfx + "/scheduler-extender", Params: noParams, Table: printRawObject})
}}
var adminQueueQuotasCmd = &cobra.Command{Use: "queue-quotas", Short: "List queue quotas", Args: noArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "queue_quotas", Path: api.AdminQueueQuotasPfx, Params: noParams, Table: printRawObject})
}}
var adminGpuAnalysesCmd = &cobra.Command{Use: "gpu-analyses", Short: "List GPU analysis records", Args: noArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "gpu_analyses", Path: api.AdminGPUAnalysisPfx, Params: noParams, Table: printSimpleTableWrapper("ID", "JobName", "UserName", "PodName", "ReviewStatus")})
}}
var adminOperationLogsCmd = &cobra.Command{Use: "operation-logs", Short: "List operation logs", Args: noArgs, RunE: runAdminOperationLogs}
var adminCronjobsCmd = &cobra.Command{Use: "cronjobs", Short: "List cronjob configs", Args: noArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "cronjobs", Path: api.AdminOperationsPfx + "/cronjob", Params: noParams, Table: printRawObject})
}}
var adminWhitelistCmd = &cobra.Command{Use: "whitelist", Short: "List operation whitelist", Args: noArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "whitelist", Path: api.AdminOperationsPfx + "/whitelist", Params: noParams, Table: printRawObject})
}}

func runAdminOperationLogs(cmd *cobra.Command, _ []string) error {
	page, _ := cmd.Flags().GetInt("page")
	limit, _ := cmd.Flags().GetInt("limit")
	var issues []usageIssue
	if page < 1 {
		issues = append(issues, invalidIssue("page", i18n.T("err_page_min")))
	}
	if limit < 1 {
		issues = append(issues, invalidIssue("limit", i18n.T("err_page_size_range", math.MaxInt)))
	}
	if page > math.MaxInt/max(limit, 1) {
		issues = append(issues, invalidIssue("page", i18n.T("err_page_overflow")))
	}
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	params := map[string]string{
		"operator":       getStringParam(cmd, "operator"),
		"operation_type": getStringParam(cmd, "operation-type"),
		"target":         getStringParam(cmd, "target"),
		"start_time":     getStringParam(cmd, "start-time"),
		"end_time":       getStringParam(cmd, "end-time"),
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	data, err := client.GetOperationLogs(page, limit, params)
	return writeRawReadResult("operation_logs", printRawObject, data, err)
}

func init() {
	adminSystemConfigCmd.AddCommand(adminSystemConfigLLMCmd, adminSystemConfigGPUCmd, adminSystemConfigSchedulerExtenderCmd)
	adminOperationLogsCmd.Flags().Int("page", 1, "Page number")
	adminOperationLogsCmd.Flags().Int("limit", 20, "Page size")
	adminOperationLogsCmd.Flags().String("operator", "", "Filter by operator")
	adminOperationLogsCmd.Flags().String("operation-type", "", "Filter by operation type")
	adminOperationLogsCmd.Flags().String("target", "", "Filter by target")
	adminOperationLogsCmd.Flags().String("start-time", "", "Filter by start time")
	adminOperationLogsCmd.Flags().String("end-time", "", "Filter by end time")
	adminCmd.AddCommand(adminSystemConfigCmd, adminQueueQuotasCmd, adminGpuAnalysesCmd, adminOperationLogsCmd, adminCronjobsCmd, adminWhitelistCmd)
	rootCmd.AddCommand(adminCmd)
}
