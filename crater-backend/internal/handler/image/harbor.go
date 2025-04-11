package image

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/utils"
)

// GetHarborIP godoc
// @Summary 获取harbor的部署地址
// @Description 通过后端获取harbor的部署地址
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/images/harbor [GET]
func (mgr *ImagePackMgr) GetHarborIP(c *gin.Context) {
	harborIP := mgr.imageRegistry.GetHarborIP()
	resp := GetHarborIPResponse{
		HarborIP: harborIP,
	}
	resputil.Success(c, resp)
}

// CheckLinkValidity godoc
// @Summary 检查镜像链接是否有效
// @Description 通过获取的镜像链接列表，遍历其中的链接，检查是否有效
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/images/valid [POST]
func (mgr *ImagePackMgr) CheckLinkValidity(c *gin.Context) {
	req := &CheckLinkValidityRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate link pairs failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	invalidPairs := []ImageInfoLinkPair{}
	for _, linkPair := range req.LinkPairs {
		if !mgr.checkLinkValidity(linkPair.ImageLink) {
			invalidPairs = append(invalidPairs, linkPair)
		}
	}
	resp := CheckLinkValidityResponse{
		InvalidPairs: invalidPairs,
	}
	fmt.Println(resp)
	resputil.Success(c, resp)
}

func (mgr *ImagePackMgr) checkLinkValidity(link string) bool {
	ip, project, repository, tag, err := utils.SplitImageLink(link)
	if err != nil {
		logutils.Log.Errorf("split image link failed, err %v", err)
		return false
	}
	encodedRepo := url.PathEscape(repository)
	encodedURL := fmt.Sprintf("https://%s/api/v2.0/projects/%s/repositories/%s/artifacts/%s", ip, project, encodedRepo, tag)
	response, err := mgr.req.R().Get(encodedURL)
	if err != nil {
		logutils.Log.Errorf("http failure between checking link validity failed, err %v", err)
		return false
	}
	return response.IsSuccessState()
}

// UserGetProjectDetail godoc
// @Summary 获取用户project的信息
// @Description 获取用户的project的详细信息
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/images/quota [GET]
func (mgr *ImagePackMgr) UserGetProjectDetail(c *gin.Context) {
	token := util.GetToken(c)
	detail, err := mgr.imageRegistry.GetProjectDetail(c, token.Username)
	if err != nil {
		logutils.Log.Errorf("fetch project quota failed, err:%v", err)
	}
	resp := GetProjectDetailResponse{
		Quota:   float64(detail.TotalSize) / float64(GBit),
		Used:    float64(detail.UsedSize) / float64(GBit),
		Project: detail.ProjectName,
	}
	resputil.Success(c, resp)
}

// UserGetProjectCredential godoc
// @Summary 创建用户的harbor项目，并返回用户的harbor项目的凭证
// @Description 获取参数，生成变量，调用接口
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Router /v1/images/credential [POST]
func (mgr *ImagePackMgr) UserGetProjectCredential(c *gin.Context) {
	token := util.GetToken(c)
	if err := mgr.imageRegistry.CheckOrCreateProjectForUser(c, token.Username); err != nil {
		logutils.Log.Errorf("check project failed")
		resputil.Error(c, "check or create project failed", resputil.NotSpecified)
		return
	}
	if exist := mgr.imageRegistry.CheckUserExist(c, token.Username); exist {
		err := mgr.imageRegistry.DeleteUser(c, token.Username)
		if err != nil {
			logutils.Log.Errorf("delete user failed")
			resputil.Error(c, "delete user failed", resputil.NotSpecified)
			return
		}
	}
	password, err := mgr.imageRegistry.CreateUser(c, token.Username)
	if err != nil {
		logutils.Log.Errorf("create user failed: %+v", err)
		resputil.Error(c, "create user failed", resputil.NotSpecified)
		return
	}
	if err = mgr.imageRegistry.AddProjectMember(c, token.Username); err != nil {
		logutils.Log.Errorf("add project member failed")
		resputil.Error(c, "add project member failed", resputil.NotSpecified)
		return
	}
	fmt.Printf("username: %s, password: %s\n", token.Username, password)
	resp := GetProjectCredentialResponse{
		Name:     &token.Username,
		Password: &password,
	}
	resputil.Success(c, resp)
}

// UpdateProjectQuota godoc
// @Summary 更新project的配额
// @Description 传入int64参数，查找用户的project，并更新镜像存储的配额
// @Tags ImagePack
// @Accept json
// @Produce json
// @Security Bearer
// @Param req body UpdateProjectQuotaRequest true "更新镜像的ID和存储大小"
// @Router /v1/images/quota [POST]
func (mgr *ImagePackMgr) UpdateProjectQuota(c *gin.Context) {
	req := &UpdateProjectQuotaRequest{}
	token := util.GetToken(c)
	if err := c.ShouldBindJSON(req); err != nil {
		logutils.Log.Errorf("validate update project quota failed, err %v", err)
		resputil.HTTPError(c, http.StatusBadRequest, "validate failed", resputil.NotSpecified)
		return
	}
	projectName := fmt.Sprintf("user-%s", token.Username)
	if err := mgr.imageRegistry.UpdateQuotaForProject(c, projectName, req.Size); err != nil {
		resputil.Error(c, "update harbor project quota failed", resputil.NotSpecified)
	}
	resputil.Success(c, "")
}
