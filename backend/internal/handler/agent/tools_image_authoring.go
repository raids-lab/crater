package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	internalutil "github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/packer"
	pkgutils "github.com/raids-lab/crater/pkg/utils"
)

type agentImageBuildMode string

const (
	agentImageBuildModePipApt  agentImageBuildMode = "pip_apt"
	agentImageBuildModeDocker  agentImageBuildMode = "dockerfile"
	agentImageBuildModeEnvd    agentImageBuildMode = "envd"
	agentImageBuildModeEnvdRaw agentImageBuildMode = "envd_raw"
	defaultImageBuildNamespace                     = "default"
)

type resolvedShareTarget struct {
	ID       uint
	Kind     string
	Name     string
	Nickname string
	Input    string
}

func normalizeImageBuildMode(value string) (agentImageBuildMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pip_apt", "pipapt", "pip-apt":
		return agentImageBuildModePipApt, nil
	case "dockerfile", "docker_file":
		return agentImageBuildModeDocker, nil
	case "envd", "envd_advanced", "advanced":
		return agentImageBuildModeEnvd, nil
	case "envd_raw", "envd-raw", "raw":
		return agentImageBuildModeEnvdRaw, nil
	default:
		return "", fmt.Errorf("mode must be one of pip_apt, dockerfile, envd, envd_raw")
	}
}

func normalizeImageBuildStatuses(values []string) []model.BuildStatus {
	result := make([]model.BuildStatus, 0, len(values))
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "initial":
			result = append(result, model.BuildJobInitial)
		case "pending":
			result = append(result, model.BuildJobPending)
		case "running":
			result = append(result, model.BuildJobRunning)
		case "finished", "succeeded", "success":
			result = append(result, model.BuildJobFinished)
		case "failed", "error":
			result = append(result, model.BuildJobFailed)
		case "canceled", "cancelled":
			result = append(result, model.BuildJobCanceled)
		}
	}
	return result
}

func normalizeImageTaskType(value string) (model.JobType, error) {
	taskTypes := normalizeJobTypes([]string{value})
	if len(taskTypes) == 0 {
		return "", fmt.Errorf("unsupported task_type %q", strings.TrimSpace(value))
	}
	return model.JobType(taskTypes[0]), nil
}

func readNamespace() string {
	if ns := strings.TrimSpace(pkgconfig.GetConfig().Namespaces.Job); ns != "" {
		return ns
	}
	return defaultImageBuildNamespace
}

func buildImageSummaryFromRecord(image *model.Image) map[string]any {
	if image == nil {
		return nil
	}
	description := ""
	if image.Description != nil {
		description = strings.TrimSpace(*image.Description)
	}
	imagePackName := ""
	if image.ImagePackName != nil {
		imagePackName = *image.ImagePackName
	}
	archs := image.Archs.Data()
	if len(archs) == 0 {
		archs = []string{"linux/amd64"}
	}
	summary := map[string]any{
		"id":            image.ID,
		"imageLink":     image.ImageLink,
		"description":   description,
		"taskType":      image.TaskType,
		"imageSource":   image.ImageSource.String(),
		"tags":          image.Tags.Data(),
		"archs":         archs,
		"imagePackName": imagePackName,
		"createdAt":     image.CreatedAt,
	}
	if image.User.ID != 0 {
		summary["owner"] = map[string]any{
			"userID":   image.User.ID,
			"username": image.User.Name,
			"nickname": image.User.Nickname,
		}
	}
	return summary
}

func buildImageBuildSummary(kaniko *model.Kaniko, finalImage *model.Image) map[string]any {
	if kaniko == nil {
		return nil
	}
	description := ""
	if kaniko.Description != nil {
		description = strings.TrimSpace(*kaniko.Description)
	}
	archs := kaniko.Archs.Data()
	if len(archs) == 0 {
		archs = []string{"linux/amd64"}
	}
	summary := map[string]any{
		"id":            kaniko.ID,
		"imagePackName": kaniko.ImagePackName,
		"imageLink":     kaniko.ImageLink,
		"status":        kaniko.Status,
		"buildSource":   kaniko.BuildSource,
		"description":   description,
		"tags":          kaniko.Tags.Data(),
		"archs":         archs,
		"size":          kaniko.Size,
		"createdAt":     kaniko.CreatedAt,
	}
	if finalImage != nil {
		summary["finalImage"] = buildImageSummaryFromRecord(finalImage)
	}
	return summary
}

func (mgr *AgentMgr) findFinalImageByPackName(c *gin.Context, imagePackName string) (*model.Image, error) {
	if strings.TrimSpace(imagePackName) == "" {
		return nil, nil
	}
	imageQuery := query.Image
	imageRecord, err := imageQuery.WithContext(c).Preload(imageQuery.User).
		Where(imageQuery.ImagePackName.Eq(strings.TrimSpace(imagePackName))).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return imageRecord, nil
}

func (mgr *AgentMgr) resolveOwnedImageBuild(c *gin.Context, token internalutil.JWTMessage, args map[string]any) (*model.Kaniko, error) {
	kanikoQuery := query.Kaniko
	specifiedQuery := kanikoQuery.WithContext(c).Where(kanikoQuery.UserID.Eq(token.UserID))

	if buildID := getToolArgInt(args, "build_id", 0); buildID > 0 {
		record, err := specifiedQuery.Where(kanikoQuery.ID.Eq(uint(buildID))).First()
		if err == nil {
			return record, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	imagePackName := getToolArgString(args, "imagepack_name", "")
	if imagePackName == "" {
		imagePackName = getToolArgString(args, "image_pack_name", "")
	}
	if imagePackName == "" {
		return nil, fmt.Errorf("build_id or imagepack_name is required")
	}

	record, err := specifiedQuery.Where(kanikoQuery.ImagePackName.Eq(imagePackName)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("image build %q not found", imagePackName)
		}
		return nil, err
	}
	return record, nil
}

func (mgr *AgentMgr) resolveOwnedImageRecord(c *gin.Context, token internalutil.JWTMessage, args map[string]any) (*model.Image, error) {
	imageQuery := query.Image
	specifiedQuery := imageQuery.WithContext(c).Preload(imageQuery.User).Where(imageQuery.UserID.Eq(token.UserID))

	if imageID := getToolArgInt(args, "image_id", 0); imageID > 0 {
		record, err := specifiedQuery.Where(imageQuery.ID.Eq(uint(imageID))).First()
		if err == nil {
			return record, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	imageLink := getToolArgString(args, "image_link", "")
	if imageLink == "" {
		return nil, fmt.Errorf("image_id or image_link is required")
	}
	record, err := specifiedQuery.Where(imageQuery.ImageLink.Eq(imageLink)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("image %q not found", imageLink)
		}
		return nil, err
	}
	return record, nil
}

func (mgr *AgentMgr) loadBuildPodInfo(c *gin.Context, imagePackName string) map[string]any {
	if mgr.kubeClient == nil || strings.TrimSpace(imagePackName) == "" {
		return map[string]any{
			"namespace": readNamespace(),
		}
	}
	podList, err := mgr.kubeClient.CoreV1().Pods(readNamespace()).List(c.Request.Context(), metav1.ListOptions{
		LabelSelector: "job-name=" + imagePackName,
	})
	if err != nil || podList == nil || len(podList.Items) == 0 {
		return map[string]any{
			"namespace": readNamespace(),
		}
	}
	sort.Slice(podList.Items, func(i, j int) bool {
		return podList.Items[i].CreationTimestamp.Time.After(podList.Items[j].CreationTimestamp.Time)
	})
	pod := podList.Items[0]
	return map[string]any{
		"name":      pod.Name,
		"namespace": pod.Namespace,
		"nodeName":  pod.Spec.NodeName,
		"phase":     pod.Status.Phase,
	}
}

func parseFlexibleTargets(args map[string]any) []string {
	targets := getToolArgStringSlice(args, "targets")
	if len(targets) == 0 {
		targets = getToolArgStringSlice(args, "target_ids")
	}
	if len(targets) == 0 {
		targets = getToolArgStringSlice(args, "target_names")
	}
	if len(targets) == 0 {
		targets = getToolArgStringSlice(args, "targets_text")
	}
	return targets
}

func parseCSVBackedSlice(args map[string]any, primaryKey, secondaryKey string, fallback []string) []string {
	values := getToolArgStringSlice(args, primaryKey)
	if len(values) == 0 && secondaryKey != "" {
		values = getToolArgStringSlice(args, secondaryKey)
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}

func parseVolumeMountsArg(args map[string]any) []internalutil.VolumeMount {
	raw, ok := args["volume_mounts"]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	result := make([]internalutil.VolumeMount, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		mount := internalutil.VolumeMount{
			Type:      internalutil.VolumeType(getToolArgInt(entry, "type", int(internalutil.FileType))),
			DatasetID: uint(getToolArgInt(entry, "dataset_id", getToolArgInt(entry, "datasetID", 0))),
			SubPath:   getToolArgString(entry, "sub_path", getToolArgString(entry, "subPath", "")),
			MountPath: getToolArgString(entry, "mount_path", getToolArgString(entry, "mountPath", "")),
		}
		if mount.SubPath == "" || mount.MountPath == "" {
			continue
		}
		result = append(result, mount)
	}
	return result
}

func extractBaseImageFromDockerfileContent(dockerfile string) (string, error) {
	lines := strings.Split(dockerfile, "\n")
	for idx := 0; idx < len(lines); idx++ {
		line := strings.TrimSpace(lines[idx])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(strings.ToUpper(line), "FROM ") {
			continue
		}
		for strings.HasSuffix(line, "\\") {
			line = strings.TrimSuffix(line, "\\")
			if idx+1 >= len(lines) {
				return "", fmt.Errorf("unexpected end of Dockerfile after FROM")
			}
			idx++
			line = strings.TrimSpace(line + " " + strings.TrimSpace(lines[idx]))
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid FROM instruction")
		}
		return parts[1], nil
	}
	return "", fmt.Errorf("Dockerfile must contain a FROM instruction")
}

func buildEnvdAdvancedScript(baseImage, pythonVersion string, aptPackages, pythonPackages []string, enableJupyter, enableZsh bool) string {
	defaultApt := []string{"openssh-server", "build-essential", "iputils-ping", "net-tools", "htop", "tree", "tzdata"}
	aptPackages = append(defaultApt, aptPackages...)

	builder := &strings.Builder{}
	builder.WriteString("# syntax=v1\n\n")
	builder.WriteString("def build():\n")
	builder.WriteString(fmt.Sprintf("    base(image=%q, dev=True)\n", baseImage))
	builder.WriteString(fmt.Sprintf("    install.python(version=%q)\n", pythonVersion))
	builder.WriteString("    install.apt_packages([")
	for idx, item := range aptPackages {
		if idx > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(strconv.Quote(item))
	}
	builder.WriteString("])\n")
	builder.WriteString("    config.repo(\n")
	builder.WriteString("        url=\"https://github.com/raids-lab/crater\",\n")
	builder.WriteString("        description=\"Crater\",\n")
	builder.WriteString("    )\n")
	builder.WriteString("    config.pip_index(url=\"https://pypi.tuna.tsinghua.edu.cn/simple\")")

	if len(pythonPackages) > 0 {
		builder.WriteString("\n    install.python_packages(name=[")
		for idx, item := range pythonPackages {
			if idx > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(strconv.Quote(item))
		}
		builder.WriteString("])")
	}

	if enableJupyter && enableZsh {
		builder.WriteString(`
    run(commands=[
      "chsh -s /bin/zsh root;",
      "git clone --depth 1 https://gitee.com/mirrors/oh-my-zsh.git;",
      "ZSH=\"/usr/share/.oh-my-zsh\" CHSH=\"no\" RUNZSH=\"no\" REMOTE=https://gitee.com/mirrors/oh-my-zsh.git sh ./oh-my-zsh/tools/install.sh;",
      "chmod a+rx /usr/share/.oh-my-zsh/oh-my-zsh.sh;",
      "rm -rf ./oh-my-zsh;",
      "git clone --depth=1 https://gitee.com/mirrors/zsh-syntax-highlighting.git /usr/share/.oh-my-zsh/custom/plugins/zsh-syntax-highlighting;",
      "git clone --depth=1 https://gitee.com/mirrors/zsh-autosuggestions.git /usr/share/.oh-my-zsh/custom/plugins/zsh-autosuggestions;",
      "echo \"export skip_global_compinit=1\" >> /etc/zsh/zshenv;",
      "echo \"export ZSH=\\\"/usr/share/.oh-my-zsh\\\"\" >> /etc/zsh/zshrc;",
      "echo \"plugins=(git extract sudo jsontools colored-man-pages zsh-autosuggestions zsh-syntax-highlighting)\" >> /etc/zsh/zshrc;",
      "echo \"ZSH_THEME=\\\"robbyrussell\\\"\" >> /etc/zsh/zshrc;",
      "echo \"export ZSH_COMPDUMP=\\$ZSH/cache/.zcompdump-\\$HOST\" >> /etc/zsh/zshrc;",
      "mkdir -p /etc/jupyter;",
      "echo \"c.ServerApp.terminado_settings = {\\\"shell_command\\\": [\\\"/bin/zsh\\\"]}\" >> /etc/jupyter/jupyter_server_config.py;",
      "echo \"source \\$ZSH/oh-my-zsh.sh\" >> /etc/zsh/zshrc;",
      "echo \"zstyle \\\":omz:update\\\" mode disabled\" >> /etc/zsh/zshrc;",
    ])`)
	}

	if enableJupyter {
		builder.WriteString("\n    config.jupyter()")
	}
	return builder.String()
}

func (mgr *AgentMgr) resolveCudaBaseImage(c *gin.Context, value string) (scriptBase string, imageLabel string, err error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", "", fmt.Errorf("cuda_base is required")
	}
	cudaQuery := query.CudaBaseImage
	if imageRecord, lookupErr := cudaQuery.WithContext(c).Where(cudaQuery.Value.Eq(trimmed)).First(); lookupErr == nil {
		return imageRecord.Value, imageRecord.ImageLabel, nil
	}
	if imageRecord, lookupErr := cudaQuery.WithContext(c).Where(cudaQuery.ImageLabel.Eq(trimmed)).First(); lookupErr == nil {
		return imageRecord.Value, imageRecord.ImageLabel, nil
	}
	if matches, lookupErr := cudaQuery.WithContext(c).Where(cudaQuery.Label.Eq(trimmed)).Find(); lookupErr == nil && len(matches) == 1 {
		return matches[0].Value, matches[0].ImageLabel, nil
	}
	if _, _, _, _, splitErr := pkgutils.SplitImageLink(trimmed); splitErr == nil {
		return trimmed, "", nil
	}
	return "", "", fmt.Errorf("cuda_base %q not found, use list_cuda_base_images first", trimmed)
}

func (mgr *AgentMgr) toolListImageBuilds(c *gin.Context, token internalutil.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	limit := getToolArgInt(args, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	statuses := normalizeImageBuildStatuses(getToolArgStringSlice(args, "statuses"))
	keyword := strings.ToLower(getToolArgString(args, "keyword", ""))

	kanikoQuery := query.Kaniko
	builds, err := kanikoQuery.WithContext(c).
		Where(kanikoQuery.UserID.Eq(token.UserID)).
		Order(kanikoQuery.CreatedAt.Desc()).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list image builds: %w", err)
	}

	allowedStatus := make(map[model.BuildStatus]struct{}, len(statuses))
	for _, status := range statuses {
		allowedStatus[status] = struct{}{}
	}

	filtered := make([]*model.Kaniko, 0, len(builds))
	statusCount := make(map[string]int)
	for _, item := range builds {
		if len(allowedStatus) > 0 {
			if _, ok := allowedStatus[item.Status]; !ok {
				continue
			}
		}
		if keyword != "" {
			searchText := strings.ToLower(item.ImagePackName + " " + item.ImageLink)
			if item.Description != nil {
				searchText += " " + strings.ToLower(*item.Description)
			}
			if !strings.Contains(searchText, keyword) {
				continue
			}
		}
		filtered = append(filtered, item)
		statusCount[string(item.Status)]++
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	items := make([]map[string]any, 0, len(filtered))
	for _, item := range filtered {
		finalImage, err := mgr.findFinalImageByPackName(c, item.ImagePackName)
		if err != nil {
			return nil, fmt.Errorf("failed to load final image for build %s: %w", item.ImagePackName, err)
		}
		items = append(items, buildImageBuildSummary(item, finalImage))
	}
	return map[string]any{
		"builds":      items,
		"count":       len(items),
		"statusCount": statusCount,
	}, nil
}

func (mgr *AgentMgr) toolGetImageBuildDetail(c *gin.Context, token internalutil.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	build, err := mgr.resolveOwnedImageBuild(c, token, args)
	if err != nil {
		return nil, err
	}
	finalImage, err := mgr.findFinalImageByPackName(c, build.ImagePackName)
	if err != nil {
		return nil, fmt.Errorf("failed to load final image record: %w", err)
	}
	detail := buildImageBuildSummary(build, finalImage)
	detail["script"] = strings.TrimSpace(derefString(build.Dockerfile))
	detail["template"] = strings.TrimSpace(build.Template)
	detail["pod"] = mgr.loadBuildPodInfo(c, build.ImagePackName)
	return map[string]any{"build": detail}, nil
}

func (mgr *AgentMgr) toolGetImageAccessDetail(c *gin.Context, token internalutil.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	imageRecord, err := mgr.resolveOwnedImageRecord(c, token, args)
	if err != nil {
		return nil, err
	}

	imageAccountQuery := query.ImageAccount
	accountShares, err := imageAccountQuery.WithContext(c).
		Preload(imageAccountQuery.Account).
		Where(imageAccountQuery.ImageID.Eq(imageRecord.ID)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to load account shares: %w", err)
	}
	accounts := make([]map[string]any, 0, len(accountShares))
	for _, share := range accountShares {
		accounts = append(accounts, map[string]any{
			"id":       share.Account.ID,
			"name":     share.Account.Name,
			"nickname": share.Account.Nickname,
		})
	}

	imageUserQuery := query.ImageUser
	userShares, err := imageUserQuery.WithContext(c).
		Preload(imageUserQuery.User).
		Where(imageUserQuery.ImageID.Eq(imageRecord.ID)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to load user shares: %w", err)
	}
	users := make([]map[string]any, 0, len(userShares))
	for _, share := range userShares {
		users = append(users, map[string]any{
			"id":       share.User.ID,
			"username": share.User.Name,
			"nickname": share.User.Nickname,
		})
	}

	return map[string]any{
		"image": buildImageSummaryFromRecord(imageRecord),
		"shares": map[string]any{
			"users":    users,
			"accounts": accounts,
		},
		"canManage": true,
	}, nil
}

func (mgr *AgentMgr) toolCreateImageBuild(c *gin.Context, token internalutil.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if mgr.imagePacker == nil || mgr.imageRegistry == nil {
		return nil, fmt.Errorf("image build dependencies are not configured")
	}
	if strings.TrimSpace(token.Username) == "" {
		return nil, fmt.Errorf("user identity is unavailable for image build")
	}
	args := parseToolArgsMap(rawArgs)
	mode, err := normalizeImageBuildMode(getToolArgString(args, "mode", ""))
	if err != nil {
		return nil, err
	}
	description := getToolArgString(args, "description", "")
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	archs := parseCSVBackedSlice(args, "archs", "archs_csv", []string{"linux/amd64"})
	tags := parseCSVBackedSlice(args, "tags", "tags_csv", nil)

	if err := mgr.imageRegistry.CheckOrCreateProjectForUser(c, token.Username); err != nil {
		return nil, fmt.Errorf("failed to prepare Harbor project: %w", err)
	}

	imagePackName := fmt.Sprintf("%s-%s", token.Username, uuid.New().String()[:5])

	switch mode {
	case agentImageBuildModePipApt:
		baseImage := getToolArgString(args, "base_image", getToolArgString(args, "image", ""))
		if baseImage == "" {
			return nil, fmt.Errorf("base_image is required for pip_apt mode")
		}
		imageLink, err := pkgutils.GenerateNewImageLinkForDockerfileBuild(
			baseImage,
			token.Username,
			getToolArgString(args, "image_name", ""),
			getToolArgString(args, "image_tag", ""),
		)
		if err != nil {
			return nil, err
		}
		requirements := getToolArgString(args, "requirements", "")
		aptPackages := strings.Join(parseCSVBackedSlice(args, "apt_packages", "apt_packages_text", nil), " ")
		dockerfile := buildPipAptDockerfile(baseImage, aptPackages, requirements)
		req := &packer.BuildKitReq{
			UserID:       token.UserID,
			Namespace:    readNamespace(),
			JobName:      imagePackName,
			Dockerfile:   &dockerfile,
			Requirements: stringPtrIfNotBlank(requirements),
			Description:  &description,
			ImageLink:    imageLink,
			Tags:         tags,
			Template:     getToolArgString(args, "template", ""),
			BuildSource:  model.PipApt,
			Archs:        archs,
			Token:        token,
		}
		if err := mgr.imagePacker.CreateFromDockerfile(c, req); err != nil {
			return nil, fmt.Errorf("failed to submit pip_apt image build: %w", err)
		}
		return mgr.buildSubmittedImageBuildResult(c, imagePackName, imageLink, string(mode), model.PipApt, archs)
	case agentImageBuildModeDocker:
		dockerfile := getToolArgString(args, "dockerfile", "")
		if dockerfile == "" {
			return nil, fmt.Errorf("dockerfile is required for dockerfile mode")
		}
		baseImage, err := extractBaseImageFromDockerfileContent(dockerfile)
		if err != nil {
			return nil, err
		}
		imageLink, err := pkgutils.GenerateNewImageLinkForDockerfileBuild(
			baseImage,
			token.Username,
			getToolArgString(args, "image_name", ""),
			getToolArgString(args, "image_tag", ""),
		)
		if err != nil {
			return nil, err
		}
		req := &packer.BuildKitReq{
			UserID:       token.UserID,
			Namespace:    readNamespace(),
			JobName:      imagePackName,
			Dockerfile:   &dockerfile,
			Description:  &description,
			ImageLink:    imageLink,
			Tags:         tags,
			Template:     getToolArgString(args, "template", ""),
			BuildSource:  model.Dockerfile,
			Archs:        archs,
			VolumeMounts: parseVolumeMountsArg(args),
			Token:        token,
		}
		if err := mgr.imagePacker.CreateFromDockerfile(c, req); err != nil {
			return nil, fmt.Errorf("failed to submit dockerfile image build: %w", err)
		}
		return mgr.buildSubmittedImageBuildResult(c, imagePackName, imageLink, string(mode), model.Dockerfile, archs)
	case agentImageBuildModeEnvd:
		pythonVersion := getToolArgString(args, "python_version", getToolArgString(args, "python", "3.10"))
		scriptBase, imageLabel, err := mgr.resolveCudaBaseImage(c, getToolArgString(args, "cuda_base", getToolArgString(args, "base", "")))
		if err != nil {
			return nil, err
		}
		pythonPackages := strings.Fields(strings.ReplaceAll(getToolArgString(args, "requirements", ""), "\n", " "))
		aptPackages := parseCSVBackedSlice(args, "apt_packages", "apt_packages_text", nil)
		enableJupyter := getToolArgBool(args, "enable_jupyter", true)
		enableZsh := getToolArgBool(args, "enable_zsh", true)
		envdScript := buildEnvdAdvancedScript(scriptBase, pythonVersion, aptPackages, pythonPackages, enableJupyter, enableZsh)
		imageLink, err := pkgutils.GenerateNewImageLinkForEnvdBuild(
			token.Username,
			pythonVersion,
			imageLabel,
			getToolArgString(args, "image_name", ""),
			getToolArgString(args, "image_tag", ""),
		)
		if err != nil {
			return nil, err
		}
		req := &packer.EnvdReq{
			UserID:      token.UserID,
			Namespace:   readNamespace(),
			JobName:     imagePackName,
			Envd:        &envdScript,
			Description: &description,
			ImageLink:   imageLink,
			Tags:        tags,
			Template:    getToolArgString(args, "template", ""),
			BuildSource: model.EnvdAdvanced,
			Archs:       archs,
		}
		if err := mgr.imagePacker.CreateFromEnvd(c, req); err != nil {
			return nil, fmt.Errorf("failed to submit envd image build: %w", err)
		}
		return mgr.buildSubmittedImageBuildResult(c, imagePackName, imageLink, string(mode), model.EnvdAdvanced, archs)
	case agentImageBuildModeEnvdRaw:
		envdScript := getToolArgString(args, "envd_script", getToolArgString(args, "envd", ""))
		if envdScript == "" {
			return nil, fmt.Errorf("envd_script is required for envd_raw mode")
		}
		imageLink, err := pkgutils.GenerateNewImageLinkForEnvdBuild(
			token.Username,
			getToolArgString(args, "python_version", getToolArgString(args, "python", "3.10")),
			getToolArgString(args, "base", ""),
			getToolArgString(args, "image_name", ""),
			getToolArgString(args, "image_tag", ""),
		)
		if err != nil {
			return nil, err
		}
		req := &packer.EnvdReq{
			UserID:      token.UserID,
			Namespace:   readNamespace(),
			JobName:     imagePackName,
			Envd:        &envdScript,
			Description: &description,
			ImageLink:   imageLink,
			Tags:        tags,
			Template:    getToolArgString(args, "template", ""),
			BuildSource: model.EnvdRaw,
			Archs:       archs,
		}
		if err := mgr.imagePacker.CreateFromEnvd(c, req); err != nil {
			return nil, fmt.Errorf("failed to submit envd_raw image build: %w", err)
		}
		return mgr.buildSubmittedImageBuildResult(c, imagePackName, imageLink, string(mode), model.EnvdRaw, archs)
	default:
		return nil, fmt.Errorf("unsupported image build mode %q", mode)
	}
}

func (mgr *AgentMgr) buildSubmittedImageBuildResult(
	c *gin.Context,
	imagePackName, imageLink, mode string,
	buildSource model.BuildSource,
	archs []string,
) (map[string]any, error) {
	result := map[string]any{
		"status":        "submitted",
		"mode":          mode,
		"buildSource":   buildSource,
		"imagePackName": imagePackName,
		"imageLink":     imageLink,
		"archs":         archs,
	}
	kanikoQuery := query.Kaniko
	buildRecord, err := kanikoQuery.WithContext(c).Where(kanikoQuery.ImagePackName.Eq(imagePackName)).First()
	if err == nil && buildRecord != nil {
		result["buildID"] = buildRecord.ID
		result["buildStatus"] = buildRecord.Status
	} else {
		result["buildStatus"] = "PendingReconcile"
	}
	return result, nil
}

func buildPipAptDockerfile(baseImage, aptPackages, requirements string) string {
	aptSection := "\n# No APT packages specified"
	if strings.TrimSpace(aptPackages) != "" {
		packages := strings.Fields(aptPackages)
		aptSection = fmt.Sprintf(`
# Install APT packages
RUN apt-get update && apt-get install -y %s && \
    rm -rf /var/lib/apt/lists/*`, strings.Join(packages, " "))
	}
	requirementsSection := "\n# No Python dependencies specified"
	if strings.TrimSpace(requirements) != "" {
		requirementsSection = `
# Install Python dependencies
COPY requirements.txt /requirements.txt
RUN pip install --extra-index-url https://mirrors.aliyun.com/pypi/simple/ --no-cache-dir -r /requirements.txt
`
	}
	return fmt.Sprintf(`FROM %s
USER root
%s
%s
`, baseImage, aptSection, requirementsSection)
}

func (mgr *AgentMgr) toolManageImageBuild(c *gin.Context, token internalutil.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if mgr.imagePacker == nil {
		return nil, fmt.Errorf("image build dependencies are not configured")
	}
	args := parseToolArgsMap(rawArgs)
	action := strings.ToLower(strings.TrimSpace(getToolArgString(args, "action", "")))
	if action != "cancel" && action != "delete" {
		return nil, fmt.Errorf("action must be cancel or delete")
	}
	build, err := mgr.resolveOwnedImageBuild(c, token, args)
	if err != nil {
		return nil, err
	}

	result := buildImageBuildSummary(build, nil)
	result["action"] = action
	result["statusBefore"] = build.Status

	kanikoQuery := query.Kaniko
	switch action {
	case "cancel":
		switch build.Status {
		case model.BuildJobFinished, model.BuildJobFailed, model.BuildJobCanceled:
			return nil, fmt.Errorf("image build %s is already finished; use delete instead", build.ImagePackName)
		}
		if err := mgr.imagePacker.DeleteJob(c, build.ImagePackName, readNamespace()); err != nil {
			return nil, fmt.Errorf("failed to cancel image build %s: %w", build.ImagePackName, err)
		}
		_, _ = kanikoQuery.WithContext(c).
			Where(kanikoQuery.ID.Eq(build.ID)).
			Update(kanikoQuery.Status, model.BuildJobCanceled)
		result["status"] = "cancellation_requested"
		return result, nil
	case "delete":
		if build.Status != model.BuildJobFinished && build.Status != model.BuildJobFailed && build.Status != model.BuildJobCanceled {
			return nil, fmt.Errorf("image build %s is still active; cancel it before deleting", build.ImagePackName)
		}
		finalImage, err := mgr.findFinalImageByPackName(c, build.ImagePackName)
		if err != nil {
			return nil, fmt.Errorf("failed to load final image before deletion: %w", err)
		}
		if finalImage != nil {
			imageQuery := query.Image
			if _, err := imageQuery.WithContext(c).Where(imageQuery.ID.Eq(finalImage.ID)).Delete(); err != nil {
				return nil, fmt.Errorf("failed to delete final image record: %w", err)
			}
			result["deletedFinalImageID"] = finalImage.ID
			if build.Status == model.BuildJobFinished && mgr.imageRegistry != nil {
				if err := mgr.imageRegistry.DeleteImageFromProject(c, build.ImageLink); err != nil {
					result["warning"] = fmt.Sprintf("image record deleted but failed to delete Harbor artifact: %v", err)
				}
			}
		}
		if _, err := kanikoQuery.WithContext(c).Where(kanikoQuery.ID.Eq(build.ID)).Delete(); err != nil {
			return nil, fmt.Errorf("failed to delete image build record: %w", err)
		}
		result["status"] = "deleted"
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported action %q", action)
	}
}

func (mgr *AgentMgr) toolRegisterExternalImage(c *gin.Context, token internalutil.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	imageLink := getToolArgString(args, "image_link", "")
	if imageLink == "" {
		return nil, fmt.Errorf("image_link is required")
	}
	if _, _, _, _, err := pkgutils.SplitImageLink(imageLink); err != nil {
		return nil, err
	}
	description := getToolArgString(args, "description", "")
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	taskType, err := normalizeImageTaskType(getToolArgString(args, "task_type", "custom"))
	if err != nil {
		return nil, err
	}
	archs := parseCSVBackedSlice(args, "archs", "archs_csv", []string{"linux/amd64"})
	tags := parseCSVBackedSlice(args, "tags", "tags_csv", nil)

	imageQuery := query.Image
	existing, err := imageQuery.WithContext(c).
		Preload(imageQuery.User).
		Where(imageQuery.UserID.Eq(token.UserID)).
		Where(imageQuery.ImageLink.Eq(imageLink)).
		First()
	if err == nil && existing != nil {
		return map[string]any{
			"status": "already_exists",
			"image":  buildImageSummaryFromRecord(existing),
		}, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing external image: %w", err)
	}

	imageRecord := &model.Image{
		UserID:      token.UserID,
		ImageLink:   imageLink,
		TaskType:    taskType,
		IsPublic:    false,
		Description: &description,
		Tags:        datatypes.NewJSONType(tags),
		Archs:       datatypes.NewJSONType(archs),
		ImageSource: model.ImageUploadType,
	}
	if err := imageQuery.WithContext(c).Create(imageRecord); err != nil {
		return nil, fmt.Errorf("failed to register external image: %w", err)
	}
	created, err := imageQuery.WithContext(c).Preload(imageQuery.User).Where(imageQuery.ID.Eq(imageRecord.ID)).First()
	if err != nil {
		return nil, fmt.Errorf("failed to reload external image: %w", err)
	}
	return map[string]any{
		"status": "registered",
		"image":  buildImageSummaryFromRecord(created),
	}, nil
}

func (mgr *AgentMgr) listMemberAccounts(c *gin.Context, userID uint) (map[uint]model.Account, error) {
	userAccountQuery := query.UserAccount
	accountIDs := []uint{}
	if err := userAccountQuery.WithContext(c).
		Where(userAccountQuery.UserID.Eq(userID)).
		Pluck(userAccountQuery.AccountID, &accountIDs); err != nil {
		return nil, err
	}
	if len(accountIDs) == 0 {
		return map[uint]model.Account{}, nil
	}
	accountQuery := query.Account
	accounts, err := accountQuery.WithContext(c).
		Where(accountQuery.ID.In(accountIDs...)).
		Find()
	if err != nil {
		return nil, err
	}
	result := make(map[uint]model.Account, len(accounts))
	for _, item := range accounts {
		result[item.ID] = *item
	}
	return result, nil
}

func resolveTargetByID(input string) (uint, bool) {
	id, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || id <= 0 {
		return 0, false
	}
	return uint(id), true
}

func (mgr *AgentMgr) resolveImageShareTargets(
	c *gin.Context,
	token internalutil.JWTMessage,
	targetType string,
	targets []string,
) ([]resolvedShareTarget, error) {
	targets = uniqueStrings(trimStringSlice(targets))
	if len(targets) == 0 {
		return nil, fmt.Errorf("targets are required")
	}

	switch strings.ToLower(strings.TrimSpace(targetType)) {
	case "user", "users":
		userQuery := query.User
		resolved := make([]resolvedShareTarget, 0, len(targets))
		for _, input := range targets {
			if targetID, ok := resolveTargetByID(input); ok {
				userRecord, err := userQuery.WithContext(c).Where(userQuery.ID.Eq(targetID)).First()
				if err != nil {
					return nil, fmt.Errorf("user %q not found", input)
				}
				resolved = append(resolved, resolvedShareTarget{
					ID:       userRecord.ID,
					Kind:     "user",
					Name:     userRecord.Name,
					Nickname: userRecord.Nickname,
					Input:    input,
				})
				continue
			}
			userRecord, err := userQuery.WithContext(c).Where(userQuery.Name.Eq(input)).First()
			if err == nil {
				resolved = append(resolved, resolvedShareTarget{
					ID:       userRecord.ID,
					Kind:     "user",
					Name:     userRecord.Name,
					Nickname: userRecord.Nickname,
					Input:    input,
				})
				continue
			}
			matches, findErr := userQuery.WithContext(c).Where(userQuery.Nickname.Eq(input)).Find()
			if findErr != nil || len(matches) == 0 {
				return nil, fmt.Errorf("user %q not found", input)
			}
			if len(matches) > 1 {
				return nil, fmt.Errorf("user nickname %q is ambiguous; use username or numeric id", input)
			}
			resolved = append(resolved, resolvedShareTarget{
				ID:       matches[0].ID,
				Kind:     "user",
				Name:     matches[0].Name,
				Nickname: matches[0].Nickname,
				Input:    input,
			})
		}
		return resolved, nil
	case "account", "accounts":
		memberAccounts, err := mgr.listMemberAccounts(c, token.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to load member accounts: %w", err)
		}
		resolved := make([]resolvedShareTarget, 0, len(targets))
		for _, input := range targets {
			if targetID, ok := resolveTargetByID(input); ok {
				accountRecord, exists := memberAccounts[targetID]
				if !exists || accountRecord.ID == model.DefaultAccountID {
					return nil, fmt.Errorf("account %q is not in the caller's accessible account set", input)
				}
				resolved = append(resolved, resolvedShareTarget{
					ID:       accountRecord.ID,
					Kind:     "account",
					Name:     accountRecord.Name,
					Nickname: accountRecord.Nickname,
					Input:    input,
				})
				continue
			}
			var matched *model.Account
			for _, accountRecord := range memberAccounts {
				if accountRecord.ID == model.DefaultAccountID {
					continue
				}
				if accountRecord.Name == input || accountRecord.Nickname == input {
					if matched != nil {
						return nil, fmt.Errorf("account name %q is ambiguous; use numeric id", input)
					}
					accountCopy := accountRecord
					matched = &accountCopy
				}
			}
			if matched == nil {
				return nil, fmt.Errorf("account %q is not in the caller's accessible account set", input)
			}
			resolved = append(resolved, resolvedShareTarget{
				ID:       matched.ID,
				Kind:     "account",
				Name:     matched.Name,
				Nickname: matched.Nickname,
				Input:    input,
			})
		}
		return resolved, nil
	default:
		return nil, fmt.Errorf("target_type must be user or account")
	}
}

func (mgr *AgentMgr) toolManageImageAccess(c *gin.Context, token internalutil.JWTMessage, rawArgs json.RawMessage) (any, error) {
	args := parseToolArgsMap(rawArgs)
	action := strings.ToLower(strings.TrimSpace(getToolArgString(args, "action", "")))
	if action != "grant" && action != "revoke" {
		return nil, fmt.Errorf("action must be grant or revoke")
	}
	targetType := strings.ToLower(strings.TrimSpace(getToolArgString(args, "target_type", "")))
	imageRecord, err := mgr.resolveOwnedImageRecord(c, token, args)
	if err != nil {
		return nil, err
	}
	resolvedTargets, err := mgr.resolveImageShareTargets(c, token, targetType, parseFlexibleTargets(args))
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0, len(resolvedTargets))
	for _, target := range resolvedTargets {
		entry := map[string]any{
			"input":      target.Input,
			"id":         target.ID,
			"targetType": target.Kind,
			"name":       target.Name,
			"nickname":   target.Nickname,
		}
		switch action {
		case "grant":
			if target.Kind == "user" {
				status, msg, err := mgr.grantImageToUser(c, imageRecord.ID, target.ID)
				if err != nil {
					return nil, err
				}
				entry["status"] = status
				if msg != "" {
					entry["message"] = msg
				}
			} else {
				status, msg, err := mgr.grantImageToAccount(c, imageRecord.ID, target.ID)
				if err != nil {
					return nil, err
				}
				entry["status"] = status
				if msg != "" {
					entry["message"] = msg
				}
			}
		case "revoke":
			if target.Kind == "user" {
				status, msg, err := mgr.revokeImageFromUser(c, imageRecord.ID, target.ID)
				if err != nil {
					return nil, err
				}
				entry["status"] = status
				if msg != "" {
					entry["message"] = msg
				}
			} else {
				status, msg, err := mgr.revokeImageFromAccount(c, imageRecord.ID, target.ID)
				if err != nil {
					return nil, err
				}
				entry["status"] = status
				if msg != "" {
					entry["message"] = msg
				}
			}
		}
		results = append(results, entry)
	}

	return map[string]any{
		"status":     "updated",
		"action":     action,
		"targetType": targetType,
		"image":      buildImageSummaryFromRecord(imageRecord),
		"results":    results,
	}, nil
}

func (mgr *AgentMgr) grantImageToUser(c *gin.Context, imageID, userID uint) (status, message string, err error) {
	imageUserQuery := query.ImageUser
	existing, err := imageUserQuery.WithContext(c).
		Where(imageUserQuery.ImageID.Eq(imageID)).
		Where(imageUserQuery.UserID.Eq(userID)).
		First()
	if err == nil && existing != nil {
		return "already_granted", "image is already shared to this user", nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", err
	}
	entity := &model.ImageUser{ImageID: imageID, UserID: userID}
	if err := imageUserQuery.WithContext(c).Create(entity); err != nil {
		return "", "", err
	}
	return "granted", "", nil
}

func (mgr *AgentMgr) grantImageToAccount(c *gin.Context, imageID, accountID uint) (status, message string, err error) {
	imageAccountQuery := query.ImageAccount
	existing, err := imageAccountQuery.WithContext(c).
		Where(imageAccountQuery.ImageID.Eq(imageID)).
		Where(imageAccountQuery.AccountID.Eq(accountID)).
		First()
	if err == nil && existing != nil {
		return "already_granted", "image is already shared to this account", nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", err
	}
	entity := &model.ImageAccount{ImageID: imageID, AccountID: accountID}
	if err := imageAccountQuery.WithContext(c).Create(entity); err != nil {
		return "", "", err
	}
	return "granted", "", nil
}

func (mgr *AgentMgr) revokeImageFromUser(c *gin.Context, imageID, userID uint) (status, message string, err error) {
	imageUserQuery := query.ImageUser
	existing, err := imageUserQuery.WithContext(c).
		Where(imageUserQuery.ImageID.Eq(imageID)).
		Where(imageUserQuery.UserID.Eq(userID)).
		First()
	if errors.Is(err, gorm.ErrRecordNotFound) || existing == nil {
		return "not_granted", "image was not shared to this user", nil
	}
	if err != nil {
		return "", "", err
	}
	if _, err := imageUserQuery.WithContext(c).Delete(existing); err != nil {
		return "", "", err
	}
	return "revoked", "", nil
}

func (mgr *AgentMgr) revokeImageFromAccount(c *gin.Context, imageID, accountID uint) (status, message string, err error) {
	imageAccountQuery := query.ImageAccount
	existing, err := imageAccountQuery.WithContext(c).
		Where(imageAccountQuery.ImageID.Eq(imageID)).
		Where(imageAccountQuery.AccountID.Eq(accountID)).
		First()
	if errors.Is(err, gorm.ErrRecordNotFound) || existing == nil {
		return "not_granted", "image was not shared to this account", nil
	}
	if err != nil {
		return "", "", err
	}
	if _, err := imageAccountQuery.WithContext(c).Delete(existing); err != nil {
		return "", "", err
	}
	return "revoked", "", nil
}

func stringPtrIfNotBlank(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
