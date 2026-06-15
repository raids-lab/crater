package api

type NodeClient interface {
	ListNodes() ([]NodeBrief, error)
	GetNode(name string) (*NodeDetail, error)
}

type NodeTaint struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Effect    string `json:"effect"`
	Reason    string `json:"reason,omitempty"`
	TimeAdded string `json:"timeAdded,omitempty"`
}

type ResourceList map[string]string

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
