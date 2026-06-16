package cmd

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

func newReadJobFamilyCommand(use, short, prefix, idLabel string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errUnknownSubcommand(cmd, args[0])
			}
			return cmd.Help()
		},
	}
	ls := &cobra.Command{Use: "ls", Short: "List jobs", Args: noArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		path := prefix
		all, _ := cmd.Flags().GetBool("all")
		if all {
			path = prefix + "/all"
		}
		return runRawRead(cmd, rawReadSpec{PayloadKey: "jobs", Path: path, Params: noParams, Table: printJobTableFromRaw})
	}}
	get := &cobra.Command{Use: "get <name>", Short: "Get a job", Args: exactArgs(1, "job-name"), RunE: func(cmd *cobra.Command, args []string) error {
		name, err := requiredArg(args, idLabel, "name")
		if err != nil {
			return err
		}
		return runRawRead(cmd, rawReadSpec{PayloadKey: "job", Path: fmt.Sprintf("%s/%s/detail", prefix, name), Params: noParams, Table: printRawObject})
	}}
	pods := &cobra.Command{Use: "pods <name>", Short: "List pods for a job", Args: exactArgs(1, "job-name"), RunE: func(cmd *cobra.Command, args []string) error {
		name, err := requiredArg(args, idLabel, "name")
		if err != nil {
			return err
		}
		return runRawRead(cmd, rawReadSpec{PayloadKey: "pods", Path: fmt.Sprintf("%s/%s/pods", prefix, name), Params: noParams, Table: printJobPodRawTable})
	}}
	events := &cobra.Command{Use: "events <name>", Short: "List events for a job", Args: exactArgs(1, "job-name"), RunE: func(cmd *cobra.Command, args []string) error {
		name, err := requiredArg(args, idLabel, "name")
		if err != nil {
			return err
		}
		suffix := "events"
		if prefix == api.AIJobsPrefix {
			suffix = "event"
		}
		return runRawRead(cmd, rawReadSpec{PayloadKey: "events", Path: fmt.Sprintf("%s/%s/%s", prefix, name, suffix), Params: noParams, Table: printRawObject})
	}}
	yamlCmd := &cobra.Command{Use: "yaml <name>", Short: "Show job YAML", Args: exactArgs(1, "job-name"), RunE: func(cmd *cobra.Command, args []string) error {
		name, err := requiredArg(args, idLabel, "name")
		if err != nil {
			return err
		}
		return runRawStringRead(cmd, fmt.Sprintf("%s/%s/yaml", prefix, name), nil, "yaml")
	}}
	ls.Flags().Bool("all", false, "List all visible records")
	cmd.AddCommand(ls, get, pods, events, yamlCmd)
	return cmd
}

func printJobTableFromRaw(data interface{}) {
	jobs := rawList(data)
	fmt.Printf("%s %s %s %s %s %s\n",
		i18nPad("table_name", 24), padLiteral("JOB_NAME", 34), i18nPad("table_type", 12),
		i18nPad("table_status", 14), i18nPad("table_queue", 18), i18nPad("table_nodes", 24))
	for _, job := range jobs {
		fmt.Printf("%s %s %s %s %s %s\n",
			pad(rawString(job, "name"), 24), pad(rawString(job, "jobName"), 34), pad(rawString(job, "jobType"), 12),
			pad(rawString(job, "status"), 14), pad(rawString(job, "queue"), 18), pad(fmt.Sprintf("%v", job["nodes"]), 24))
	}
}

func printJobPodRawTable(data interface{}) {
	printSimpleTable(data, "name", "namespace", "nodename", "ip", "phase")
}

func pad(v string, width int) string {
	return i18n.PadRight(v, width)
}

func padLiteral(v string, width int) string {
	return i18n.PadRight(v, width)
}

func i18nPad(key string, width int) string {
	return i18n.PadRight(i18n.T(key), width)
}

func init() {
	// AIJob/SPJob routes use different identifiers and are intentionally not
	// exposed in this broad read surface until their CLI contract is designed.
}
