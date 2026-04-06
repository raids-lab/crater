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
		return nil, fmt.Errorf("failed to marshal admin ops patrol request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+adminOpsReportPipelinePath,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin ops patrol request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Agent-Internal-Token", internalToken)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call admin ops patrol pipeline: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read admin ops patrol response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf(
			"admin ops patrol pipeline returned status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	payload := make(map[string]any)
	if len(responseBody) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode admin ops patrol response: %w", err)
	}
	return payload, nil
}
