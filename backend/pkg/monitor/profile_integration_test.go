package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

func writeProfileReadResponse(t *testing.T, w http.ResponseWriter, series []*prompb.TimeSeries) {
	t.Helper()
	response := prompb.ReadResponse{Results: []*prompb.QueryResult{{Timeseries: series}}}
	body, err := response.Marshal()
	if err != nil {
		t.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("Content-Encoding", "snappy")
	if _, err := w.Write(snappy.Encode(nil, body)); err != nil {
		t.Error(err)
	}
}

func readProfileTestRequest(t *testing.T, r *http.Request) (*prompb.Query, map[string]string, error) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, err
	}
	body, err = snappy.Decode(nil, body)
	if err != nil {
		return nil, nil, err
	}
	var request prompb.ReadRequest
	if err := request.Unmarshal(body); err != nil {
		return nil, nil, err
	}
	if len(request.Queries) != 1 {
		return nil, nil, fmt.Errorf("expected one query, got %d", len(request.Queries))
	}
	q := request.Queries[0]
	matchers := make(map[string]string)
	for _, matcher := range q.Matchers {
		if matcher.Type != prompb.LabelMatcher_NEQ {
			matchers[matcher.Name] = matcher.Value
		}
	}
	return q, matchers, nil
}

func writeProfileVector(t *testing.T, w http.ResponseWriter, values ...string) {
	t.Helper()
	result := make([]map[string]any, 0, len(values))
	for i, v := range values {
		result = append(result, map[string]any{"metric": map[string]string{"instance": strconv.Itoa(i)}, "value": []any{0, v}})
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status": "success", "data": map[string]any{"resultType": "vector", "result": result},
	}); err != nil {
		t.Error(err)
	}
}

func TestProfileFixedBoundsAndChunkContinuity(t *testing.T) {
	from := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC).Add(123 * time.Microsecond)
	to := from.Add(12*time.Hour + 123*time.Millisecond)
	window := newProfileWindow("", "", from, to)
	boundary := float64(window.start + profileReadWindow.Milliseconds())
	first := profileSeries(cpuMemoryMetric, "master-0", "container", "c", boundary-1, bytesPerMiB)
	second := profileSeries(cpuMemoryMetric, "worker-0", "container", "c", boundary, 9*bytesPerMiB)
	raw, scalar := atomic.Int32{}, atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == profileReadPath {
			raw.Add(1)
			checkChunkAndRespond(t, w, r, window, first, second)
			return
		}
		scalar.Add(1)
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		at, err := strconv.ParseFloat(r.Form.Get("time"), 64)
		if err != nil || math.Abs(at-float64(to.UnixNano())/1e9) > 1e-6 {
			t.Errorf("instant query did not use completion time: %s", r.Form.Get("time"))
		}
		query := r.Form.Get("query")
		if !strings.Contains(query, `pod=~"master-0|worker-0"`) || !strings.Contains(query, "43200122ms") {
			t.Errorf("query lost pods or duration: %s", query)
		}
		writeProfileVector(t, w, "3")
	}))
	defer server.Close()
	profile, err := NewPrometheusClient(server.URL).QueryProfileDataContext(t.Context(), "workspace", []string{"master-0", "worker-0"}, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if profile == nil || profile.Version != ProfileDataVersion {
		t.Fatalf("expected a versioned all-Pod profile, got %+v", profile)
	}
	assertProfileFields(t, [3]*float32{profile.CPUMemAvg, profile.CPUMemMax, profile.CPUMemStd},
		&metricSummary{avg: 5.5, max: 10, std: 4.5}, 1)
	if raw.Load() != 12 || scalar.Load() != 4 {
		t.Fatalf("expected four independent remote reads across three chunks and four scalar queries, got raw=%d scalar=%d",
			raw.Load(), scalar.Load())
	}
}

func checkChunkAndRespond(t *testing.T, w http.ResponseWriter, r *http.Request, window profileWindow, streams ...*prompb.TimeSeries) {
	t.Helper()
	q, matchers, err := readProfileTestRequest(t, r)
	if err != nil {
		t.Error(err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if q.StartTimestampMs < window.start || (q.StartTimestampMs-window.start)%profileReadWindow.Milliseconds() != 0 ||
		q.EndTimestampMs != min(q.StartTimestampMs+profileReadWindow.Milliseconds()-1, window.end) {
		t.Errorf("raw read lost bounded inclusive chunks: %+v", q)
	}
	if matchers["pod"] != "master-0|worker-0" || matchers["namespace"] != "workspace" {
		t.Errorf("raw read lost job selection: %v", matchers)
	}
	var result []*prompb.TimeSeries
	if matchers["__name__"] == cpuMemoryMetric {
		for _, stream := range streams {
			at := stream.Samples[0].Timestamp
			if at >= q.StartTimestampMs && at <= q.EndTimestampMs {
				result = append(result, stream)
			}
		}
	}
	writeProfileReadResponse(t, w, result)
}

func TestProfileReadFailureDiscardsSnapshot(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			var fail atomic.Bool
			fail.Store(true)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != profileReadPath {
					writeProfileVector(t, w, "4")
					return
				}
				_, matchers, err := readProfileTestRequest(t, r)
				if err != nil {
					t.Error(err)
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}
				if fail.Load() && matchers["__name__"] == cpuMemoryMetric {
					http.Error(w, "temporarily unavailable", code)
					return
				}
				writeProfileReadResponse(t, w, nil)
			}))
			defer server.Close()
			client := NewPrometheusClient(server.URL)
			from, to := time.Unix(1, 0), time.Unix(30, 0)
			got, err := client.QueryProfileDataContext(t.Context(), "workspace", []string{"pod"}, from, to)
			if err == nil || got != nil || !strings.Contains(err.Error(), strconv.Itoa(code)) {
				t.Fatalf("expected a retryable error and no partial profile, got %+v, %v", got, err)
			}
			fail.Store(false)
			got, err = client.QueryProfileDataContext(t.Context(), "workspace", []string{"pod"}, from, to)
			if err != nil || got == nil || got.CPURequest == nil || *got.CPURequest != 4 || got.CPUMemAvg != nil {
				t.Fatalf("reachable empty usage should preserve valid resource data, got %+v, %v", got, err)
			}
		})
	}
}

func TestProfileInstantFailureDiscardsSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == profileReadPath {
			writeProfileReadResponse(t, w, nil)
			return
		}
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	data, err := NewPrometheusClient(server.URL).QueryProfileDataContext(t.Context(), "workspace", []string{"pod"}, time.Unix(1, 0), time.Unix(30, 0))
	if err == nil || data != nil {
		t.Fatalf("expected failed instant request to invalidate collection: %+v, %v", data, err)
	}
}

func TestProfileQueriesUseBoundedConcurrency(t *testing.T) {
	var active, seen, peak atomic.Int32
	var once sync.Once
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); old < current && !peak.CompareAndSwap(old, current); old = peak.Load() {
		}
		if seen.Add(1) == profileQueryConcurrency {
			once.Do(func() { close(started) })
		}
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Error("independent queries did not run concurrently")
		}
		if r.URL.Path == profileReadPath {
			writeProfileReadResponse(t, w, nil)
		} else {
			writeProfileVector(t, w)
		}
	}))
	defer server.Close()
	got, err := NewPrometheusClient(server.URL).QueryProfileDataContext(t.Context(), "workspace", []string{"pod"}, time.Unix(1, 0), time.Unix(30, 0))
	if err != nil || got != nil || peak.Load() != profileQueryConcurrency || seen.Load() != 8 {
		t.Fatalf("expected all 8 empty queries in bounded concurrent batches; got %+v, %v, peak=%d, queries=%d",
			got, err, peak.Load(), seen.Load())
	}
}

func TestProfileFiniteMetricExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == profileReadPath {
			writeProfileReadResponse(t, w, nil)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.Form.Get("query") == "vector" {
			writeProfileVector(t, w, "NaN", "+Inf", "-Inf", "1e39", "2")
			return
		}
		writeProfileVector(t, w, "NaN", "+Inf", "-Inf", "1e80")
	}))
	defer server.Close()
	client := NewPrometheusClient(server.URL).(*PrometheusClient)
	if got, err := client.queryMetricAt("vector", time.Now()); err != nil || got != 2 {
		t.Fatalf("finite instant scalar missing: %v, %v", got, err)
	}
	if got, err := client.float32MapQuery("vector", "instance"); err != nil || len(got) != 1 || got["4"] != 2 {
		t.Fatalf("non-finite samples leaked into map: %v, %v", got, err)
	}
	got, err := client.QueryProfileDataContext(t.Context(), "workspace", []string{"pod"}, time.Unix(1, 0), time.Unix(30, 0))
	if err != nil || got != nil {
		t.Fatalf("non-finite or float32 overflow produced resource fields: %+v, %v", got, err)
	}
}

func TestProfileInvalidIdentifiersAndLifetimeDoNotQuery(t *testing.T) {
	client := &PrometheusClient{}
	now := time.Now()
	if profile, err := client.QueryProfileDataContext(t.Context(), "workspace", nil, now.Add(-time.Minute), now); profile != nil || err != nil {
		t.Fatalf("missing pods generated profile: %+v, %v", profile, err)
	}
	for _, bounds := range [][2]time.Time{{{}, now}, {now, {}}, {now, now}, {now, now.Add(-time.Minute)}} {
		if profile, err := client.QueryProfileDataContext(t.Context(), "workspace", []string{"pod"}, bounds[0], bounds[1]); profile != nil || err != nil {
			t.Fatalf("invalid lifetime generated profile: %+v, %v", profile, err)
		}
	}
}

func TestProfileWindowsShrinkOnlyOnPrometheusSampleLimit(t *testing.T) {
	from, to := time.UnixMilli(1000), time.UnixMilli(1005)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q, _, err := readProfileTestRequest(t, r)
		if err != nil {
			t.Error(err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if q.EndTimestampMs-q.StartTimestampMs >= 2 {
			http.Error(w, "exceeded sample limit (2)", http.StatusBadRequest)
			return
		}
		stream := &prompb.TimeSeries{}
		for at := q.StartTimestampMs; at <= q.EndTimestampMs; at++ {
			stream.Samples = append(stream.Samples, prompb.Sample{Timestamp: at})
		}
		writeProfileReadResponse(t, w, []*prompb.TimeSeries{stream})
	}))
	defer server.Close()
	p := NewPrometheusClient(server.URL).(*PrometheusClient)
	var last = from.UnixMilli() - 1
	err := p.readProfileSamples(t.Context(), newProfileWindow("workspace", "pod", from, to), "counter", true,
		func(series []*prompb.TimeSeries, stop int64) error {
			for _, sample := range series[0].Samples {
				if sample.Timestamp != last+1 {
					t.Errorf("unexpected sample after %d: %d", last, sample.Timestamp)
				}
				last = sample.Timestamp
			}
			if last != stop {
				t.Errorf("chunk ended at %d, last sample at %d", stop, last)
			}
			return nil
		})
	if err != nil || last != to.UnixMilli() {
		t.Fatalf("adaptive windows left samples unread: last=%d, err=%v", last, err)
	}
}
