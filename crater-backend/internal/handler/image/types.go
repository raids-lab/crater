package image

import (
	"time"

	"github.com/raids-lab/crater/dao/model"
)

type (
	CreateKanikoRequest struct {
		SourceImage        string   `json:"image"`
		PythonRequirements string   `json:"requirements"`
		APTPackages        string   `json:"packages"`
		Description        string   `json:"description"`
		ImageName          string   `json:"name"`
		ImageTag           string   `json:"tag"`
		Tags               []string `json:"tags"`
		Template           string   `json:"template"`
		Archs              []string `json:"archs"`
	}

	CreateByDockerfileRequest struct {
		Description string   `json:"description"`
		Dockerfile  string   `json:"dockerfile"`
		ImageName   string   `json:"name"`
		ImageTag    string   `json:"tag"`
		Tags        []string `json:"tags"`
		Template    string   `json:"template"`
		Archs       []string `json:"archs"`
	}

	CreateByEnvdRequest struct {
		Description string            `json:"description"`
		Envd        string            `json:"envd"`
		ImageName   string            `json:"name"`
		ImageTag    string            `json:"tag"`
		Python      string            `json:"python"`
		Base        string            `json:"base"`
		Tags        []string          `json:"tags"`
		Template    string            `json:"template"`
		BuildSource model.BuildSource `json:"buildSource"`
		Archs       []string          `json:"archs"`
	}

	UploadImageRequest struct {
		ImageLink   string        `json:"imageLink"`
		TaskType    model.JobType `json:"taskType"`
		Description string        `json:"description"`
		Tags        []string      `json:"tags"`
		Archs       []string      `json:"archs"`
	}

	DeleteKanikoByIDRequest struct {
		ID uint `uri:"id" binding:"required"`
	}

	CancelKanikoByIDRequest struct {
		ID uint `json:"id" binding:"required"`
	}

	DeleteKanikoByIDListRequest struct {
		IDList []uint `json:"idList" binding:"required"`
	}

	DeleteImageByIDRequest struct {
		ID uint `uri:"id" binding:"required"`
	}

	DeleteImageByIDListRequest struct {
		IDList []uint `json:"idList" binding:"required"`
	}

	GetKanikoRequest struct {
		ImagePackName string `form:"name" binding:"required"`
	}

	GetKanikoPodRequest struct {
		ID uint `form:"id" binding:"required"`
	}

	ListAvailableImageRequest struct {
		Type model.JobType `form:"type" binding:"required"`
	}

	UpdateProjectQuotaRequest struct {
		Size int64 `json:"size" binding:"required"`
	}

	ChangeImagePublicStatusRequest struct {
		ID uint `uri:"id" binding:"required"`
	}

	CheckLinkValidityRequest struct {
		LinkPairs []ImageInfoLinkPair `json:"linkPairs"`
	}

	ChangeImageDescriptionRequest struct {
		ID          uint   `json:"id"`
		Description string `json:"description"`
	}

	ChangeImageTaskTypeRequest struct {
		ID       uint          `json:"id"`
		TaskType model.JobType `json:"taskType"`
	}

	ChangeImageTagsRequest struct {
		ID   uint     `json:"id"`
		Tags []string `json:"tags"`
	}

	ShareImageRequest struct {
		IDList  []uint `json:"idList"`
		ImageID uint   `json:"imageID"`
		Type    string `json:"type"` // user: share with user, account: share with account
	}

	CancelShareImageRequest struct {
		ID      uint   `json:"id"`
		ImageID uint   `json:"imageID"`
		Type    string `json:"type"` // user: cancel share with user, account: cancel share with account
	}

	ImageGrantRequest struct {
		ImageID uint `form:"imageID" binding:"required"`
	}

	UserSearchRequest struct {
		ImageID uint   `form:"imageID" binding:"required"`
		Name    string `form:"name"`
	}

	AccountSearchRequest struct {
		ImageID uint `form:"imageID" binding:"required"`
	}

	CudaBaseImageCreateRequest struct {
		ImageLabel string `json:"imageLabel" binding:"required"`
		Label      string `json:"label" binding:"required"`
		Value      string `json:"value" binding:"required"`
	}

	CudaBaseImageDeleteRequest struct {
		ID uint `uri:"id" binding:"required"`
	}

	UpdateImageArchRequest struct {
		ID    uint     `json:"id" binding:"required"`
		Archs []string `json:"archs" binding:"required"`
	}
)

type (
	ListImageResponse struct {
		ImageInfoList []*ImageInfo `json:"imageList"`
	}

	GetKanikoResponse struct {
		ID            uint              `json:"ID"`
		ImageLink     string            `json:"imageLink"`
		Status        model.BuildStatus `json:"status"`
		BuildSource   model.BuildSource `json:"buildSource"`
		CreatedAt     time.Time         `json:"createdAt"`
		ImagePackName string            `json:"imagepackName"`
		Description   string            `json:"description"`
		Dockerfile    string            `json:"dockerfile"`
		PodName       string            `json:"podName"`
		PodNameSpace  string            `json:"podNameSpace"`
	}

	GetKanikoPodResponse struct {
		PodName      string `json:"name"`
		PodNameSpace string `json:"namespace"`
	}

	ListAvailableImageResponse struct {
		Images []*ImageInfo `json:"images"`
	}

	GetProjectCredentialResponse struct {
		Name     *string `json:"name"`
		Password *string `json:"password"`
	}

	GetProjectDetailResponse struct {
		Used    float64 `json:"used"`
		Quota   float64 `json:"quota"`
		Project string  `json:"project"`
		Total   int64   `json:"total"`
	}

	CheckLinkValidityResponse struct {
		InvalidPairs []ImageInfoLinkPair `json:"linkPairs"`
	}

	GetHarborIPResponse struct {
		HarborIP string `json:"ip"`
	}

	ImageGrantResponse struct {
		UserList    []ImageGrantedUsers    `json:"userList"`
		AccountList []ImageGrantedAccounts `json:"accountList"`
	}

	SearchUserResponse struct {
		UserList []ImageGrantedUsers `json:"userList"`
	}

	CudaBaseImagesResponse struct {
		CudaBaseImages []CudaBaseImage `json:"cudaBaseImages"`
	}
)

type (
	KanikoInfo struct {
		ID            uint              `json:"ID"`
		ImageLink     string            `json:"imageLink"`
		Status        model.BuildStatus `json:"status"`
		CreatedAt     time.Time         `json:"createdAt"`
		Size          int64             `json:"size"`
		Description   string            `json:"description"`
		UserInfo      model.UserInfo    `json:"userInfo"`
		Tags          []string          `json:"tags"`
		ImagePackName string            `json:"imagepackName"`
		BuildSource   model.BuildSource `json:"buildSource"`
		Archs         []string          `json:"archs"`
	}

	ListKanikoResponse struct {
		KanikoInfoList []KanikoInfo `json:"kanikoList"`
	}

	ImageInfo struct {
		ID               uint                  `json:"ID"`
		ImageLink        string                `json:"imageLink"`
		Description      *string               `json:"description"`
		CreatedAt        time.Time             `json:"createdAt"`
		TaskType         model.JobType         `json:"taskType"`
		IsPublic         bool                  `json:"isPublic"`
		UserInfo         model.UserInfo        `json:"userInfo"`
		Tags             []string              `json:"tags"`
		ImageBuildSource model.ImageSourceType `json:"imageBuildSource"`
		ImagePackName    *string               `json:"imagepackName"`
		ImageShareStatus model.ImageShareType  `json:"imageShareStatus"`
		Archs            []string              `json:"archs"`
	}

	ImageInfoLinkPair struct {
		ID          uint           `json:"id"`
		ImageLink   string         `json:"imageLink"`
		Description string         `json:"description"`
		Creator     model.UserInfo `json:"creator"`
	}

	DockerfileBuildData struct {
		BaseImage    string
		UserName     string
		UserID       uint
		Description  string
		Dockerfile   string
		ImageName    string
		ImageTag     string
		Requirements *string
		Tags         []string
		Template     string
		BuildSource  model.BuildSource
		Archs        []string
	}

	EnvdBuildData struct {
		Python      string
		Base        string
		Description string
		Envd        string
		ImageName   string
		ImageTag    string
		UserName    string
		UserID      uint
		Tags        []string
		Template    string
		BuildSource model.BuildSource
		Archs       []string
	}

	ImageGrantedUsers struct {
		Nickname string `json:"nickname"`
		Name     string `json:"name"`
		ID       uint   `json:"id"`
	}
	ImageGrantedAccounts struct {
		Name string `json:"name"`
		ID   uint   `json:"id"`
	}

	CudaBaseImage struct {
		ID         uint   `json:"id"`
		Label      string `json:"label"`
		ImageLabel string `json:"imageLabel"`
		Value      string `json:"value"`
	}
)
