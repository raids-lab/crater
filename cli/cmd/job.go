package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

const (
	scheduleBackfill  = 0
	scheduleNormal    = 1
	volumeTypeFile    = 1
	volumeTypeDataset = 2
)

var (
	jobStatuses = []string{
		"Prequeue", "Pending", "Aborting", "Aborted", "Running", "Restarting",
		"Completing", "Completed", "Terminating", "Terminated", "Failed",
		"Deleted", "Freed", "Cancelled",
	}
	jobTypes = []string{
		"jupyter", "webide", "custom", "pytorch", "tensorflow", "kuberay", "deepspeed", "openmpi",
	}
)

var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "Manage jobs",
	Long:  "List, inspect, create, stop, and snapshot Volcano jobs on the active Crater platform.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var jobLsCmd = &cobra.Command{Use: "ls", Short: "List jobs", Args: noArgs, RunE: runJobLs}
var jobGetCmd = &cobra.Command{Use: "get <name>", Short: "Get a job", Args: exactArgs(1, "job-name"), RunE: runJobGet}
var jobPodsCmd = &cobra.Command{Use: "pods <name>", Short: "List pods for a job", Args: exactArgs(1, "job-name"), RunE: runJobPods}
var jobEventsCmd = &cobra.Command{Use: "events <name>", Short: "List events for a job", Args: exactArgs(1, "job-name"), RunE: runJobEvents}
var jobYAMLCmd = &cobra.Command{Use: "yaml <name>", Short: "Show job YAML", Args: exactArgs(1, "job-name"), RunE: runJobYAML}
var jobTemplateCmd = &cobra.Command{Use: "template <name>", Short: "Show job template JSON", Args: exactArgs(1, "job-name"), RunE: runJobTemplate}
var jobTokenCmd = &cobra.Command{Use: "token <name>", Short: "Get Jupyter token", Args: exactArgs(1, "job-name"), RunE: runJobToken}
var jobSecretCmd = &cobra.Command{Use: "secret <name>", Short: "Get WebIDE secret", Args: exactArgs(1, "job-name"), RunE: runJobSecret}
var jobSSHCmd = &cobra.Command{Use: "ssh <name>", Short: "Open SSH for a running job", Args: exactArgs(1, "job-name"), RunE: runJobSSH}
var jobSnapshotCmd = &cobra.Command{Use: "snapshot <name>", Short: "Create a job image snapshot", Args: exactArgs(1, "job-name"), RunE: runJobSnapshot}
var jobAlertCmd = &cobra.Command{Use: "alert <name>", Short: "Toggle job alert state", Args: exactArgs(1, "job-name"), RunE: runJobAlert}
var jobDeleteCmd = &cobra.Command{Use: "delete <name>", Short: "Stop or delete a job", Args: exactArgs(1, "job-name"), RunE: runJobDelete}

var jobCreateCmd = &cobra.Command{Use: "create", Short: "Create jobs"}
var jobCreateJupyterCmd = &cobra.Command{Use: "jupyter", Short: "Create a Jupyter job", Args: noArgs, RunE: runJobCreateJupyter}
var jobCreateWebIDECmd = &cobra.Command{Use: "webide", Short: "Create a WebIDE job", Args: noArgs, RunE: runJobCreateWebIDE}
var jobCreateCustomCmd = &cobra.Command{Use: "custom", Short: "Create a custom single-node job", Args: noArgs, RunE: runJobCreateCustom}
var jobCreateTensorflowCmd = &cobra.Command{Use: "tensorflow", Short: "Create a TensorFlow distributed job from JSON", Args: noArgs, RunE: runJobCreateTensorflow}
var jobCreatePytorchCmd = &cobra.Command{Use: "pytorch", Short: "Create a PyTorch distributed job from JSON", Args: noArgs, RunE: runJobCreatePytorch}

var adminJobCmd = &cobra.Command{Use: "job", Short: "Admin job operations"}
var adminJobLsCmd = &cobra.Command{Use: "ls", Short: "List jobs", Args: noArgs, RunE: runAdminJobLs}
var adminJobDeleteCmd = &cobra.Command{Use: "delete <name>", Short: "Delete a job", Args: exactArgs(1, "job-name"), RunE: runAdminJobDelete}
var jobAdminLockCmd = &cobra.Command{Use: "lock <name>", Short: "Lock a job cleanup window", Args: exactArgs(1, "job-name"), RunE: runJobAdminLock}
var jobAdminUnlockCmd = &cobra.Command{Use: "unlock <name>", Short: "Unlock a job cleanup window", Args: exactArgs(1, "job-name"), RunE: runJobAdminUnlock}
var jobAdminKeepCmd = &cobra.Command{Use: "keep <name>", Short: "Toggle keep-when-low-usage", Args: exactArgs(1, "job-name"), RunE: runJobAdminKeep}
var jobAdminCleanCmd = &cobra.Command{Use: "clean", Short: "Run admin job cleanup actions"}
var jobAdminCleanWaitingJupyterCmd = &cobra.Command{Use: "waiting-jupyter", Short: "Cancel waiting Jupyter jobs", Args: noArgs, RunE: runJobAdminCleanWaitingJupyter}
var jobAdminCleanWaitingCustomCmd = &cobra.Command{Use: "waiting-custom", Short: "Cancel waiting custom jobs", Args: noArgs, RunE: runJobAdminCleanWaitingCustom}
var jobAdminCleanLongRunningCmd = &cobra.Command{Use: "long-running", Short: "Clean long-running jobs", Args: noArgs, RunE: runJobAdminCleanLongRunning}
var jobAdminCleanLowGPUCmd = &cobra.Command{Use: "low-gpu", Short: "Clean low GPU usage jobs", Args: noArgs, RunE: runJobAdminCleanLowGPU}

func runJobLs(cmd *cobra.Command, _ []string) error {
	opts, err := readJobListOptions(cmd, false)
	if err != nil {
		return err
	}
	return listJobs(cmd, opts)
}

func runAdminJobLs(cmd *cobra.Command, _ []string) error {
	opts, err := readJobListOptions(cmd, true)
	if err != nil {
		return err
	}
	return listJobs(cmd, opts)
}

func readJobListOptions(cmd *cobra.Command, admin bool) (api.JobListOptions, error) {
	all, _ := cmd.Flags().GetBool("all")
	username, _ := cmd.Flags().GetString("user")
	username = strings.TrimSpace(username)
	days, _ := cmd.Flags().GetInt("days")
	if err := validateJobListFilters(cmd); err != nil {
		return api.JobListOptions{}, err
	}
	return api.JobListOptions{
		All:      all,
		Admin:    admin,
		Username: username,
		Days:     days,
	}, nil
}

func listJobs(cmd *cobra.Command, opts api.JobListOptions) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	jobs, err := client.ListJobs(opts)
	if err != nil {
		return cliErrFromAPI(err)
	}
	filtered, err := filterJobs(cmd, jobs)
	if err != nil {
		return err
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"jobs": filtered}))
	}
	printJobTable(filtered)
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
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"job": job}))
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
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"pods": pods}))
	}
	printPodTable(pods)
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
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"events": events}))
	}
	printEvents(events)
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
	yamlText, err := client.GetJobYAML(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"yaml": yamlText}))
	}
	fmt.Print(yamlText)
	if !strings.HasSuffix(yamlText, "\n") {
		fmt.Println()
	}
	return nil
}

func runJobTemplate(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	template, err := client.GetJobTemplate(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"template": template}))
	}
	fmt.Println(template)
	return nil
}

func runJobToken(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	token, err := client.GetJupyterToken(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	return writeToken(token)
}

func runJobSecret(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	token, err := client.GetWebIDESecret(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	return writeToken(token)
}

func runJobSSH(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	ssh, err := client.OpenJobSSH(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"ssh": ssh}))
	}
	fmt.Printf("%s:%s\n", ssh.IP, ssh.Port)
	return nil
}

func runJobSnapshot(_ *cobra.Command, args []string) error {
	return runJobMessage(args, func(client *api.Client, name string) (string, error) {
		return client.SnapshotJob(name)
	})
}

func runJobAlert(_ *cobra.Command, args []string) error {
	return runJobMessage(args, func(client *api.Client, name string) (string, error) {
		return client.ToggleJobAlert(name)
	})
}

func runJobDelete(cmd *cobra.Command, args []string) error {
	return runConfirmedJobMessage(cmd, args, "job_delete_confirm", "job_delete_succeeded", func(client *api.Client, name string) (string, error) {
		return client.DeleteJob(name)
	})
}

func runAdminJobDelete(cmd *cobra.Command, args []string) error {
	return runConfirmedJobMessage(cmd, args, "admin_job_delete_confirm", "admin_job_delete_succeeded", func(client *api.Client, name string) (string, error) {
		return client.AdminDeleteJob(name)
	})
}

func runJobMessage(args []string, call func(*api.Client, string) (string, error)) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := call(client, name)
	return writeMessage(msg, err)
}

func runConfirmedJobMessage(
	cmd *cobra.Command,
	args []string,
	confirmKey string,
	successKey string,
	call func(*api.Client, string) (string, error),
) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	if err := confirmJobAction(cmd, confirmKey, name); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := call(client, name)
	if err == nil && strings.TrimSpace(msg) == "" {
		msg = i18n.T(successKey, name)
	}
	return writeMessage(msg, err)
}

func runJobCreateJupyter(cmd *cobra.Command, _ []string) error {
	req, err := collectInteractiveCreate(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	data, err := client.CreateJupyterJob(req)
	return writeCreateResult(data, err)
}

func runJobCreateWebIDE(cmd *cobra.Command, _ []string) error {
	req, err := collectInteractiveCreate(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	data, err := client.CreateWebIDEJob(req)
	return writeCreateResult(data, err)
}

func runJobCreateCustom(cmd *cobra.Command, _ []string) error {
	file, _ := cmd.Flags().GetString("file")
	var req api.CreateTrainingJobRequest
	var err error
	if strings.TrimSpace(file) != "" {
		err = readJSONFile(file, &req)
	} else {
		req, err = collectCustomCreate(cmd)
	}
	if err != nil {
		return err
	}
	if err := validateTrainingRequest(req); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	data, err := client.CreateTrainingJob(req)
	return writeCreateResult(data, err)
}

func runJobCreateTensorflow(cmd *cobra.Command, _ []string) error {
	var req api.CreateDistributedJobRequest
	if err := readJSONFlag(cmd, &req); err != nil {
		return err
	}
	if err := validateDistributedRequest(req); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	data, err := client.CreateTensorflowJob(req)
	return writeCreateResult(data, err)
}

func runJobCreatePytorch(cmd *cobra.Command, _ []string) error {
	var req api.CreateDistributedJobRequest
	if err := readJSONFlag(cmd, &req); err != nil {
		return err
	}
	if err := validateDistributedRequest(req); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	data, err := client.CreatePytorchJob(req)
	return writeCreateResult(data, err)
}

func runJobAdminLock(cmd *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	permanent, _ := cmd.Flags().GetBool("permanent")
	days, _ := cmd.Flags().GetInt("days")
	hours, _ := cmd.Flags().GetInt("hours")
	minutes, _ := cmd.Flags().GetInt("minutes")
	if days < 0 || hours < 0 || minutes < 0 {
		return errUsageFromIssues([]usageIssue{invalidIssue("duration", i18n.T("err_invalid_non_negative_int", "duration"))})
	}
	if !permanent && days == 0 && hours == 0 && minutes == 0 {
		return errUsageFromIssues([]usageIssue{invalidIssue("duration", i18n.T("err_value_required_when_flag_enabled", "duration", "permanent=false"))})
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.AdminLockJob(api.LockJobRequest{Name: name, IsPermanent: permanent, Days: days, Hours: hours, Minutes: minutes})
	return writeMessage(msg, err)
}

func runJobAdminUnlock(_ *cobra.Command, args []string) error {
	return runJobMessage(args, func(client *api.Client, name string) (string, error) {
		return client.AdminUnlockJob(name)
	})
}

func runJobAdminKeep(_ *cobra.Command, args []string) error {
	return runJobMessage(args, func(client *api.Client, name string) (string, error) {
		return client.AdminToggleJobKeep(name)
	})
}

func runJobAdminCleanWaitingJupyter(cmd *cobra.Command, _ []string) error {
	wait, err := waitMinutesFlag(cmd)
	if err != nil {
		return err
	}
	if err := confirmJobAction(cmd, "admin_job_clean_confirm"); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	res, err := client.AdminCleanWaitingJupyter(wait)
	return writeCleanupResult(res, err)
}

func runJobAdminCleanWaitingCustom(cmd *cobra.Command, _ []string) error {
	wait, err := waitMinutesFlag(cmd)
	if err != nil {
		return err
	}
	if err := confirmJobAction(cmd, "admin_job_clean_confirm"); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	res, err := client.AdminCleanWaitingCustom(wait)
	return writeCleanupResult(res, err)
}

func runJobAdminCleanLongRunning(cmd *cobra.Command, _ []string) error {
	batchDays, _ := cmd.Flags().GetInt("batch-days")
	interactiveDays, _ := cmd.Flags().GetInt("interactive-days")
	issues := []usageIssue{}
	if batchDays <= 0 {
		issues = append(issues, invalidIssue("batch-days", i18n.T("err_invalid_positive_int", "batch-days")))
	}
	if interactiveDays <= 0 {
		issues = append(issues, invalidIssue("interactive-days", i18n.T("err_invalid_positive_int", "interactive-days")))
	}
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	if err := confirmJobAction(cmd, "admin_job_clean_confirm"); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	res, err := client.AdminCleanLongRunning(api.CleanLongTimeRequest{
		BatchDays:       batchDays,
		InteractiveDays: interactiveDays,
	})
	return writeCleanupResult(res, err)
}

func runJobAdminCleanLowGPU(cmd *cobra.Command, _ []string) error {
	timeRange, _ := cmd.Flags().GetInt("time-range")
	waitTime, _ := cmd.Flags().GetInt("wait-time")
	util, _ := cmd.Flags().GetInt("util")
	issues := []usageIssue{}
	if timeRange <= 0 {
		issues = append(issues, invalidIssue("time-range", i18n.T("err_invalid_positive_int", "time-range")))
	}
	if waitTime <= 0 {
		issues = append(issues, invalidIssue("wait-time", i18n.T("err_invalid_positive_int", "wait-time")))
	}
	if util < 0 || util > 100 {
		issues = append(issues, invalidIssue("util", i18n.T("err_invalid_percentage", "util")))
	}
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	if err := confirmJobAction(cmd, "admin_job_clean_confirm"); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	res, err := client.AdminCleanLowGPUUsage(api.CleanLowGPUUsageRequest{
		TimeRange: timeRange,
		WaitTime:  waitTime,
		Util:      util,
	})
	return writeCleanupResult(res, err)
}

func collectInteractiveCreate(cmd *cobra.Command) (api.CreateInteractiveJobRequest, error) {
	file, _ := cmd.Flags().GetString("file")
	if strings.TrimSpace(file) != "" {
		var req api.CreateInteractiveJobRequest
		if err := readJSONFile(file, &req); err != nil {
			return req, err
		}
		return req, validateInteractiveRequest(req)
	}
	common, resource, image, err := collectBasicCreate(cmd)
	if err != nil {
		return api.CreateInteractiveJobRequest{}, err
	}
	req := api.CreateInteractiveJobRequest{JobCommonRequest: common, Resource: resource, Image: image}
	return req, validateInteractiveRequest(req)
}

func collectCustomCreate(cmd *cobra.Command) (api.CreateTrainingJobRequest, error) {
	common, resource, image, err := collectBasicCreate(cmd)
	if err != nil {
		return api.CreateTrainingJobRequest{}, err
	}
	workingDir, _ := cmd.Flags().GetString("working-dir")
	command, _ := cmd.Flags().GetString("command")
	shell, _ := cmd.Flags().GetString("shell")
	req := api.CreateTrainingJobRequest{
		CreateInteractiveJobRequest: api.CreateInteractiveJobRequest{JobCommonRequest: common, Resource: resource, Image: image},
		WorkingDir:                  workingDir,
	}
	if strings.TrimSpace(command) != "" {
		req.Command = &command
	}
	if strings.TrimSpace(shell) != "" {
		req.Shell = &shell
	}
	return req, validateTrainingRequest(req)
}

func collectBasicCreate(cmd *cobra.Command) (api.JobCommonRequest, api.ResourceList, api.ImageBaseInfo, error) {
	name, _ := cmd.Flags().GetString("name")
	imageLink, _ := cmd.Flags().GetString("image")
	archs, _ := cmd.Flags().GetStringSlice("arch")
	cpu, _ := cmd.Flags().GetFloat64("cpu")
	memory, _ := cmd.Flags().GetString("memory")
	gpu, _ := cmd.Flags().GetInt("gpu")
	gpuResource, _ := cmd.Flags().GetString("gpu-resource")
	template, _ := cmd.Flags().GetString("template")
	alert, _ := cmd.Flags().GetBool("alert")
	cpuPinning, _ := cmd.Flags().GetBool("cpu-pinning")
	schedule, _ := cmd.Flags().GetString("schedule")

	issues := []usageIssue{}
	if strings.TrimSpace(name) == "" {
		issues = append(issues, missingIssue("name", "job_label_display_name"))
	}
	if strings.TrimSpace(imageLink) == "" {
		issues = append(issues, missingIssue("image", "job_label_image"))
	}
	if cpu < 0 {
		issues = append(issues, invalidIssue("cpu", i18n.T("err_invalid_non_negative_float", "cpu")))
	}
	if strings.TrimSpace(memory) == "" {
		issues = append(issues, missingIssue("memory", "job_label_memory"))
	} else if strings.HasPrefix(strings.TrimSpace(memory), "-") {
		issues = append(issues, invalidIssue("memory", i18n.T("err_invalid_non_negative_int", "memory")))
	}
	if gpu < 0 {
		issues = append(issues, invalidIssue("gpu", i18n.T("err_invalid_non_negative_int", "gpu")))
	}
	if gpu > 0 && strings.TrimSpace(gpuResource) == "" {
		issues = append(issues, missingIssue("gpu-resource", "job_label_gpu_resource"))
	}
	scheduleValue, err := parseScheduleType(schedule)
	if err != nil {
		issues = append(issues, invalidIssue("schedule", err.Error()))
	}
	if len(issues) > 0 {
		return api.JobCommonRequest{}, nil, api.ImageBaseInfo{}, errUsageFromIssues(issues)
	}

	envs, err := parseEnvFlags(cmd)
	if err != nil {
		return api.JobCommonRequest{}, nil, api.ImageBaseInfo{}, err
	}
	volumes, err := parseVolumeFlags(cmd)
	if err != nil {
		return api.JobCommonRequest{}, nil, api.ImageBaseInfo{}, err
	}
	datasets, err := parseDatasetFlags(cmd)
	if err != nil {
		return api.JobCommonRequest{}, nil, api.ImageBaseInfo{}, err
	}
	volumes = append(volumes, datasets...)
	selectors, err := parseSelectorFlags(cmd)
	if err != nil {
		return api.JobCommonRequest{}, nil, api.ImageBaseInfo{}, err
	}
	forwards, err := parseForwardFlags(cmd)
	if err != nil {
		return api.JobCommonRequest{}, nil, api.ImageBaseInfo{}, err
	}

	resources := api.ResourceList{
		"cpu":    strconv.FormatFloat(cpu, 'f', -1, 64),
		"memory": memory,
	}
	if gpu > 0 {
		resources[gpuResource] = strconv.Itoa(gpu)
	}
	common := api.JobCommonRequest{
		Name:              name,
		VolumeMounts:      volumes,
		Envs:              envs,
		Selectors:         selectors,
		Template:          template,
		AlertEnabled:      alert,
		CpuPinningEnabled: cpuPinning,
		Forwards:          forwards,
		ScheduleType:      scheduleValue,
	}
	return common, resources, api.ImageBaseInfo{ImageLink: imageLink, Archs: archs}, nil
}

func parseScheduleType(raw string) (*int, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return nil, nil
	}
	switch raw {
	case "normal", "1":
		v := scheduleNormal
		return &v, nil
	case "backfill", "0":
		v := scheduleBackfill
		return &v, nil
	default:
		return nil, fmt.Errorf("%s", i18n.T("err_invalid_job_schedule", raw))
	}
}

func parseEnvFlags(cmd *cobra.Command) ([]api.EnvVar, error) {
	values, _ := cmd.Flags().GetStringArray("env")
	out := make([]api.EnvVar, 0, len(values))
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, errUsageFromIssues([]usageIssue{invalidIssue("env", i18n.T("err_invalid_enum", "env", value))})
		}
		out = append(out, api.EnvVar{Name: strings.TrimSpace(key), Value: val})
	}
	return out, nil
}

func parseVolumeFlags(cmd *cobra.Command) ([]api.VolumeMount, error) {
	values, _ := cmd.Flags().GetStringArray("volume")
	out := make([]api.VolumeMount, 0, len(values))
	for _, value := range values {
		subPath, mountPath, ok := strings.Cut(value, ":")
		if !ok || strings.TrimSpace(mountPath) == "" {
			return nil, errUsageFromIssues([]usageIssue{invalidIssue("volume", i18n.T("err_invalid_enum", "volume", value))})
		}
		if strings.TrimSpace(subPath) == "" {
			return nil, errUsageFromIssues([]usageIssue{invalidIssue("volume", i18n.T("err_invalid_enum", "volume", value))})
		}
		out = append(out, api.VolumeMount{
			Type:      volumeTypeFile,
			SubPath:   strings.TrimSpace(subPath),
			MountPath: strings.TrimSpace(mountPath),
		})
	}
	return out, nil
}

func parseDatasetFlags(cmd *cobra.Command) ([]api.VolumeMount, error) {
	values, _ := cmd.Flags().GetStringArray("dataset")
	out := make([]api.VolumeMount, 0, len(values))
	for _, value := range values {
		rawID, mountPath, ok := strings.Cut(value, ":")
		id, parseErr := strconv.ParseUint(strings.TrimSpace(rawID), 10, 0)
		if !ok || parseErr != nil || id == 0 || strings.TrimSpace(mountPath) == "" {
			return nil, errUsageFromIssues([]usageIssue{invalidIssue("dataset", i18n.T("err_invalid_enum", "dataset", value))})
		}
		out = append(out, api.VolumeMount{
			Type:      volumeTypeDataset,
			DatasetID: uint(id),
			MountPath: strings.TrimSpace(mountPath),
		})
	}
	return out, nil
}

func parseSelectorFlags(cmd *cobra.Command) ([]api.NodeSelectorRequirement, error) {
	values, _ := cmd.Flags().GetStringArray("selector")
	out := make([]api.NodeSelectorRequirement, 0, len(values))
	for _, value := range values {
		key, rest, ok := strings.Cut(value, "=")
		op, rawValues, ok2 := strings.Cut(rest, ":")
		if !ok || !ok2 || strings.TrimSpace(key) == "" || strings.TrimSpace(op) == "" {
			return nil, errUsageFromIssues([]usageIssue{invalidIssue("selector", i18n.T("err_invalid_enum", "selector", value))})
		}
		selector := api.NodeSelectorRequirement{Key: strings.TrimSpace(key), Operator: strings.TrimSpace(op)}
		if strings.TrimSpace(rawValues) != "" {
			selector.Values = splitCSV(rawValues)
		}
		out = append(out, selector)
	}
	return out, nil
}

func parseForwardFlags(cmd *cobra.Command) ([]api.Forward, error) {
	values, _ := cmd.Flags().GetStringArray("forward")
	out := make([]api.Forward, 0, len(values))
	for _, value := range values {
		parts := strings.Split(value, ":")
		if len(parts) != 2 {
			return nil, errUsageFromIssues([]usageIssue{invalidIssue("forward", i18n.T("err_invalid_enum", "forward", value))})
		}
		port, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		name := strings.TrimSpace(parts[0])
		if err != nil || port <= 0 || port > 65535 {
			return nil, errUsageFromIssues([]usageIssue{invalidIssue("forward", i18n.T("err_invalid_positive_int", "forward port"))})
		}
		if !validForwardName(name) {
			return nil, errUsageFromIssues([]usageIssue{invalidIssue("forward", i18n.T("err_invalid_forward_name", name))})
		}
		out = append(out, api.Forward{Name: name, Port: int32(port)})
	}
	return out, nil
}

func validForwardName(name string) bool {
	if len(name) == 0 || len(name) > 20 {
		return false
	}
	for _, r := range name {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

func readJSONFlag(cmd *cobra.Command, dst interface{}) error {
	file, _ := cmd.Flags().GetString("file")
	if strings.TrimSpace(file) == "" {
		return errUsageFromIssues([]usageIssue{missingIssue("file", "job_label_file")})
	}
	return readJSONFile(file, dst)
}

func readJSONFile(path string, dst interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrCommandExecution, Message: i18n.T("err_read_file", path, err.Error())}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrInvalidFlagValue, Message: i18n.T("err_unmarshal_file", path, err.Error())}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrInvalidFlagValue, Message: i18n.T("err_unmarshal_file", path, err.Error())}
	}
	return nil
}

func validateInteractiveRequest(req api.CreateInteractiveJobRequest) error {
	return validateBasicRequest(req.JobCommonRequest, req.Resource, req.Image)
}

func validateTrainingRequest(req api.CreateTrainingJobRequest) error {
	issues := validateBasicIssues(req.JobCommonRequest, req.Resource, req.Image)
	if strings.TrimSpace(req.WorkingDir) == "" {
		issues = append(issues, missingIssue("working-dir", "job_label_working_dir"))
	}
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	return nil
}

func validateDistributedRequest(req api.CreateDistributedJobRequest) error {
	issues := validateCommonIssues(req.JobCommonRequest, false)
	if len(req.Tasks) == 0 {
		issues = append(issues, missingIssue("tasks", "job_label_tasks"))
	}
	for i, task := range req.Tasks {
		prefix := fmt.Sprintf("tasks[%d]", i)
		if strings.TrimSpace(task.Name) == "" {
			issues = append(issues, missingIssue(prefix+".name", "job_label_task_name"))
		}
		if task.Replicas <= 0 {
			issues = append(issues, invalidIssue(prefix+".replicas", i18n.T("err_invalid_positive_int", prefix+".replicas")))
		}
		issues = append(issues, validateResourceIssues(prefix+".resource", task.Resource)...)
		if strings.TrimSpace(task.Image.ImageLink) == "" {
			issues = append(issues, missingIssue(prefix+".image", "job_label_image"))
		}
		for j, port := range task.Ports {
			portField := fmt.Sprintf("%s.ports[%d]", prefix, j)
			if strings.TrimSpace(port.Name) == "" {
				issues = append(issues, invalidIssue(portField+".name", i18n.T("err_value_empty", portField+".name")))
			}
			if port.Port <= 0 || port.Port > 65535 {
				issues = append(issues, invalidIssue(portField+".port", i18n.T("err_invalid_port", portField+".port")))
			}
		}
	}
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	return nil
}

func validateBasicRequest(common api.JobCommonRequest, resource api.ResourceList, image api.ImageBaseInfo) error {
	issues := validateBasicIssues(common, resource, image)
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	return nil
}

func validateBasicIssues(common api.JobCommonRequest, resource api.ResourceList, image api.ImageBaseInfo) []usageIssue {
	issues := validateCommonIssues(common, true)
	issues = append(issues, validateResourceIssues("resource", resource)...)
	if strings.TrimSpace(image.ImageLink) == "" {
		issues = append(issues, missingIssue("image", "job_label_image"))
	}
	return issues
}

func validateCommonIssues(common api.JobCommonRequest, allowBackfill bool) []usageIssue {
	issues := []usageIssue{}
	if strings.TrimSpace(common.Name) == "" {
		issues = append(issues, missingIssue("name", "job_label_display_name"))
	}
	if common.ScheduleType != nil {
		switch *common.ScheduleType {
		case scheduleNormal:
		case scheduleBackfill:
			if !allowBackfill {
				issues = append(issues, invalidIssue("scheduleType", i18n.T("err_job_backfill_distributed")))
			}
		default:
			issues = append(issues, invalidIssue("scheduleType", i18n.T("err_invalid_job_schedule_value", *common.ScheduleType)))
		}
	}
	for i, mount := range common.VolumeMounts {
		field := fmt.Sprintf("volumeMounts[%d]", i)
		if mount.Type != volumeTypeFile && mount.Type != volumeTypeDataset {
			issues = append(issues, invalidIssue(field+".type", i18n.T("err_invalid_volume_type", mount.Type)))
		}
		if !validMountPath(mount.MountPath) {
			issues = append(issues, invalidIssue(field+".mountPath", i18n.T("err_invalid_mount_path", mount.MountPath)))
		}
		switch mount.Type {
		case volumeTypeFile:
			if strings.TrimSpace(mount.SubPath) == "" {
				issues = append(issues, invalidIssue(field+".subPath", i18n.T("err_value_empty", field+".subPath")))
			}
		case volumeTypeDataset:
			if mount.DatasetID == 0 {
				issues = append(issues, invalidIssue(field+".datasetID", i18n.T("err_invalid_positive_int", field+".datasetID")))
			}
		}
	}
	for i, env := range common.Envs {
		if strings.TrimSpace(env.Name) == "" {
			field := fmt.Sprintf("envs[%d].name", i)
			issues = append(issues, invalidIssue(field, i18n.T("err_value_empty", field)))
		}
	}
	operators := []string{"In", "NotIn", "Exists", "DoesNotExist", "Gt", "Lt"}
	for i, selector := range common.Selectors {
		field := fmt.Sprintf("selectors[%d]", i)
		if strings.TrimSpace(selector.Key) == "" {
			issues = append(issues, invalidIssue(field+".key", i18n.T("err_value_empty", field+".key")))
		}
		if !slices.Contains(operators, selector.Operator) {
			issues = append(issues, invalidIssue(field+".operator", i18n.T("err_invalid_enum", field+".operator", selector.Operator)))
		}
		if (selector.Operator == "In" || selector.Operator == "NotIn") && len(selector.Values) == 0 {
			issues = append(issues, invalidIssue(field+".values", i18n.T("err_value_empty", field+".values")))
		}
	}
	for i, forward := range common.Forwards {
		field := fmt.Sprintf("forwards[%d]", i)
		if !validForwardName(strings.TrimSpace(forward.Name)) {
			issues = append(issues, invalidIssue(field+".name", i18n.T("err_invalid_forward_name", forward.Name)))
		}
		if forward.Port <= 0 || forward.Port > 65535 {
			issues = append(issues, invalidIssue(field+".port", i18n.T("err_invalid_port", field+".port")))
		}
	}
	return issues
}

func validMountPath(path string) bool {
	path = strings.TrimSpace(path)
	return strings.HasPrefix(path, "/") &&
		path != "/" &&
		!strings.Contains(path, "..") &&
		!strings.Contains(path, "//")
}

func validateResourceIssues(field string, resources api.ResourceList) []usageIssue {
	issues := []usageIssue{}
	if len(resources) == 0 {
		return append(issues, missingIssue(field, "job_label_resources"))
	}
	for key, value := range resources {
		value = strings.TrimSpace(value)
		if value == "" {
			issues = append(issues, invalidIssue(field+"."+key, i18n.T("err_value_empty", field+"."+key)))
		} else if strings.HasPrefix(value, "-") {
			issues = append(issues, invalidIssue(field+"."+key, i18n.T("err_invalid_non_negative_int", field+"."+key)))
		}
	}
	return issues
}

func filterJobs(cmd *cobra.Command, jobs []api.JobInfo) ([]api.JobInfo, error) {
	if err := validateJobListFilters(cmd); err != nil {
		return nil, err
	}
	status, _ := cmd.Flags().GetString("status")
	jobType, _ := cmd.Flags().GetString("type")
	node, _ := cmd.Flags().GetString("node")
	owner, _ := cmd.Flags().GetString("owner")
	interactive, _ := cmd.Flags().GetBool("interactive")
	batch, _ := cmd.Flags().GetBool("batch")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	fromTime, err := parseOptionalTime(from)
	if err != nil {
		return nil, err
	}
	toTime, err := parseOptionalTime(to)
	if err != nil {
		return nil, err
	}

	out := jobs[:0]
	for _, job := range jobs {
		if status != "" && job.Status != status {
			continue
		}
		if jobType != "" && job.JobType != jobType {
			continue
		}
		if node != "" && !slices.Contains(job.Nodes, node) {
			continue
		}
		if owner != "" && job.UserInfo.Username != owner && job.Owner != owner {
			continue
		}
		isInteractive := job.JobType == "jupyter" || job.JobType == "webide"
		if interactive && !isInteractive {
			continue
		}
		if batch && isInteractive {
			continue
		}
		createdAt := job.CreatedAt
		if fromTime != nil && !createdAt.IsZero() && createdAt.Before(*fromTime) {
			continue
		}
		if toTime != nil && !createdAt.IsZero() && createdAt.After(*toTime) {
			continue
		}
		out = append(out, job)
	}
	return out, nil
}

func validateJobListFilters(cmd *cobra.Command) error {
	issues := jobListFilterIssues(cmd)
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	return nil
}

func jobListFilterIssues(cmd *cobra.Command) []usageIssue {
	days, _ := cmd.Flags().GetInt("days")
	status, _ := cmd.Flags().GetString("status")
	jobType, _ := cmd.Flags().GetString("type")
	interactive, _ := cmd.Flags().GetBool("interactive")
	batch, _ := cmd.Flags().GetBool("batch")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	status = strings.TrimSpace(status)
	jobType = strings.TrimSpace(jobType)
	issues := []usageIssue{}
	if days < -1 {
		issues = append(issues, invalidIssue("days", i18n.T("err_invalid_job_days")))
	}
	if status != "" && !slices.Contains(jobStatuses, status) {
		issues = append(issues, invalidIssue("status", i18n.T("err_invalid_job_status", status)))
	}
	if jobType != "" && !slices.Contains(jobTypes, jobType) {
		issues = append(issues, invalidIssue("type", i18n.T("err_invalid_job_type", jobType)))
	}
	if interactive && batch {
		issues = append(issues, invalidIssue("interactive", i18n.T("err_job_interactive_batch_conflict")))
	}
	var fromTime *time.Time
	if strings.TrimSpace(from) != "" {
		parsed, err := parseAPITime(from)
		if err != nil {
			issues = append(issues, invalidIssue("from", i18n.T("err_invalid_time", "from", from)))
		} else {
			fromTime = &parsed
		}
	}
	var toTime *time.Time
	if strings.TrimSpace(to) != "" {
		parsed, err := parseAPITime(to)
		if err != nil {
			issues = append(issues, invalidIssue("to", i18n.T("err_invalid_time", "to", to)))
		} else {
			toTime = &parsed
		}
	}
	if fromTime != nil && toTime != nil && fromTime.After(*toTime) {
		issues = append(issues, invalidIssue("from", i18n.T("err_invalid_time_range")))
	}
	return issues
}

func parseOptionalTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := parseAPITime(value)
	if err != nil {
		return nil, errUsageFromIssues([]usageIssue{invalidIssue("time", i18n.T("err_invalid_time", "time", value))})
	}
	return &parsed, nil
}

func parseAPITime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "0001-") {
		return time.Time{}, nil
	}
	formats := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"}
	var last error
	for _, layout := range formats {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
		last = err
	}
	return time.Time{}, last
}

func formatAPITime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format(time.RFC3339)
}

func waitMinutesFlag(cmd *cobra.Command) (int, error) {
	wait, _ := cmd.Flags().GetInt("wait-minutes")
	if wait <= 0 {
		return 0, errUsageFromIssues([]usageIssue{invalidIssue("wait-minutes", i18n.T("err_invalid_positive_int", "wait-minutes"))})
	}
	return wait, nil
}

func confirmJobAction(cmd *cobra.Command, messageKey string, args ...interface{}) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		return nil
	}
	if noInteractive {
		return &clierror.Error{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrMissingRequiredFlag,
			Message:  i18n.T("err_confirm_required"),
		}
	}
	var confirmed bool
	prompt := &survey.Confirm{Message: i18n.T(messageKey, args...), Default: false}
	if err := survey.AskOne(prompt, &confirmed); err != nil {
		return errSurveyOrSame(err)
	}
	if !confirmed {
		return errOperationCancelled()
	}
	return nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func writeMessage(msg string, err error) error {
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"message": msg}))
	}
	fmt.Println(msg)
	return nil
}

func writeCreateResult(data map[string]interface{}, err error) error {
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"job": data}))
	}
	fmt.Println(i18n.T("job_create_submitted"))
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok {
			fmt.Printf("%s: %s\n", i18n.T("table_job_name"), name)
		}
	}
	return nil
}

func writeCleanupResult(res *api.CleanupResult, err error) error {
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"cleanup": res}))
	}
	fmt.Printf("%s: %s\n", i18n.T("table_reminded"), strings.Join(res.Reminded, ","))
	fmt.Printf("%s: %s\n", i18n.T("table_deleted"), strings.Join(res.Deleted, ","))
	return nil
}

func writeToken(token *api.JobToken) error {
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"token": token}))
	}
	fmt.Printf("%s: %s\n", i18n.T("table_url"), token.FullURL)
	fmt.Printf("%s: %s\n", i18n.T("table_token"), token.Token)
	fmt.Printf("%s: %s\n", i18n.T("table_pod"), token.PodName)
	return nil
}

func printJobTable(jobs []api.JobInfo) {
	fmt.Printf("%s %s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_name"), 24),
		i18n.PadRight(i18n.T("table_job_name"), 34),
		i18n.PadRight(i18n.T("table_type"), 12),
		i18n.PadRight(i18n.T("table_status"), 14),
		i18n.PadRight(i18n.T("table_queue"), 18),
		i18n.PadRight(i18n.T("table_nodes"), 22),
		i18n.PadRight(i18n.T("table_resources"), 24))
	for _, job := range jobs {
		fmt.Printf("%s %s %s %s %s %s %s\n",
			i18n.PadRight(job.Name, 24),
			i18n.PadRight(job.JobName, 34),
			i18n.PadRight(job.JobType, 12),
			i18n.PadRight(job.Status, 14),
			i18n.PadRight(job.Queue, 18),
			i18n.PadRight(strings.Join(job.Nodes, ","), 22),
			i18n.PadRight(formatResources(job.Resources), 24))
	}
}

func printJobDetail(job *api.JobDetail) {
	if job == nil {
		return
	}
	fmt.Printf("%s: %s\n", i18n.T("table_name"), job.Name)
	fmt.Printf("%s: %s\n", i18n.T("table_job_name"), job.JobName)
	fmt.Printf("%s: %s\n", i18n.T("table_type"), job.JobType)
	fmt.Printf("%s: %s\n", i18n.T("table_status"), job.Status)
	fmt.Printf("%s: %s\n", i18n.T("table_owner"), job.Username)
	fmt.Printf("%s: %s\n", i18n.T("table_queue"), job.Queue)
	fmt.Printf("%s: %s\n", i18n.T("table_resources"), formatResources(job.Resources))
	fmt.Printf("%s: %s\n", i18n.T("table_created_at"), formatAPITime(job.CreatedAt))
	fmt.Printf("%s: %s\n", i18n.T("table_started_at"), formatAPITime(job.StartedAt))
	fmt.Printf("%s: %s\n", i18n.T("table_completed_at"), formatAPITime(job.CompletedAt))
}

func printPodTable(pods []api.PodDetail) {
	fmt.Printf("%s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_name"), 32),
		i18n.PadRight(i18n.T("table_namespace"), 18),
		i18n.PadRight(i18n.T("table_node"), 24),
		i18n.PadRight(i18n.T("table_ip"), 16),
		i18n.PadRight(i18n.T("table_phase"), 12),
		i18n.PadRight(i18n.T("table_resources"), 24))
	for _, pod := range pods {
		fmt.Printf("%s %s %s %s %s %s\n",
			i18n.PadRight(pod.Name, 32),
			i18n.PadRight(pod.Namespace, 18),
			i18n.PadRight(emptyDash(stringValue(pod.NodeName)), 24),
			i18n.PadRight(emptyDash(pod.IP), 16),
			i18n.PadRight(pod.Phase, 12),
			i18n.PadRight(formatResources(pod.Resource), 24))
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func printEvents(events []map[string]interface{}) {
	fmt.Printf("%s %s %s\n",
		i18n.PadRight(i18n.T("table_type"), 10),
		i18n.PadRight(i18n.T("table_reason"), 24),
		i18n.T("table_message"))
	for _, event := range events {
		fmt.Printf("%s %s %s\n",
			i18n.PadRight(fmt.Sprint(event["type"]), 10),
			i18n.PadRight(fmt.Sprint(event["reason"]), 24),
			fmt.Sprint(event["message"]))
	}
}

func formatResources(resources api.ResourceList) string {
	if len(resources) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(resources))
	for key := range resources {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+resources[key])
	}
	return strings.Join(parts, ",")
}

func addCreateCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("file", "", "Read exact JSON request body from file")
	cmd.Flags().String("name", "", "Display name")
	cmd.Flags().String("image", "", "Image link")
	cmd.Flags().StringSlice("arch", nil, "Image architecture, repeatable or comma-separated")
	cmd.Flags().Float64("cpu", 1, "CPU request")
	cmd.Flags().String("memory", "", "Memory request, for example 8Gi")
	cmd.Flags().Int("gpu", 0, "GPU count")
	cmd.Flags().String("gpu-resource", "nvidia.com/gpu", "GPU resource name")
	cmd.Flags().String("template", "", "Template name or JSON")
	cmd.Flags().Bool("alert", false, "Enable alert")
	cmd.Flags().Bool("cpu-pinning", false, "Enable CPU pinning")
	cmd.Flags().String("schedule", "", "Schedule type: normal or backfill")
	cmd.Flags().StringArray("env", nil, "Environment variable KEY=VALUE, repeatable")
	cmd.Flags().StringArray("volume", nil, "Workspace mount subPath:mountPath, repeatable")
	cmd.Flags().StringArray("dataset", nil, "Dataset mount id:mountPath, repeatable")
	cmd.Flags().StringArray("selector", nil, "Node selector key=Operator:value1,value2, repeatable")
	cmd.Flags().StringArray("forward", nil, "Forward name:port, repeatable")
}

func init() {
	jobLsCmd.Flags().Bool("all", false, "List all jobs visible to this account")
	jobLsCmd.Flags().String("user", "", "List jobs for a username")
	jobLsCmd.Flags().Int("days", 0, "Look back days for --all or --user; -1 means all")
	jobLsCmd.Flags().String("status", "", "Filter by job status")
	jobLsCmd.Flags().String("type", "", "Filter by job type")
	jobLsCmd.Flags().String("node", "", "Filter by node name")
	jobLsCmd.Flags().String("owner", "", "Filter by owner username or display name")
	jobLsCmd.Flags().String("from", "", "Filter createdAt from time, RFC3339 or YYYY-MM-DD")
	jobLsCmd.Flags().String("to", "", "Filter createdAt until time, RFC3339 or YYYY-MM-DD")
	jobLsCmd.Flags().Bool("interactive", false, "Only show interactive jobs")
	jobLsCmd.Flags().Bool("batch", false, "Only show batch jobs")

	adminJobLsCmd.Flags().String("user", "", "List jobs for a username")
	adminJobLsCmd.Flags().Int("days", 0, "Look back days; -1 means all")
	adminJobLsCmd.Flags().String("status", "", "Filter by job status")
	adminJobLsCmd.Flags().String("type", "", "Filter by job type")
	adminJobLsCmd.Flags().String("node", "", "Filter by node name")
	adminJobLsCmd.Flags().String("owner", "", "Filter by owner username or display name")
	adminJobLsCmd.Flags().String("from", "", "Filter createdAt from time, RFC3339 or YYYY-MM-DD")
	adminJobLsCmd.Flags().String("to", "", "Filter createdAt until time, RFC3339 or YYYY-MM-DD")
	adminJobLsCmd.Flags().Bool("interactive", false, "Only show interactive jobs")
	adminJobLsCmd.Flags().Bool("batch", false, "Only show batch jobs")

	addCreateCommonFlags(jobCreateJupyterCmd)
	addCreateCommonFlags(jobCreateWebIDECmd)
	addCreateCommonFlags(jobCreateCustomCmd)
	jobCreateCustomCmd.Flags().String("working-dir", "/workspace", "Working directory")
	jobCreateCustomCmd.Flags().String("command", "", "Command to run")
	jobCreateCustomCmd.Flags().String("shell", "sh", "Shell for --command")
	jobCreateTensorflowCmd.Flags().String("file", "", "Read exact JSON request body from file")
	jobCreatePytorchCmd.Flags().String("file", "", "Read exact JSON request body from file")

	jobDeleteCmd.Flags().BoolP("yes", "y", false, "Delete without confirmation")
	adminJobDeleteCmd.Flags().BoolP("yes", "y", false, "Delete without confirmation")
	jobAdminLockCmd.Flags().Bool("permanent", false, "Lock permanently")
	jobAdminLockCmd.Flags().Int("days", 0, "Lock days")
	jobAdminLockCmd.Flags().Int("hours", 0, "Lock hours")
	jobAdminLockCmd.Flags().Int("minutes", 0, "Lock minutes")
	jobAdminCleanWaitingJupyterCmd.Flags().Int("wait-minutes", 0, "Waiting minutes threshold")
	jobAdminCleanWaitingCustomCmd.Flags().Int("wait-minutes", 0, "Waiting minutes threshold")
	jobAdminCleanLongRunningCmd.Flags().Int("batch-days", 0, "Batch job running days threshold")
	jobAdminCleanLongRunningCmd.Flags().Int("interactive-days", 0, "Interactive job running days threshold")
	jobAdminCleanLowGPUCmd.Flags().Int("time-range", 0, "GPU usage lookback range")
	jobAdminCleanLowGPUCmd.Flags().Int("wait-time", 0, "Wait time before cleanup")
	jobAdminCleanLowGPUCmd.Flags().Int("util", 0, "GPU utilization threshold")
	for _, cleanCmd := range []*cobra.Command{
		jobAdminCleanWaitingJupyterCmd,
		jobAdminCleanWaitingCustomCmd,
		jobAdminCleanLongRunningCmd,
		jobAdminCleanLowGPUCmd,
	} {
		cleanCmd.Flags().BoolP("yes", "y", false, "Run cleanup without confirmation")
	}

	completion.RegisterFlagValue([]string{"job", "ls"}, "status", staticValueCompleter(jobStatuses, nil))
	completion.RegisterFlagValue([]string{"job", "ls"}, "type", staticValueCompleter(jobTypes, nil))
	completion.RegisterFlagValue([]string{"admin", "job", "ls"}, "status", staticValueCompleter(jobStatuses, nil))
	completion.RegisterFlagValue([]string{"admin", "job", "ls"}, "type", staticValueCompleter(jobTypes, nil))
	scheduleValues := []string{"normal", "backfill"}
	for _, path := range [][]string{{"job", "create", "jupyter"}, {"job", "create", "webide"}, {"job", "create", "custom"}} {
		completion.RegisterFlagValue(path, "schedule", staticValueCompleter(scheduleValues, nil))
	}

	jobCreateCmd.AddCommand(jobCreateJupyterCmd, jobCreateWebIDECmd, jobCreateCustomCmd, jobCreateTensorflowCmd, jobCreatePytorchCmd)
	jobAdminCleanCmd.AddCommand(jobAdminCleanWaitingJupyterCmd, jobAdminCleanWaitingCustomCmd, jobAdminCleanLongRunningCmd, jobAdminCleanLowGPUCmd)
	adminJobCmd.AddCommand(adminJobLsCmd, adminJobDeleteCmd, jobAdminLockCmd, jobAdminUnlockCmd, jobAdminKeepCmd, jobAdminCleanCmd)
	jobCmd.AddCommand(jobLsCmd, jobGetCmd, jobPodsCmd, jobEventsCmd, jobYAMLCmd, jobTemplateCmd, jobTokenCmd, jobSecretCmd, jobSSHCmd, jobSnapshotCmd, jobAlertCmd, jobDeleteCmd, jobCreateCmd)
	adminCmd.AddCommand(adminJobCmd)
	rootCmd.AddCommand(jobCmd)
}
