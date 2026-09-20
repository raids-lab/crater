package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

var jobLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show logs for a job",
	Args:  exactArgs(1, "job-name"),
	RunE:  runJobLogs,
}

type jobLogOptions struct {
	Pod           string
	AllPods       bool
	Container     string
	AllContainers bool
	TailLines     int64
	Timestamps    bool
	Previous      bool
	Follow        bool
	Prefix        bool
}

type jobLogSource struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
}

type jobLogResult struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
	Content   string `json:"content"`
}

func runJobLogs(cmd *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	options, err := readJobLogOptions(cmd)
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
	selectedPods, err := selectJobLogPods(pods, options)
	if err != nil {
		return err
	}
	if options.Follow && len(selectedPods) != 1 {
		return newJobLogUsageError(
			errorcodes.ErrInvalidFlagValue,
			i18n.T("err_job_logs_follow_multiple_pods", len(selectedPods)),
			map[string]interface{}{"pods": podNames(selectedPods)},
		)
	}
	sources, err := resolveJobLogSources(client, selectedPods, options)
	if err != nil {
		return err
	}

	logOptions := api.PodLogOptions{
		TailLines:  options.TailLines,
		Timestamps: options.Timestamps,
		Previous:   options.Previous,
	}
	if options.Follow {
		if len(sources) != 1 {
			return newJobLogUsageError(
				errorcodes.ErrInvalidFlagValue,
				i18n.T("err_job_logs_follow_multiple", len(sources)),
				map[string]interface{}{"sources": sources},
			)
		}
		dst := io.Writer(os.Stdout)
		if options.Prefix {
			dst = &linePrefixWriter{
				dst:         os.Stdout,
				prefix:      []byte(jobLogPrefix(sources[0])),
				atLineStart: true,
			}
		}
		source := sources[0]
		// Keep SIGINT handling local to the operation that consumes this context.
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()
		if err := client.StreamPodLogs(
			ctx,
			dst,
			source.Namespace,
			source.Pod,
			source.Container,
			logOptions,
		); err != nil {
			return cliErrFromPodLog(err)
		}
		return nil
	}

	if !outputJSON {
		return writeJobLogSources(
			os.Stdout,
			client,
			sources,
			logOptions,
			options.Prefix || len(sources) > 1,
		)
	}

	results := make([]jobLogResult, 0, len(sources))
	for _, source := range sources {
		data, err := client.GetPodLogs(
			source.Namespace,
			source.Pod,
			source.Container,
			logOptions,
		)
		if err != nil {
			return cliErrFromPodLog(err)
		}
		results = append(results, jobLogResult{
			Namespace: source.Namespace,
			Pod:       source.Pod,
			Container: source.Container,
			Content:   string(data),
		})
	}

	return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
		"logs": results,
	}))
}

func readJobLogOptions(cmd *cobra.Command) (jobLogOptions, error) {
	options := jobLogOptions{}
	options.Pod, _ = cmd.Flags().GetString("pod")
	options.Pod = strings.TrimSpace(options.Pod)
	options.AllPods, _ = cmd.Flags().GetBool("all-pods")
	options.Container, _ = cmd.Flags().GetString("container")
	options.Container = strings.TrimSpace(options.Container)
	options.AllContainers, _ = cmd.Flags().GetBool("all-containers")
	options.TailLines, _ = cmd.Flags().GetInt64("tail")
	options.Timestamps, _ = cmd.Flags().GetBool("timestamps")
	options.Previous, _ = cmd.Flags().GetBool("previous")
	options.Follow, _ = cmd.Flags().GetBool("follow")
	options.Prefix, _ = cmd.Flags().GetBool("prefix")

	issues := []usageIssue{}
	if options.Pod != "" && options.AllPods {
		issues = append(issues, invalidIssue(
			"pod",
			i18n.T("err_job_logs_mutually_exclusive", "--pod", "--all-pods"),
		))
	}
	if options.Container != "" && options.AllContainers {
		issues = append(issues, invalidIssue(
			"container",
			i18n.T("err_job_logs_mutually_exclusive", "--container", "--all-containers"),
		))
	}
	if options.TailLines < 0 {
		issues = append(issues, invalidIssue("tail", i18n.T("err_invalid_non_negative_int", "tail")))
	}
	if options.Follow && outputJSON {
		issues = append(issues, invalidIssue("follow", i18n.T("err_job_logs_follow_json")))
	}
	if options.Follow && options.Previous {
		issues = append(issues, invalidIssue("follow", i18n.T("err_job_logs_follow_previous")))
	}
	if len(issues) > 0 {
		return options, errUsageFromIssues(issues)
	}
	return options, nil
}

func selectJobLogPods(pods []api.PodDetail, options jobLogOptions) ([]api.PodDetail, error) {
	candidates := make([]api.PodDetail, 0, len(pods))
	for _, pod := range pods {
		if strings.TrimSpace(pod.Name) == "" || strings.TrimSpace(pod.Namespace) == "" {
			continue
		}
		candidates = append(candidates, pod)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Namespace == candidates[j].Namespace {
			return candidates[i].Name < candidates[j].Name
		}
		return candidates[i].Namespace < candidates[j].Namespace
	})

	if options.Pod != "" {
		for _, pod := range candidates {
			if pod.Name == options.Pod {
				return []api.PodDetail{pod}, nil
			}
		}
		return nil, newJobLogUsageError(
			errorcodes.ErrNotFound,
			i18n.T("err_job_logs_pod_not_found", options.Pod, joinPodNames(candidates)),
			map[string]interface{}{"pod": options.Pod, "available_pods": podNames(candidates)},
		)
	}
	if len(candidates) == 0 {
		return nil, newJobLogUsageError(
			errorcodes.ErrNotFound,
			i18n.T("err_job_logs_no_pods"),
			nil,
		)
	}
	if options.AllPods || len(candidates) == 1 {
		return candidates, nil
	}
	return nil, newJobLogUsageError(
		errorcodes.ErrMissingRequiredFlag,
		i18n.T("err_job_logs_multiple_pods", joinPodNames(candidates)),
		map[string]interface{}{"available_pods": podNames(candidates)},
	)
}

func resolveJobLogSources(
	client *api.Client,
	pods []api.PodDetail,
	options jobLogOptions,
) ([]jobLogSource, error) {
	sources := make([]jobLogSource, 0, len(pods))
	for _, pod := range pods {
		containers, err := client.GetPodContainers(pod.Namespace, pod.Name)
		if err != nil {
			return nil, cliErrFromAPI(err)
		}
		selected, err := selectJobLogContainers(pod.Name, containers, options)
		if err != nil {
			return nil, err
		}
		for _, container := range selected {
			sources = append(sources, jobLogSource{
				Namespace: pod.Namespace,
				Pod:       pod.Name,
				Container: container.Name,
			})
		}
	}
	sort.Slice(sources, func(i, j int) bool {
		if sources[i].Namespace != sources[j].Namespace {
			return sources[i].Namespace < sources[j].Namespace
		}
		if sources[i].Pod != sources[j].Pod {
			return sources[i].Pod < sources[j].Pod
		}
		return sources[i].Container < sources[j].Container
	})
	return sources, nil
}

func selectJobLogContainers(
	podName string,
	containers []api.PodContainerInfo,
	options jobLogOptions,
) ([]api.PodContainerInfo, error) {
	all := make([]api.PodContainerInfo, 0, len(containers))
	regular := make([]api.PodContainerInfo, 0, len(containers))
	for _, container := range containers {
		if strings.TrimSpace(container.Name) == "" {
			continue
		}
		all = append(all, container)
		if !container.IsInitContainer {
			regular = append(regular, container)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	sort.Slice(regular, func(i, j int) bool { return regular[i].Name < regular[j].Name })

	if options.Container != "" {
		for _, container := range all {
			if container.Name == options.Container {
				return []api.PodContainerInfo{container}, nil
			}
		}
		return nil, newJobLogUsageError(
			errorcodes.ErrNotFound,
			i18n.T("err_job_logs_container_not_found", options.Container, podName, joinContainerNames(all)),
			map[string]interface{}{
				"pod":                  podName,
				"container":            options.Container,
				"available_containers": containerNames(all),
			},
		)
	}
	if len(all) == 0 {
		return nil, newJobLogUsageError(
			errorcodes.ErrNotFound,
			i18n.T("err_job_logs_no_containers", podName),
			map[string]interface{}{"pod": podName},
		)
	}
	if options.AllContainers {
		return all, nil
	}
	if len(regular) == 0 {
		return nil, newJobLogUsageError(
			errorcodes.ErrNotFound,
			i18n.T("err_job_logs_no_regular_containers", podName, joinContainerNames(all)),
			map[string]interface{}{
				"pod":                  podName,
				"available_containers": containerNames(all),
			},
		)
	}
	if len(regular) == 1 {
		return regular, nil
	}
	return nil, newJobLogUsageError(
		errorcodes.ErrMissingRequiredFlag,
		i18n.T("err_job_logs_multiple_containers", podName, joinContainerNames(regular)),
		map[string]interface{}{
			"pod":                  podName,
			"available_containers": containerNames(regular),
		},
	)
}

func writeJobLogs(dst io.Writer, results []jobLogResult, prefix bool) error {
	for _, result := range results {
		if err := writeJobLog(dst, jobLogSource{
			Pod:       result.Pod,
			Container: result.Container,
		}, []byte(result.Content), prefix); err != nil {
			return err
		}
	}
	return nil
}

func writeJobLog(dst io.Writer, source jobLogSource, data []byte, prefix bool) error {
	if prefix {
		return writePrefixedLog(dst, jobLogPrefix(source), data)
	}
	if _, err := dst.Write(data); err != nil {
		return err
	}
	if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
		_, err := io.WriteString(dst, "\n")
		return err
	}
	return nil
}

type jobLogGetter interface {
	GetPodLogs(namespace, pod, container string, options api.PodLogOptions) ([]byte, error)
}

func writeJobLogSources(
	dst io.Writer,
	client jobLogGetter,
	sources []jobLogSource,
	options api.PodLogOptions,
	prefix bool,
) error {
	for _, source := range sources {
		data, err := client.GetPodLogs(
			source.Namespace,
			source.Pod,
			source.Container,
			options,
		)
		if err != nil {
			return cliErrFromPodLog(err)
		}
		if err := writeJobLog(dst, source, data, prefix); err != nil {
			return cliErrFromPodLog(&api.PodLogWriteError{Cause: err})
		}
	}
	return nil
}

func writePrefixedLog(dst io.Writer, prefix string, data []byte) error {
	writer := &linePrefixWriter{
		dst:         dst,
		prefix:      []byte(prefix),
		atLineStart: true,
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
		_, err := io.WriteString(dst, "\n")
		return err
	}
	return nil
}

type linePrefixWriter struct {
	dst         io.Writer
	prefix      []byte
	atLineStart bool
}

func (w *linePrefixWriter) Write(data []byte) (int, error) {
	written := 0
	for len(data) > 0 {
		if w.atLineStart {
			n, err := w.dst.Write(w.prefix)
			if err != nil {
				return written, err
			}
			if n != len(w.prefix) {
				return written, io.ErrShortWrite
			}
			w.atLineStart = false
		}
		index := bytes.IndexByte(data, '\n')
		size := len(data)
		if index >= 0 {
			size = index + 1
		}
		n, err := w.dst.Write(data[:size])
		written += n
		if err != nil {
			return written, err
		}
		if n != size {
			return written, io.ErrShortWrite
		}
		if index >= 0 {
			w.atLineStart = true
		}
		data = data[size:]
	}
	return written, nil
}

func jobLogPrefix(source jobLogSource) string {
	return fmt.Sprintf("[%s/%s] ", source.Pod, source.Container)
}

func podNames(pods []api.PodDetail) []string {
	names := make([]string, len(pods))
	for i := range pods {
		names[i] = pods[i].Name
	}
	return names
}

func joinPodNames(pods []api.PodDetail) string {
	return strings.Join(podNames(pods), ", ")
}

func containerNames(containers []api.PodContainerInfo) []string {
	names := make([]string, len(containers))
	for i := range containers {
		names[i] = containers[i].Name
	}
	return names
}

func joinContainerNames(containers []api.PodContainerInfo) string {
	return strings.Join(containerNames(containers), ", ")
}

func newJobLogUsageError(
	code, message string,
	context map[string]interface{},
) *clierror.Error {
	return &clierror.Error{
		Category: errorcodes.CategoryUsage,
		Code:     code,
		Message:  message,
		Context:  context,
	}
}

func cliErrFromPodLog(err error) error {
	if errors.Is(err, context.Canceled) {
		return errOperationCancelled()
	}
	var writeErr *api.PodLogWriteError
	if errors.As(err, &writeErr) {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrCommandExecution,
			Message:  i18n.T("err_job_logs_write", writeErr.Cause),
		}
	}
	var protocolErr *api.PodLogProtocolError
	if errors.As(err, &protocolErr) {
		return &clierror.Error{
			Category: errorcodes.CategoryAPI,
			Code:     errorcodes.ErrAPIOther,
			Message:  i18n.T("err_job_logs_protocol", protocolErr.Cause),
		}
	}
	return cliErrFromAPI(err)
}

func addJobLogFlags(cmd *cobra.Command) {
	cmd.Flags().String("pod", "", "Select a pod by name")
	cmd.Flags().Bool("all-pods", false, "Show logs from every pod in the job")
	cmd.Flags().StringP("container", "c", "", "Select a container by name")
	cmd.Flags().Bool("all-containers", false, "Show logs from every container")
	cmd.Flags().Int64("tail", 0, "Number of recent log lines; 0 means all")
	cmd.Flags().Bool("timestamps", false, "Include timestamps in logs")
	cmd.Flags().BoolP("previous", "p", false, "Return previous terminated container logs")
	cmd.Flags().BoolP("follow", "f", false, "Follow a single pod container log")
	cmd.Flags().Bool("prefix", false, "Prefix text output lines with pod and container names")
}

func init() {
	addJobLogFlags(jobLogsCmd)
	jobCmd.AddCommand(jobLogsCmd)
}
