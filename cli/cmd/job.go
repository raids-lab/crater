package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

var (
	jobStatuses = []string{
		"Prequeue", "Pending", "Running", "Completed", "Failed", "Terminated", "Deleted", "Freed", "Cancelled",
	}
	jobTypes = []string{
		"jupyter", "webide", "custom", "pytorch", "tensorflow", "kuberay", "deepspeed", "openmpi",
	}
	interactiveJobTypes = []string{"jupyter", "webide"}
)

var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "View jobs",
	Long:  "View Volcano job lists and job details from the active Crater platform.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var jobLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List jobs",
	Args:  noArgs,
	RunE:  runJobLs,
}

var jobGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get a job",
	Args:  exactArgs(1, "name"),
	RunE:  runJobGet,
}

var jobPodsCmd = &cobra.Command{
	Use:   "pods <name>",
	Short: "List pods for a job",
	Args:  exactArgs(1, "name"),
	RunE:  runJobPods,
}

var jobEventsCmd = &cobra.Command{
	Use:   "events <name>",
	Short: "List events for a job",
	Args:  exactArgs(1, "name"),
	RunE:  runJobEvents,
}

var jobYAMLCmd = &cobra.Command{
	Use:   "yaml <name>",
	Short: "Show job YAML",
	Args:  exactArgs(1, "name"),
	RunE:  runJobYAML,
}

func runJobLs(cmd *cobra.Command, _ []string) error {
	opts, err := readJobListOptions(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	jobs, err := client.ListJobs(opts)
	if err != nil {
		return cliErrFromAPI(err)
	}
	jobs = filterJobs(cmd, jobs)
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"jobs": jobs,
		}))
	}
	printJobTable(jobs)
	return nil
}

func runJobGet(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	job, err := client.GetJob(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"job": job,
		}))
	}
	printJobDetail(job)
	return nil
}

func runJobPods(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	pods, err := client.GetJobPods(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"pods": pods,
		}))
	}
	printJobPodTable(pods)
	return nil
}

func runJobEvents(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	events, err := client.GetJobEvents(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"events": events,
		}))
	}
	for _, event := range events {
		fmt.Printf("%v\n", event)
	}
	return nil
}

func runJobYAML(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	yaml, err := client.GetJobYAML(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"yaml": yaml,
		}))
	}
	fmt.Print(yaml)
	if yaml != "" && !strings.HasSuffix(yaml, "\n") {
		fmt.Println()
	}
	return nil
}

func readJobListOptions(cmd *cobra.Command) (api.JobListOptions, error) {
	all, _ := cmd.Flags().GetBool("all")
	username, _ := cmd.Flags().GetString("user")
	username = strings.TrimSpace(username)
	days, _ := cmd.Flags().GetInt("days")
	status, _ := cmd.Flags().GetString("status")
	jobType, _ := cmd.Flags().GetString("type")
	status = strings.TrimSpace(status)
	jobType = strings.TrimSpace(jobType)
	interactive, _ := cmd.Flags().GetBool("interactive")
	batch, _ := cmd.Flags().GetBool("batch")

	var issues []usageIssue
	if days < -1 {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_days"),
			Field:   "days",
		})
	}
	if status != "" && !slices.Contains(jobStatuses, status) {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_job_status", status),
			Field:   "status",
		})
	}
	if jobType != "" && !slices.Contains(jobTypes, jobType) {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_job_type", jobType),
			Field:   "type",
		})
	}
	if interactive && batch {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_job_interactive_batch_conflict"),
			Field:   "interactive",
		})
	}
	if len(issues) > 0 {
		return api.JobListOptions{}, errUsageFromIssues(issues)
	}
	return api.JobListOptions{
		All:      all,
		Username: username,
		Days:     days,
	}, nil
}

func filterJobs(cmd *cobra.Command, jobs []api.JobInfo) []api.JobInfo {
	status, _ := cmd.Flags().GetString("status")
	jobType, _ := cmd.Flags().GetString("type")
	nodeName, _ := cmd.Flags().GetString("node")
	interactive, _ := cmd.Flags().GetBool("interactive")
	batch, _ := cmd.Flags().GetBool("batch")
	status = strings.TrimSpace(status)
	jobType = strings.TrimSpace(jobType)
	nodeName = strings.TrimSpace(nodeName)
	out := jobs[:0]
	for _, job := range jobs {
		if status != "" && job.Status != status {
			continue
		}
		if jobType != "" && job.JobType != jobType {
			continue
		}
		if nodeName != "" && !slices.Contains(job.Nodes, nodeName) {
			continue
		}
		isInteractive := slices.Contains(interactiveJobTypes, job.JobType)
		if interactive && !isInteractive {
			continue
		}
		if batch && isInteractive {
			continue
		}
		out = append(out, job)
	}
	return out
}

func printJobTable(jobs []api.JobInfo) {
	fmt.Printf("%s %s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_name"), 24),
		i18n.PadRight("JOB_NAME", 34),
		i18n.PadRight(i18n.T("table_type"), 12),
		i18n.PadRight(i18n.T("table_status"), 14),
		i18n.PadRight(i18n.T("table_queue"), 18),
		i18n.PadRight(i18n.T("table_nodes"), 24),
		i18n.PadRight(i18n.T("table_resources"), 24))
	for _, job := range jobs {
		fmt.Printf("%s %s %s %s %s %s %s\n",
			i18n.PadRight(job.Name, 24),
			i18n.PadRight(job.JobName, 34),
			i18n.PadRight(job.JobType, 12),
			i18n.PadRight(job.Status, 14),
			i18n.PadRight(job.Queue, 18),
			i18n.PadRight(strings.Join(job.Nodes, ","), 24),
			i18n.PadRight(formatResources(job.Resources), 24))
	}
}

func printJobDetail(job *api.JobDetail) {
	if job == nil {
		return
	}
	fmt.Printf("%s: %s\n", i18n.T("table_name"), job.Name)
	fmt.Printf("%s: %s\n", "JobName", job.JobName)
	fmt.Printf("%s: %s\n", i18n.T("table_type"), job.JobType)
	fmt.Printf("%s: %s\n", i18n.T("table_status"), job.Status)
	fmt.Printf("%s: %s\n", i18n.T("table_queue"), job.Queue)
	fmt.Printf("%s: %s\n", i18n.T("table_owner"), job.UserInfo.Nickname)
	fmt.Printf("%s: %s\n", i18n.T("table_resources"), formatResources(job.Resources))
}

func printJobPodTable(pods []api.PodDetail) {
	fmt.Printf("%s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_name"), 36),
		i18n.PadRight(i18n.T("table_namespace"), 22),
		i18n.PadRight(i18n.T("table_node"), 24),
		i18n.PadRight("IP", 16),
		i18n.PadRight(i18n.T("table_phase"), 14),
		i18n.PadRight(i18n.T("table_resources"), 24))
	for _, pod := range pods {
		fmt.Printf("%s %s %s %s %s %s\n",
			i18n.PadRight(pod.Name, 36),
			i18n.PadRight(pod.Namespace, 22),
			i18n.PadRight(pod.NodeName, 24),
			i18n.PadRight(emptyDash(pod.IP), 16),
			i18n.PadRight(pod.Phase, 14),
			i18n.PadRight(formatResources(pod.Resource), 24))
	}
}

func formatResources(resources api.ResourceList) string {
	if len(resources) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(resources))
	for k, v := range resources {
		parts = append(parts, k+"="+v)
	}
	slices.Sort(parts)
	return strings.Join(parts, ",")
}

func init() {
	jobLsCmd.Flags().Bool("all", false, "List all jobs visible to this account")
	jobLsCmd.Flags().String("user", "", "List jobs for a specific username")
	jobLsCmd.Flags().Int("days", 0, "Look back days for --all or --user; -1 means all")
	jobLsCmd.Flags().String("status", "", "Filter by job status")
	jobLsCmd.Flags().String("type", "", "Filter by job type")
	jobLsCmd.Flags().String("node", "", "Filter by node name")
	jobLsCmd.Flags().Bool("interactive", false, "Only show interactive jobs")
	jobLsCmd.Flags().Bool("batch", false, "Only show batch jobs")
	completion.RegisterFlagValue([]string{"job", "ls"}, "status", staticValueCompleter(jobStatuses, nil))
	completion.RegisterFlagValue([]string{"job", "ls"}, "type", staticValueCompleter(jobTypes, nil))

	jobCmd.AddCommand(jobLsCmd)
	jobCmd.AddCommand(jobGetCmd)
	jobCmd.AddCommand(jobPodsCmd)
	jobCmd.AddCommand(jobEventsCmd)
	jobCmd.AddCommand(jobYAMLCmd)
	rootCmd.AddCommand(jobCmd)
}
