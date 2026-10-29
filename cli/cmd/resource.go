package cmd

import (
	"fmt"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

var resourceCmd = &cobra.Command{
	Use:   "resource",
	Short: "View resources",
	Long:  "View cluster resource definitions, billing prices, GPU networks, and vGPU links.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var resourceLsCmd = &cobra.Command{Use: "ls", Short: "List resources", Args: noArgs, RunE: runResourceLs}
var resourceNetworksCmd = &cobra.Command{Use: "networks <id>", Short: "List networks for a GPU resource", Args: exactArgs(1, "id"), RunE: runResourceNetworks}
var resourceVGPUCmd = &cobra.Command{Use: "vgpu <id>", Short: "List vGPU resources linked to a GPU resource", Args: exactArgs(1, "id"), RunE: runResourceVGPU}
var resourcePricesCmd = &cobra.Command{Use: "prices", Short: "List billing prices", Args: noArgs, RunE: runResourcePrices}
var adminResourceCmd = &cobra.Command{Use: "resource", Short: "View admin resource links"}
var adminResourceNetworksCmd = &cobra.Command{Use: "networks <id>", Short: "List networks for a GPU resource", Args: exactArgs(1, "id"), RunE: runAdminResourceNetworks}
var adminResourceVGPUCmd = &cobra.Command{Use: "vgpu <id>", Short: "List vGPU resources linked to a GPU resource", Args: exactArgs(1, "id"), RunE: runAdminResourceVGPU}

func runResourceLs(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{
		PayloadKey: "resources",
		Path:       api.ResourcesPrefix,
		Params: func(cmd *cobra.Command) map[string]string {
			return map[string]string{"withVendorDomain": getBoolParam(cmd, "with-vendor-domain")}
		},
		Table: printResourceTable,
	})
}

func runResourceNetworks(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "resource_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "networks", Path: fmt.Sprintf("%s/%s/networks", api.ResourcesPrefix, api.UintPath(id)), Params: noParams, Table: printResourceTable})
}

func runAdminResourceNetworks(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "resource_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "networks", Path: fmt.Sprintf("%s/%s/networks", api.AdminResourcesPfx, api.UintPath(id)), Params: noParams, Table: printResourceTable})
}

func runResourceVGPU(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "resource_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "vgpu", Path: fmt.Sprintf("%s/%s/vgpu", api.ResourcesPrefix, api.UintPath(id)), Params: noParams, Table: printSimpleTableWrapper("id", "vgpuResourceId", "min", "max")})
}

func runAdminResourceVGPU(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "resource_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "vgpu", Path: fmt.Sprintf("%s/%s/vgpu", api.AdminResourcesPfx, api.UintPath(id)), Params: noParams, Table: printSimpleTableWrapper("id", "vgpuResourceId", "min", "max")})
}

func runResourcePrices(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "prices", Path: api.ResourcesPrefix + "/billing/prices", Params: noParams, Table: printSimpleTableWrapper("id", "name", "label", "unitPrice")})
}

func printSimpleTableWrapper(cols ...string) func(interface{}) {
	return func(data interface{}) { printSimpleTable(data, cols...) }
}

func printResourceTable(data interface{}) {
	fmt.Printf("%s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_id"), 8),
		i18n.PadRight(i18n.T("table_name"), 28),
		i18n.PadRight(i18n.T("table_type"), 12),
		i18n.PadRight("LABEL", 20),
		i18n.PadRight("AMOUNT", 10),
		i18n.PadRight("FORMAT", 12))
	for _, row := range rawList(data) {
		fmt.Printf("%s %s %s %s %s %s\n",
			i18n.PadRight(rawString(row, "ID"), 8),
			i18n.PadRight(rawString(row, "name"), 28),
			i18n.PadRight(rawString(row, "type"), 12),
			i18n.PadRight(rawString(row, "label"), 20),
			i18n.PadRight(rawString(row, "amount"), 10),
			i18n.PadRight(rawString(row, "format"), 12))
	}
}

func init() {
	resourceLsCmd.Flags().Bool("with-vendor-domain", false, "Include vendor domain in resource names")
	resourceCmd.AddCommand(resourceLsCmd, resourceNetworksCmd, resourceVGPUCmd, resourcePricesCmd)
	rootCmd.AddCommand(resourceCmd)
	adminResourceCmd.AddCommand(adminResourceNetworksCmd, adminResourceVGPUCmd)
	adminCmd.AddCommand(adminResourceCmd)
}
