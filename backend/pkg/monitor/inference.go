package monitor

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	prommodel "github.com/prometheus/common/model"
)

// Missing series and empty histograms remain null, not misleading zeros.
type InferenceMetrics struct {
	RequestsPerSecond *float64  `json:"requestsPerSecond"`
	TokensPerSecond   *float64  `json:"tokensPerSecond"`
	TTFTP95Seconds    *float64  `json:"ttftP95Seconds"`
	LatencyP95Seconds *float64  `json:"latencyP95Seconds"`
	WaitingRequests   *float64  `json:"waitingRequests"`
	RunningRequests   *float64  `json:"runningRequests"`
	HTTPErrorRatio    *float64  `json:"httpErrorRatio"`
	ObservedAt        time.Time `json:"observedAt"`
	WindowSeconds     int       `json:"windowSeconds"`
}

func inferenceQueries(namespace, engine, scope string, pods []string) map[string]string {
	names := make([]string, len(pods))
	for i, name := range pods {
		names[i] = regexp.QuoteMeta(name)
	}
	selector := fmt.Sprintf("namespace=%s,pod=~%s", strconv.Quote(namespace), strconv.Quote(strings.Join(names, "|")))
	rate := func(metric string) string { return "sum(rate(" + metric + "{" + selector + "}[5m]))" }
	percentile := func(metric string) string {
		return "histogram_quantile(0.95, sum by (le) (rate(" + metric + "{" + selector + "}[5m])))"
	}
	httpSelector := selector + `,handler=~"/v1/(chat/completions|completions|embeddings)"`
	httpTotal := "sum(rate(http_requests_total{" + httpSelector + "}[5m]))"
	httpErrors := "sum(rate(http_requests_total{" + httpSelector + `,status=~"[45].."` + "}[5m]))"

	if scope == "router" {
		selector = "namespace=" + strconv.Quote(namespace) + ",model=" + strconv.Quote(pods[0])
		total := rate("kthena_router_requests_total")
		errors := "sum(rate(kthena_router_requests_total{" + selector + `,status_code=~"[45].."` + "}[5m]))"
		return map[string]string{
			"requests": total,
			"latency":  percentile("kthena_router_request_duration_seconds_bucket"),
			"errors":   "(" + errors + " or on() (0 * " + total + ")) / " + total,
		}
	}
	if engine == "SGLang" {
		return map[string]string{
			"requests": rate("sglang:num_requests_total"),
			"tokens":   rate("sglang:generation_tokens_total"),
			"ttft":     percentile("sglang:time_to_first_token_seconds_bucket"),
			"latency":  percentile("sglang:e2e_request_latency_seconds_bucket"),
			"waiting":  "sum(sglang:num_queue_reqs{" + selector + "})",
			"running":  "sum(sglang:num_running_reqs{" + selector + "})",
		}
	}

	return map[string]string{
		"requests": rate("vllm:request_success_total"), "tokens": rate("vllm:generation_tokens_total"),
		"ttft": percentile(
			"vllm:time_to_first_token_seconds_bucket",
		), "latency": percentile("vllm:e2e_request_latency_seconds_bucket"),
		"waiting": "sum(vllm:num_requests_waiting{" + selector + "})", "running": "sum(vllm:num_requests_running{" + selector + "})",
		"errors": "(" + httpErrors + " or on() (0 * " + httpTotal + ")) / " + httpTotal,
	}
}

func (p *PrometheusClient) QueryInferenceMetrics(
	ctx context.Context,
	namespace, engine, scope string,
	pods []string,
) (InferenceMetrics, error) {
	const windowSeconds = 300
	result := InferenceMetrics{ObservedAt: time.Now(), WindowSeconds: windowSeconds}
	if len(pods) == 0 {
		return result, nil
	}
	const metricsTimeout = 8 * time.Second
	ctx, cancel := context.WithTimeout(ctx, metricsTimeout)
	defer cancel()
	fields := map[string]**float64{"requests": &result.RequestsPerSecond, "tokens": &result.TokensPerSecond,
		"ttft": &result.TTFTP95Seconds, "latency": &result.LatencyP95Seconds, "waiting": &result.WaitingRequests,
		"running": &result.RunningRequests, "errors": &result.HTTPErrorRatio}
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(len(fields))
	for key, expression := range inferenceQueries(namespace, engine, scope, pods) {
		group.Go(func() error {
			value, _, err := p.v1api.Query(ctx, expression, result.ObservedAt)
			if err != nil {
				return fmt.Errorf("query inference %s: %w", key, err)
			}
			vector, ok := value.(prommodel.Vector)
			if !ok {
				return fmt.Errorf("unexpected inference metric result")
			}
			if len(vector) == 0 {
				return nil
			}
			sample := float64(vector[0].Value)
			if !math.IsNaN(sample) && !math.IsInf(sample, 0) {
				*fields[key] = &sample
			}
			return nil
		})
	}
	err := group.Wait()
	return result, err
}
