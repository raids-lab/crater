package monitor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQueryPodProfileMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if !strings.Contains(r.Form.Get("query"), `namespace="workspace",pod="job-0"`) {
			t.Errorf("query lost pod name or namespace: %s", r.Form.Get("query"))
		}
		writeProfileVector(t, w, "2")
	}))
	defer server.Close()

	data, err := NewPrometheusClient(server.URL).QueryPodProfileMetric("workspace", "job-0")
	if err != nil || data.CPUMemMax != 2 || data.GPUUtilAvg != 2 {
		t.Fatalf("unexpected pod metrics: %+v, %v", data, err)
	}
}

func TestQueryNodeGPUUtilOmitsNonFiniteSamples(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeProfileVector(t, w, "NaN", "20", "-Inf", "+Inf")
	}))
	defer server.Close()

	data := NewPrometheusClient(server.URL).QueryNodeGPUUtil()
	if len(data) != 1 || data[0].Instance != "1" || data[0].Util != 20 {
		t.Fatalf("expected one finite GPU sample, got %+v", data)
	}
}
