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

var imageTaskTypes = []string{"jupyter", "webide", "custom", "pytorch", "tensorflow", "all"}
var imageVisibilityTypes = []string{"Public", "Private", "UserShare", "AccountShare"}

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "View images",
	Long:  "View container image lists from the active Crater platform.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var imageLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List images",
	Args:  noArgs,
	RunE:  runImageLs,
}

func runImageLs(cmd *cobra.Command, _ []string) error {
	available, _ := cmd.Flags().GetBool("available")
	taskType, _ := cmd.Flags().GetString("type")
	visibility, _ := cmd.Flags().GetString("visibility")
	taskType = strings.TrimSpace(taskType)
	visibility = strings.TrimSpace(visibility)
	if taskType != "" && !slices.Contains(imageTaskTypes, taskType) {
		return errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_image_type", taskType),
			Field:   "type",
		}})
	}
	if visibility != "" && !slices.Contains(imageVisibilityTypes, visibility) {
		return errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_image_visibility", visibility),
			Field:   "visibility",
		}})
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	images, err := client.ListImages(available)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if taskType != "" {
		images = filterImagesByTaskType(images, taskType)
	}
	images = filterImages(cmd, images)
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"images": images,
		}))
	}
	printImageTable(images)
	return nil
}

func filterImagesByTaskType(images []api.ImageInfo, taskType string) []api.ImageInfo {
	out := images[:0]
	for _, image := range images {
		if image.TaskType == taskType {
			out = append(out, image)
		}
	}
	return out
}

func filterImages(cmd *cobra.Command, images []api.ImageInfo) []api.ImageInfo {
	arch, _ := cmd.Flags().GetString("arch")
	visibility, _ := cmd.Flags().GetString("visibility")
	owner, _ := cmd.Flags().GetString("owner")
	search, _ := cmd.Flags().GetString("search")
	arch = strings.TrimSpace(arch)
	visibility = strings.TrimSpace(visibility)
	owner = strings.ToLower(strings.TrimSpace(owner))
	search = strings.ToLower(strings.TrimSpace(search))

	out := images[:0]
	for _, image := range images {
		if arch != "" && !slices.Contains(image.Archs, arch) {
			continue
		}
		if visibility != "" && image.ImageShareStatus != visibility {
			continue
		}
		if owner != "" && !strings.Contains(strings.ToLower(image.UserInfo.Username), owner) &&
			!strings.Contains(strings.ToLower(image.UserInfo.Nickname), owner) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(image.ImageLink), search) &&
			(image.Description == nil || !strings.Contains(strings.ToLower(*image.Description), search)) {
			continue
		}
		out = append(out, image)
	}
	return out
}

func printImageTable(images []api.ImageInfo) {
	fmt.Printf("%s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_id"), 8),
		i18n.PadRight(i18n.T("table_image"), 48),
		i18n.PadRight(i18n.T("table_type"), 12),
		i18n.PadRight(i18n.T("table_visibility"), 14),
		i18n.PadRight(i18n.T("table_arch"), 24),
		i18n.PadRight(i18n.T("table_owner"), 18))
	for _, image := range images {
		fmt.Printf("%s %s %s %s %s %s\n",
			i18n.PadRight(fmt.Sprintf("%d", image.ID), 8),
			i18n.PadRight(image.ImageLink, 48),
			i18n.PadRight(image.TaskType, 12),
			i18n.PadRight(image.ImageShareStatus, 14),
			i18n.PadRight(strings.Join(image.Archs, ","), 24),
			i18n.PadRight(image.UserInfo.Nickname, 18))
	}
}

func init() {
	imageLsCmd.Flags().Bool("available", false, "List images available for creating jobs")
	imageLsCmd.Flags().String("type", "", "Filter by job type")
	imageLsCmd.Flags().String("arch", "", "Filter by image architecture")
	imageLsCmd.Flags().String("visibility", "", "Filter by image visibility")
	imageLsCmd.Flags().String("owner", "", "Filter by owner username or nickname")
	imageLsCmd.Flags().String("search", "", "Filter by image link or description substring")
	completion.RegisterFlagValue([]string{"image", "ls"}, "type", staticValueCompleter(imageTaskTypes, nil))
	completion.RegisterFlagValue([]string{"image", "ls"}, "arch", staticValueCompleter([]string{"linux/amd64", "linux/arm64"}, nil))
	completion.RegisterFlagValue([]string{"image", "ls"}, "visibility", staticValueCompleter(imageVisibilityTypes, nil))
	imageCmd.AddCommand(imageLsCmd)
	rootCmd.AddCommand(imageCmd)
}
