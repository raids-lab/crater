package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/patrol"
)

const (
	defaultAgentServiceURL            = "http://localhost:8000"
	adminOpsReportPipelinePath        = "/pipeline/admin-ops-report"
	storageAuditPipelinePath          = "/pipeline/storage-audit"
	adminOpsReportRequestTimeout      = 3 * time.Minute
	adminOpsReportInternalTokenEnvKey = "CRATER_AGENT_INTERNAL_TOKEN"
)

type AdminOpsReportService struct {
	httpClient *http.Client
}

func NewAdminOpsReportService() *AdminOpsReportService {
	return &AdminOpsReportService{
		httpClient: &http.Client{Timeout: adminOpsReportRequestTimeout},
	}
}

func (s *AdminOpsReportService) TriggerAdminOpsReport(
	ctx context.Context,
	req patrol.TriggerAdminOpsReportRequest,
) (map[string]any, error) {
	return s.triggerPipeline(ctx, adminOpsReportPipelinePath, req, "admin ops patrol")
}

func (s *AdminOpsReportService) TriggerStorageAudit(
	ctx context.Context,
	req patrol.TriggerStorageAuditRequest,
) (map[string]any, error) {
	return s.triggerPipeline(ctx, storageAuditPipelinePath, req, "storage audit patrol")
}

func (s *AdminOpsReportService) triggerPipeline(
	ctx context.Context,
	pipelinePath string,
	req any,
	label string,
) (map[string]any, error) {
	baseURL := strings.TrimRight(pkgconfig.GetConfig().Agent.ServiceURL, "/")
	if baseURL == "" {
		baseURL = defaultAgentServiceURL
	}

	internalToken := strings.TrimSpace(pkgconfig.GetConfig().Agent.InternalToken)
	if internalToken == "" {
		internalToken = strings.TrimSpace(os.Getenv(adminOpsReportInternalTokenEnvKey))
	}
	if internalToken == "" {
		return nil, fmt.Errorf("python agent internal token is not configured")
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s request: %w", label, err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+pipelinePath,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s request: %w", label, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Agent-Internal-Token", internalToken)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call %s pipeline: %w", label, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s response: %w", label, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf(
			"%s pipeline returned status %d: %s",
			label,
			resp.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	payload := make(map[string]any)
	if len(responseBody) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode %s response: %w", label, err)
	}
	return payload, nil
}
