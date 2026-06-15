package api

import (
	"fmt"
	"time"
)

// ModelDownloadClient handles model and dataset download task APIs.
type ModelDownloadClient interface {
	CreateDownload(req CreateModelDownloadReq) (*ModelDownloadResp, string, error)
	ListDownloads(category string) ([]ModelDownloadResp, error)
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
	ID              uint      `json:"id"`
	Name            string    `json:"name"`
	Source          string    `json:"source"`
	Category        string    `json:"category"`
	Revision        string    `json:"revision"`
	Path            string    `json:"path"`
	SizeBytes       int64     `json:"sizeBytes"`
	DownloadedBytes int64     `json:"downloadedBytes"`
	DownloadSpeed   string    `json:"downloadSpeed"`
	Status          string    `json:"status"`
	Message         string    `json:"message"`
	JobName         string    `json:"jobName"`
	CreatorID       uint      `json:"creatorId"`
	ReferenceCount  int       `json:"referenceCount"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
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

// ListDownloads lists model or dataset download tasks for the active user.
func (c *Client) ListDownloads(category string) ([]ModelDownloadResp, error) {
	var result Response[[]ModelDownloadResp]
	req := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result)
	if category != "" {
		req.SetQueryParam("category", category)
	}
	resp, err := req.Get(ModelDownloadListPath)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
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

func errorFromResponse(resp interface {
	GetStatusCode() int
	IsSuccessState() bool
}, craterCode int, msg string) error {
	status := resp.GetStatusCode()
	if !resp.IsSuccessState() {
		return &RequestError{
			HTTPStatus: status,
			CraterCode: craterCode,
			Msg:        msg,
		}
	}
	if craterCode != 0 {
		return &RequestError{
			HTTPStatus: status,
			CraterCode: craterCode,
			Msg:        msg,
		}
	}
	return nil
}
