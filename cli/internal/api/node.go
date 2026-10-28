package api

import (
	"fmt"
	"net/url"
	"strconv"
)

type NodeClient interface {
	ListNodes() ([]NodeBrief, error)
	GetNode(name string) (*NodeDetail, error)
	GetNodePods(name string) ([]PodInfo, error)
	GetNodeGPU(name string) (*NodeGPUInfo, error)
}

type NodePodsSnapshotPage struct {
	Items           []PodInfo `json:"items"`
	NextCursor      *string   `json:"next_cursor"`
	HasMore         bool      `json:"has_more"`
	SnapshotID      string    `json:"snapshot_id"`
	ResourceVersion string    `json:"resource_version"`
	ObservedAt      string    `json:"observed_at"`
}

type NodeTaint struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Effect    string `json:"effect"`
	Reason    string `json:"reason,omitempty"`
	TimeAdded string `json:"timeAdded,omitempty"`
}

type ResourceList map[string]string

type OwnerReference struct {
	APIVersion         string `json:"apiVersion,omitempty"`
	Kind               string `json:"kind,omitempty"`
	Name               string `json:"name,omitempty"`
	UID                string `json:"uid,omitempty"`
	Controller         *bool  `json:"controller,omitempty"`
	BlockOwnerDeletion *bool  `json:"blockOwnerDeletion,omitempty"`
}

type PodInfo struct {
	Name             string           `json:"name"`
	Namespace        string           `json:"namespace"`
	OwnerReference   []OwnerReference `json:"ownerReference"`
	IP               string           `json:"ip"`
	CreateTime       string           `json:"createTime"`
	Status           string           `json:"status"`
	Resources        ResourceList     `json:"resources"`
	RequestResources ResourceList     `json:"requestResources,omitempty"`
	Locked           bool             `json:"locked"`
	PermanentLocked  bool             `json:"permanentLocked"`
	LockedTimestamp  string           `json:"lockedTimestamp,omitempty"`
	Type             string           `json:"type,omitempty"`
	UserName         string           `json:"userName,omitempty"`
	UserID           uint             `json:"userID,omitempty"`
	UserRealName     string           `json:"userRealName,omitempty"`
	AccountName      string           `json:"accountName,omitempty"`
	AccountID        uint             `json:"accountID,omitempty"`
	AccountRealName  string           `json:"accountRealName,omitempty"`
}

type NodeBrief struct {
	Name          string            `json:"name"`
	Role          string            `json:"role"`
	Arch          string            `json:"arch"`
	Status        string            `json:"status"`
	Vendor        string            `json:"vendor"`
	Taints        []NodeTaint       `json:"taints"`
	Capacity      ResourceList      `json:"capacity"`
	Allocatable   ResourceList      `json:"allocatable"`
	Used          ResourceList      `json:"used"`
	Workloads     int               `json:"workloads"`
	Annotations   map[string]string `json:"annotations"`
	KernelVersion string            `json:"kernelVersion,omitempty"`
	GPUDriver     string            `json:"gpuDriver,omitempty"`
	Address       string            `json:"address,omitempty"`
}

type NodeDetail struct {
	Name                    string       `json:"name"`
	Role                    string       `json:"role"`
	IsReady                 string       `json:"isReady"`
	Time                    string       `json:"time"`
	Address                 string       `json:"address"`
	OS                      string       `json:"os"`
	OSVersion               string       `json:"osVersion"`
	Arch                    string       `json:"arch"`
	KubeletVersion          string       `json:"kubeletVersion"`
	ContainerRuntimeVersion string       `json:"containerRuntimeVersion"`
	KernelVersion           string       `json:"kernelVersion,omitempty"`
	Capacity                ResourceList `json:"capacity,omitempty"`
	Allocatable             ResourceList `json:"allocatable,omitempty"`
	Used                    ResourceList `json:"used,omitempty"`
	GPUDriver               string       `json:"gpuDriver,omitempty"`
}

type GPUDeviceInfo struct {
	ResourceName   string `json:"resourceName"`
	Label          string `json:"label"`
	Product        string `json:"product"`
	VendorDomain   string `json:"vendorDomain"`
	Count          int    `json:"count"`
	Memory         string `json:"memory"`
	Arch           string `json:"arch"`
	Driver         string `json:"driver"`
	RuntimeVersion string `json:"runtimeVersion"`
}

type NodeGPUInfo struct {
	Name        string              `json:"name"`
	HaveGPU     bool                `json:"haveGPU"`
	GPUCount    int                 `json:"gpuCount"`
	GPUDevices  []GPUDeviceInfo     `json:"gpuDevices"`
	GPUUtil     map[string]float64  `json:"gpuUtil"`
	RelateJobs  map[string][]string `json:"relateJobs"`
	GPUMemory   string              `json:"gpuMemory"`
	GPUArch     string              `json:"gpuArch"`
	GPUDriver   string              `json:"gpuDriver"`
	CUDAVersion string              `json:"cudaVersion"`
	GPUProduct  string              `json:"gpuProduct"`
}

func (c *Client) ListNodes() ([]NodeBrief, error) {
	var result Response[[]NodeBrief]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(NodeListPath)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) GetNode(name string) (*NodeDetail, error) {
	var result Response[NodeDetail]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(NodeListPath + "/" + name)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) GetNodePods(name string) ([]PodInfo, error) {
	const pageLimit = 200
	items := make([]PodInfo, 0)
	cursor := ""
	seenCursors := make(map[string]struct{})
	snapshotID := ""
	for pageRequests := 0; pageRequests < MaxPageRequests; pageRequests++ {
		page, err := c.ListNodePodsSnapshot(name, pageLimit, cursor)
		if err != nil {
			return nil, err
		}
		if page.SnapshotID == "" {
			return nil, fmt.Errorf("get node pods: missing snapshot ID")
		}
		if pageRequests == 0 {
			snapshotID = page.SnapshotID
		} else if page.SnapshotID != snapshotID {
			return nil, fmt.Errorf("get node pods: snapshot ID changed between pages; refresh and retry")
		}
		nextCursor := ""
		if page.NextCursor != nil {
			nextCursor = *page.NextCursor
		}
		if page.HasMore != (nextCursor != "") {
			return nil, fmt.Errorf("get node pods: has_more does not match next_cursor")
		}
		items = append(items, page.Items...)
		if nextCursor == "" {
			return items, nil
		}
		if _, seen := seenCursors[nextCursor]; seen {
			return nil, fmt.Errorf("get node pods: repeated pagination cursor")
		}
		seenCursors[nextCursor] = struct{}{}
		cursor = nextCursor
	}
	return nil, fmt.Errorf("get node pods: exceeded maximum of %d page requests", MaxPageRequests)
}

func (c *Client) ListNodePodsSnapshot(name string, limit int, cursor string) (NodePodsSnapshotPage, error) {
	values := url.Values{"limit": {strconv.Itoa(limit)}}
	if cursor != "" {
		values.Set("cursor", cursor)
	}
	var result Response[NodePodsSnapshotPage]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		SetQueryParamsFromValues(values).
		Get(NodeListPath + "/" + url.PathEscape(name) + "/pods")
	if err != nil {
		return NodePodsSnapshotPage{}, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return NodePodsSnapshotPage{}, err
	}
	return result.Data, nil
}

func (c *Client) GetNodeGPU(name string) (*NodeGPUInfo, error) {
	var result Response[NodeGPUInfo]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(NodeListPath + "/" + name + "/gpu")
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return &result.Data, nil
}
