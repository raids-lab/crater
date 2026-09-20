package api

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

type PodContainerInfo struct {
	Name            string                 `json:"name"`
	Image           string                 `json:"image"`
	State           map[string]interface{} `json:"state,omitempty"`
	Resources       ResourceList           `json:"resources,omitempty"`
	RestartCount    int32                  `json:"restartCount"`
	IsInitContainer bool                   `json:"isInitContainer"`
	Node            string                 `json:"node"`
}

type PodLogOptions struct {
	TailLines  int64
	Timestamps bool
	Previous   bool
}

type PodLogProtocolError struct {
	Cause error
}

func (e *PodLogProtocolError) Error() string {
	return fmt.Sprintf("invalid pod log protocol: %v", e.Cause)
}

func (e *PodLogProtocolError) Unwrap() error {
	return e.Cause
}

type PodLogWriteError struct {
	Cause error
}

func (e *PodLogWriteError) Error() string {
	return fmt.Sprintf("write pod logs: %v", e.Cause)
}

func (e *PodLogWriteError) Unwrap() error {
	return e.Cause
}

type podContainersResponse struct {
	Containers []PodContainerInfo `json:"containers"`
}

func podContainerPath(namespace, pod, container, suffix string) string {
	path := NamespacesPrefix + "/" + url.PathEscape(namespace) +
		"/pods/" + url.PathEscape(pod) +
		"/containers"
	if container != "" {
		path += "/" + url.PathEscape(container)
	}
	if suffix != "" {
		path += "/" + suffix
	}
	return path
}

func podLogQuery(options PodLogOptions) map[string]string {
	params := map[string]string{
		"timestamps": strconv.FormatBool(options.Timestamps),
		"previous":   strconv.FormatBool(options.Previous),
	}
	if options.TailLines > 0 {
		params["tailLines"] = strconv.FormatInt(options.TailLines, 10)
	}
	return params
}

func (c *Client) GetPodContainers(namespace, pod string) ([]PodContainerInfo, error) {
	var result Response[podContainersResponse]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(podContainerPath(namespace, pod, "", ""))
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	if result.Data.Containers == nil {
		return []PodContainerInfo{}, nil
	}
	return result.Data.Containers, nil
}

// GetPodLogs returns decoded UTF-8 log bytes. The backend wraps Kubernetes log
// bytes in JSON, so encoding/json represents the data field as Base64.
func (c *Client) GetPodLogs(namespace, pod, container string, options PodLogOptions) ([]byte, error) {
	var result Response[string]
	request := c.httpClient.R()
	for key, value := range podLogQuery(options) {
		request.SetQueryParam(key, value)
	}
	resp, err := request.
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(podContainerPath(namespace, pod, container, "log"))
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	logs, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return nil, &PodLogProtocolError{Cause: fmt.Errorf("decode response: %w", err)}
	}
	return logs, nil
}

// StreamPodLogs decodes the backend's newline-delimited Base64 stream and
// writes the original log bytes to dst until the stream closes or ctx is
// cancelled.
func (c *Client) StreamPodLogs(
	ctx context.Context,
	dst io.Writer,
	namespace, pod, container string,
	options PodLogOptions,
) error {
	request := c.httpClient.R().
		SetContext(ctx).
		DisableAutoReadResponse()
	for key, value := range podLogQuery(options) {
		request.SetQueryParam(key, value)
	}
	resp, err := request.Get(podContainerPath(namespace, pod, container, "log/stream"))
	if err != nil {
		return &NetworkError{Cause: err}
	}
	if resp.Response == nil || resp.Body == nil {
		return &NetworkError{Cause: fmt.Errorf("empty streaming response")}
	}
	defer resp.Body.Close()

	if !resp.IsSuccessState() {
		var result Response[interface{}]
		if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
			return &RequestError{HTTPStatus: resp.GetStatusCode(), Msg: decodeErr.Error()}
		}
		return &RequestError{
			HTTPStatus: resp.GetStatusCode(),
			CraterCode: result.Code,
			Msg:        result.Message,
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		decoded, decodeErr := base64.StdEncoding.DecodeString(line)
		if decodeErr != nil {
			return &PodLogProtocolError{Cause: fmt.Errorf("decode stream: %w", decodeErr)}
		}
		if _, writeErr := dst.Write(decoded); writeErr != nil {
			return &PodLogWriteError{Cause: writeErr}
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return &NetworkError{Cause: err}
	}
	return ctx.Err()
}
