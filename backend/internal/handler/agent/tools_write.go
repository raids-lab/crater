package agent

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/util"
)

const (
	agentForwardTypeIngress  = 1
	agentForwardTypeNodePort = 2
)

func agentWriteErrorf(format string, args ...any) error {
	return bizerr.BadRequest.ParameterError.New(fmt.Sprintf(strings.ReplaceAll(format, "%w", "%v"), args...))
}

func normalizeForwardTypeValue(value any) (int, error) {
	switch typed := value.(type) {
	case nil:
		return agentForwardTypeIngress, nil
	case float64:
		switch int(typed) {
		case agentForwardTypeIngress, agentForwardTypeNodePort:
			return int(typed), nil
		}
	case int:
		switch typed {
		case agentForwardTypeIngress, agentForwardTypeNodePort:
			return typed, nil
		}
	case string:
		switch strings.TrimSpace(strings.ToLower(typed)) {
		case "", "ingress":
			return agentForwardTypeIngress, nil
		case "nodeport", "node_port":
			return agentForwardTypeNodePort, nil
		case "1":
			return agentForwardTypeIngress, nil
		case "2":
			return agentForwardTypeNodePort, nil
		}
	}
	return 0, agentWriteErrorf("forward type must be ingress or nodeport")
}

func parseForwardTextSpecs(raw string) ([]map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})
	result := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		parts := strings.Split(field, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return nil, agentWriteErrorf("invalid forward spec %q, expected name:port[:ingress|nodeport]", field)
		}
		port, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || port <= 0 {
			return nil, agentWriteErrorf("invalid forward port in %q", field)
		}
		forwardType := agentForwardTypeIngress
		if len(parts) == 3 {
			forwardType, err = normalizeForwardTypeValue(parts[2])
			if err != nil {
				return nil, err
			}
		}
		result = append(result, map[string]any{
			"type": forwardType,
			"name": strings.TrimSpace(parts[0]),
			"port": port,
		})
	}
	return result, nil
}

func parseForwardArgs(args map[string]any) ([]map[string]any, error) {
	raw, ok := args["forwards"]
	if ok && raw != nil {
		items, ok := raw.([]any)
		if !ok {
			return nil, agentWriteErrorf("forwards must be a list")
		}
		result := make([]map[string]any, 0, len(items))
		for _, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				return nil, agentWriteErrorf("forwards entries must be objects")
			}
			name := getToolArgString(entry, "name", "")
			if name == "" {
				return nil, agentWriteErrorf("forward name is required")
			}
			port := getToolArgInt(entry, "port", 0)
			if port <= 0 {
				return nil, agentWriteErrorf("forward %q requires a positive port", name)
			}
			forwardType, err := normalizeForwardTypeValue(entry["type"])
			if err != nil {
				return nil, err
			}
			result = append(result, map[string]any{
				"type": forwardType,
				"name": name,
				"port": port,
			})
		}
		return result, nil
	}
	return parseForwardTextSpecs(getToolArgString(args, "forwards_text", ""))
}

func lookupToolArgValue(args map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		value, ok := args[key]
		if !ok || value == nil {
			continue
		}
		return value, true
	}
	return nil, false
}

func getToolArgStringAny(args map[string]any, fallback string, keys ...string) string {
	for _, key := range keys {
		if value := getToolArgString(args, key, ""); value != "" {
			return value
		}
	}
	return fallback
}

func getToolArgIntAny(args map[string]any, fallback int, keys ...string) int {
	for _, key := range keys {
		if value, ok := lookupToolArgValue(args, key); ok {
			switch typed := value.(type) {
			case float64:
				return int(typed)
			case int:
				return typed
			case int32:
				return int(typed)
			case int64:
				return int(typed)
			case string:
				parsed, err := strconv.Atoi(strings.TrimSpace(typed))
				if err == nil {
					return parsed
				}
			}
		}
	}
	return fallback
}

func parseDistributedPorts(raw any) ([]map[string]any, error) {
	if raw == nil {
		return nil, nil
	}

	if text, ok := raw.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, nil
		}
		if strings.HasPrefix(text, "[") {
			var decoded []any
			if err := json.Unmarshal([]byte(text), &decoded); err != nil {
				return nil, agentWriteErrorf("ports_json must be a JSON array: %w", err)
			}
			raw = decoded
		} else {
			specs, err := parseForwardTextSpecs(text)
			if err != nil {
				return nil, err
			}
			ports := make([]map[string]any, 0, len(specs))
			for _, spec := range specs {
				ports = append(ports, map[string]any{
					"name": spec["name"],
					"port": spec["port"],
				})
			}
			return ports, nil
		}
	}

	items, ok := raw.([]any)
	if !ok {
		return nil, agentWriteErrorf("ports must be a list or text specs")
	}

	ports := make([]map[string]any, 0, len(items))
	for idx, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, agentWriteErrorf("port #%d must be an object", idx+1)
		}
		name := getToolArgStringAny(entry, "", "name")
		if name == "" {
			return nil, agentWriteErrorf("port #%d requires name", idx+1)
		}
		port := getToolArgIntAny(entry, 0, "port")
		if port <= 0 {
			return nil, agentWriteErrorf("port %q requires a positive port number", name)
		}
		ports = append(ports, map[string]any{
			"name": name,
			"port": port,
		})
	}
	return ports, nil
}

func normalizeDistributedTask(entry map[string]any) (map[string]any, error) {
	name := getToolArgStringAny(entry, "", "name")
	if name == "" {
		return nil, agentWriteErrorf("task name is required")
	}
	imageLink := getToolArgStringAny(entry, "", "image_link", "imageLink")
	if imageLink == "" {
		return nil, agentWriteErrorf("task %q requires image_link", name)
	}

	replicas := getToolArgIntAny(entry, 1, "replicas")
	if replicas <= 0 {
		return nil, agentWriteErrorf("task %q requires replicas > 0", name)
	}

	resourceMap := map[string]string{
		"cpu":    getToolArgStringAny(entry, "4", "cpu"),
		"memory": getToolArgStringAny(entry, "16Gi", "memory"),
	}
	if gpuCount := getToolArgIntAny(entry, 0, "gpu_count", "gpuCount"); gpuCount > 0 {
		gpuResourceName := normalizeGPUResourceName("", getToolArgStringAny(entry, "gpu", "gpu_model", "gpuModel"))
		resourceMap[string(gpuResourceName)] = strconv.Itoa(gpuCount)
	}

	ports, err := parseDistributedPorts(func() any {
		if value, ok := lookupToolArgValue(entry, "ports"); ok {
			return value
		}
		if value, ok := lookupToolArgValue(entry, "ports_text", "ports_json"); ok {
			return value
		}
		return nil
	}())
	if err != nil {
		return nil, err
	}

	task := map[string]any{
		"name":     name,
		"replicas": replicas,
		"resource": resourceMap,
		"image": map[string]any{
			"imageLink": imageLink,
			"archs":     []string{},
		},
		"ports": ports,
	}

	if command := getToolArgStringAny(entry, "", "command"); command != "" {
		task["command"] = command
		task["shell"] = getToolArgStringAny(entry, "bash", "shell")
	}
	if workingDir := getToolArgStringAny(entry, "", "working_dir", "workingDir"); workingDir != "" {
		task["workingDir"] = workingDir
	}

	return task, nil
}

func parseDistributedTasks(args map[string]any) ([]map[string]any, error) {
	if raw, ok := lookupToolArgValue(args, "tasks"); ok {
		items, ok := raw.([]any)
		if !ok {
			return nil, agentWriteErrorf("tasks must be a list")
		}
		result := make([]map[string]any, 0, len(items))
		for idx, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				return nil, agentWriteErrorf("task #%d must be an object", idx+1)
			}
			task, err := normalizeDistributedTask(entry)
			if err != nil {
				return nil, err
			}
			result = append(result, task)
		}
		return result, nil
	}

	rawJSON := getToolArgString(args, "tasks_json", "")
	if rawJSON == "" {
		return nil, nil
	}
	var items []any
	if err := json.Unmarshal([]byte(rawJSON), &items); err != nil {
		return nil, agentWriteErrorf("tasks_json must be a JSON array: %w", err)
	}
	result := make([]map[string]any, 0, len(items))
	for idx, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, agentWriteErrorf("task #%d must be an object", idx+1)
		}
		task, err := normalizeDistributedTask(entry)
		if err != nil {
			return nil, err
		}
		result = append(result, task)
	}
	return result, nil
}

func (mgr *AgentMgr) toolDeleteJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if mgr.jobSubmitter == nil {
		return nil, agentWriteErrorf("job submitter is not configured")
	}
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentWriteErrorf("invalid args: %w", err)
	}
	return mgr.jobSubmitter.DeleteJob(c.Request.Context(), token, args.JobName)
}

func (mgr *AgentMgr) toolStopJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if mgr.jobSubmitter == nil {
		return nil, agentWriteErrorf("job submitter is not configured")
	}
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentWriteErrorf("invalid args: %w", err)
	}
	return mgr.jobSubmitter.StopJob(c.Request.Context(), token, args.JobName)
}

func (mgr *AgentMgr) toolResubmitJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if mgr.jobSubmitter == nil {
		return nil, agentWriteErrorf("job submitter is not configured")
	}
	return mgr.jobSubmitter.ResubmitJob(c.Request.Context(), token, rawArgs)
}

func (mgr *AgentMgr) toolCreateJupyterJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	argsMap := parseToolArgsMap(rawArgs)
	var args struct {
		Name      string  `json:"name"`
		ImageLink string  `json:"image_link"`
		CPU       string  `json:"cpu"`
		Memory    string  `json:"memory"`
		GPUCount  *int    `json:"gpu_count"`
		GPUModel  *string `json:"gpu_model"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentWriteErrorf("invalid args: %w", err)
	}
	if args.Name == "" {
		return nil, agentWriteErrorf("name is required")
	}
	if args.ImageLink == "" {
		return nil, agentWriteErrorf("image_link is required")
	}
	if args.CPU == "" {
		args.CPU = "2"
	}
	if args.Memory == "" {
		args.Memory = "8Gi"
	}

	if mgr.jobSubmitter == nil {
		return nil, agentWriteErrorf("job submitter is not configured")
	}
	forwards, err := parseForwardArgs(argsMap)
	if err != nil {
		return nil, err
	}

	resourceMap := map[string]string{
		"cpu":    args.CPU,
		"memory": args.Memory,
	}
	if args.GPUCount != nil && *args.GPUCount > 0 {
		gpuResourceName := normalizeGPUResourceName("", "gpu")
		if args.GPUModel != nil && strings.TrimSpace(*args.GPUModel) != "" {
			gpuResourceName = normalizeGPUResourceName(gpuResourceName, *args.GPUModel)
		}
		resourceMap[string(gpuResourceName)] = strconv.Itoa(*args.GPUCount)
	}

	requestBody, err := json.Marshal(map[string]any{
		"name":     args.Name,
		"resource": resourceMap,
		"image": map[string]any{
			"imageLink": args.ImageLink,
			"archs":     []string{},
		},
		"forwards": forwards,
	})
	if err != nil {
		return nil, agentWriteErrorf("failed to marshal jupyter request: %w", err)
	}

	result, err := mgr.jobSubmitter.SubmitJupyterJob(c, token, requestBody)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}

func (mgr *AgentMgr) toolCreateWebIDEJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	argsMap := parseToolArgsMap(rawArgs)
	var args struct {
		Name      string  `json:"name"`
		ImageLink string  `json:"image_link"`
		CPU       string  `json:"cpu"`
		Memory    string  `json:"memory"`
		GPUCount  *int    `json:"gpu_count"`
		GPUModel  *string `json:"gpu_model"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentWriteErrorf("invalid args: %w", err)
	}
	if strings.TrimSpace(args.Name) == "" {
		return nil, agentWriteErrorf("name is required")
	}
	if strings.TrimSpace(args.ImageLink) == "" {
		return nil, agentWriteErrorf("image_link is required")
	}
	if strings.TrimSpace(args.CPU) == "" {
		args.CPU = "2"
	}
	if strings.TrimSpace(args.Memory) == "" {
		args.Memory = "8Gi"
	}
	if mgr.jobSubmitter == nil {
		return nil, agentWriteErrorf("job submitter is not configured")
	}
	forwards, err := parseForwardArgs(argsMap)
	if err != nil {
		return nil, err
	}

	resourceMap := map[string]string{
		"cpu":    args.CPU,
		"memory": args.Memory,
	}
	if args.GPUCount != nil && *args.GPUCount > 0 {
		gpuResourceName := normalizeGPUResourceName("", "gpu")
		if args.GPUModel != nil && strings.TrimSpace(*args.GPUModel) != "" {
			gpuResourceName = normalizeGPUResourceName(gpuResourceName, *args.GPUModel)
		}
		resourceMap[string(gpuResourceName)] = strconv.Itoa(*args.GPUCount)
	}

	requestBody, err := json.Marshal(map[string]any{
		"name":     args.Name,
		"resource": resourceMap,
		"image": map[string]any{
			"imageLink": args.ImageLink,
			"archs":     []string{},
		},
		"forwards": forwards,
	})
	if err != nil {
		return nil, agentWriteErrorf("failed to marshal webide request: %w", err)
	}

	result, err := mgr.jobSubmitter.SubmitWebIDEJob(c, token, requestBody)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}

//nolint:gocyclo // Custom-job creation validates many optional form fields before delegating to vcjob submitter.
func (mgr *AgentMgr) toolCreateCustomJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	argsMap := parseToolArgsMap(rawArgs)
	var args struct {
		Name       string  `json:"name"`
		ImageLink  string  `json:"image_link"`
		Command    string  `json:"command"`
		WorkingDir string  `json:"working_dir"`
		CPU        string  `json:"cpu"`
		Memory     string  `json:"memory"`
		GPUCount   *int    `json:"gpu_count"`
		GPUModel   *string `json:"gpu_model"`
		Shell      string  `json:"shell"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, agentWriteErrorf("invalid args: %w", err)
	}
	if strings.TrimSpace(args.Name) == "" {
		return nil, agentWriteErrorf("name is required")
	}
	if strings.TrimSpace(args.ImageLink) == "" {
		return nil, agentWriteErrorf("image_link is required")
	}
	if strings.TrimSpace(args.Command) == "" {
		return nil, agentWriteErrorf("command is required")
	}
	if strings.TrimSpace(args.WorkingDir) == "" {
		return nil, agentWriteErrorf("working_dir is required")
	}
	if strings.TrimSpace(args.CPU) == "" {
		args.CPU = "4"
	}
	if strings.TrimSpace(args.Memory) == "" {
		args.Memory = "16Gi"
	}
	if strings.TrimSpace(args.Shell) == "" {
		args.Shell = "bash"
	}

	resourceMap := map[string]string{
		"cpu":    args.CPU,
		"memory": args.Memory,
	}
	if args.GPUCount != nil && *args.GPUCount > 0 {
		gpuResourceName := normalizeGPUResourceName("", "gpu")
		if args.GPUModel != nil && strings.TrimSpace(*args.GPUModel) != "" {
			gpuResourceName = normalizeGPUResourceName("", *args.GPUModel)
		}
		resourceMap[string(gpuResourceName)] = strconv.Itoa(*args.GPUCount)
	}

	if mgr.jobSubmitter == nil {
		return nil, agentWriteErrorf("job submitter is not configured")
	}
	forwards, err := parseForwardArgs(argsMap)
	if err != nil {
		return nil, err
	}

	requestBody, err := json.Marshal(map[string]any{
		"name":       args.Name,
		"resource":   resourceMap,
		"workingDir": args.WorkingDir,
		"command":    args.Command,
		"shell":      args.Shell,
		"image": map[string]any{
			"imageLink": args.ImageLink,
			"archs":     []string{},
		},
		"forwards": forwards,
	})
	if err != nil {
		return nil, agentWriteErrorf("failed to marshal custom job request: %w", err)
	}

	result, err := mgr.jobSubmitter.SubmitTrainingJob(c, token, requestBody)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}

func (mgr *AgentMgr) toolCreatePytorchJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	return mgr.toolCreateDistributedJob(c, token, rawArgs, agentToolCreatePytorch)
}

func (mgr *AgentMgr) toolCreateTensorflowJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	return mgr.toolCreateDistributedJob(c, token, rawArgs, agentToolCreateTensorflow)
}

func (mgr *AgentMgr) toolCreateDistributedJob(
	c *gin.Context,
	token util.JWTMessage,
	rawArgs json.RawMessage,
	toolName string,
) (any, error) {
	argsMap := parseToolArgsMap(rawArgs)
	name := getToolArgString(argsMap, "name", "")
	if strings.TrimSpace(name) == "" {
		return nil, agentWriteErrorf("name is required")
	}
	tasks, err := parseDistributedTasks(argsMap)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, agentWriteErrorf("tasks or tasks_json is required")
	}
	forwards, err := parseForwardArgs(argsMap)
	if err != nil {
		return nil, err
	}
	if mgr.jobSubmitter == nil {
		return nil, agentWriteErrorf("job submitter is not configured")
	}

	requestBody, err := json.Marshal(map[string]any{
		"name":     name,
		"tasks":    tasks,
		"forwards": forwards,
	})
	if err != nil {
		return nil, agentWriteErrorf("failed to marshal %s request: %w", toolName, err)
	}

	var result any
	switch toolName {
	case agentToolCreatePytorch:
		result, err = mgr.jobSubmitter.SubmitPytorchJob(c, token, requestBody)
	case agentToolCreateTensorflow:
		result, err = mgr.jobSubmitter.SubmitTensorflowJob(c, token, requestBody)
	default:
		return nil, agentWriteErrorf("distributed job tool %s is not supported", toolName)
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}
