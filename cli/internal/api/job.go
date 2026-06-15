package api

import (
	"fmt"
	"time"
)

type JobClient interface {
	ListJobs(opts JobListOptions) ([]JobInfo, error)
	GetJob(name string) (*JobDetail, error)
	GetJobPods(name string) ([]PodDetail, error)
	GetJobEvents(name string) ([]interface{}, error)
	GetJobYAML(name string) (string, error)
}

type JobListOptions struct {
	All      bool
	Username string
	Days     int
}

type UserInfo struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

type JobInfo struct {
	Name                    string       `json:"name"`
	JobName                 string       `json:"jobName"`
	Owner                   string       `json:"owner"`
	UserInfo                UserInfo     `json:"userInfo"`
	JobType                 string       `json:"jobType"`
	ScheduleType            int          `json:"scheduleType"`
	WaitingToleranceSeconds *int64       `json:"waitingToleranceSeconds,omitempty"`
	Queue                   string       `json:"queue"`
	Status                  string       `json:"status"`
	CreatedAt               time.Time    `json:"createdAt"`
	StartedAt               time.Time    `json:"startedAt"`
	CompletedAt             time.Time    `json:"completedAt"`
	Nodes                   []string     `json:"nodes"`
	Resources               ResourceList `json:"resources"`
	Locked                  bool         `json:"locked"`
	PermanentLocked         bool         `json:"permanentLocked"`
	LockedTimestamp         time.Time    `json:"lockedTimestamp"`
}

type JobDetail struct {
	Name           string       `json:"name"`
	Namespace      string       `json:"namespace"`
	Username       string       `json:"username"`
	Nickname       string       `json:"nickname"`
	UserInfo       UserInfo     `json:"userInfo"`
	JobName        string       `json:"jobName"`
	JobType        string       `json:"jobType"`
	ScheduleType   int          `json:"scheduleType"`
	Retry          string       `json:"retry"`
	Queue          string       `json:"queue"`
	Status         string       `json:"status"`
	Resources      ResourceList `json:"resources"`
	CreatedAt      time.Time    `json:"createdAt"`
	StartedAt      time.Time    `json:"startedAt"`
	CompletedAt    time.Time    `json:"completedAt"`
	ProfileData    interface{}  `json:"profileData,omitempty"`
	ScheduleData   interface{}  `json:"scheduleData,omitempty"`
	Events         interface{}  `json:"events,omitempty"`
	TerminatedInfo interface{}  `json:"terminatedStates,omitempty"`
}

type PodDetail struct {
	Name      string       `json:"name"`
	Namespace string       `json:"namespace"`
	NodeName  string       `json:"nodename"`
	IP        string       `json:"ip"`
	Port      string       `json:"port"`
	Resource  ResourceList `json:"resource,omitempty"`
	Phase     string       `json:"phase"`
}

func (c *Client) ListJobs(opts JobListOptions) ([]JobInfo, error) {
	path := VCJobListPath
	var result Response[[]JobInfo]
	req := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result)
	switch {
	case opts.Username != "":
		path = fmt.Sprintf("%s/user/%s", VCJobListPath, opts.Username)
		if opts.Days != 0 {
			req.SetQueryParam("days", fmt.Sprintf("%d", opts.Days))
		}
	case opts.All:
		path = VCJobListPath + "/all"
		if opts.Days != 0 {
			req.SetQueryParam("days", fmt.Sprintf("%d", opts.Days))
		}
	}
	resp, err := req.Get(path)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) GetJob(name string) (*JobDetail, error) {
	var result Response[JobDetail]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(VCJobListPath + "/" + name + "/detail")
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) GetJobPods(name string) ([]PodDetail, error) {
	var result Response[[]PodDetail]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(VCJobListPath + "/" + name + "/pods")
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) GetJobEvents(name string) ([]interface{}, error) {
	var result Response[[]interface{}]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(VCJobListPath + "/" + name + "/event")
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) GetJobYAML(name string) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(VCJobListPath + "/" + name + "/yaml")
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}
