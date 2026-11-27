package monitor

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/value"
	"github.com/prometheus/prometheus/prompb"
)

func memorySeries(points ...float64) *prompb.TimeSeries {
	s := &prompb.TimeSeries{}
	for i := 0; i < len(points); i += 2 {
		s.Samples = append(s.Samples, prompb.Sample{Timestamp: int64(points[i]), Value: points[i+1]})
	}
	return s
}

func TestSummarizeMemory(t *testing.T) {
	stale := math.Float64frombits(value.StaleNaN)
	for _, tt := range []struct {
		name          string
		series        []*prompb.TimeSeries
		avg, max, std float64
		absent        bool
	}{
		{
			name:   "single container retains original sample statistics",
			series: []*prompb.TimeSeries{memorySeries(0, 10, 10000, 20, 20000, 30)},
			avg:    20, max: 30, std: math.Sqrt(200.0 / 3),
		},
		{
			name: "simultaneous updates do not add staggered peaks",
			series: []*prompb.TimeSeries{
				memorySeries(0, 10, 15000, 0, 30000, 10),
				memorySeries(0, 0, 15000, 10, 30000, 0),
			},
			avg: 10, max: 10,
		},
		{
			name:   "subsecond peak between thirty-second boundaries survives",
			series: []*prompb.TimeSeries{memorySeries(0, 0, 15100, 900, 15200, 0, 30000, 0)},
			avg:    225, max: 900, std: 225 * math.Sqrt(3),
		},
		{
			name:   "asynchronous containers use their latest observed values",
			series: []*prompb.TimeSeries{memorySeries(0, 10, 20000, 20), memorySeries(10000, 5, 30000, 10)},
			avg:    20, max: 30, std: math.Sqrt(62.5),
		},
		{
			name:   "stale container stops contributing before workers finish",
			series: []*prompb.TimeSeries{memorySeries(0, 10, 10000, stale), memorySeries(0, 20, 10000, 20, 20000, 0)},
			avg:    50.0 / 3, max: 30, std: math.Sqrt(1400.0 / 9),
		},
		{
			name: "long gaps expire at Prometheus lookback and later samples resume",
			series: []*prompb.TimeSeries{
				memorySeries(0, 10, 601000, 10),
				memorySeries(0, 1, 300000, 1, 300001, 1, 600000, 1),
			},
			avg: 7, max: 11, std: math.Sqrt(24),
		},
		{
			name: "zero memory is valid", series: []*prompb.TimeSeries{memorySeries(0, 0, 1000, 0)},
		},
		{
			name: "one scrape still provides a summary", series: []*prompb.TimeSeries{memorySeries(200, 7)},
			avg: 7, max: 7,
		},
		{name: "no samples are absent", absent: true},
		{name: "only stale markers are absent", series: []*prompb.TimeSeries{memorySeries(0, stale)}, absent: true},
		{name: "non-finite samples are absent", series: []*prompb.TimeSeries{memorySeries(0, math.NaN(), 1, math.Inf(1), 2, math.Inf(-1))}, absent: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for i, stream := range tt.series {
				stream.Labels = []prompb.Label{{Name: "id", Value: strconv.Itoa(i)}}
			}
			timeline := memoryTimeline{devices: make(map[string]*memoryDevice)}
			if err := timeline.consume(tt.series, 1000000); err != nil {
				t.Fatal(err)
			}
			got, _ := timeline.summary()

			if tt.absent {
				if got != nil {
					t.Fatalf("expected no summary, got %+v", got)
				}
				return
			}
			if got == nil || math.Abs(got.avg-tt.avg) > 1e-9 || math.Abs(got.max-tt.max) > 1e-9 ||
				math.Abs(got.std-tt.std) > 1e-9 {
				t.Fatalf("got %+v, want avg=%g max=%g std=%g", got, tt.avg, tt.max, tt.std)
			}
		})
	}
}

func TestMemorySummaryFixedWindow(t *testing.T) {
	server := newProfileReadServer(t, []*prompb.TimeSeries{
		profileSeries(cpuMemoryMetric, "master-0", "container", "work", 0, 50, 1000, 10, 2000, 20, 3000, 100),
	})
	p := NewPrometheusClient(server.URL).(*PrometheusClient)
	got, _, err := p.queryMemorySummary(t.Context(), newProfileWindow("workspace", "master-0", time.UnixMilli(1000), time.UnixMilli(2000)), false)
	if err != nil {
		t.Fatal(err)
	}
	assertSummary(t, got, &metricSummary{avg: 15, max: 20, std: 5})
}

func TestGPUMemoryPairsOnlySameDeviceAndTimestamp(t *testing.T) {
	const boundary = 21601000
	stale := math.Float64frombits(value.StaleNaN)
	for _, tc := range []struct {
		name              string
		series            []*prompb.TimeSeries
		maxUsed, capacity float64
	}{
		{
			name: "agrees across all cards and pods, with carry across chunk boundary",
			series: []*prompb.TimeSeries{
				profileSeries(gpuFBUsed, "master-0", "gpu", "0", boundary-1, 20),
				profileSeries(gpuFBFree, "master-0", "gpu", "0", boundary-1, 80),
				profileSeries(gpuFBUsed, "worker-0", "gpu", "0", boundary, 30),
				profileSeries(gpuFBFree, "worker-0", "gpu", "0", boundary, 70),
				profileSeries(gpuFBUsed, "worker-0", "gpu", "1", boundary, 40),
				profileSeries(gpuFBFree, "worker-0", "gpu", "1", boundary, 60),
			},
			maxUsed: 90, capacity: 300,
		},
		{
			name: "shifted samples cannot mix to exceed device capacity",
			series: []*prompb.TimeSeries{
				profileSeries(gpuFBUsed, "master-0", "gpu", "0", 1000, 10, 2000, 80),
				profileSeries(gpuFBFree, "master-0", "gpu", "0", 1000, 90, 3000, 20),
			},
			maxUsed: 80, capacity: 100,
		},
		{
			name: "stale free samples remove that device from pair coverage",
			series: []*prompb.TimeSeries{
				profileSeries(gpuFBUsed, "master-0", "gpu", "0", 1000, 10, 2000, 90),
				profileSeries(gpuFBFree, "master-0", "gpu", "0", 1000, 90, 2000, stale),
				profileSeries(gpuFBUsed, "worker-0", "gpu", "0", 2000, 90),
				profileSeries(gpuFBFree, "worker-0", "gpu", "0", 2000, 10),
			},
			maxUsed: 180,
		},
		{
			name: "stale used and a different GPU's free do not form capacity",
			series: []*prompb.TimeSeries{
				profileSeries(gpuFBUsed, "master-0", "gpu", "0", 1000, 60, 2000, stale),
				profileSeries(gpuFBFree, "master-0", "gpu", "0", 2000, 20),
				profileSeries(gpuFBUsed, "master-0", "gpu", "1", 2000, 30),
			},
			maxUsed: 60,
		},
		{
			name:    "used without free reports usage only",
			series:  []*prompb.TimeSeries{profileSeries(gpuFBUsed, "master-0", "gpu", "0", 1000, 20)},
			maxUsed: 20,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newProfileReadServer(t, tc.series)
			p := NewPrometheusClient(s.URL).(*PrometheusClient)
			w := newProfileWindow("workspace", "master-0|worker-0", time.UnixMilli(1000), time.UnixMilli(boundary+1))
			used, capacity, err := p.queryMemorySummary(t.Context(), w, true)
			if err != nil {
				t.Fatal(err)
			}
			if used == nil || used.max != tc.maxUsed {
				t.Fatalf("used = %+v, want peak %g", used, tc.maxUsed)
			}
			if tc.capacity == 0 && capacity != nil || tc.capacity != 0 && (capacity == nil || capacity.max != tc.capacity) {
				t.Fatalf("capacity = %+v, want %g (0 means absent)", capacity, tc.capacity)
			}
		})
	}
}
