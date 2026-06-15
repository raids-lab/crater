package cmd

import (
	"github.com/raids-lab/crater/cli/internal/api"
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
var adminSystemConfigLLMCmd = &cobra.Command{Use: "llm", Short: "Get LLM configuration", RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "llm", Path: api.AdminSysConfigPfx + "/llm", Params: noParams, Table: printRawObject})
}}
var adminSystemConfigGPUCmd = &cobra.Command{Use: "gpu-analysis", Short: "Get GPU analysis status", RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "gpu_analysis", Path: api.AdminSysConfigPfx + "/gpu-analysis", Params: noParams, Table: printRawObject})
}}
var adminSystemConfigPrequeueCmd = &cobra.Command{Use: "prequeue", Short: "Get prequeue configuration", RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "prequeue", Path: api.AdminSysConfigPfx + "/prequeue", Params: noParams, Table: printRawObject})
}}
var adminQueueQuotasCmd = &cobra.Command{Use: "queue-quotas", Short: "List queue quotas", RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "queue_quotas", Path: api.AdminQueueQuotasPfx, Params: noParams, Table: printRawObject})
}}
var adminGpuAnalysesCmd = &cobra.Command{Use: "gpu-analyses", Short: "List GPU analysis records", RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "gpu_analyses", Path: api.AdminGPUAnalysisPfx, Params: noParams, Table: printSimpleTableWrapper("ID", "JobName", "UserName", "PodName", "ReviewStatus")})
}}
var adminOperationLogsCmd = &cobra.Command{Use: "operation-logs", Short: "List operation logs", RunE: runAdminOperationLogs}
var adminCronjobsCmd = &cobra.Command{Use: "cronjobs", Short: "List cronjob configs", RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "cronjobs", Path: api.AdminOperationsPfx + "/cronjob", Params: noParams, Table: printRawObject})
}}
var adminWhitelistCmd = &cobra.Command{Use: "whitelist", Short: "List operation whitelist", RunE: func(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "whitelist", Path: api.AdminOperationsPfx + "/whitelist", Params: noParams, Table: printRawObject})
}}

func runAdminOperationLogs(cmd *cobra.Command, _ []string) error {
	params := map[string]string{
		"page":           getIntParam(cmd, "page"),
		"limit":          getIntParam(cmd, "limit"),
		"operator":       getStringParam(cmd, "operator"),
		"operation_type": getStringParam(cmd, "operation-type"),
		"target":         getStringParam(cmd, "target"),
		"start_time":     getStringParam(cmd, "start-time"),
		"end_time":       getStringParam(cmd, "end-time"),
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "operation_logs", Path: api.AdminOperationLogs, Params: func(*cobra.Command) map[string]string { return params }, Table: printRawObject})
}

func init() {
	adminSystemConfigCmd.AddCommand(adminSystemConfigLLMCmd, adminSystemConfigGPUCmd, adminSystemConfigPrequeueCmd)
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
