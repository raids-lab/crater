package cmd

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	imageTaskWriteTypes  = []string{"jupyter", "webide", "custom", "pytorch", "tensorflow"}
	imageTaskFilterTypes = []string{"jupyter", "webide", "custom", "pytorch", "tensorflow", "all"}
	imageVisibilityTypes = []string{"Public", "Private", "UserShare", "AccountShare"}
	imageShareTypes      = []string{"user", "account"}
	imageBuildSources    = []string{"EnvdAdvanced", "EnvdRaw"}
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Manage images and image builds",
	Long:  "Build, upload, delete, share, and update Crater images and image build records.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var imageBuildCmd = &cobra.Command{Use: "build", Short: "Manage image builds"}
var imageBuildLsCmd = &cobra.Command{Use: "ls", Short: "List image build records", Args: noArgs, RunE: runImageBuildLs}
var imageBuildPipAptCmd = &cobra.Command{Use: "pip-apt", Short: "Build an image from base image plus pip/apt packages", Args: noArgs, RunE: runImageBuildPipApt}
var imageBuildDockerfileCmd = &cobra.Command{Use: "dockerfile", Short: "Build an image from Dockerfile", Args: noArgs, RunE: runImageBuildDockerfile}
var imageBuildEnvdCmd = &cobra.Command{Use: "envd", Short: "Build an image from envd", Args: noArgs, RunE: runImageBuildEnvd}
var imageBuildRemoveCmd = &cobra.Command{Use: "remove", Short: "Cancel or remove image build records", Args: noArgs, RunE: runImageBuildRemove}
var imageBuildGetCmd = &cobra.Command{Use: "get <name>", Short: "Get an image build record", Args: exactArgs(1, "name"), RunE: runImageBuildGet}
var imageBuildTemplateCmd = &cobra.Command{Use: "template <name>", Short: "Get image build template", Args: exactArgs(1, "name"), RunE: runImageBuildTemplate}
var imageBuildPodCmd = &cobra.Command{Use: "pod <id>", Short: "Get image build pod", Args: exactArgs(1, "id"), RunE: runImageBuildPod}

var imageLsCmd = &cobra.Command{Use: "ls", Short: "List images", Args: noArgs, RunE: runImageLs}
var imageUploadCmd = &cobra.Command{Use: "upload", Short: "Upload/register an existing image link", Args: noArgs, RunE: runImageUpload}
var imageDeleteCmd = &cobra.Command{Use: "delete <id>", Short: "Delete an image", Args: exactArgs(1, "id"), RunE: runImageDelete}
var imageDeleteManyCmd = &cobra.Command{Use: "delete-many", Short: "Delete multiple images", Args: noArgs, RunE: runImageDeleteMany}
var imageDescriptionCmd = &cobra.Command{Use: "description <id>", Short: "Update image description", Args: exactArgs(1, "id"), RunE: runImageDescription}
var imageTypeCmd = &cobra.Command{Use: "type <id>", Short: "Update image task type", Args: exactArgs(1, "id"), RunE: runImageType}
var imageTagsCmd = &cobra.Command{Use: "tags <id>", Short: "Update image tags", Args: exactArgs(1, "id"), RunE: runImageTags}
var imageArchCmd = &cobra.Command{Use: "arch <id>", Short: "Update image architectures", Args: exactArgs(1, "id"), RunE: runImageArch}
var imageValidCmd = &cobra.Command{Use: "valid", Short: "Validate image links", Args: noArgs, RunE: runImageValid}

var imageShareCmd = &cobra.Command{Use: "share", Short: "Manage image sharing"}
var imageShareLsCmd = &cobra.Command{Use: "ls <image-id>", Short: "List image grants", Args: exactArgs(1, "image-id"), RunE: runImageShareLs}
var imageShareUsersCmd = &cobra.Command{Use: "users <image-id>", Short: "List users not granted an image", Args: exactArgs(1, "image-id"), RunE: runImageShareUsers}
var imageShareAccountsCmd = &cobra.Command{Use: "accounts <image-id>", Short: "List accounts not granted an image", Args: exactArgs(1, "image-id"), RunE: runImageShareAccounts}
var imageShareAddCmd = &cobra.Command{Use: "add <image-id>", Short: "Share an image with users or accounts", Args: exactArgs(1, "image-id"), RunE: runImageShareAdd}
var imageShareRemoveCmd = &cobra.Command{Use: "remove <image-id>", Short: "Cancel image sharing", Args: exactArgs(1, "image-id"), RunE: runImageShareRemove}

var imageCudaCmd = &cobra.Command{Use: "cuda", Short: "Manage CUDA base images"}
var imageCudaLsCmd = &cobra.Command{Use: "ls", Short: "List CUDA base images", Args: noArgs, RunE: runImageCudaLs}
var imageCudaAddCmd = &cobra.Command{Use: "add", Short: "Add a CUDA base image", Args: noArgs, RunE: runImageCudaAdd}
var imageCudaDeleteCmd = &cobra.Command{Use: "delete <id>", Short: "Delete a CUDA base image", Args: exactArgs(1, "id"), RunE: runImageCudaDelete}

var imageHarborCmd = &cobra.Command{Use: "harbor", Short: "View Harbor information"}
var imageHarborInfoCmd = &cobra.Command{Use: "info", Short: "Get Harbor address", Args: noArgs, RunE: runImageHarborInfo}
var imageHarborCredentialCmd = &cobra.Command{Use: "credential", Short: "Create and show Harbor project credentials", Args: noArgs, RunE: runImageHarborCredential}

var imageQuotaCmd = &cobra.Command{Use: "quota", Short: "View or update Harbor project quota"}
var imageQuotaGetCmd = &cobra.Command{Use: "get", Short: "Get Harbor project quota", Args: noArgs, RunE: runImageQuotaGet}
var imageQuotaSetCmd = &cobra.Command{Use: "set", Short: "Update Harbor project quota", Args: noArgs, RunE: runImageQuotaSet}

var adminImageCmd = &cobra.Command{Use: "image", Short: "Manage admin image resources"}
var adminImageBuildLsCmd = &cobra.Command{Use: "build-ls", Short: "List all image build records", Args: noArgs, RunE: runAdminImageBuildLs}
var adminImageBuildRemoveCmd = &cobra.Command{Use: "build-remove", Short: "Cancel or remove image build records", Args: noArgs, RunE: runAdminImageBuildRemove}
var adminImageLsCmd = &cobra.Command{Use: "ls", Short: "List all images", Args: noArgs, RunE: runAdminImageLs}
var adminImageDeleteManyCmd = &cobra.Command{Use: "delete-many", Short: "Delete multiple images", Args: noArgs, RunE: runAdminImageDeleteMany}
var adminImageDescriptionCmd = &cobra.Command{Use: "description <id>", Short: "Update image description", Args: exactArgs(1, "id"), RunE: runAdminImageDescription}
var adminImageTypeCmd = &cobra.Command{Use: "type <id>", Short: "Update image task type", Args: exactArgs(1, "id"), RunE: runAdminImageType}
var adminImageTagsCmd = &cobra.Command{Use: "tags <id>", Short: "Update image tags", Args: exactArgs(1, "id"), RunE: runAdminImageTags}
var adminImageArchCmd = &cobra.Command{Use: "arch <id>", Short: "Update image architectures", Args: exactArgs(1, "id"), RunE: runAdminImageArch}
var adminImagePublicCmd = &cobra.Command{Use: "public <id>", Short: "Toggle public visibility", Args: exactArgs(1, "id"), RunE: runAdminImagePublic}

func runImageBuildLs(cmd *cobra.Command, _ []string) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.ListKaniko(false)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"builds": resp.KanikoList}))
	}
	printKanikoTable(resp.KanikoList)
	return nil
}

func runAdminImageBuildLs(cmd *cobra.Command, _ []string) error {
	return runImageBuildList(cmd, true)
}

func runImageBuildList(_ *cobra.Command, admin bool) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.ListKaniko(admin)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"builds": resp.KanikoList}))
	}
	printKanikoTable(resp.KanikoList)
	return nil
}

func runImageBuildPipApt(cmd *cobra.Command, _ []string) error {
	req, err := collectPipAptBuild(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.CreatePipApt(req)
	return writeImageMessage(msg, err)
}

func runImageBuildDockerfile(cmd *cobra.Command, _ []string) error {
	req, err := collectDockerfileBuild(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.CreateDockerfile(req)
	return writeImageMessage(msg, err)
}

func runImageBuildEnvd(cmd *cobra.Command, _ []string) error {
	req, err := collectEnvdBuild(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.CreateEnvd(req)
	return writeImageMessage(msg, err)
}

func runImageBuildRemove(cmd *cobra.Command, _ []string) error {
	ids, err := idsFlag(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.RemoveKaniko(ids, false)
	return writeImageMessage(msg, err)
}

func runAdminImageBuildRemove(cmd *cobra.Command, _ []string) error {
	ids, err := idsFlag(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.RemoveKaniko(ids, true)
	return writeImageMessage(msg, err)
}

func runImageBuildGet(_ *cobra.Command, args []string) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.GetKanikoByName(args[0])
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"build": resp}))
	}
	printRawObject(map[string]interface{}{
		"ID":            resp.ID,
		"imageLink":     resp.ImageLink,
		"status":        resp.Status,
		"imagepackName": resp.ImagePackName,
		"podName":       resp.PodName,
		"namespace":     resp.PodNameSpace,
		"nodeName":      resp.NodeName,
	})
	return nil
}

func runImageBuildTemplate(_ *cobra.Command, args []string) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	template, err := client.GetKanikoTemplateByName(args[0])
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"template": template}))
	}
	fmt.Println(template)
	return nil
}

func runImageBuildPod(_ *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.GetKanikoPod(id)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"pod": resp}))
	}
	printRawObject(map[string]interface{}{"name": resp.Name, "namespace": resp.Namespace, "nodeName": resp.NodeName})
	return nil
}

func runImageLs(cmd *cobra.Command, _ []string) error {
	taskType, _ := cmd.Flags().GetString("type")
	visibility, _ := cmd.Flags().GetString("visibility")
	taskType = strings.TrimSpace(taskType)
	visibility = strings.TrimSpace(visibility)
	if taskType != "" && !slices.Contains(imageTaskFilterTypes, taskType) {
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
	available, _ := cmd.Flags().GetBool("available")
	var images []api.ImageInfo
	if available {
		images, err = client.ListAvailableImages()
	} else {
		var resp *api.ListImageResponse
		resp, err = client.ListImageRecords(false)
		if resp != nil {
			images = resp.ImageList
		}
	}
	if err != nil {
		return cliErrFromAPI(err)
	}
	if taskType != "" && taskType != "all" {
		images = filterImagesByTaskType(images, taskType)
	}
	images = filterImages(cmd, images)
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"images": images}))
	}
	printImageTable(images)
	return nil
}

func runAdminImageLs(cmd *cobra.Command, _ []string) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.ListImageRecords(true)
	if err != nil {
		return cliErrFromAPI(err)
	}
	images := filterImages(cmd, resp.ImageList)
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"images": images}))
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

func runImageUpload(cmd *cobra.Command, _ []string) error {
	req, err := collectUpload(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.UploadImage(req)
	return writeImageMessage(msg, err)
}

func runImageDelete(_ *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.DeleteImage(id)
	return writeImageMessage(msg, err)
}

func runImageDeleteMany(cmd *cobra.Command, _ []string) error {
	ids, err := idsFlag(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.DeleteImages(ids, false)
	return writeImageMessage(msg, err)
}

func runAdminImageDeleteMany(cmd *cobra.Command, _ []string) error {
	ids, err := idsFlag(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.DeleteImages(ids, true)
	return writeImageMessage(msg, err)
}

func runImageDescription(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	description, _ := cmd.Flags().GetString("description")
	if strings.TrimSpace(description) == "" {
		return errUsageFromIssues([]usageIssue{missingIssue("description", "image_flag_description")})
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.UpdateImageDescription(api.ImageDescriptionRequest{ID: id, Description: description}, false)
	return writeImageMessage(msg, err)
}

func runAdminImageDescription(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	description, _ := cmd.Flags().GetString("description")
	if strings.TrimSpace(description) == "" {
		return errUsageFromIssues([]usageIssue{missingIssue("description", "image_flag_description")})
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.UpdateImageDescription(api.ImageDescriptionRequest{ID: id, Description: description}, true)
	return writeImageMessage(msg, err)
}

func runImageType(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	taskType, _ := cmd.Flags().GetString("type")
	if err := validateEnum("type", taskType, imageTaskWriteTypes); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.UpdateImageType(api.ImageTypeRequest{ID: id, TaskType: taskType}, false)
	return writeImageMessage(msg, err)
}

func runAdminImageType(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	taskType, _ := cmd.Flags().GetString("type")
	if err := validateEnum("type", taskType, imageTaskWriteTypes); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.UpdateImageType(api.ImageTypeRequest{ID: id, TaskType: taskType}, true)
	return writeImageMessage(msg, err)
}

func runImageTags(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	tags := csvFlag(cmd, "tags")
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.UpdateImageTags(api.ImageTagsRequest{ID: id, Tags: tags}, false)
	return writeImageMessage(msg, err)
}

func runAdminImageTags(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	tags := csvFlag(cmd, "tags")
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.UpdateImageTags(api.ImageTagsRequest{ID: id, Tags: tags}, true)
	return writeImageMessage(msg, err)
}

func runImageArch(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	archs := csvFlag(cmd, "archs")
	if len(archs) == 0 {
		archs = []string{"linux/amd64"}
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.UpdateImageArch(api.ImageArchRequest{ID: id, Archs: archs}, false)
	return writeImageMessage(msg, err)
}

func runAdminImageArch(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	archs := csvFlag(cmd, "archs")
	if len(archs) == 0 {
		archs = []string{"linux/amd64"}
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.UpdateImageArch(api.ImageArchRequest{ID: id, Archs: archs}, true)
	return writeImageMessage(msg, err)
}

func runImageShareAdd(cmd *cobra.Command, args []string) error {
	imageID, err := requiredUintArg(args, "image_label_id", "image-id")
	if err != nil {
		return err
	}
	targets, err := idsFlag(cmd)
	if err != nil {
		return err
	}
	shareType, _ := cmd.Flags().GetString("share-type")
	if err := validateEnum("share-type", shareType, imageShareTypes); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.ShareImage(api.ImageShareRequest{ImageID: imageID, IDList: targets, Type: shareType})
	return writeImageMessage(msg, err)
}

func runImageShareRemove(cmd *cobra.Command, args []string) error {
	imageID, err := requiredUintArg(args, "image_label_id", "image-id")
	if err != nil {
		return err
	}
	targetID, _ := cmd.Flags().GetUint("target-id")
	if targetID == 0 {
		return errUsageFromIssues([]usageIssue{missingIssue("target-id", "image_flag_target-id")})
	}
	shareType, _ := cmd.Flags().GetString("share-type")
	if err := validateEnum("share-type", shareType, imageShareTypes); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.CancelShareImage(api.ImageCancelShareRequest{ImageID: imageID, ID: targetID, Type: shareType})
	return writeImageMessage(msg, err)
}

func runImageShareLs(_ *cobra.Command, args []string) error {
	imageID, err := requiredUintArg(args, "image_label_id", "image-id")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.GetImageGrants(imageID)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"grants": resp}))
	}
	printGrantTable(resp)
	return nil
}

func runImageShareUsers(cmd *cobra.Command, args []string) error {
	imageID, err := requiredUintArg(args, "image_label_id", "image-id")
	if err != nil {
		return err
	}
	name, _ := cmd.Flags().GetString("name")
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.ListUngrantedUsers(imageID, name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"users": resp.UserList}))
	}
	printUserGrantTable(resp.UserList)
	return nil
}

func runImageShareAccounts(_ *cobra.Command, args []string) error {
	imageID, err := requiredUintArg(args, "image_label_id", "image-id")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.ListUngrantedAccounts(imageID)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"accounts": resp.AccountList}))
	}
	printAccountGrantTable(resp.AccountList)
	return nil
}

func runImageValid(cmd *cobra.Command, _ []string) error {
	pairs, err := linkPairsFlag(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.CheckImageLinks(pairs)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"invalid_pairs": resp.InvalidPairs}))
	}
	printLinkPairTable(resp.InvalidPairs)
	return nil
}

func runImageCudaLs(_ *cobra.Command, _ []string) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.ListCudaBaseImages()
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"cuda_base_images": resp.CudaBaseImages}))
	}
	printCudaTable(resp.CudaBaseImages)
	return nil
}

func runImageCudaAdd(cmd *cobra.Command, _ []string) error {
	imageLabel, _ := cmd.Flags().GetString("image-label")
	label, _ := cmd.Flags().GetString("label")
	value, _ := cmd.Flags().GetString("value")
	var issues []usageIssue
	if strings.TrimSpace(imageLabel) == "" {
		issues = append(issues, missingIssue("image-label", "image_flag_image-label"))
	}
	if strings.TrimSpace(label) == "" {
		issues = append(issues, missingIssue("label", "image_flag_label"))
	}
	if strings.TrimSpace(value) == "" {
		issues = append(issues, missingIssue("value", "image_flag_value"))
	}
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.AddCudaBaseImage(api.CudaBaseImageRequest{ImageLabel: imageLabel, Label: label, Value: value})
	return writeImageMessage(msg, err)
}

func runImageCudaDelete(_ *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.DeleteCudaBaseImage(id)
	return writeImageMessage(msg, err)
}

func runImageHarborInfo(_ *cobra.Command, _ []string) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.GetHarbor()
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"harbor": resp}))
	}
	fmt.Printf("Harbor: %s\n", resp.IP)
	return nil
}

func runImageHarborCredential(cmd *cobra.Command, _ []string) error {
	if err := requireYes(cmd); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.GetCredential()
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"credential": resp}))
	}
	fmt.Printf("Username: %s\nPassword: %s\n", deref(resp.Name), deref(resp.Password))
	return nil
}

func runImageQuotaGet(_ *cobra.Command, _ []string) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	resp, err := client.GetQuota()
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"quota": resp}))
	}
	fmt.Printf("Project: %s\nUsed: %.2f GB\nQuota: %.2f GB\n", resp.Project, resp.Used, resp.Quota)
	return nil
}

func runImageQuotaSet(cmd *cobra.Command, _ []string) error {
	size, _ := cmd.Flags().GetInt64("size")
	if size <= 0 {
		return errUsageFromIssues([]usageIssue{missingIssue("size", "image_flag_size")})
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.UpdateQuota(size)
	return writeImageMessage(msg, err)
}

func runAdminImagePublic(_ *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "image_label_id", "id")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	msg, err := client.TogglePublic(id)
	return writeImageMessage(msg, err)
}

func collectPipAptBuild(cmd *cobra.Command) (api.PipAptBuildRequest, error) {
	name, tag, description, tags, archs, template, err := commonBuildFlags(cmd)
	if err != nil {
		return api.PipAptBuildRequest{}, err
	}
	image, _ := cmd.Flags().GetString("image")
	if strings.TrimSpace(image) == "" {
		return api.PipAptBuildRequest{}, errUsageFromIssues([]usageIssue{missingIssue("image", "image_flag_image")})
	}
	requirements, _ := cmd.Flags().GetString("requirements")
	packages, _ := cmd.Flags().GetString("packages")
	return api.PipAptBuildRequest{Image: image, Requirements: requirements, Packages: packages, Description: description, Name: name, Tag: tag, Tags: tags, Template: template, Archs: archs}, nil
}

func collectDockerfileBuild(cmd *cobra.Command) (api.DockerfileBuildRequest, error) {
	name, tag, description, tags, archs, template, err := commonBuildFlags(cmd)
	if err != nil {
		return api.DockerfileBuildRequest{}, err
	}
	dockerfile, err := contentFromFlags(cmd, "dockerfile", "file")
	if err != nil {
		return api.DockerfileBuildRequest{}, err
	}
	return api.DockerfileBuildRequest{Dockerfile: dockerfile, Description: description, Name: name, Tag: tag, Tags: tags, Template: template, Archs: archs}, nil
}

func collectEnvdBuild(cmd *cobra.Command) (api.EnvdBuildRequest, error) {
	name, tag, description, tags, archs, template, err := commonBuildFlags(cmd)
	if err != nil {
		return api.EnvdBuildRequest{}, err
	}
	envd, err := contentFromFlags(cmd, "envd", "file")
	if err != nil {
		return api.EnvdBuildRequest{}, err
	}
	python, _ := cmd.Flags().GetString("python")
	base, _ := cmd.Flags().GetString("base")
	buildSource, _ := cmd.Flags().GetString("build-source")
	if buildSource == "" {
		buildSource = "EnvdAdvanced"
	}
	if err := validateEnum("build-source", buildSource, imageBuildSources); err != nil {
		return api.EnvdBuildRequest{}, err
	}
	return api.EnvdBuildRequest{Envd: envd, Description: description, Name: name, Tag: tag, Python: python, Base: base, Tags: tags, Template: template, BuildSource: buildSource, Archs: archs}, nil
}

func commonBuildFlags(cmd *cobra.Command) (string, string, string, []string, []string, string, error) {
	name, _ := cmd.Flags().GetString("name")
	tag, _ := cmd.Flags().GetString("tag")
	description, _ := cmd.Flags().GetString("description")
	template, _ := cmd.Flags().GetString("template")
	tags := csvFlag(cmd, "tags")
	archs := csvFlag(cmd, "archs")
	if len(archs) == 0 {
		archs = []string{"linux/amd64"}
	}
	var issues []usageIssue
	if strings.TrimSpace(name) == "" {
		issues = append(issues, missingIssue("name", "image_flag_name"))
	}
	if strings.TrimSpace(tag) == "" {
		issues = append(issues, missingIssue("tag", "image_flag_tag"))
	}
	if len(issues) > 0 {
		return "", "", "", nil, nil, "", errUsageFromIssues(issues)
	}
	return name, tag, description, tags, archs, template, nil
}

func collectUpload(cmd *cobra.Command) (api.ImageUploadRequest, error) {
	image, _ := cmd.Flags().GetString("image")
	taskType, _ := cmd.Flags().GetString("type")
	description, _ := cmd.Flags().GetString("description")
	if taskType == "" {
		taskType = "custom"
	}
	var issues []usageIssue
	if strings.TrimSpace(image) == "" {
		issues = append(issues, missingIssue("image", "image_flag_image"))
	}
	if taskType != "" && !slices.Contains(imageTaskWriteTypes, taskType) {
		issues = append(issues, invalidIssue("type", fmt.Sprintf("invalid type: %s", taskType)))
	}
	if len(issues) > 0 {
		return api.ImageUploadRequest{}, errUsageFromIssues(issues)
	}
	return api.ImageUploadRequest{ImageLink: image, TaskType: taskType, Description: description, Tags: csvFlag(cmd, "tags"), Archs: csvFlag(cmd, "archs")}, nil
}

func contentFromFlags(cmd *cobra.Command, directFlag, fileFlag string) (string, error) {
	direct, _ := cmd.Flags().GetString(directFlag)
	file, _ := cmd.Flags().GetString(fileFlag)
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		direct = string(b)
	}
	if strings.TrimSpace(direct) == "" {
		return "", errUsageFromIssues([]usageIssue{missingIssue(directFlag, "image_flag_"+directFlag)})
	}
	return direct, nil
}

func idsFlag(cmd *cobra.Command) ([]uint, error) {
	raw, _ := cmd.Flags().GetString("ids")
	ids, err := parseIDs(raw)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errUsageFromIssues([]usageIssue{missingIssue("ids", "image_flag_ids")})
	}
	return ids, nil
}

func parseIDs(raw string) ([]uint, error) {
	parts := splitCSV(raw)
	out := make([]uint, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.ParseUint(part, 10, 0)
		if err != nil {
			return nil, errUsageFromIssues([]usageIssue{{Code: errorcodes.ErrInvalidFlagValue, Message: i18n.T("err_invalid_ids", raw), Field: "ids"}})
		}
		out = append(out, uint(v))
	}
	return out, nil
}

func csvFlag(cmd *cobra.Command, name string) []string {
	raw, _ := cmd.Flags().GetString(name)
	return splitCSV(raw)
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func validateEnum(field, value string, allowed []string) error {
	if strings.TrimSpace(value) == "" {
		return errUsageFromIssues([]usageIssue{missingIssue(field, "image_flag_"+field)})
	}
	if !slices.Contains(allowed, value) {
		return errUsageFromIssues([]usageIssue{invalidIssue(field, fmt.Sprintf("invalid %s: %s", field, value))})
	}
	return nil
}

func linkPairsFlag(cmd *cobra.Command) ([]api.ImageInfoLinkPair, error) {
	raw, _ := cmd.Flags().GetString("links")
	parts := splitCSV(raw)
	if len(parts) == 0 {
		return nil, errUsageFromIssues([]usageIssue{missingIssue("links", "image_flag_links")})
	}
	pairs := make([]api.ImageInfoLinkPair, 0, len(parts))
	for _, part := range parts {
		pairs = append(pairs, api.ImageInfoLinkPair{ImageLink: part})
	}
	return pairs, nil
}

func missingIssue(field, labelKey string) usageIssue {
	return usageIssue{Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", i18n.T(labelKey), field), Field: field}
}

func invalidIssue(field, message string) usageIssue {
	return usageIssue{Code: errorcodes.ErrInvalidFlagValue, Message: message, Field: field}
}

func requireYes(cmd *cobra.Command) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if viper.GetBool("no-interactive") && !yes {
		return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_confirm_needed")}
	}
	return nil
}

func writeImageMessage(msg string, err error) error {
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"message": msg}))
	}
	fmt.Println(i18n.T("image_success", emptyDash(msg)))
	return nil
}

func printKanikoTable(items []api.KanikoInfo) {
	fmt.Printf("%s %s %s %s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight(i18n.T("table_name"), 28), i18n.PadRight(i18n.T("table_status"), 12), i18n.PadRight(i18n.T("table_image"), 42), i18n.PadRight(i18n.T("table_owner"), 18), i18n.PadRight(i18n.T("table_created"), 22))
	for _, item := range items {
		fmt.Printf("%s %s %s %s %s %s\n", i18n.PadRight(strconv.FormatUint(uint64(item.ID), 10), 8), i18n.PadRight(item.ImagePackName, 28), i18n.PadRight(item.Status, 12), i18n.PadRight(item.ImageLink, 42), i18n.PadRight(item.UserInfo.Nickname, 18), i18n.PadRight(item.CreatedAt.Format("2006-01-02 15:04:05"), 22))
	}
}

func printImageTable(items []api.ImageInfo) {
	fmt.Printf("%s %s %s %s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight(i18n.T("table_image"), 48), i18n.PadRight(i18n.T("table_type"), 12), i18n.PadRight(i18n.T("table_arch"), 24), i18n.PadRight(i18n.T("table_owner"), 18), i18n.PadRight(i18n.T("table_created"), 22))
	for _, item := range items {
		fmt.Printf("%s %s %s %s %s %s\n", i18n.PadRight(strconv.FormatUint(uint64(item.ID), 10), 8), i18n.PadRight(item.ImageLink, 48), i18n.PadRight(item.TaskType, 12), i18n.PadRight(strings.Join(item.Archs, ","), 24), i18n.PadRight(item.UserInfo.Nickname, 18), i18n.PadRight(item.CreatedAt.Format("2006-01-02 15:04:05"), 22))
	}
}

func printCudaTable(items []api.CudaBaseImage) {
	fmt.Printf("%s %s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight("LABEL", 24), i18n.PadRight("IMAGE_LABEL", 24), i18n.PadRight("VALUE", 48))
	for _, item := range items {
		fmt.Printf("%s %s %s %s\n", i18n.PadRight(strconv.FormatUint(uint64(item.ID), 10), 8), i18n.PadRight(item.Label, 24), i18n.PadRight(item.ImageLabel, 24), i18n.PadRight(item.Value, 48))
	}
}

func printGrantTable(resp *api.ImageGrantResponse) {
	if resp == nil {
		return
	}
	printUserGrantTable(resp.UserList)
	printAccountGrantTable(resp.AccountList)
}

func printUserGrantTable(items []api.ImageGrantedUser) {
	fmt.Printf("%s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight("USERNAME", 24), i18n.PadRight("NICKNAME", 24))
	for _, item := range items {
		fmt.Printf("%s %s %s\n", i18n.PadRight(strconv.FormatUint(uint64(item.ID), 10), 8), i18n.PadRight(item.Name, 24), i18n.PadRight(item.Nickname, 24))
	}
}

func printAccountGrantTable(items []api.ImageGrantedAccount) {
	fmt.Printf("%s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight("NAME", 24))
	for _, item := range items {
		fmt.Printf("%s %s\n", i18n.PadRight(strconv.FormatUint(uint64(item.ID), 10), 8), i18n.PadRight(item.Name, 24))
	}
}

func printLinkPairTable(items []api.ImageInfoLinkPair) {
	fmt.Printf("%s\n", i18n.PadRight("IMAGE", 64))
	for _, item := range items {
		fmt.Printf("%s\n", i18n.PadRight(item.ImageLink, 64))
	}
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func init() {
	addCommonBuildFlags(imageBuildPipAptCmd)
	imageBuildPipAptCmd.Flags().String("image", "", "Base image")
	imageBuildPipAptCmd.Flags().String("packages", "", "APT packages")
	imageBuildPipAptCmd.Flags().String("requirements", "", "Python requirements")
	addCommonBuildFlags(imageBuildDockerfileCmd)
	imageBuildDockerfileCmd.Flags().String("dockerfile", "", "Dockerfile content")
	imageBuildDockerfileCmd.Flags().String("file", "", "Read Dockerfile from file")
	addCommonBuildFlags(imageBuildEnvdCmd)
	imageBuildEnvdCmd.Flags().String("envd", "", "envd content")
	imageBuildEnvdCmd.Flags().String("file", "", "Read envd from file")
	imageBuildEnvdCmd.Flags().String("python", "", "Python version")
	imageBuildEnvdCmd.Flags().String("base", "", "envd base image")
	imageBuildEnvdCmd.Flags().String("build-source", "EnvdAdvanced", "envd build source")
	imageBuildRemoveCmd.Flags().String("ids", "", "Comma-separated IDs")
	imageBuildCmd.AddCommand(imageBuildLsCmd, imageBuildPipAptCmd, imageBuildDockerfileCmd, imageBuildEnvdCmd, imageBuildRemoveCmd, imageBuildGetCmd, imageBuildTemplateCmd, imageBuildPodCmd)

	imageLsCmd.Flags().Bool("available", false, "List images available for creating jobs")
	imageLsCmd.Flags().String("type", "", "Filter by job type")
	imageLsCmd.Flags().String("arch", "", "Filter by image architecture")
	imageLsCmd.Flags().String("visibility", "", "Filter by image visibility")
	imageLsCmd.Flags().String("owner", "", "Filter by owner username or nickname")
	imageLsCmd.Flags().String("search", "", "Filter by image link or description substring")
	adminImageLsCmd.Flags().String("arch", "", "Filter by image architecture")
	adminImageLsCmd.Flags().String("visibility", "", "Filter by image visibility")
	adminImageLsCmd.Flags().String("owner", "", "Filter by owner username or nickname")
	adminImageLsCmd.Flags().String("search", "", "Filter by image link or description substring")
	imageUploadCmd.Flags().String("image", "", "Image link")
	imageUploadCmd.Flags().String("type", "custom", "Image task type")
	imageUploadCmd.Flags().String("description", "", "Description")
	imageUploadCmd.Flags().String("tags", "", "Comma-separated tags")
	imageUploadCmd.Flags().String("archs", "", "Comma-separated architectures")
	imageDeleteManyCmd.Flags().String("ids", "", "Comma-separated IDs")
	imageDescriptionCmd.Flags().String("description", "", "Description")
	imageTypeCmd.Flags().String("type", "", "Image task type")
	imageTagsCmd.Flags().String("tags", "", "Comma-separated tags")
	imageArchCmd.Flags().String("archs", "", "Comma-separated architectures")
	imageValidCmd.Flags().String("links", "", "Comma-separated image links")

	imageShareAddCmd.Flags().String("ids", "", "Comma-separated target IDs")
	imageShareAddCmd.Flags().String("share-type", "user", "Share type")
	imageShareRemoveCmd.Flags().Uint("target-id", 0, "Share target ID")
	imageShareRemoveCmd.Flags().String("share-type", "user", "Share type")
	imageShareUsersCmd.Flags().String("name", "", "Filter by username or nickname")
	imageShareCmd.AddCommand(imageShareLsCmd, imageShareUsersCmd, imageShareAccountsCmd, imageShareAddCmd, imageShareRemoveCmd)

	imageCudaAddCmd.Flags().String("image-label", "", "Image label")
	imageCudaAddCmd.Flags().String("label", "", "Display label")
	imageCudaAddCmd.Flags().String("value", "", "Image value")
	imageCudaCmd.AddCommand(imageCudaLsCmd, imageCudaAddCmd, imageCudaDeleteCmd)

	imageHarborCredentialCmd.Flags().BoolP("yes", "y", false, "Confirm sensitive operation")
	imageHarborCmd.AddCommand(imageHarborInfoCmd, imageHarborCredentialCmd)
	imageQuotaSetCmd.Flags().Int64("size", 0, "Quota size")
	imageQuotaCmd.AddCommand(imageQuotaGetCmd, imageQuotaSetCmd)

	adminImageBuildRemoveCmd.Flags().String("ids", "", "Comma-separated IDs")
	adminImageDeleteManyCmd.Flags().String("ids", "", "Comma-separated IDs")
	adminImageDescriptionCmd.Flags().String("description", "", "Description")
	adminImageTypeCmd.Flags().String("type", "", "Image task type")
	adminImageTagsCmd.Flags().String("tags", "", "Comma-separated tags")
	adminImageArchCmd.Flags().String("archs", "", "Comma-separated architectures")

	completion.RegisterFlagValue([]string{"image", "ls"}, "type", staticValueCompleter(imageTaskFilterTypes, nil))
	completion.RegisterFlagValue([]string{"image", "ls"}, "arch", staticValueCompleter([]string{"linux/amd64", "linux/arm64"}, nil))
	completion.RegisterFlagValue([]string{"image", "ls"}, "visibility", staticValueCompleter(imageVisibilityTypes, nil))
	completion.RegisterFlagValue([]string{"image", "upload"}, "type", staticValueCompleter(imageTaskWriteTypes, nil))
	completion.RegisterFlagValue([]string{"image", "type"}, "type", staticValueCompleter(imageTaskWriteTypes, nil))
	completion.RegisterFlagValue([]string{"admin", "image", "type"}, "type", staticValueCompleter(imageTaskWriteTypes, nil))
	completion.RegisterFlagValue([]string{"image", "share", "add"}, "share-type", staticValueCompleter(imageShareTypes, nil))
	completion.RegisterFlagValue([]string{"image", "share", "remove"}, "share-type", staticValueCompleter(imageShareTypes, nil))
	completion.RegisterFlagValue([]string{"image", "build", "envd"}, "build-source", staticValueCompleter(imageBuildSources, nil))

	imageCmd.AddCommand(imageBuildCmd, imageLsCmd, imageUploadCmd, imageDeleteCmd, imageDeleteManyCmd, imageDescriptionCmd, imageTypeCmd, imageTagsCmd, imageArchCmd, imageValidCmd, imageShareCmd, imageCudaCmd, imageHarborCmd, imageQuotaCmd)
	rootCmd.AddCommand(imageCmd)
	adminImageCmd.AddCommand(adminImageBuildLsCmd, adminImageBuildRemoveCmd, adminImageLsCmd, adminImageDeleteManyCmd, adminImageDescriptionCmd, adminImageTypeCmd, adminImageTagsCmd, adminImageArchCmd, adminImagePublicCmd)
	adminCmd.AddCommand(adminImageCmd)
}

func addCommonBuildFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "Image name")
	cmd.Flags().String("tag", "", "Image tag")
	cmd.Flags().String("description", "", "Description")
	cmd.Flags().String("tags", "", "Comma-separated tags")
	cmd.Flags().String("archs", "", "Comma-separated architectures")
	cmd.Flags().String("template", "", "Template text")
}
