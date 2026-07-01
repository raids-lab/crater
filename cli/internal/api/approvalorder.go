package api

import (
	"fmt"
	"time"
)

type ApprovalOrderClient interface {
	ListApprovalOrders(admin bool) ([]ApprovalOrder, error)
	GetApprovalOrder(id uint, admin bool) (*ApprovalOrder, error)
	ListApprovalOrdersByName(name string) ([]ApprovalOrder, error)
	CreateApprovalOrder(req ApprovalOrderRequest) (string, error)
	UpdateApprovalOrder(id uint, req ApprovalOrderRequest) (string, error)
	ReviewApprovalOrder(id uint, req ApprovalOrderReviewRequest) (string, error)
	DeleteApprovalOrder(id uint) (string, error)
	CheckApprovalOrders() (string, error)
	LockJob(req JobLockRequest) (string, error)
}

type ApprovalOrderContent struct {
	ApprovalOrderTypeID         uint   `json:"approvalorderTypeID"`
	ApprovalOrderReason         string `json:"approvalorderReason"`
	ApprovalOrderExtensionHours uint   `json:"approvalorderExtensionHours"`
}

type ApprovalUserInfo struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

type ApprovalOrder struct {
	ID          uint                 `json:"id"`
	Name        string               `json:"name"`
	Type        string               `json:"type"`
	Status      string               `json:"status"`
	Content     ApprovalOrderContent `json:"content"`
	ReviewNotes string               `json:"reviewNotes"`
	CreatedAt   time.Time            `json:"createdAt"`
	CreatorID   uint                 `json:"creatorID"`
	Creator     ApprovalUserInfo     `json:"creator"`
	ReviewerID  uint                 `json:"reviewerID"`
	Reviewer    ApprovalUserInfo     `json:"reviewer"`
}

type ApprovalOrderRequest struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	Status         string `json:"status,omitempty"`
	TypeID         uint   `json:"approvalorderTypeID"`
	Reason         string `json:"approvalOrderReason"`
	ExtensionHours uint   `json:"approvalOrderExtensionHours"`
}

type ApprovalOrderReviewRequest struct {
	Status      string `json:"status"`
	ReviewNotes string `json:"reviewNotes,omitempty"`
}

type JobLockRequest struct {
	Name        string `json:"name"`
	IsPermanent bool   `json:"isPermanent"`
	Days        int    `json:"days"`
	Hours       int    `json:"hours"`
	Minutes     int    `json:"minutes"`
}

func (c *Client) ListApprovalOrders(admin bool) ([]ApprovalOrder, error) {
	path := ApprovalOrderPrefix
	if admin {
		path = AdminApprovalPrefix
	}
	var result Response[[]ApprovalOrder]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Get(path)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) GetApprovalOrder(id uint, admin bool) (*ApprovalOrder, error) {
	prefix := ApprovalOrderPrefix
	if admin {
		prefix = AdminApprovalPrefix
	}
	var result Response[ApprovalOrder]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Get(fmt.Sprintf("%s/%d", prefix, id))
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) ListApprovalOrdersByName(name string) ([]ApprovalOrder, error) {
	var result Response[[]ApprovalOrder]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Get(ApprovalOrderPrefix + "/name/" + name)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) CreateApprovalOrder(req ApprovalOrderRequest) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetBody(&req).SetSuccessResult(&result).SetErrorResult(&result).Post(ApprovalOrderPrefix)
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) UpdateApprovalOrder(id uint, req ApprovalOrderRequest) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetBody(&req).SetSuccessResult(&result).SetErrorResult(&result).Put(fmt.Sprintf("%s/%d", ApprovalOrderPrefix, id))
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) ReviewApprovalOrder(id uint, req ApprovalOrderReviewRequest) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetBody(&req).SetSuccessResult(&result).SetErrorResult(&result).Put(fmt.Sprintf("%s/%d/review", AdminApprovalPrefix, id))
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) DeleteApprovalOrder(id uint) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Delete(fmt.Sprintf("%s/%d", ApprovalOrderPrefix, id))
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) CheckApprovalOrders() (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Put(AdminApprovalPrefix + "/check")
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) LockJob(req JobLockRequest) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetBody(&req).SetSuccessResult(&result).SetErrorResult(&result).Put(AdminOperationsPfx + "/add/locktime")
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}
