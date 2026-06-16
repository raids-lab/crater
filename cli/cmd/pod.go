package cmd

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/api"
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

var podContainersCmd = &cobra.Command{Use: "containers <namespace> <pod>", Short: "List pod containers", Args: exactArgs(2, "namespace", "pod"), RunE: runPodContainers}
var podEventsCmd = &cobra.Command{Use: "events <namespace> <pod>", Short: "List pod events", Args: exactArgs(2, "namespace", "pod"), RunE: runPodEvents}
var podLogsCmd = &cobra.Command{Use: "logs <namespace> <pod> <container>", Short: "Show pod container logs", Args: exactArgs(3, "namespace", "pod", "container"), RunE: runPodLogs}
var podIngressesCmd = &cobra.Command{Use: "ingresses <namespace> <pod>", Short: "List pod ingresses", Args: exactArgs(2, "namespace", "pod"), RunE: runPodIngresses}
var podNodeportsCmd = &cobra.Command{Use: "nodeports <namespace> <pod>", Short: "List pod nodeports", Args: exactArgs(2, "namespace", "pod"), RunE: runPodNodeports}

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

func podNSAndName(args []string) (string, string, error) {
	ns, err := requiredArg(args, "pod_label_namespace", "namespace")
	if err != nil {
		return "", "", err
	}
	name, err := requiredArg(args[1:], "pod_label_name", "pod")
	if err != nil {
		return "", "", err
	}
	return ns, name, nil
}

func runPodContainers(cmd *cobra.Command, args []string) error {
	ns, name, err := podNSAndName(args)
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "containers", Path: fmt.Sprintf("%s/%s/pods/%s/containers", api.NamespacesPrefix, ns, name), Params: noParams, Table: printRawObject})
}

func runPodEvents(cmd *cobra.Command, args []string) error {
	ns, name, err := podNSAndName(args)
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "events", Path: fmt.Sprintf("%s/%s/pods/%s/events", api.NamespacesPrefix, ns, name), Params: noParams, Table: printRawObject})
}

func runPodLogs(cmd *cobra.Command, args []string) error {
	ns, name, err := podNSAndName(args)
	if err != nil {
		return err
	}
	container, err := requiredArg(args[2:], "container_label_name", "container")
	if err != nil {
		return err
	}
	params := map[string]string{
		"timestamps": getBoolParam(cmd, "timestamps"),
		"previous":   getBoolParam(cmd, "previous"),
	}
	tail := getIntParam(cmd, "tail")
	if tail != "0" {
		params["tailLines"] = tail
	}
	return runRawStringRead(cmd, fmt.Sprintf("%s/%s/pods/%s/containers/%s/log", api.NamespacesPrefix, ns, name, container), params, "logs")
}

func runPodIngresses(cmd *cobra.Command, args []string) error {
	ns, name, err := podNSAndName(args)
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "ingresses", Path: fmt.Sprintf("%s/%s/pods/%s/ingresses", api.NamespacesPrefix, ns, name), Params: noParams, Table: printRawObject})
}

func runPodNodeports(cmd *cobra.Command, args []string) error {
	ns, name, err := podNSAndName(args)
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "nodeports", Path: fmt.Sprintf("%s/%s/pods/%s/nodeports", api.NamespacesPrefix, ns, name), Params: noParams, Table: printRawObject})
}

func init() {
	podLogsCmd.Flags().Bool("timestamps", false, "Include timestamps in logs")
	podLogsCmd.Flags().Bool("previous", false, "Return previous terminated container logs")
	podLogsCmd.Flags().Int("tail", 0, "Number of recent log lines to show")
	podCmd.AddCommand(podContainersCmd, podEventsCmd, podLogsCmd, podIngressesCmd, podNodeportsCmd)
	rootCmd.AddCommand(podCmd)
}
