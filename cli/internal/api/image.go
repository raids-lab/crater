package api

import "time"

type ImageClient interface {
	ListImages(available bool) ([]ImageInfo, error)
}

type ImageListResp struct {
	ImageList []ImageInfo `json:"imageList"`
}

type AvailableImageListResp struct {
	Images []ImageInfo `json:"images"`
}

type ImageInfo struct {
	ID               uint      `json:"ID"`
	ImageLink        string    `json:"imageLink"`
	Description      *string   `json:"description"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	IsPublic         bool      `json:"isPublic"`
	TaskType         string    `json:"taskType"`
	UserInfo         UserInfo  `json:"userInfo"`
	Tags             []string  `json:"tags"`
	ImageBuildSource int       `json:"imageBuildSource"`
	ImagePackName    *string   `json:"imagepackName"`
	ImageShareStatus string    `json:"imageShareStatus"`
	Archs            []string  `json:"archs"`
}

func (c *Client) ListImages(available bool) ([]ImageInfo, error) {
	if available {
		var result Response[AvailableImageListResp]
		resp, err := c.httpClient.R().
			SetSuccessResult(&result).
			SetErrorResult(&result).
			Get(ImageAvailablePath)
		if err != nil {
			return nil, &NetworkError{Cause: err}
		}
		if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
			return nil, err
		}
		return result.Data.Images, nil
	}

	var result Response[ImageListResp]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(ImageListPath)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data.ImageList, nil
}
