package monitor

import (
	"math"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/value"
	"github.com/prometheus/prometheus/prompb"
)

func TestCPUUsageFromRawSamples(t *testing.T) {
	stale := math.Float64frombits(value.StaleNaN)
	for _, tc := range []struct {
		name     string
		from, to float64
		streams  []*prompb.TimeSeries
		want     *metricSummary
	}{
		{
			name: "all pods and containers add cores; infrastructure and other pods are excluded",
			from: 1000, to: 21000,
			streams: []*prompb.TimeSeries{
				cpuSeries("master-0", "main", 1000, 0, 21000, 20),
				cpuSeries("worker-0", "main", 1000, 0, 21000, 40),
				cpuSeries("worker-0", "sidecar", 1000, 0, 21000, 10),
				cpuSeries("worker-0", "POD", 1000, 0, 21000, 500),
				cpuSeries("worker-0", "", 1000, 0, 21000, 500),
				cpuSeries("worker-0-extra", "main", 1000, 0, 21000, 500),
			},
			want: &metricSummary{avg: 3.5, max: 3.5},
		},
		{
			name: "short workload peak has no 30 second grid requirement",
			from: 1000, to: 21000,
			streams: []*prompb.TimeSeries{cpuSeries("master-0", "main", 1000, 0, 11000, 10, 21000, 40)},
			want:    &metricSummary{avg: 2, max: 3, std: 1},
		},
		{
			name: "job-end peak after a grid point is included",
			from: 1000, to: 56000,
			streams: []*prompb.TimeSeries{cpuSeries("master-0", "main", 1000, 0, 51000, 50, 56000, 100, 57000, 9000)},
			want:    &metricSummary{avg: 100.0 / 55, max: 10, std: 9 * math.Sqrt(50*5) / 55},
		},
		{
			name: "one in-window sample does not use a counter from before the job",
			from: 11001, to: 21000,
			streams: []*prompb.TimeSeries{cpuSeries("master-0", "main", 1000, 0, 11000, 100, 21000, 110)},
		},
		{
			name: "early exit with stale marker does not extrapolate",
			from: 1000, to: 121000,
			streams: []*prompb.TimeSeries{
				cpuSeries("master-0", "main", 1000, 0, 61000, 60, 91000, stale),
				cpuSeries("worker-0", "main", 1000, 0, 61000, 120, 121000, 240),
			},
			want: &metricSummary{avg: 2.5, max: 3, std: 0.5},
		},
		{
			name: "early exit without stale marker does not hold a rate",
			from: 1000, to: 121000,
			streams: []*prompb.TimeSeries{
				cpuSeries("master-0", "main", 1000, 0, 61000, 60),
				cpuSeries("worker-0", "main", 1000, 0, 61000, 120, 121000, 240),
			},
			want: &metricSummary{avg: 2.5, max: 3, std: 0.5},
		},
		{
			name: "reset and late-starting container contribute over measured intervals",
			from: 1000, to: 41000,
			streams: []*prompb.TimeSeries{
				cpuSeries("master-0", "main", 1000, 90, 21000, 10, 41000, 30),
				cpuSeries("worker-0", "main", 11000, 0, 31000, 20),
			},
			want: &metricSummary{avg: 1.25, max: 2, std: math.Sqrt(0.3125)},
		},
		{
			name: "stale marker prevents bridging a restarted stream",
			from: 1000, to: 31000,
			streams: []*prompb.TimeSeries{cpuSeries("master-0", "main", 1000, 5, 11000, stale, 21000, 100, 31000, 110)},
			want:    &metricSummary{avg: 1.0 / 3, max: 1},
		},
		{
			name: "counter intervals span remote-read window boundaries",
			from: 1000, to: 3611000,
			streams: []*prompb.TimeSeries{cpuSeries("master-0", "main", 3591000, 100, 3611000, 160)},
			want:    &metricSummary{avg: 60.0 / 3610, max: 3},
		},
		{
			name: "idle CPU stays zero",
			from: 1000, to: 21000,
			streams: []*prompb.TimeSeries{cpuSeries("master-0", "main", 1000, 20, 21000, 20)},
			want:    &metricSummary{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := newProfileReadServer(t, tc.streams)
			p := NewPrometheusClient(server.URL).(*PrometheusClient)
			w := newProfileWindow("workspace", "master-0|worker-0", time.UnixMilli(int64(tc.from)), time.UnixMilli(int64(tc.to)))
			got, err := p.queryCPUUsage(t.Context(), w)
			if err != nil {
				t.Fatal(err)
			}
			assertSummary(t, got, tc.want)
		})
	}
}

func cpuSeries(pod, container string, points ...float64) *prompb.TimeSeries {
	return profileSeries("container_cpu_usage_seconds_total", pod, "container", container, points...)
}
