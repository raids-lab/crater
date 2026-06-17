package api

import (
	"fmt"
	"net/url"
	"time"
)

type JobClient interface {
	ListJobs(opts JobListOptions) ([]JobInfo, error)
	GetJob(name string) (*JobDetail, error)
	GetJobPods(name string) ([]PodDetail, error)
	GetJobEvents(name string) ([]map[string]interface{}, error)
	GetJobYAML(name string) (string, error)
	GetJobTemplate(name string) (string, error)
	GetJupyterToken(name string) (*JobToken, error)
	GetWebIDESecret(name string) (*JobToken, error)
	OpenJobSSH(name string) (*SSHInfo, error)
	SnapshotJob(name string) (string, error)
	ToggleJobAlert(name string) (string, error)
	DeleteJob(name string) (string, error)
	AdminDeleteJob(name string) (string, error)
	CreateJupyterJob(req CreateInteractiveJobRequest) (map[string]interface{}, error)
	CreateWebIDEJob(req CreateInteractiveJobRequest) (map[string]interface{}, error)
	CreateTrainingJob(req CreateTrainingJobRequest) (map[string]interface{}, error)
	CreateTensorflowJob(req CreateDistributedJobRequest) (map[string]interface{}, error)
	CreatePytorchJob(req CreateDistributedJobRequest) (map[string]interface{}, error)
	ToggleJobKeep(name string) (string, error)
	LockJob(req LockJobRequest) (string, error)
	UnlockJob(name string) (string, error)
	CleanWaitingJupyter(waitMinutes int) (*CleanupResult, error)
	CleanWaitingCustom(waitMinutes int) (*CleanupResult, error)
	CleanLongRunning(req CleanLongTimeRequest) (*CleanupResult, error)
	CleanLowGPUUsage(req CleanLowGPUUsageRequest) (*CleanupResult, error)
}

type JobListOptions struct {
	All      bool
	Admin    bool
	Username string
	Days     int
}

type UserInfo struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

type ImageBaseInfo struct {
	ImageLink string   `json:"imageLink"`
	Archs     []string `json:"archs,omitempty"`
}

type VolumeMount struct {
	SubPath   string `json:"subPath"`
	MountPath string `json:"mountPath"`
}

type DatasetMount struct {
	DatasetID uint   `json:"datasetID"`
	MountPath string `json:"mountPath"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type NodeSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

type Forward struct {
	Name string `json:"name"`
	Port int    `json:"port"`
	Type string `json:"type,omitempty"`
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
	Name                    string                   `json:"name"`
	Namespace               string                   `json:"namespace"`
	Username                string                   `json:"username"`
	Nickname                string                   `json:"nickname"`
	UserInfo                UserInfo                 `json:"userInfo"`
	JobName                 string                   `json:"jobName"`
	JobType                 string                   `json:"jobType"`
	ScheduleType            int                      `json:"scheduleType"`
	WaitingToleranceSeconds *int64                   `json:"waitingToleranceSeconds,omitempty"`
	Queue                   string                   `json:"queue"`
	Resources               ResourceList             `json:"resources"`
	Status                  string                   `json:"status"`
	Retry                   string                   `json:"retry"`
	ProfileData             map[string]interface{}   `json:"profileData,omitempty"`
	ScheduleData            map[string]interface{}   `json:"scheduleData,omitempty"`
	Events                  []map[string]interface{} `json:"events,omitempty"`
	TerminatedStates        []map[string]interface{} `json:"terminatedStates,omitempty"`
	TerminatedInfo          interface{}              `json:"terminatedStates,omitempty"`
	CreatedAt               time.Time                `json:"createdAt"`
	StartedAt               time.Time                `json:"startedAt"`
	CompletedAt             time.Time                `json:"completedAt"`
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

type JobToken struct {
	BaseURL   string `json:"baseURL"`
	FullURL   string `json:"fullURL"`
	Token     string `json:"token"`
	PodName   string `json:"podName"`
	Namespace string `json:"namespace"`
}

type SSHInfo struct {
	IP   string `json:"ip"`
	Port string `json:"port"`
}

type JobCommonRequest struct {
	Name              string                    `json:"name"`
	VolumeMounts      []VolumeMount             `json:"volumeMounts,omitempty"`
	DatasetMounts     []DatasetMount            `json:"datasetMounts,omitempty"`
	Envs              []EnvVar                  `json:"envs,omitempty"`
	Selectors         []NodeSelectorRequirement `json:"selectors,omitempty"`
	Template          string                    `json:"template,omitempty"`
	AlertEnabled      bool                      `json:"alertEnabled"`
	CpuPinningEnabled bool                      `json:"cpuPinningEnabled,omitempty"`
	Forwards          []Forward                 `json:"forwards,omitempty"`
	ScheduleType      *int                      `json:"scheduleType,omitempty"`
}

type CreateInteractiveJobRequest struct {
	JobCommonRequest
	Resource ResourceList  `json:"resource"`
	Image    ImageBaseInfo `json:"image"`
}

type CreateTrainingJobRequest struct {
	CreateInteractiveJobRequest
	Shell      *string `json:"shell,omitempty"`
	Command    *string `json:"command,omitempty"`
	WorkingDir string  `json:"workingDir"`
}

type PortRequest struct {
	Name string `json:"name"`
	Port int32  `json:"port"`
}

type TaskRequest struct {
	Name       string        `json:"name"`
	Replicas   int32         `json:"replicas"`
	Resource   ResourceList  `json:"resource"`
	Image      ImageBaseInfo `json:"image"`
	Shell      *string       `json:"shell,omitempty"`
	Command    *string       `json:"command,omitempty"`
	WorkingDir *string       `json:"workingDir,omitempty"`
	Ports      []PortRequest `json:"ports,omitempty"`
}

type CreateDistributedJobRequest struct {
	JobCommonRequest
	Tasks []TaskRequest `json:"tasks"`
}

type LockJobRequest struct {
	Name        string `json:"name"`
	IsPermanent bool   `json:"isPermanent"`
	Days        int    `json:"days"`
	Hours       int    `json:"hours"`
	Minutes     int    `json:"minutes"`
}

type CleanupResult struct {
	Reminded []string `json:"reminded"`
	Deleted  []string `json:"deleted"`
}

type CleanLongTimeRequest struct {
	BatchDays       *int `json:"batchDays,omitempty"`
	InteractiveDays *int `json:"interactiveDays,omitempty"`
}

type CleanLowGPUUsageRequest struct {
	TimeRange int  `json:"timeRange"`
	WaitTime  *int `json:"waitTime,omitempty"`
	Util      *int `json:"util,omitempty"`
}

func (c *Client) ListJobs(opts JobListOptions) ([]JobInfo, error) {
	prefix := VCJobsPrefix
	if opts.Admin {
		prefix = AdminVCJobsPrefix
	}
	path := prefix
	query := url.Values{}
	switch {
	case opts.Username != "":
		path += "/user/" + url.PathEscape(opts.Username)
		if opts.Days != 0 {
			query.Set("days", fmt.Sprintf("%d", opts.Days))
		}
	case opts.All && !opts.Admin:
		path += "/all"
		if opts.Days != 0 {
			query.Set("days", fmt.Sprintf("%d", opts.Days))
		}
	case opts.Admin:
		if opts.Days != 0 {
			query.Set("days", fmt.Sprintf("%d", opts.Days))
		}
	}
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result Response[[]JobInfo]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Get(path)
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
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Get(VCJobsPrefix + "/" + url.PathEscape(name) + "/detail")
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
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Get(VCJobsPrefix + "/" + url.PathEscape(name) + "/pods")
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) GetJobEvents(name string) ([]map[string]interface{}, error) {
	var result Response[[]map[string]interface{}]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Get(VCJobsPrefix + "/" + url.PathEscape(name) + "/event")
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
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Get(VCJobsPrefix + "/" + url.PathEscape(name) + "/yaml")
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) GetJobTemplate(name string) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Get(VCJobsPrefix + "/" + url.PathEscape(name) + "/template")
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) GetJupyterToken(name string) (*JobToken, error) {
	return c.getJobToken(name, "token")
}

func (c *Client) GetWebIDESecret(name string) (*JobToken, error) {
	return c.getJobToken(name, "secret")
}

func (c *Client) getJobToken(name string, suffix string) (*JobToken, error) {
	var result Response[JobToken]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Get(VCJobsPrefix + "/" + url.PathEscape(name) + "/" + suffix)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) OpenJobSSH(name string) (*SSHInfo, error) {
	var result Response[SSHInfo]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Post(VCJobsPrefix + "/" + url.PathEscape(name) + "/ssh")
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) SnapshotJob(name string) (string, error) {
	return c.messagePost(VCJobsPrefix+"/"+url.PathEscape(name)+"/snapshot", nil)
}

func (c *Client) ToggleJobAlert(name string) (string, error) {
	return c.messagePut(VCJobsPrefix+"/"+url.PathEscape(name)+"/alert", nil)
}

func (c *Client) DeleteJob(name string) (string, error) {
	return c.messageDelete(VCJobsPrefix + "/" + url.PathEscape(name))
}

func (c *Client) AdminDeleteJob(name string) (string, error) {
	return c.messageDelete(AdminVCJobsPrefix + "/" + url.PathEscape(name))
}

func (c *Client) CreateJupyterJob(req CreateInteractiveJobRequest) (map[string]interface{}, error) {
	return c.createJob("jupyter", req)
}

func (c *Client) CreateWebIDEJob(req CreateInteractiveJobRequest) (map[string]interface{}, error) {
	return c.createJob("webide", req)
}

func (c *Client) CreateTrainingJob(req CreateTrainingJobRequest) (map[string]interface{}, error) {
	return c.createJob("training", req)
}

func (c *Client) CreateTensorflowJob(req CreateDistributedJobRequest) (map[string]interface{}, error) {
	return c.createJob("tensorflow", req)
}

func (c *Client) CreatePytorchJob(req CreateDistributedJobRequest) (map[string]interface{}, error) {
	return c.createJob("pytorch", req)
}

func (c *Client) createJob(kind string, body interface{}) (map[string]interface{}, error) {
	var result Response[map[string]interface{}]
	resp, err := c.httpClient.R().
		SetBody(body).
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Post(VCJobsPrefix + "/" + kind)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) ToggleJobKeep(name string) (string, error) {
	return c.messagePut(AdminOperationsPfx+"/keep/"+url.PathEscape(name), nil)
}

func (c *Client) LockJob(req LockJobRequest) (string, error) {
	return c.messagePut(AdminOperationsPfx+"/add/locktime", req)
}

func (c *Client) UnlockJob(name string) (string, error) {
	return c.messagePut(AdminOperationsPfx+"/clear/locktime", map[string]string{"name": name})
}

func (c *Client) CleanWaitingJupyter(waitMinutes int) (*CleanupResult, error) {
	return c.cleanup(AdminOperationsPfx+"/clean/clean-waiting-jupyter-job", map[string]interface{}{
		"waitMinitues": waitMinutes,
		"jobTypes":     []string{"jupyter"},
	})
}

func (c *Client) CleanWaitingCustom(waitMinutes int) (*CleanupResult, error) {
	return c.cleanup(AdminOperationsPfx+"/clean/clean-waiting-custom-job", map[string]interface{}{
		"waitMinitues": waitMinutes,
		"jobTypes":     []string{"custom"},
	})
}

func (c *Client) CleanLongRunning(req CleanLongTimeRequest) (*CleanupResult, error) {
	return c.cleanup(AdminOperationsPfx+"/clean/clean-long-running-job", req)
}

func (c *Client) CleanLowGPUUsage(req CleanLowGPUUsageRequest) (*CleanupResult, error) {
	return c.cleanup(AdminOperationsPfx+"/clean/clean-low-gpu-usage-job", req)
}

func (c *Client) cleanup(path string, body interface{}) (*CleanupResult, error) {
	var result Response[CleanupResult]
	resp, err := c.httpClient.R().
		SetBody(body).
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Post(path)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) messagePost(path string, body interface{}) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetBody(body).SetSuccessResult(&result).SetErrorResult(&result).Post(path)
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) messagePut(path string, body interface{}) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetBody(body).SetSuccessResult(&result).SetErrorResult(&result).Put(path)
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}

func (c *Client) messageDelete(path string) (string, error) {
	var result Response[string]
	resp, err := c.httpClient.R().SetSuccessResult(&result).SetErrorResult(&result).Delete(path)
	if err != nil {
		return "", &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return "", err
	}
	return result.Data, nil
}
