package cmd

import (
	"fmt"
	"io"
	"os"
	"regexp"
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
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

const (
	downloadCategoryModel     = "model"
	downloadCategoryDataset   = "dataset"
	downloadSourceDefault     = downloadSourceModelScope
	downloadSourceModelScope  = "modelscope"
	downloadSourceHuggingFace = "huggingface"
	downloadSourceAliasMS     = "ms"
	downloadSourceAliasHF     = "hf"
	downloadStatusReady       = "Ready"
	downloadStatusFailed      = "Failed"
	downloadStatusPaused      = "Paused"
)

var (
	downloadCategories = []string{downloadCategoryModel, downloadCategoryDataset}
	downloadSources    = []string{downloadSourceModelScope, downloadSourceHuggingFace, downloadSourceAliasMS, downloadSourceAliasHF}
	modelNamePattern   = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
)

type downloadCreateInput struct {
	name     string
	category string
	source   string
	revision string
	token    string
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Create and manage model or dataset download tasks",
	Long:  "Create and manage platform-side model or dataset download tasks.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var downloadCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a model or dataset download task",
	Long:  "Submit a platform-side download task for a ModelScope or HuggingFace model or dataset.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errTooManyArgs(cmd, len(args), 0)
		}
		return nil
	},
	RunE: runDownloadCreate,
}

var downloadModelCmd = &cobra.Command{
	Use:   "model <name>",
	Short: "Create a model download task",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDownloadShortcut(cmd, args, downloadCategoryModel)
	},
}

var downloadDatasetCmd = &cobra.Command{
	Use:   "dataset <name>",
	Short: "Create a dataset download task",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDownloadShortcut(cmd, args, downloadCategoryDataset)
	},
}

var downloadLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List model and dataset download tasks",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errTooManyArgs(cmd, len(args), 0)
		}
		return nil
	},
	RunE: runDownloadLs,
}

var downloadGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a download task",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: runDownloadGet,
}

var downloadLogsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "Show download task logs",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: runDownloadLogs,
}

var downloadPauseCmd = &cobra.Command{
	Use:   "pause <id>",
	Short: "Pause a download task",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: runDownloadPause,
}

var downloadResumeCmd = &cobra.Command{
	Use:   "resume <id>",
	Short: "Resume a download task",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: runDownloadResume,
}

var downloadRetryCmd = &cobra.Command{
	Use:   "retry <id>",
	Short: "Retry a failed download task",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: runDownloadRetry,
}

var downloadRmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: "Remove a download task",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: runDownloadRm,
}

func readDownloadCreateInput(cmd *cobra.Command) downloadCreateInput {
	name, _ := cmd.Flags().GetString("name")
	category, _ := cmd.Flags().GetString("category")
	source, _ := cmd.Flags().GetString("source")
	revision, _ := cmd.Flags().GetString("revision")
	token, _ := cmd.Flags().GetString("token")
	return downloadCreateInput{
		name:     strings.TrimSpace(name),
		category: strings.TrimSpace(category),
		source:   strings.TrimSpace(source),
		revision: strings.TrimSpace(revision),
		token:    strings.TrimSpace(token),
	}
}

func downloadCategoryValid(v string) bool {
	return slices.Contains(downloadCategories, v)
}

func downloadSourceValid(v string) bool {
	return slices.Contains(downloadSources, v)
}

func normalizeDownloadSource(v string) string {
	switch v {
	case downloadSourceAliasMS:
		return downloadSourceModelScope
	case downloadSourceAliasHF:
		return downloadSourceHuggingFace
	default:
		return v
	}
}

func collectDownloadCreateUsageIssues(in downloadCreateInput) []usageIssue {
	var issues []usageIssue
	if in.name == "" {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_flag_non_interactive", i18n.T("download_label_name"), "name"),
			Field:   "name",
		})
	} else if !modelNamePattern.MatchString(in.name) {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_download_name", in.name),
			Field:   "name",
		})
	}
	if in.category == "" {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_flag_non_interactive", i18n.T("download_label_category"), "category"),
			Field:   "category",
		})
	} else if !downloadCategoryValid(in.category) {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_download_category", in.category),
			Field:   "category",
		})
	}
	if in.source == "" {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_flag_non_interactive", i18n.T("download_label_source"), "source"),
			Field:   "source",
		})
	} else if !downloadSourceValid(in.source) {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_download_source", in.source),
			Field:   "source",
		})
	}
	return issues
}

func collectTokenUsageIssues(cmd *cobra.Command) []usageIssue {
	tokenSet := flagChanged(cmd, "token")
	tokenEnvSet := flagChanged(cmd, "token-env")
	tokenStdinSet := flagChanged(cmd, "token-stdin")
	count := 0
	for _, set := range []bool{tokenSet, tokenEnvSet, tokenStdinSet} {
		if set {
			count++
		}
	}
	if count <= 1 {
		return nil
	}
	return []usageIssue{{
		Code:    errorcodes.ErrInvalidFlagValue,
		Message: i18n.T("err_download_token_source_conflict"),
		Field:   "token",
	}}
}

func flagChanged(cmd *cobra.Command, name string) bool {
	f := cmd.Flags().Lookup(name)
	return f != nil && f.Changed
}

func resolveDownloadToken(cmd *cobra.Command, current string) (string, error) {
	if flagChanged(cmd, "token-env") {
		envName, _ := cmd.Flags().GetString("token-env")
		envName = strings.TrimSpace(envName)
		if envName == "" {
			return "", errUsageFromIssues([]usageIssue{{
				Code:    errorcodes.ErrMissingRequiredFlag,
				Message: i18n.T("err_missing_required", i18n.T("download_label_token_env"), "token-env"),
				Field:   "token-env",
			}})
		}
		v, ok := os.LookupEnv(envName)
		if !ok || strings.TrimSpace(v) == "" {
			return "", errUsageFromIssues([]usageIssue{{
				Code:    errorcodes.ErrMissingRequiredFlag,
				Message: i18n.T("err_download_token_env_missing", envName),
				Field:   "token-env",
			}})
		}
		return strings.TrimSpace(v), nil
	}
	if flagChanged(cmd, "token-stdin") {
		useStdin, _ := cmd.Flags().GetBool("token-stdin")
		if useStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", &clierror.Error{
					Category: errorcodes.CategorySystem,
					Code:     errorcodes.ErrCommandExecution,
					Message:  i18n.T("err_download_token_stdin_read", err.Error()),
				}
			}
			token := strings.TrimSpace(string(data))
			if token == "" {
				return "", errUsageFromIssues([]usageIssue{{
					Code:    errorcodes.ErrMissingRequiredFlag,
					Message: i18n.T("err_download_token_stdin_empty"),
					Field:   "token-stdin",
				}})
			}
			return token, nil
		}
	}
	return current, nil
}

func activeModelDownloadClient() (api.ModelDownloadClient, error) {
	st, err := session.LoadState()
	if err != nil {
		return nil, &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  i18n.T("err_config_write", err.Error()),
		}
	}
	active := st.ActiveContext
	if active.PlatformURL == "" {
		return nil, &clierror.Error{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrNotFound,
			Message:  i18n.T("err_no_active"),
		}
	}
	token, err := session.LoadToken(active)
	if err != nil {
		return nil, &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrSecureStorageError,
			Message:  i18n.T("err_token_load_failed", err.Error()),
		}
	}
	return api.NewModelDownloadClient(active.PlatformURL, token), nil
}

func runDownloadCreate(cmd *cobra.Command, _ []string) error {
	in := readDownloadCreateInput(cmd)
	issues := collectDownloadCreateUsageIssues(in)
	issues = append(issues, collectTokenUsageIssues(cmd)...)
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	token, err := resolveDownloadToken(cmd, in.token)
	if err != nil {
		return err
	}
	in.token = token
	return submitDownload(cmd, in)
}

func runDownloadShortcut(cmd *cobra.Command, args []string, category string) error {
	source, _ := cmd.Flags().GetString("source")
	revision, _ := cmd.Flags().GetString("revision")
	token, _ := cmd.Flags().GetString("token")
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	in := downloadCreateInput{
		name:     strings.TrimSpace(name),
		category: category,
		source:   strings.TrimSpace(source),
		revision: strings.TrimSpace(revision),
		token:    strings.TrimSpace(token),
	}
	issues := collectDownloadShortcutUsageIssues(in)
	issues = append(issues, collectTokenUsageIssues(cmd)...)
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	resolvedToken, err := resolveDownloadToken(cmd, in.token)
	if err != nil {
		return err
	}
	in.token = resolvedToken
	return submitDownload(cmd, in)
}

func submitDownload(cmd *cobra.Command, in downloadCreateInput) error {
	client, err := activeModelDownloadClient()
	if err != nil {
		return err
	}

	download, msg, err := client.CreateDownload(api.CreateModelDownloadReq{
		Name:     in.name,
		Revision: in.revision,
		Source:   normalizeDownloadSource(in.source),
		Category: in.category,
		Token:    in.token,
	})
	if err != nil {
		return cliErrFromAPI(err)
	}

	wait, _ := cmd.Flags().GetBool("wait")
	if outputJSON && wait {
		return waitForDownload(cmd, client, download.ID)
	}
	if outputJSON {
		env := output.SuccessEnvelope(map[string]interface{}{
			"download": download,
		})
		if msg != "" {
			env["message"] = msg
		}
		return output.WriteSuccessJSON(os.Stdout, env)
	}

	if msg != "" {
		fmt.Printf("%s\n", msg)
	}
	printDownload(download)

	if wait {
		return waitForDownload(cmd, client, download.ID)
	}
	return nil
}

func runDownloadLs(cmd *cobra.Command, _ []string) error {
	category, _ := cmd.Flags().GetString("category")
	category = strings.TrimSpace(category)
	if category != "" && !downloadCategoryValid(category) {
		return errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_download_category", category),
			Field:   "category",
		}})
	}
	client, err := activeModelDownloadClient()
	if err != nil {
		return err
	}
	downloads, err := client.ListDownloads(category)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"downloads": downloads,
		}))
	}
	printDownloadTable(downloads)
	return nil
}

func runDownloadGet(cmd *cobra.Command, args []string) error {
	id, err := parseDownloadID(args)
	if err != nil {
		return err
	}
	client, err := activeModelDownloadClient()
	if err != nil {
		return err
	}
	download, err := client.GetDownload(id)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"download": download,
		}))
	}
	printDownload(download)
	return nil
}

func runDownloadLogs(cmd *cobra.Command, args []string) error {
	id, err := parseDownloadID(args)
	if err != nil {
		return err
	}
	follow, _ := cmd.Flags().GetBool("follow")
	interval, _ := cmd.Flags().GetDuration("poll-interval")
	if follow && outputJSON {
		return errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_download_follow_json"),
			Field:   "follow",
		}})
	}
	client, err := activeModelDownloadClient()
	if err != nil {
		return err
	}
	if !follow {
		logs, err := client.GetDownloadLogs(id)
		if err != nil {
			return cliErrFromAPI(err)
		}
		if outputJSON {
			return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
				"logs": logs,
			}))
		}
		fmt.Print(logs)
		if logs != "" && !strings.HasSuffix(logs, "\n") {
			fmt.Println()
		}
		return nil
	}
	return followDownloadLogs(client, id, interval)
}

func runDownloadPause(cmd *cobra.Command, args []string) error {
	return runDownloadAction(cmd, args, func(c api.ModelDownloadClient, id uint) (*api.ModelDownloadResp, error) {
		return c.PauseDownload(id)
	})
}

func runDownloadResume(cmd *cobra.Command, args []string) error {
	return runDownloadAction(cmd, args, func(c api.ModelDownloadClient, id uint) (*api.ModelDownloadResp, error) {
		return c.ResumeDownload(id)
	})
}

func runDownloadRetry(cmd *cobra.Command, args []string) error {
	return runDownloadAction(cmd, args, func(c api.ModelDownloadClient, id uint) (*api.ModelDownloadResp, error) {
		return c.RetryDownload(id)
	})
}

func runDownloadAction(_ *cobra.Command, args []string, action func(api.ModelDownloadClient, uint) (*api.ModelDownloadResp, error)) error {
	id, err := parseDownloadID(args)
	if err != nil {
		return err
	}
	client, err := activeModelDownloadClient()
	if err != nil {
		return err
	}
	download, err := action(client, id)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"download": download,
		}))
	}
	printDownload(download)
	return nil
}

func runDownloadRm(cmd *cobra.Command, args []string) error {
	id, err := parseDownloadID(args)
	if err != nil {
		return err
	}
	force, _ := cmd.Flags().GetBool("yes")
	if noInteractive && !force {
		return &clierror.Error{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrMissingRequiredFlag,
			Message:  i18n.T("err_missing_required", "yes", "yes"),
		}
	}
	if !force {
		var confirm bool
		prompt := &survey.Confirm{Message: i18n.T("download_remove_confirm", id), Default: false}
		if err := survey.AskOne(prompt, &confirm); err != nil {
			return errSurveyOrSame(err)
		}
		if !confirm {
			return errOperationCancelled()
		}
	}
	client, err := activeModelDownloadClient()
	if err != nil {
		return err
	}
	message, err := client.DeleteDownload(id)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"id":      id,
			"message": message,
		}))
	}
	fmt.Printf("%s\n", i18n.T("download_remove_success", id))
	return nil
}

func parseDownloadID(args []string) (uint, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return 0, errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_download_id_arg"),
			Field:   "id",
		}})
	}
	id64, err := strconv.ParseUint(args[0], 10, 0)
	if err != nil || id64 == 0 {
		return 0, errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_download_id", args[0]),
			Field:   "id",
		}})
	}
	return uint(id64), nil
}

func collectDownloadShortcutUsageIssues(in downloadCreateInput) []usageIssue {
	var issues []usageIssue
	if in.name == "" {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_download_name_arg"),
			Field:   "name",
		})
	} else if !modelNamePattern.MatchString(in.name) {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_download_name", in.name),
			Field:   "name",
		})
	}
	if in.source == "" {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_flag_non_interactive", i18n.T("download_label_source"), "source"),
			Field:   "source",
		})
	} else if !downloadSourceValid(in.source) {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_download_source", in.source),
			Field:   "source",
		})
	}
	return issues
}

func printDownload(download *api.ModelDownloadResp) {
	if download == nil {
		return
	}
	fmt.Printf("%s\n", i18n.T(
		"download_task_summary",
		download.ID,
		download.Name,
		download.Category,
		download.Source,
		download.Status,
		download.Path,
	))
}

func printDownloadTable(downloads []api.ModelDownloadResp) {
	header := fmt.Sprintf("%s %s %s %s %s %s",
		i18n.PadRight(i18n.T("table_id"), 8),
		i18n.PadRight(i18n.T("table_name"), 34),
		i18n.PadRight(i18n.T("table_category"), 10),
		i18n.PadRight(i18n.T("table_source"), 12),
		i18n.PadRight(i18n.T("table_status"), 12),
		i18n.PadRight(i18n.T("table_path"), 20))
	fmt.Println(header)
	for _, d := range downloads {
		row := fmt.Sprintf("%s %s %s %s %s %s",
			i18n.PadRight(fmt.Sprintf("%d", d.ID), 8),
			i18n.PadRight(d.Name, 34),
			i18n.PadRight(d.Category, 10),
			i18n.PadRight(d.Source, 12),
			i18n.PadRight(d.Status, 12),
			i18n.PadRight(d.Path, 20))
		fmt.Println(row)
	}
}

func waitForDownload(cmd *cobra.Command, client api.ModelDownloadClient, id uint) error {
	interval, _ := cmd.Flags().GetDuration("poll-interval")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Time{}
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}
	for {
		d, err := client.GetDownload(id)
		if err != nil {
			return cliErrFromAPI(err)
		}
		if d.Status == downloadStatusReady || d.Status == downloadStatusFailed || d.Status == downloadStatusPaused {
			if outputJSON {
				return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
					"download": d,
				}))
			}
			fmt.Printf("%s\n", i18n.T("download_wait_done", d.Status))
			printDownload(d)
			return nil
		}
		if !outputJSON {
			fmt.Printf("%s\n", i18n.T("download_wait_status", d.Status))
		}
		if !deadline.IsZero() && time.Now().Add(interval).After(deadline) {
			return &clierror.Error{
				Category: errorcodes.CategorySystem,
				Code:     errorcodes.ErrCommandExecution,
				Message:  i18n.T("err_download_wait_timeout", timeout.String()),
			}
		}
		time.Sleep(interval)
	}
}

func followDownloadLogs(client api.ModelDownloadClient, id uint, interval time.Duration) error {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	last := ""
	for {
		logs, err := client.GetDownloadLogs(id)
		if err != nil {
			return cliErrFromAPI(err)
		}
		if logs != last {
			if strings.HasPrefix(logs, last) {
				fmt.Print(strings.TrimPrefix(logs, last))
			} else {
				fmt.Print(logs)
			}
			if logs != "" && !strings.HasSuffix(logs, "\n") {
				fmt.Println()
			}
			last = logs
		}
		d, err := client.GetDownload(id)
		if err != nil {
			return cliErrFromAPI(err)
		}
		if d.Status == downloadStatusReady || d.Status == downloadStatusFailed || d.Status == downloadStatusPaused {
			return nil
		}
		time.Sleep(interval)
	}
}

func staticValueCompleter(values []string, descKey func(string) string) func(completion.Context) ([]completion.Candidate, error) {
	return func(ctx completion.Context) ([]completion.Candidate, error) {
		prefix := strings.ToLower(completion.CurrentWordPrefix(ctx))
		out := make([]completion.Candidate, 0, len(values))
		for _, v := range values {
			if prefix != "" && !strings.HasPrefix(strings.ToLower(v), prefix) {
				continue
			}
			c := completion.Candidate{Value: v}
			if descKey != nil {
				c.Description = i18n.T(descKey(v))
			}
			out = append(out, c)
		}
		return out, nil
	}
}

func init() {
	addDownloadCreateFlags(downloadCreateCmd, true)
	addDownloadCreateFlags(downloadModelCmd, false)
	addDownloadCreateFlags(downloadDatasetCmd, false)
	downloadLsCmd.Flags().String("category", "", "Filter by category (model | dataset)")
	downloadLogsCmd.Flags().Bool("follow", false, "Follow logs until the download reaches a terminal status")
	downloadLogsCmd.Flags().Duration("poll-interval", 5*time.Second, "Polling interval for --follow")
	downloadRmCmd.Flags().BoolP("yes", "y", false, "Remove without confirmation")

	completion.RegisterFlagValue([]string{"download", "create"}, "category", staticValueCompleter(downloadCategories, func(v string) string {
		return "download_category_" + v + "_desc"
	}))
	completion.RegisterFlagValue([]string{"download", "create"}, "source", staticValueCompleter(downloadSources, func(v string) string {
		return "download_source_" + v + "_desc"
	}))
	completion.RegisterFlagValue([]string{"download", "model"}, "source", staticValueCompleter(downloadSources, func(v string) string {
		return "download_source_" + v + "_desc"
	}))
	completion.RegisterFlagValue([]string{"download", "dataset"}, "source", staticValueCompleter(downloadSources, func(v string) string {
		return "download_source_" + v + "_desc"
	}))
	completion.RegisterFlagValue([]string{"download", "ls"}, "category", staticValueCompleter(downloadCategories, func(v string) string {
		return "download_category_" + v + "_desc"
	}))

	downloadCmd.AddCommand(downloadCreateCmd)
	downloadCmd.AddCommand(downloadModelCmd)
	downloadCmd.AddCommand(downloadDatasetCmd)
	downloadCmd.AddCommand(downloadLsCmd)
	downloadCmd.AddCommand(downloadGetCmd)
	downloadCmd.AddCommand(downloadLogsCmd)
	downloadCmd.AddCommand(downloadPauseCmd)
	downloadCmd.AddCommand(downloadResumeCmd)
	downloadCmd.AddCommand(downloadRetryCmd)
	downloadCmd.AddCommand(downloadRmCmd)
	rootCmd.AddCommand(downloadCmd)
}

func addDownloadCreateFlags(cmd *cobra.Command, includeCategory bool) {
	if includeCategory {
		cmd.Flags().String("category", "", "Download category (model | dataset)")
	}
	if includeCategory {
		cmd.Flags().String("name", "", "Resource name in owner/name format")
	}
	cmd.Flags().String("source", downloadSourceDefault, "Download source (modelscope | ms | huggingface | hf)")
	cmd.Flags().String("revision", "", "Optional branch, tag, or revision")
	cmd.Flags().String("token", "", "Optional access token for gated or private repositories")
	cmd.Flags().String("token-env", "", "Read the optional repository access token from an environment variable")
	cmd.Flags().Bool("token-stdin", false, "Read the optional repository access token from stdin")
	cmd.Flags().Bool("wait", false, "Wait until the download reaches Ready, Failed, or Paused")
	cmd.Flags().Duration("poll-interval", 5*time.Second, "Polling interval for --wait")
	cmd.Flags().Duration("timeout", 0, "Maximum time to wait; 0 means no timeout")
}
