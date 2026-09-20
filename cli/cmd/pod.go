package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/spf13/cobra"
)

var podCmd = &cobra.Command{
	Use:   "pod",
	Short: "View pod diagnostics",
	Long:  "View pod containers, events, logs, ingresses, and nodeports.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var podContainersCmd = &cobra.Command{Use: "containers [namespace] <pod>", Short: "List pod containers", Args: maxTwoArgs, RunE: runPodContainers}
var podEventsCmd = &cobra.Command{Use: "events [namespace] <pod>", Short: "List pod events", Args: maxTwoArgs, RunE: runPodEvents}
var podLogsCmd = &cobra.Command{Use: "logs [namespace] <pod> <container>", Short: "Show pod container logs", Args: maxThreeArgs, RunE: runPodLogs}
var podIngressesCmd = &cobra.Command{Use: "ingresses [namespace] <pod>", Short: "List pod ingresses", Args: maxTwoArgs, RunE: runPodIngresses}
var podNodeportsCmd = &cobra.Command{Use: "nodeports [namespace] <pod>", Short: "List pod nodeports", Args: maxTwoArgs, RunE: runPodNodeports}

func maxTwoArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 2 {
		return errTooManyArgs(cmd, len(args), 2)
	}
	return nil
}

func maxThreeArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 3 {
		return errTooManyArgs(cmd, len(args), 3)
	}
	return nil
}

func podNamespaceFlag(cmd *cobra.Command) (string, error) {
	namespace, _ := cmd.Flags().GetString("namespace")
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return "", errUsageFromIssues([]usageIssue{missingIssue("namespace", "pod_label_namespace")})
	}
	return namespace, nil
}

func podNamespaceConflict() error {
	return errUsageFromIssues([]usageIssue{
		invalidIssue("namespace", i18n.T("err_pod_namespace_conflict")),
	})
}

func podNSAndName(cmd *cobra.Command, args []string) (string, string, error) {
	if len(args) >= 2 {
		if cmd.Flags().Changed("namespace") {
			return "", "", podNamespaceConflict()
		}
		namespace, err := requiredArg(args, "pod_label_namespace", "namespace")
		if err != nil {
			return "", "", err
		}
		name, err := requiredArg(args[1:], "pod_label_name", "pod")
		if err != nil {
			return "", "", err
		}
		return namespace, name, nil
	}

	name, err := requiredArg(args, "pod_label_name", "pod")
	if err != nil {
		return "", "", err
	}
	namespace, err := podNamespaceFlag(cmd)
	if err != nil {
		return "", "", err
	}
	return namespace, name, nil
}

func podNSNameAndContainer(cmd *cobra.Command, args []string) (string, string, string, error) {
	if len(args) >= 3 {
		if cmd.Flags().Changed("namespace") {
			return "", "", "", podNamespaceConflict()
		}
		namespace, err := requiredArg(args, "pod_label_namespace", "namespace")
		if err != nil {
			return "", "", "", err
		}
		name, err := requiredArg(args[1:], "pod_label_name", "pod")
		if err != nil {
			return "", "", "", err
		}
		container, err := requiredArg(args[2:], "container_label_name", "container")
		if err != nil {
			return "", "", "", err
		}
		return namespace, name, container, nil
	}

	name, err := requiredArg(args, "pod_label_name", "pod")
	if err != nil {
		return "", "", "", err
	}
	container, err := requiredArg(args[1:], "container_label_name", "container")
	if err != nil {
		return "", "", "", err
	}
	namespace, err := podNamespaceFlag(cmd)
	if err != nil {
		return "", "", "", err
	}
	return namespace, name, container, nil
}

func runPodContainers(cmd *cobra.Command, args []string) error {
	ns, name, err := podNSAndName(cmd, args)
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "containers", Path: fmt.Sprintf("%s/%s/pods/%s/containers", api.NamespacesPrefix, ns, name), Params: noParams, Table: printRawObject})
}

func runPodEvents(cmd *cobra.Command, args []string) error {
	ns, name, err := podNSAndName(cmd, args)
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "events", Path: fmt.Sprintf("%s/%s/pods/%s/events", api.NamespacesPrefix, ns, name), Params: noParams, Table: printRawObject})
}

func runPodLogs(cmd *cobra.Command, args []string) error {
	ns, name, container, err := podNSNameAndContainer(cmd, args)
	if err != nil {
		return err
	}
	tail, _ := cmd.Flags().GetInt("tail")
	if tail < 0 {
		return errUsageFromIssues([]usageIssue{
			invalidIssue("tail", i18n.T("err_invalid_non_negative_int", "tail")),
		})
	}
	timestamps, _ := cmd.Flags().GetBool("timestamps")
	previous, _ := cmd.Flags().GetBool("previous")
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	logs, err := client.GetPodLogs(ns, name, container, api.PodLogOptions{
		TailLines:  int64(tail),
		Timestamps: timestamps,
		Previous:   previous,
	})
	if err != nil {
		return cliErrFromPodLog(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"logs": string(logs),
		}))
	}
	if err := writeJobLogs(os.Stdout, []jobLogResult{{Content: string(logs)}}, false); err != nil {
		return cliErrFromPodLog(&api.PodLogWriteError{Cause: err})
	}
	return nil
}

func runPodIngresses(cmd *cobra.Command, args []string) error {
	ns, name, err := podNSAndName(cmd, args)
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "ingresses", Path: fmt.Sprintf("%s/%s/pods/%s/ingresses", api.NamespacesPrefix, ns, name), Params: noParams, Table: printRawObject})
}

func runPodNodeports(cmd *cobra.Command, args []string) error {
	ns, name, err := podNSAndName(cmd, args)
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "nodeports", Path: fmt.Sprintf("%s/%s/pods/%s/nodeports", api.NamespacesPrefix, ns, name), Params: noParams, Table: printRawObject})
}

func init() {
	for _, cmd := range []*cobra.Command{podContainersCmd, podEventsCmd, podLogsCmd, podIngressesCmd, podNodeportsCmd} {
		cmd.Flags().String("namespace", "", i18n.T("flag_namespace"))
	}
	podLogsCmd.Flags().Bool("timestamps", false, "Include timestamps in logs")
	podLogsCmd.Flags().Bool("previous", false, "Return previous terminated container logs")
	podLogsCmd.Flags().Int("tail", 0, "Number of recent log lines to show")
	podCmd.AddCommand(podContainersCmd, podEventsCmd, podLogsCmd, podIngressesCmd, podNodeportsCmd)
	rootCmd.AddCommand(podCmd)
}
