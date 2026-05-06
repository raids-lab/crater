package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	agentLocalToolCatalogTTL       = 30 * time.Second
	agentLocalToolCatalogTimeout   = 5 * time.Second
	agentLocalWriteDispatchTimeout = 6 * time.Minute
)

func cloneLocalToolCatalog(entries []agentLocalToolCatalogEntry) []agentLocalToolCatalogEntry {
	if len(entries) == 0 {
		return nil
	}
	cloned := make([]agentLocalToolCatalogEntry, len(entries))
	copy(cloned, entries)
	return cloned
}

func (mgr *AgentMgr) getPythonLocalToolCatalog(ctx context.Context) ([]agentLocalToolCatalogEntry, error) {
	if mgr == nil {
		return nil, fmt.Errorf("agent manager is not configured")
	}

	now := time.Now()
	mgr.localToolCatalogMu.RLock()
	if now.Before(mgr.localToolCatalogExpiresAt) {
		cached := cloneLocalToolCatalog(mgr.localToolCatalog)
		mgr.localToolCatalogMu.RUnlock()
		return cached, nil
	}
	mgr.localToolCatalogMu.RUnlock()

	fetched, err := mgr.fetchPythonLocalToolCatalog(ctx)
	mgr.localToolCatalogMu.Lock()
	defer mgr.localToolCatalogMu.Unlock()
	if err != nil {
		mgr.localToolCatalog = nil
		mgr.localToolCatalogExpiresAt = time.Time{}
		return nil, err
	}
	mgr.localToolCatalog = cloneLocalToolCatalog(fetched)
	mgr.localToolCatalogExpiresAt = now.Add(agentLocalToolCatalogTTL)
	return cloneLocalToolCatalog(mgr.localToolCatalog), nil
}

func (mgr *AgentMgr) fetchPythonLocalToolCatalog(ctx context.Context) ([]agentLocalToolCatalogEntry, error) {
	internalToken := mgr.getPythonAgentInternalToken()
	if internalToken == "" {
		return nil, fmt.Errorf("python agent internal token is not configured")
	}
	reqCtx, cancel := context.WithTimeout(ctx, agentLocalToolCatalogTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, mgr.getPythonAgentURL()+"/internal/tools/catalog", http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Agent-Internal-Token", internalToken)

	resp, err := mgr.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("python agent returned status %d for local tool catalog", resp.StatusCode)
	}

	var payload agentLocalToolCatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Tools, nil
}

func (mgr *AgentMgr) executePythonLocalWrite(
	ctx context.Context,
	reqBody agentLocalWriteExecuteRequest,
) (map[string]any, error) {
	internalToken := mgr.getPythonAgentInternalToken()
	if internalToken == "" {
		return nil, fmt.Errorf("python agent internal token is not configured")
	}

	payloadBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, agentLocalWriteDispatchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		reqCtx,
		http.MethodPost,
		mgr.getPythonAgentURL()+"/internal/tools/execute-local-write",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Agent-Internal-Token", internalToken)

	resp, err := mgr.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("python local write returned status %d", resp.StatusCode)
	}

	result := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
