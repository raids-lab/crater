package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "View cluster nodes",
	Long:  "View cluster node lists and node details from the active Crater platform.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var nodeLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List cluster nodes",
	RunE:  runNodeLs,
}

var nodeGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get a cluster node",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: runNodeGet,
}

var nodePodsCmd = &cobra.Command{
	Use:   "pods <name>",
	Short: "List pods on a cluster node",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: runNodePods,
}

var nodeGPUCmd = &cobra.Command{
	Use:   "gpu <name>",
	Short: "Get GPU information for a cluster node",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: runNodeGPU,
}

func runNodeLs(cmd *cobra.Command, _ []string) error {
	if err := validateNodeLsFlags(cmd); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	nodes, err := client.ListNodes()
	if err != nil {
		return cliErrFromAPI(err)
	}
	nodes = filterNodes(cmd, nodes)
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"nodes": nodes,
		}))
	}
	printNodeTable(nodes)
	return nil
}

func validateNodeLsFlags(cmd *cobra.Command) error {
	gpu, _ := cmd.Flags().GetString("gpu")
	gpuAvailable, _ := cmd.Flags().GetBool("gpu-available")
	if gpuAvailable && strings.TrimSpace(gpu) == "" {
		return errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_node_gpu_required_for_available"),
			Field:   "gpu",
		}})
	}
	return nil
}

func filterNodes(cmd *cobra.Command, nodes []api.NodeBrief) []api.NodeBrief {
	name, _ := cmd.Flags().GetString("name")
	status, _ := cmd.Flags().GetString("status")
	arch, _ := cmd.Flags().GetString("arch")
	gpu, _ := cmd.Flags().GetString("gpu")
	gpuAvailable, _ := cmd.Flags().GetBool("gpu-available")
	name = strings.ToLower(strings.TrimSpace(name))
	status = strings.TrimSpace(status)
	arch = strings.TrimSpace(arch)
	gpu = strings.ToLower(strings.TrimSpace(gpu))

	out := nodes[:0]
	for _, node := range nodes {
		if name != "" && !strings.Contains(strings.ToLower(node.Name), name) {
			continue
		}
		if status != "" && node.Status != status {
			continue
		}
		if arch != "" && node.Arch != arch {
			continue
		}
		if gpu != "" && !nodeHasGPU(node, gpu, gpuAvailable) {
			continue
		}
		out = append(out, node)
	}
	return out
}

func nodeHasGPU(node api.NodeBrief, gpu string, availableOnly bool) bool {
	for key, allocatable := range node.Allocatable {
		keyLower := strings.ToLower(key)
		if !strings.Contains(keyLower, "gpu") && !strings.Contains(keyLower, "nvidia.com/") {
			continue
		}
		if gpu != "" && !strings.Contains(keyLower, gpu) {
			continue
		}
		if !availableOnly {
			return true
		}
		if resourceNumber(allocatable) > resourceNumber(node.Used[key]) {
			return true
		}
	}
	return false
}

func resourceNumber(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if strings.HasSuffix(raw, "m") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(raw, "m"), 64)
		return v / 1000
	}
	v, _ := strconv.ParseFloat(raw, 64)
	return v
}

func runNodeGet(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "node_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	node, err := client.GetNode(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"node": node,
		}))
	}
	printNodeDetail(node)
	return nil
}

func runNodePods(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "node_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	pods, err := client.GetNodePods(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"pods": pods,
		}))
	}
	printNodePodTable(pods)
	return nil
}

func runNodeGPU(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "node_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	gpu, err := client.GetNodeGPU(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"gpu": gpu,
		}))
	}
	printNodeGPU(gpu)
	return nil
}

func printNodeTable(nodes []api.NodeBrief) {
	fmt.Printf("%s %s %s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_name"), 32),
		i18n.PadRight(i18n.T("table_status"), 16),
		i18n.PadRight(i18n.T("table_role"), 16),
		i18n.PadRight(i18n.T("table_arch"), 12),
		i18n.PadRight(i18n.T("table_address"), 16),
		i18n.PadRight("CPU", 14),
		i18n.PadRight("MEMORY", 18),
		i18n.PadRight(i18n.T("table_workloads"), 10))
	for _, n := range nodes {
		fmt.Printf("%s %s %s %s %s %s %s %s\n",
			i18n.PadRight(n.Name, 32),
			i18n.PadRight(n.Status, 16),
			i18n.PadRight(n.Role, 16),
			i18n.PadRight(n.Arch, 12),
			i18n.PadRight(emptyDash(n.Address), 16),
			i18n.PadRight(resourcePair(n.Used, n.Allocatable, "cpu"), 14),
			i18n.PadRight(resourcePair(n.Used, n.Allocatable, "memory"), 18),
			i18n.PadRight(fmt.Sprintf("%d", n.Workloads), 10))
	}
}

func printNodeDetail(n *api.NodeDetail) {
	if n == nil {
		return
	}
	fmt.Printf("%s: %s\n", i18n.T("table_name"), n.Name)
	fmt.Printf("%s: %s\n", i18n.T("table_status"), n.IsReady)
	fmt.Printf("%s: %s\n", i18n.T("table_role"), n.Role)
	fmt.Printf("%s: %s\n", i18n.T("table_arch"), n.Arch)
	fmt.Printf("%s: %s\n", i18n.T("table_address"), emptyDash(n.Address))
	fmt.Printf("%s: %s\n", "OS", strings.TrimSpace(n.OS+" "+n.OSVersion))
	fmt.Printf("%s: %s\n", "Kubelet", n.KubeletVersion)
	fmt.Printf("%s: %s\n", "Runtime", n.ContainerRuntimeVersion)
	fmt.Printf("%s: %s\n", "CPU", resourcePair(n.Used, n.Allocatable, "cpu"))
	fmt.Printf("%s: %s\n", "Memory", resourcePair(n.Used, n.Allocatable, "memory"))
}

func printNodePodTable(pods []api.PodInfo) {
	fmt.Printf("%s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_name"), 36),
		i18n.PadRight(i18n.T("table_namespace"), 22),
		i18n.PadRight("IP", 16),
		i18n.PadRight(i18n.T("table_status"), 14),
		i18n.PadRight(i18n.T("table_type"), 12),
		i18n.PadRight(i18n.T("table_resources"), 24))
	for _, pod := range pods {
		fmt.Printf("%s %s %s %s %s %s\n",
			i18n.PadRight(pod.Name, 36),
			i18n.PadRight(pod.Namespace, 22),
			i18n.PadRight(emptyDash(pod.IP), 16),
			i18n.PadRight(pod.Status, 14),
			i18n.PadRight(emptyDash(pod.Type), 12),
			i18n.PadRight(formatResources(pod.Resources), 24))
	}
}

func printNodeGPU(gpu *api.NodeGPUInfo) {
	if gpu == nil {
		return
	}
	fmt.Printf("%s: %s\n", i18n.T("table_name"), gpu.Name)
	fmt.Printf("%s: %t\n", "HaveGPU", gpu.HaveGPU)
	fmt.Printf("%s: %d\n", "GPUCount", gpu.GPUCount)
	for _, device := range gpu.GPUDevices {
		fmt.Printf("- %s %s count=%d memory=%s driver=%s runtime=%s\n",
			device.ResourceName,
			device.Product,
			device.Count,
			device.Memory,
			device.Driver,
			device.RuntimeVersion)
	}
}

func requiredArg(args []string, labelKey string, field string) (string, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return "", errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_required", i18n.T(labelKey), field),
			Field:   field,
		}})
	}
	return strings.TrimSpace(args[0]), nil
}

func emptyDash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

func resourcePair(used api.ResourceList, allocatable api.ResourceList, key string) string {
	u := "0"
	a := "0"
	if used != nil && used[key] != "" {
		u = used[key]
	}
	if allocatable != nil && allocatable[key] != "" {
		a = allocatable[key]
	}
	return u + "/" + a
}

func init() {
	nodeLsCmd.Flags().String("name", "", "Filter nodes by name substring")
	nodeLsCmd.Flags().String("status", "", "Filter nodes by status")
	nodeLsCmd.Flags().String("arch", "", "Filter nodes by CPU architecture")
	nodeLsCmd.Flags().String("gpu", "", "Filter nodes by GPU resource name or model substring")
	nodeLsCmd.Flags().Bool("gpu-available", false, "Only show nodes with available matching GPU")
	completion.RegisterFlagValue([]string{"node", "ls"}, "status", staticValueCompleter([]string{"Ready", "NotReady", "Unschedulable", "Occupied"}, nil))
	completion.RegisterFlagValue([]string{"node", "ls"}, "arch", staticValueCompleter([]string{"amd64", "arm64"}, nil))
	nodeCmd.AddCommand(nodeLsCmd)
	nodeCmd.AddCommand(nodeGetCmd)
	nodeCmd.AddCommand(nodePodsCmd)
	nodeCmd.AddCommand(nodeGPUCmd)
	rootCmd.AddCommand(nodeCmd)
}
