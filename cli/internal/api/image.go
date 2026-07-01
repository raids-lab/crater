package api

import (
	"fmt"
	"time"
)

type ImageClient interface {
	ListKaniko(admin bool) (*ListKanikoResponse, error)
	GetKanikoByName(name string) (*KanikoDetailResponse, error)
	GetKanikoTemplateByName(name string) (string, error)
	GetKanikoPod(id uint) (*KanikoPodResponse, error)
	CreatePipApt(req PipAptBuildRequest) (string, error)
	CreateDockerfile(req DockerfileBuildRequest) (string, error)
	CreateEnvd(req EnvdBuildRequest) (string, error)
	RemoveKaniko(ids []uint, admin bool) (string, error)
	ListImageRecords(admin bool) (*ListImageResponse, error)
	ListAvailableImages() ([]ImageInfo, error)
	UploadImage(req ImageUploadRequest) (string, error)
	DeleteImage(id uint) (string, error)
	DeleteImages(ids []uint, admin bool) (string, error)
	UpdateImageDescription(req ImageDescriptionRequest, admin bool) (string, error)
	UpdateImageType(req ImageTypeRequest, admin bool) (string, error)
	UpdateImageTags(req ImageTagsRequest, admin bool) (string, error)
	UpdateImageArch(req ImageArchRequest, admin bool) (string, error)
	TogglePublic(id uint) (string, error)
	ShareImage(req ImageShareRequest) (string, error)
	CancelShareImage(req ImageCancelShareRequest) (string, error)
	GetImageGrants(imageID uint) (*ImageGrantResponse, error)
	ListUngrantedUsers(imageID uint, name string) (*ImageGrantResponse, error)
	ListUngrantedAccounts(imageID uint) (*ImageGrantResponse, error)
	CheckImageLinks(pairs []ImageInfoLinkPair) (*CheckLinkValidityResponse, error)
	GetHarbor() (*HarborResponse, error)
	GetCredential() (*ProjectCredentialResponse, error)
	GetQuota() (*ProjectDetailResponse, error)
	UpdateQuota(size int64) (string, error)
	ListCudaBaseImages() (*CudaBaseImagesResponse, error)
	AddCudaBaseImage(req CudaBaseImageRequest) (string, error)
	DeleteCudaBaseImage(id uint) (string, error)
}

type KanikoInfo struct {
	ID            uint      `json:"ID"`
	ImageLink     string    `json:"imageLink"`
	Status        string    `json:"status"`
	BuildSource   string    `json:"buildSource"`
	CreatedAt     time.Time `json:"createdAt"`
	Size          int64     `json:"size"`
	Description   string    `json:"description"`
	UserInfo      UserInfo  `json:"userInfo"`
	Tags          []string  `json:"tags"`
	ImagePackName string    `json:"imagepackName"`
	Archs         []string  `json:"archs"`
}

type ListKanikoResponse struct {
	KanikoList []KanikoInfo `json:"kanikoList"`
}

type KanikoDetailResponse struct {
	ID            uint      `json:"ID"`
	ImageLink     string    `json:"imageLink"`
	Status        string    `json:"status"`
	BuildSource   string    `json:"buildSource"`
	CreatedAt     time.Time `json:"createdAt"`
	ImagePackName string    `json:"imagepackName"`
	Description   string    `json:"description"`
	Dockerfile    string    `json:"dockerfile"`
	PodName       string    `json:"podName"`
	PodNameSpace  string    `json:"podNameSpace"`
	NodeName      string    `json:"nodeName"`
}

type KanikoPodResponse struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	NodeName  string `json:"nodeName"`
}

type ImageInfo struct {
	ID               uint      `json:"ID"`
	ImageLink        string    `json:"imageLink"`
	Description      *string   `json:"description"`
	CreatedAt        time.Time `json:"createdAt"`
	TaskType         string    `json:"taskType"`
	IsPublic         bool      `json:"isPublic"`
	UserInfo         UserInfo  `json:"userInfo"`
	Tags             []string  `json:"tags"`
	ImageBuildSource int       `json:"imageBuildSource"`
	ImagePackName    *string   `json:"imagepackName"`
	ImageShareStatus string    `json:"imageShareStatus"`
	Archs            []string  `json:"archs"`
}

type ListImageResponse struct {
	ImageList []ImageInfo `json:"imageList"`
}

type AvailableImageListResp struct {
	Images []ImageInfo `json:"images"`
}

type PipAptBuildRequest struct {
	Image        string   `json:"image"`
	Requirements string   `json:"requirements"`
	Packages     string   `json:"packages"`
	Description  string   `json:"description"`
	Name         string   `json:"name"`
	Tag          string   `json:"tag"`
	Tags         []string `json:"tags"`
	Template     string   `json:"template"`
	Archs        []string `json:"archs"`
}

type DockerfileBuildRequest struct {
	Description string   `json:"description"`
	Dockerfile  string   `json:"dockerfile"`
	Name        string   `json:"name"`
	Tag         string   `json:"tag"`
	Tags        []string `json:"tags"`
	Template    string   `json:"template"`
	Archs       []string `json:"archs"`
}

type EnvdBuildRequest struct {
	Description string   `json:"description"`
	Envd        string   `json:"envd"`
	Name        string   `json:"name"`
	Tag         string   `json:"tag"`
	Python      string   `json:"python"`
	Base        string   `json:"base"`
	Tags        []string `json:"tags"`
	Template    string   `json:"template"`
	BuildSource string   `json:"buildSource"`
	Archs       []string `json:"archs"`
}

type IDListRequest struct {
	IDList []uint `json:"idList"`
}

type ImageUploadRequest struct {
	ImageLink   string   `json:"imageLink"`
	TaskType    string   `json:"taskType"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Archs       []string `json:"archs"`
}

type ImageDescriptionRequest struct {
	ID          uint   `json:"id"`
	Description string `json:"description"`
}

type ImageTypeRequest struct {
	ID       uint   `json:"id"`
	TaskType string `json:"taskType"`
}

type ImageTagsRequest struct {
	ID   uint     `json:"id"`
	Tags []string `json:"tags"`
}

type ImageArchRequest struct {
	ID    uint     `json:"id"`
	Archs []string `json:"archs"`
}

type ImageShareRequest struct {
	IDList  []uint `json:"idList"`
	ImageID uint   `json:"imageID"`
	Type    string `json:"type"`
}

type ImageCancelShareRequest struct {
	ID      uint   `json:"id"`
	ImageID uint   `json:"imageID"`
	Type    string `json:"type"`
}

type ImageInfoLinkPair struct {
	ImageLink string `json:"imageLink"`
}

type CheckLinkValidityRequest struct {
	LinkPairs []ImageInfoLinkPair `json:"linkPairs"`
}

type CheckLinkValidityResponse struct {
	InvalidPairs []ImageInfoLinkPair `json:"linkPairs"`
}

type ImageGrantedUser struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
}

type ImageGrantedAccount struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ImageGrantResponse struct {
	UserList    []ImageGrantedUser    `json:"userList"`
	AccountList []ImageGrantedAccount `json:"accountList"`
}

type HarborResponse struct {
	IP string `json:"ip"`
}

type ProjectCredentialResponse struct {
	Name     *string `json:"name"`
	Password *string `json:"password"`
}

type ProjectDetailResponse struct {
	Used    float64 `json:"used"`
	Quota   float64 `json:"quota"`
	Project string  `json:"project"`
	Total   int64   `json:"total"`
}

type CudaBaseImage struct {
	ID         uint   `json:"id"`
	Label      string `json:"label"`
	ImageLabel string `json:"imageLabel"`
	Value      string `json:"value"`
}

type CudaBaseImagesResponse struct {
	CudaBaseImages []CudaBaseImage `json:"cudaBaseImages"`
}

type CudaBaseImageRequest struct {
	ImageLabel string `json:"imageLabel"`
	Label      string `json:"label"`
	Value      string `json:"value"`
}

func (c *Client) ListKaniko(admin bool) (*ListKanikoResponse, error) {
	path := ImagesPrefix + "/kaniko"
	if admin {
		path = AdminImagesPrefix + "/kaniko"
	}
	var result Response[ListKanikoResponse]
	if err := c.get(path, nil, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) GetKanikoByName(name string) (*KanikoDetailResponse, error) {
	var result Response[KanikoDetailResponse]
	if err := c.get(ImagesPrefix+"/getbyname", map[string]string{"name": name}, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) GetKanikoTemplateByName(name string) (string, error) {
	var result Response[string]
	if err := c.get(ImagesPrefix+"/template", map[string]string{"name": name}, &result); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) GetKanikoPod(id uint) (*KanikoPodResponse, error) {
	var result Response[KanikoPodResponse]
	if err := c.get(ImagesPrefix+"/podname", map[string]string{"id": fmt.Sprintf("%d", id)}, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) CreatePipApt(req PipAptBuildRequest) (string, error) {
	return c.postString(ImagesPrefix+"/kaniko", req)
}

func (c *Client) CreateDockerfile(req DockerfileBuildRequest) (string, error) {
	return c.postString(ImagesPrefix+"/dockerfile", req)
}

func (c *Client) CreateEnvd(req EnvdBuildRequest) (string, error) {
	return c.postString(ImagesPrefix+"/envd", req)
}

func (c *Client) RemoveKaniko(ids []uint, admin bool) (string, error) {
	path := ImagesPrefix + "/remove"
	if admin {
		path = AdminImagesPrefix + "/remove"
	}
	return c.postString(path, IDListRequest{IDList: ids})
}

func (c *Client) ListImageRecords(admin bool) (*ListImageResponse, error) {
	path := ImagesPrefix + "/image"
	if admin {
		path = AdminImagesPrefix + "/image"
	}
	var result Response[ListImageResponse]
	if err := c.get(path, nil, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) ListAvailableImages() ([]ImageInfo, error) {
	var result Response[AvailableImageListResp]
	if err := c.get(ImageAvailablePath, nil, &result); err != nil {
		return nil, err
	}
	return result.Data.Images, nil
}

func (c *Client) UploadImage(req ImageUploadRequest) (string, error) {
	return c.postString(ImagesPrefix+"/image", req)
}

func (c *Client) DeleteImage(id uint) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Delete(fmt.Sprintf("%s/image/%d", ImagesPrefix, id))
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) DeleteImages(ids []uint, admin bool) (string, error) {
	path := ImagesPrefix + "/deleteimage"
	if admin {
		path = AdminImagesPrefix + "/deleteimage"
	}
	return c.postString(path, IDListRequest{IDList: ids})
}

func (c *Client) UpdateImageDescription(req ImageDescriptionRequest, admin bool) (string, error) {
	return c.postString(imageAdminPath("/description", admin), req)
}

func (c *Client) UpdateImageType(req ImageTypeRequest, admin bool) (string, error) {
	return c.postString(imageAdminPath("/type", admin), req)
}

func (c *Client) UpdateImageTags(req ImageTagsRequest, admin bool) (string, error) {
	return c.postString(imageAdminPath("/tags", admin), req)
}

func (c *Client) UpdateImageArch(req ImageArchRequest, admin bool) (string, error) {
	return c.postString(imageAdminPath("/arch", admin), req)
}

func (c *Client) TogglePublic(id uint) (string, error) {
	return c.postString(fmt.Sprintf("%s/change/%d", AdminImagesPrefix, id), nil)
}

func (c *Client) ShareImage(req ImageShareRequest) (string, error) {
	return c.postString(ImagesPrefix+"/share", req)
}

func (c *Client) CancelShareImage(req ImageCancelShareRequest) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetBody(&req).SetSuccessResult(&result).SetErrorResult(&result).Delete(ImagesPrefix + "/share")
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) GetImageGrants(imageID uint) (*ImageGrantResponse, error) {
	var result Response[ImageGrantResponse]
	if err := c.get(ImagesPrefix+"/share", map[string]string{"imageID": fmt.Sprintf("%d", imageID)}, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) ListUngrantedUsers(imageID uint, name string) (*ImageGrantResponse, error) {
	var result Response[ImageGrantResponse]
	if err := c.get(ImagesPrefix+"/user", map[string]string{"imageID": fmt.Sprintf("%d", imageID), "name": name}, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) ListUngrantedAccounts(imageID uint) (*ImageGrantResponse, error) {
	var result Response[ImageGrantResponse]
	if err := c.get(ImagesPrefix+"/account", map[string]string{"imageID": fmt.Sprintf("%d", imageID)}, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) CheckImageLinks(pairs []ImageInfoLinkPair) (*CheckLinkValidityResponse, error) {
	var result Response[CheckLinkValidityResponse]
	if err := c.post(ImagesPrefix+"/valid", CheckLinkValidityRequest{LinkPairs: pairs}, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) GetHarbor() (*HarborResponse, error) {
	var result Response[HarborResponse]
	if err := c.get(ImagesPrefix+"/harbor", nil, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) GetCredential() (*ProjectCredentialResponse, error) {
	var result Response[ProjectCredentialResponse]
	if err := c.post(ImagesPrefix+"/credential", nil, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) GetQuota() (*ProjectDetailResponse, error) {
	var result Response[ProjectDetailResponse]
	if err := c.get(ImagesPrefix+"/quota", nil, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) UpdateQuota(size int64) (string, error) {
	return c.postString(ImagesPrefix+"/quota", map[string]int64{"size": size})
}

func (c *Client) ListCudaBaseImages() (*CudaBaseImagesResponse, error) {
	var result Response[CudaBaseImagesResponse]
	if err := c.get(ImagesPrefix+"/cudabaseimage", nil, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) AddCudaBaseImage(req CudaBaseImageRequest) (string, error) {
	return c.postString(ImagesPrefix+"/cudabaseimage", req)
}

func (c *Client) DeleteCudaBaseImage(id uint) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Delete(fmt.Sprintf("%s/cudabaseimage/%d", ImagesPrefix, id))
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func imageAdminPath(suffix string, admin bool) string {
	if admin {
		return AdminImagesPrefix + suffix
	}
	return ImagesPrefix + suffix
}

func (c *Client) get(path string, params map[string]string, result interface{}) error {
	req := c.httpClient.R().SetSuccessResult(result).SetErrorResult(result)
	for k, v := range params {
		if v != "" {
			req.SetQueryParam(k, v)
		}
	}
	resp, err := req.Get(path)
	if err != nil {
		return &NetworkError{Cause: err}
	}
	code, msg := responseCodeMsg(result)
	return errorFromResponse(resp, code, msg)
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
	resp, err := c.httpClient.R().SetBody(body).SetSuccessResult(result).SetErrorResult(result).Post(path)
	if err != nil {
		return &NetworkError{Cause: err}
	}
	code, msg := responseCodeMsg(result)
	return errorFromResponse(resp, code, msg)
}

func (c *Client) postString(path string, body interface{}) (string, error) {
	var result Response[string]
	if err := c.post(path, body, &result); err != nil {
		return "", err
	}
	return result.Data, nil
}

func responseCodeMsg(result interface{}) (int, string) {
	switch r := result.(type) {
	case *Response[ListKanikoResponse]:
		return r.Code, r.Message
	case *Response[KanikoDetailResponse]:
		return r.Code, r.Message
	case *Response[KanikoPodResponse]:
		return r.Code, r.Message
	case *Response[ListImageResponse]:
		return r.Code, r.Message
	case *Response[AvailableImageListResp]:
		return r.Code, r.Message
	case *Response[string]:
		return r.Code, r.Message
	case *Response[ImageGrantResponse]:
		return r.Code, r.Message
	case *Response[CheckLinkValidityResponse]:
		return r.Code, r.Message
	case *Response[HarborResponse]:
		return r.Code, r.Message
	case *Response[ProjectCredentialResponse]:
		return r.Code, r.Message
	case *Response[ProjectDetailResponse]:
		return r.Code, r.Message
	case *Response[CudaBaseImagesResponse]:
		return r.Code, r.Message
	default:
		return 0, ""
	}
}
