package monitor

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/value"
	"github.com/prometheus/prometheus/prompb"
)

func TestGPUProfilePodSelection(t *testing.T) {
	pods := profilePodPattern([]string{"worker.0", "worker-1"})
	query := fmt.Sprintf(`max(max_over_time(%s{namespace=%q,pod=~%q}[1m]))`, gpuUtilMetric, "workspace", pods)
	testProfileExpressions(t, "1m", []map[string]string{
		{"series": `DCGM_FI_DEV_GPU_UTIL{namespace="workspace",pod="worker.0",gpu="0"}`, "values": "20 20"},
		{"series": `DCGM_FI_DEV_GPU_UTIL{namespace="workspace",pod="worker-1",gpu="1"}`, "values": "40 80"},
		{"series": `DCGM_FI_DEV_GPU_UTIL{namespace="workspace",pod="workerx0",gpu="0"}`, "values": "99 99"},
		{"series": `DCGM_FI_DEV_GPU_UTIL{namespace="other",pod="worker.0",gpu="0"}`, "values": "99 99"},
	}, []map[string]any{
		{"expr": query, "eval_time": "1m", "exp_samples": []map[string]any{{"labels": "{}", "value": 80}}},
	})
}

func TestGPUDurationWeights(t *testing.T) {
	stale := math.Float64frombits(value.StaleNaN)
	for _, tc := range []struct {
		name   string
		series []*prompb.TimeSeries
		want   *metricSummary
	}{
		{
			name: "irregular intervals weight observed durations from both pods",
			series: []*prompb.TimeSeries{
				profileSeries(gpuUtilMetric, "worker.0", "gpu", "0", 1000, 0, 11000, 100, 41000, 100),
				profileSeries(gpuUtilMetric, "worker-1", "gpu", "0", 1000, 0, 41000, 0),
				profileSeries(gpuUtilMetric, "workerx0", "gpu", "0", 1000, 100, 41000, 100),
			},
			want: &metricSummary{avg: 43.75, max: 100, std: 100 * math.Sqrt(35.0*45/6400)},
		},
		{
			name: "stale and long gaps do not weight the unobserved interval",
			series: []*prompb.TimeSeries{
				profileSeries(gpuUtilMetric, "worker.0", "gpu", "0", 1000, 15, 11000, stale, 500000, 80, 530000, 80),
				profileSeries(gpuUtilMetric, "worker.0", "gpu", "1", 1000, 20, 500000, 100),
			},
			want: &metricSummary{avg: 63.75, max: 100, std: 65 * math.Sqrt(3) / 4},
		},
		{
			name: "chunk boundary preserves the interval and a zero is a measurement",
			series: []*prompb.TimeSeries{
				profileSeries(gpuUtilMetric, "worker.0", "gpu", "0", 21591000, 0, 21611000, 0),
			},
			want: &metricSummary{},
		},
		{
			name:   "single observation still supplies statistics",
			series: []*prompb.TimeSeries{profileSeries(gpuUtilMetric, "worker.0", "gpu", "1", 1000, 75)},
			want:   &metricSummary{avg: 75, max: 75},
		},
		{name: "absent GPUs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := newProfileReadServer(t, tc.series)
			client := NewPrometheusClient(server.URL).(*PrometheusClient)
			profile := &ProfileData{}
			err := client.queryGPUProfile(t.Context(),
				newProfileWindow("workspace", profilePodPattern([]string{"coordinator-0", "worker.0", "worker-1"}),
					time.UnixMilli(1000), time.UnixMilli(21611000)), profile)
			if err != nil {
				t.Fatal(err)
			}
			assertProfileFields(t, [3]*float32{profile.GPUUtilAvg, profile.GPUUtilMax, profile.GPUUtilStd}, tc.want, .01)
		})
	}
}

func TestSecondaryGPUMetricsWithoutGPUUtil(t *testing.T) {
	var streams []*prompb.TimeSeries
	for _, field := range gpuProfileMetrics(&ProfileData{}) {
		if field.metric == gpuUtilMetric || field.metric == "DCGM_FI_PROF_DRAM_ACTIVE" {
			continue
		}
		streams = append(streams, secondaryGPUSeries(field.metric, field.scale)...)
	}
	streams = append(streams, secondaryGPUSeries("DCGM_FI_PROF_DRAM_ACTIVE", 1)...)
	server := newProfileReadServer(t, streams)
	client := NewPrometheusClient(server.URL).(*PrometheusClient)
	profile := &ProfileData{}
	err := client.queryGPUProfile(t.Context(), newProfileWindow("workspace", "coordinator-0|worker-0|worker-1", time.UnixMilli(1000), time.UnixMilli(21000)), profile)
	if err != nil {
		t.Fatal(err)
	}
	if fields := [3]*float32{profile.GPUUtilAvg, profile.GPUUtilMax, profile.GPUUtilStd}; fields != [3]*float32{} {
		t.Fatal("empty GPU_UTIL generated fields")
	}
	for _, field := range gpuProfileMetrics(profile) {
		if field.metric == gpuUtilMetric {
			continue
		}
		values := []*float32{*field.fields.avg, *field.fields.max}
		if field.fields.std != nil {
			values = append(values, *field.fields.std)
		}
		t.Run(field.metric, func(t *testing.T) {
			assertProfileValues(t, values, &metricSummary{avg: 2, max: 3, std: math.Sqrt(2.0 / 3)}, 1)
		})
	}
}

func secondaryGPUSeries(metric string, scale float64) []*prompb.TimeSeries {
	return []*prompb.TimeSeries{
		profileSeries(metric, "worker-0", "gpu", "0", 1000, 1/scale, 2000, math.NaN()),
		profileSeries(metric, "worker-0", "gpu", "1", 1000, 2/scale, 2000, math.Inf(1)),
		profileSeries(metric, "worker-1", "gpu", "0", 1000, 3/scale, 2000, math.Inf(-1)),
		profileSeries(metric, "unrelated", "gpu", "0", 1000, 99/scale),
	}
}

func assertProfileFields(t *testing.T, fields [3]*float32, want *metricSummary, scale float64) {
	t.Helper()
	if want == nil {
		if fields != [3]*float32{} {
			t.Fatalf("expected absent summary, got %v", fields)
		}
		return
	}
	assertProfileValues(t, fields[:], want, scale)
}

func assertProfileValues(t *testing.T, fields []*float32, want *metricSummary, scale float64) {
	t.Helper()
	expected := []float64{want.avg, want.max, want.std}
	for i, field := range fields {
		expectedValue := expected[i] * scale
		if field == nil || !isFinite(float64(*field)) || math.Abs(float64(*field)-expectedValue) > 1e-6 {
			t.Fatalf("field %d got %v, want %.6f", i, field, expectedValue)
		}
	}
}
