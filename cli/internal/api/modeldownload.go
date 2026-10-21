package api

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ModelDownloadClient handles model and dataset download task APIs.
type ModelDownloadClient interface {
	CreateDownload(req CreateModelDownloadReq) (*ModelDownloadResp, string, error)
	ListDownloads(category string) ([]ModelDownloadResp, error)
	ListDownloadPage(opts ModelDownloadListOptions) (ModelDownloadPage, error)
	ListAdminDownloads() ([]ModelDownloadResp, error)
	GetDownload(id uint) (*ModelDownloadResp, error)
	GetDownloadLogs(id uint) (string, error)
	RetryDownload(id uint) (*ModelDownloadResp, error)
	PauseDownload(id uint) (*ModelDownloadResp, error)
	ResumeDownload(id uint) (*ModelDownloadResp, error)
	DeleteDownload(id uint) (string, error)
}

// NewModelDownloadClient returns the default model download client.
func NewModelDownloadClient(baseURL, token string) ModelDownloadClient {
	return NewClient(baseURL).SetToken(token)
}

// CreateModelDownloadReq is the request body for creating a model or dataset download task.
type CreateModelDownloadReq struct {
	Name     string `json:"name"`
	Revision string `json:"revision,omitempty"`
	Source   string `json:"source,omitempty"`
	Category string `json:"category"`
	Token    string `json:"token,omitempty"`
}

// ModelDownloadResp mirrors the platform model download task summary.
type ModelDownloadResp struct {
	ID              uint       `json:"id"`
	Name            string     `json:"name"`
	Source          string     `json:"source"`
	Category        string     `json:"category"`
	Revision        string     `json:"revision"`
	Path            string     `json:"path"`
	SizeBytes       int64      `json:"sizeBytes"`
	DownloadedBytes int64      `json:"downloadedBytes"`
	DownloadSpeed   string     `json:"downloadSpeed"`
	Status          string     `json:"status"`
	Message         string     `json:"message"`
	JobName         string     `json:"jobName"`
	CreatorID       uint       `json:"creatorId"`
	ReferenceCount  int        `json:"referenceCount"`
	RequesterCount  int        `json:"requesterCount"`
	Requesters      []UserInfo `json:"requesters"`
	Relation        string     `json:"relation"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	SourceUpdatedAt *time.Time `json:"sourceUpdatedAt"`
	UserInfo        UserInfo   `json:"userInfo"`
	CanManage       bool       `json:"canManage"`
	CanDelete       bool       `json:"canDelete"`
	CanViewLogs     bool       `json:"canViewLogs"`
	SourceURL       string     `json:"sourceUrl"`
	DisplayName     string     `json:"displayName"`
	License         string     `json:"license"`
	Task            string     `json:"task"`
	Library         string     `json:"library"`
	ModelType       string     `json:"modelType"`
	ParameterCount  int64      `json:"parameterCount"`
	SourceCreatedAt *time.Time `json:"sourceCreatedAt"`
}

// ModelDownloadListOptions contains server-side pagination and filters for
// model and dataset download tasks.
type ModelDownloadListOptions struct {
	ListOptions
	Category string
	Status   string
	Search   string
}

// ModelDownloadPage is one page of download tasks and the backend's
// category-scoped status summary.
type ModelDownloadPage struct {
	Page[ModelDownloadResp]
	Summary map[string]int64 `json:"summary"`
}

// CreateDownload submits a model or dataset download task.
func (c *Client) CreateDownload(req CreateModelDownloadReq) (*ModelDownloadResp, string, error) {
	var result Response[ModelDownloadResp]

	resp, err := c.httpClient.R().
		SetBody(&req).
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Post(ModelDownloadCreatePath)
	if err != nil {
		return nil, "", &NetworkError{Cause: err}
	}

	status := resp.GetStatusCode()
	if !resp.IsSuccessState() {
		return nil, "", &RequestError{
			HTTPStatus: status,
			CraterCode: result.Code,
			Msg:        result.Message,
		}
	}

	if result.Code != 0 {
		return nil, "", &RequestError{
			HTTPStatus: status,
			CraterCode: result.Code,
			Msg:        result.Message,
		}
	}

	return &result.Data, result.Message, nil
}

func (c *Client) ListDownloads(category string) ([]ModelDownloadResp, error) {
	return fetchAllListPagesWithParams[ModelDownloadResp](c, ModelDownloadListPath, map[string]string{"category": category, "sort": "-updatedAt"})
}

func (c *Client) ListAdminDownloads() ([]ModelDownloadResp, error) {
	return fetchAllListPages[ModelDownloadResp](c, AdminModelDLPfx+"/models/downloads", "-updatedAt")
}

// ListDownloadPage lists one server-side page of model or dataset download
// tasks for the active user.
func (c *Client) ListDownloadPage(opts ModelDownloadListOptions) (ModelDownloadPage, error) {
	opts.ListOptions = opts.ListOptions.Normalize()
	var result Response[struct {
		Items   []ModelDownloadResp `json:"items"`
		Total   int64               `json:"total"`
		Summary map[string]int64    `json:"summary"`
	}]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		SetQueryParamsFromValues(modelDownloadListValues(opts)).
		Get(ModelDownloadListPath)
	if err != nil {
		return ModelDownloadPage{}, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return ModelDownloadPage{}, err
	}
	if result.Data.Items == nil {
		result.Data.Items = []ModelDownloadResp{}
	}
	if result.Data.Summary == nil {
		result.Data.Summary = map[string]int64{}
	}
	return ModelDownloadPage{
		Page: Page[ModelDownloadResp]{
			Items:    result.Data.Items,
			Total:    result.Data.Total,
			Page:     opts.Page,
			PageSize: opts.PageSize,
		},
		Summary: result.Data.Summary,
	}, nil
}

func modelDownloadListValues(opts ModelDownloadListOptions) url.Values {
	values := url.Values{
		"page":      {strconv.Itoa(opts.Page)},
		"page_size": {strconv.Itoa(opts.PageSize)},
	}
	if opts.Category != "" {
		values.Set("category", opts.Category)
	}
	if opts.Status != "" {
		values.Set("status", opts.Status)
	}
	if opts.Search != "" {
		values.Set("search", opts.Search)
	}
	return values
}

// GetDownload returns one download task by id.
func (c *Client) GetDownload(id uint) (*ModelDownloadResp, error) {
	var result Response[ModelDownloadResp]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(downloadItemPath(id))
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// GetDownloadLogs returns the current logs for one download task.
func (c *Client) GetDownloadLogs(id uint) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(downloadItemPath(id) + "/logs")
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

// RetryDownload retries a failed download task.
func (c *Client) RetryDownload(id uint) (*ModelDownloadResp, error) {
	return c.postDownloadAction(id, "retry")
}

// PauseDownload pauses a downloading task.
func (c *Client) PauseDownload(id uint) (*ModelDownloadResp, error) {
	return c.postDownloadAction(id, "pause")
}

// ResumeDownload resumes a paused download task.
func (c *Client) ResumeDownload(id uint) (*ModelDownloadResp, error) {
	return c.postDownloadAction(id, "resume")
}

// DeleteDownload removes the active user's association with a download task.
func (c *Client) DeleteDownload(id uint) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Delete(downloadItemPath(id))
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) postDownloadAction(id uint, action string) (*ModelDownloadResp, error) {
	var result Response[ModelDownloadResp]
	resp, err := c.httpClient.R().
		SetBody(map[string]interface{}{}).
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Post(downloadItemPath(id) + "/" + action)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func downloadItemPath(id uint) string {
	return fmt.Sprintf("%s/%d", ModelDownloadListPath, id)
}
