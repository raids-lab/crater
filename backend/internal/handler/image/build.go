package image

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/packer"
	"github.com/raids-lab/crater/pkg/utils"
)

// UserCreateByPipApt godoc
//
//	@Summary		创建ImagePack CRD和数据库Kaniko entity
//	@Description	获取参数，生成变量，调用接口
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body	CreateKanikoRequest	true	"创建ImagePack CRD & Kaniko entity"
//	@Router			/v1/images/kaniko [POST]
func (mgr *ImagePackMgr) UserCreateByPipApt(c *gin.Context) {
	req := &CreateKanikoRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}
	dockerfile := mgr.generateDockerfile(req)
	buildData := &DockerfileBuildData{
		BaseImage:    req.SourceImage,
		Description:  req.Description,
		Dockerfile:   dockerfile,
		ImageName:    req.ImageName,
		ImageTag:     req.ImageTag,
		UserName:     token.Username,
		UserID:       token.UserID,
		Requirements: &req.PythonRequirements,
		Tags:         req.Tags,
		Template:     req.Template,
		BuildSource:  model.PipApt,
		Archs:        req.Archs,
	}
	mgr.buildFromDockerfile(c, buildData)
}

// UserCreateByDockerfile godoc
//
//	@Summary		接受用户传入的Dockerfile和描述，创建镜像
//	@Description	获取参数，提取Dockerfile中的基础镜像，调用接口
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body	CreateByDockerfileRequest	true	"创建ImagePack CRD"
//	@Router			/v1/images/dockerfile [POST]
func (mgr *ImagePackMgr) UserCreateByDockerfile(c *gin.Context) {
	req := &CreateByDockerfileRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}
	baseImage, err := extractBaseImageFromDockerfile(req.Dockerfile)
	if err != nil {
		msg := fmt.Sprintf("failed to extract base image from Dockerfile, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}
	buildData := &DockerfileBuildData{
		Description: req.Description,
		Dockerfile:  req.Dockerfile,
		ImageName:   req.ImageName,
		ImageTag:    req.ImageTag,
		BaseImage:   baseImage,
		UserName:    token.Username,
		UserID:      token.UserID,
		Tags:        req.Tags,
		Template:    req.Template,
		BuildSource: model.Dockerfile,
		Archs:       req.Archs,
	}
	mgr.buildFromDockerfile(c, buildData)
}

func extractBaseImageFromDockerfile(dockerfile string) (string, error) {
	lines := strings.Split(dockerfile, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "FROM") {
			for strings.HasSuffix(line, "\\") {
				line = line[:len(line)-1]
				line = strings.TrimSpace(line)
				// last line check
				if i+1 >= len(lines) {
					return "", fmt.Errorf("unexpected end of Dockerfile after line: %s", line)
				}
				nextLine := strings.TrimSpace(lines[i+1])
				line += " " + nextLine
				i++ // move to next line
			}

			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
			return "", fmt.Errorf("invalid FROM instruction: %s", line)
		}
	}
	return "", fmt.Errorf("no FROM instruction found in Dockerfile")
}

// UserCreateByEnvd godoc
//
//	@Summary		接受用户传入的Envd内容和描述，创建镜像
//	@Description	获取参数，提取Dockerfile中的基础镜像，调用接口
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body	CreateByEnvdRequest	true	"创建ImagePack CRD"
//	@Router			/v1/images/envd [POST]
func (mgr *ImagePackMgr) UserCreateByEnvd(c *gin.Context) {
	req := &CreateByEnvdRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}

	buildData := &EnvdBuildData{
		Description: req.Description,
		Envd:        req.Envd,
		ImageName:   req.ImageName,
		ImageTag:    req.ImageTag,
		Python:      req.Python,
		Base:        req.Base,
		UserName:    token.Username,
		UserID:      token.UserID,
		Tags:        req.Tags,
		Template:    req.Template,
		BuildSource: req.BuildSource,
		Archs:       req.Archs,
	}
	mgr.buildFromEnvd(c, buildData)
}

// AdminCreate godoc
//
//	@Summary		创建ImagePack CRD和数据库kaniko entity
//	@Description	获取参数，生成变量，调用接口
//	@Tags			ImagePack
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			data	body	CreateKanikoRequest	true	"创建ImagePack"
//	@Router			/v1/admin/images/kaniko [POST]
func (mgr *ImagePackMgr) AdminCreate(c *gin.Context) {
	req := &CreateKanikoRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		resputil.BadRequestError(c, msg)
		return
	}
	klog.Infof("create params: %+v", req)
	dockerfile := mgr.generateDockerfile(req)
	buildData := &DockerfileBuildData{
		BaseImage:   req.SourceImage,
		Description: req.Description,
		Dockerfile:  dockerfile,
		ImageName:   req.ImageName,
		ImageTag:    req.ImageTag,
		UserName:    token.Username,
		UserID:      token.UserID,
		Tags:        req.Tags,
		Template:    req.Template,
		BuildSource: model.Dockerfile,
	}
	mgr.buildFromDockerfile(c, buildData)
}

func (mgr *ImagePackMgr) generateDockerfile(req *CreateKanikoRequest) string {
	// Handle APT packages
	aptInstallSection := "\n# No APT packages specified"
	if req.APTPackages != "" {
		aptPackages := strings.Fields(req.APTPackages) // split by space
		aptInstallSection = fmt.Sprintf(`
# Install APT packages
RUN apt-get update && apt-get install -y %s && \
    rm -rf /var/lib/apt/lists/*`, strings.Join(aptPackages, " "))
	}

	// Generate requirements.txt and install Python dependencies
	requirementsSection := "\n# No Python dependencies specified"
	if req.PythonRequirements != "" {
		requirementsSection =
			`
# Install Python dependencies
COPY requirements.txt /requirements.txt
RUN pip install --extra-index-url https://mirrors.aliyun.com/pypi/simple/ --no-cache-dir -r /requirements.txt
`
	}
	// Generate Dockerfile
	dockerfile := fmt.Sprintf(`FROM %s
USER root
%s
%s
`, req.SourceImage, aptInstallSection, requirementsSection)

	return dockerfile
}

func (mgr *ImagePackMgr) buildFromDockerfile(c *gin.Context, data *DockerfileBuildData) {
	if err := mgr.imageRegistry.CheckOrCreateProjectForUser(c, data.UserName); err != nil {
		resputil.Error(c, "create harbor project failed", resputil.NotSpecified)
		return
	}
	imagepackName := fmt.Sprintf("%s-%s", data.UserName, uuid.New().String()[:5])
	imageLink, err := utils.GenerateNewImageLinkForDockerfileBuild(data.BaseImage, data.UserName, data.ImageName, data.ImageTag)
	if err != nil {
		resputil.Error(c, "generate new image link failed", resputil.NotSpecified)
		return
	}

	// create ImagePack CRD
	buildkitData := &packer.BuildKitReq{
		JobName:      imagepackName,
		Namespace:    UserNameSpace,
		Dockerfile:   &data.Dockerfile,
		ImageLink:    imageLink,
		UserID:       data.UserID,
		Description:  &data.Description,
		Requirements: data.Requirements,
		Tags:         data.Tags,
		Template:     data.Template,
		BuildSource:  data.BuildSource,
		Archs:        data.Archs,
	}

	if err := mgr.imagePacker.CreateFromDockerfile(c, buildkitData); err != nil {
		klog.Errorf("create imagepack failed, err:%+v", err)
		resputil.Error(c, "create imagepack failed", resputil.NotSpecified)
		return
	}

	resputil.Success(c, "")
}

func (mgr *ImagePackMgr) buildFromEnvd(c *gin.Context, data *EnvdBuildData) {
	if err := mgr.imageRegistry.CheckOrCreateProjectForUser(c, data.UserName); err != nil {
		resputil.Error(c, "create harbor project failed", resputil.NotSpecified)
		return
	}
	imagepackName := fmt.Sprintf("%s-%s", data.UserName, uuid.New().String()[:5])
	imageLink, err := utils.GenerateNewImageLinkForEnvdBuild(data.UserName, data.Python, data.Base, data.ImageName, data.ImageTag)
	if err != nil {
		resputil.Error(c, "generate new image link failed", resputil.NotSpecified)
		return
	}

	// create ImagePack CRD
	envdData := &packer.EnvdReq{
		JobName:     imagepackName,
		Namespace:   UserNameSpace,
		Envd:        &data.Envd,
		ImageLink:   imageLink,
		UserID:      data.UserID,
		Description: &data.Description,
		Tags:        data.Tags,
		Template:    data.Template,
		BuildSource: data.BuildSource,
		Archs:       data.Archs,
	}

	if err := mgr.imagePacker.CreateFromEnvd(c, envdData); err != nil {
		klog.Errorf("create imagepack failed, err:%+v", err)
		resputil.Error(c, "create imagepack failed", resputil.NotSpecified)
		return
	}

	resputil.Success(c, "")
}
