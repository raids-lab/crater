package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func TestInferenceMetricsScopeAndMissingSamples(t *testing.T) {
	for _, q := range inferenceQueries("jobs", "vLLM", "server", []string{"qwen-0", "qwen.1"}) {
		if !strings.Contains(q, `namespace="jobs"`) || !strings.Contains(q, `pod=~"qwen-0|qwen\\.1"`) {
			t.Fatalf("unscoped query: %s", q)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer server.Close()
	c, err := api.NewClient(api.Config{Address: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	p := &PrometheusClient{v1api: v1.NewAPI(c)}
	result, err := p.QueryInferenceMetrics(context.Background(), "jobs", "vLLM", "server", []string{"qwen"})
	if err != nil {
		t.Fatal(err)
	}
	if result.RequestsPerSecond != nil || result.HTTPErrorRatio != nil {
		t.Fatalf("missing series became zero: %+v", result)
	}
}

func TestInferenceMetricsEngineAndRouterBoundaries(t *testing.T) {
	sglang := inferenceQueries("jobs", "SGLang", "decode", []string{"decode-entry"})
	for _, q := range sglang {
		if strings.Contains(q, "vllm:") || !strings.Contains(q, `pod=~"decode-entry"`) {
			t.Fatalf("wrong engine/role scope: %s", q)
		}
	}
	if !strings.Contains(sglang["tokens"], "sglang:generation_tokens_total") {
		t.Fatal("missing SGLang tokens")
	}
	router := inferenceQueries("kthena-system", "vLLM", "router", []string{"deployment-route"})
	if len(router) != 3 {
		t.Fatal("router must expose request metrics only")
	}
	for _, q := range router {
		if !strings.Contains(q, `model="deployment-route"`) || strings.Contains(q, "pod=~") {
			t.Fatalf("wrong end-to-end boundary: %s", q)
		}
	}
}
