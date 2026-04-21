package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/klog/v2"

	pkgconfig "github.com/raids-lab/crater/pkg/config"
)

// ApprovalEvalRequest is sent to the Python agent service.
type ApprovalEvalRequest struct {
	OrderID        int    `json:"order_id"`
	JobName        string `json:"job_name"`
	ExtensionHours int    `json:"extension_hours"`
	UserReason     string `json:"user_reason"`
	UserID         int    `json:"user_id"`
	Username       string `json:"username"`
	JobType        string `json:"job_type"`
}

// ApprovalEvalResponse is returned by the Python agent service.
type ApprovalEvalResponse struct {
	Verdict       string       `json:"verdict"` // "approve" or "escalate"
	Confidence    float64      `json:"confidence"`
	Reason        string       `json:"reason"`
	ApprovedHours *int         `json:"approved_hours"` // agent-adjusted hours, nil = use original
	UserMessage   string       `json:"user_message"`
	AdminSummary  string       `json:"admin_summary"`
	Trace         []TraceEntry `json:"trace"`
}

type TraceEntry struct {
	Tool        string `json:"tool"`
	ArgsSummary string `json:"args_summary"`
}

// AgentApprovalEvaluator calls the crater-agent service to evaluate approval orders.
// It has three-layer protection: rate limiting, concurrency control, and circuit breaker.
// All methods are safe to call concurrently.
type AgentApprovalEvaluator struct {
	client    *http.Client
	agentURL  string
	timeout   time.Duration
	semaphore chan struct{}

	// Rate limiting (simple sliding window counter)
	rateMu       sync.Mutex
	rateCount    int
	rateResetAt  time.Time
	maxPerMinute int

	// Circuit breaker
	consecutiveFailures atomic.Int32
	circuitOpenUntil    atomic.Int64 // unix timestamp
	breakerThreshold    int
	breakerCooldown     time.Duration
}

// NewAgentApprovalEvaluator creates a new evaluator from config.
// Returns nil if the feature is disabled.
func NewAgentApprovalEvaluator() *AgentApprovalEvaluator {
	cfg := pkgconfig.GetConfig()
	if !cfg.Agent.ApprovalHook.Enabled {
		return nil
	}

	agentURL := cfg.Agent.ServiceURL
	if agentURL == "" {
		agentURL = "http://localhost:8000"
	}

	timeout := time.Duration(cfg.Agent.ApprovalHook.TotalTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	maxPerMin := cfg.Agent.ApprovalHook.MaxPerMinute
	if maxPerMin <= 0 {
		maxPerMin = 10
	}

	maxConcurrent := cfg.Agent.ApprovalHook.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}

	threshold := cfg.Agent.ApprovalHook.CircuitBreakerThreshold
	if threshold <= 0 {
		threshold = 5
	}

	cooldown := time.Duration(cfg.Agent.ApprovalHook.CircuitBreakerCooldownSeconds) * time.Second
	if cooldown <= 0 {
		cooldown = 60 * time.Second
	}

	return &AgentApprovalEvaluator{
		client:           &http.Client{Timeout: timeout},
		agentURL:         agentURL,
		timeout:          timeout,
		semaphore:        make(chan struct{}, maxConcurrent),
		maxPerMinute:     maxPerMin,
		rateResetAt:      time.Now().Add(time.Minute),
		breakerThreshold: threshold,
		breakerCooldown:  cooldown,
	}
}

var (
	errRateLimited    = fmt.Errorf("agent approval: rate limited")
	errCircuitOpen    = fmt.Errorf("agent approval: circuit breaker open")
	errConcurrencyMax = fmt.Errorf("agent approval: concurrency limit reached")
)

// Evaluate calls the agent service to evaluate an approval order.
// Returns nil on any infrastructure error (rate limit, circuit breaker, timeout, HTTP error).
// The caller should treat nil as "fallback to manual review".
func (e *AgentApprovalEvaluator) Evaluate(ctx context.Context, req *ApprovalEvalRequest) (*ApprovalEvalResponse, error) {
	// Layer 1: Rate limit
	if !e.allowRate() {
		klog.V(2).Infof("agent approval rate limited for order %d", req.OrderID)
		return nil, errRateLimited
	}

	// Layer 2: Circuit breaker
	if e.isCircuitOpen() {
		klog.V(2).Infof("agent approval circuit open for order %d", req.OrderID)
		return nil, errCircuitOpen
	}

	// Layer 3: Concurrency semaphore (non-blocking)
	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	default:
		klog.V(2).Infof("agent approval concurrency limit for order %d", req.OrderID)
		return nil, errConcurrencyMax
	}

	// Make the HTTP call
	result, err := e.doCall(ctx, req)
	if err != nil {
		e.recordFailure()
		return nil, err
	}

	e.recordSuccess()
	return result, nil
}

func (e *AgentApprovalEvaluator) doCall(ctx context.Context, req *ApprovalEvalRequest) (*ApprovalEvalResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := e.agentURL + "/evaluate/approval"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("agent service call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("agent service busy (429)")
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("agent service returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result ApprovalEvalResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Validate verdict
	if result.Verdict != "approve" && result.Verdict != "approve_emergency" && result.Verdict != "escalate" {
		return nil, fmt.Errorf("invalid verdict: %q", result.Verdict)
	}

	return &result, nil
}

// --- Rate limiting ---

func (e *AgentApprovalEvaluator) allowRate() bool {
	e.rateMu.Lock()
	defer e.rateMu.Unlock()

	now := time.Now()
	if now.After(e.rateResetAt) {
		e.rateCount = 0
		e.rateResetAt = now.Add(time.Minute)
	}

	if e.rateCount >= e.maxPerMinute {
		return false
	}
	e.rateCount++
	return true
}

// --- Circuit breaker ---

func (e *AgentApprovalEvaluator) isCircuitOpen() bool {
	openUntil := e.circuitOpenUntil.Load()
	if openUntil == 0 {
		return false
	}
	return time.Now().Unix() < openUntil
}

func (e *AgentApprovalEvaluator) recordFailure() {
	count := e.consecutiveFailures.Add(1)
	if int(count) >= e.breakerThreshold {
		e.circuitOpenUntil.Store(time.Now().Add(e.breakerCooldown).Unix())
		klog.Warningf("agent approval circuit breaker opened for %v", e.breakerCooldown)
	}
}

func (e *AgentApprovalEvaluator) recordSuccess() {
	e.consecutiveFailures.Store(0)
}
