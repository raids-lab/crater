package monitor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/prompb"
)

func TestProfileSixHourChunks(t *testing.T) {
	for _, tc := range []struct {
		name     string
		duration time.Duration
		count    int
	}{
		{"short job", 2 * time.Hour, 1},
		{"six hour inclusive range", 6*time.Hour - time.Millisecond, 1},
		{"multi day job", 49 * time.Hour, 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			from := time.UnixMilli(1000)
			to := from.Add(tc.duration)
			var bounds [][2]int64
			var mu sync.Mutex
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query, _, err := readProfileTestRequest(t, r)
				if err != nil {
					t.Error(err)
					return
				}
				mu.Lock()
				bounds = append(bounds, [2]int64{query.StartTimestampMs, query.EndTimestampMs})
				mu.Unlock()
				writeProfileReadResponse(t, w, nil)
			}))
			defer server.Close()
			p := NewPrometheusClient(server.URL).(*PrometheusClient)
			err := p.readProfileSamples(t.Context(), newProfileWindow("workspace", "pod", from, to), "metric", false,
				func(_ []*prompb.TimeSeries, _ int64) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(bounds) != tc.count || bounds[0][0] != from.UnixMilli() || bounds[len(bounds)-1][1] != to.UnixMilli() {
				t.Fatalf("unexpected chunk count or bounds: %v", bounds)
			}
			for i, bound := range bounds {
				if bound[1]-bound[0]+1 > (6*time.Hour).Milliseconds() ||
					(i > 0 && bound[0] != bounds[i-1][1]+1) {
					t.Fatalf("chunks exceed six hours or are not contiguous: %v", bounds)
				}
			}
		})
	}
}

func TestProfileSixHourSampleLimitHalving(t *testing.T) {
	var bounds [][2]int64
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, _, err := readProfileTestRequest(t, r)
		if err != nil {
			t.Error(err)
			return
		}
		mu.Lock()
		bounds = append(bounds, [2]int64{query.StartTimestampMs, query.EndTimestampMs})
		mu.Unlock()
		if query.EndTimestampMs-query.StartTimestampMs+1 > (3 * time.Hour).Milliseconds() {
			http.Error(w, "exceeded sample limit (100)", http.StatusBadRequest)
			return
		}
		writeProfileReadResponse(t, w, nil)
	}))
	defer server.Close()
	p := NewPrometheusClient(server.URL).(*PrometheusClient)
	err := p.readProfileSamples(t.Context(), newProfileWindow("workspace", "pod", time.UnixMilli(1), time.UnixMilli(21600000)),
		"metric", false, func(_ []*prompb.TimeSeries, _ int64) error { return nil })
	want := [][2]int64{{1, 21600000}, {1, 10800000}, {10800001, 21600000}}
	mu.Lock()
	defer mu.Unlock()
	if err != nil || !reflect.DeepEqual(bounds, want) {
		t.Fatalf("sample limit did not halve six hours into contiguous three-hour chunks: %v, %v", bounds, err)
	}
}

type profileDeadlineClient struct {
	t        *testing.T
	deadline time.Time
}

func (*profileDeadlineClient) URL(string, map[string]string) *url.URL {
	return &url.URL{Scheme: "http", Host: "prometheus"}
}

func (c *profileDeadlineClient) Do(ctx context.Context, _ *http.Request) (*http.Response, []byte, error) {
	deadline, ok := ctx.Deadline()
	if !ok || deadline.After(c.deadline) {
		c.t.Errorf("profile request deadline %v exceeds %v", deadline, c.deadline)
	}
	<-ctx.Done()
	return nil, nil, ctx.Err()
}

func TestProfileQueryHonorsParentDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	deadline, _ := ctx.Deadline()
	client := &profileDeadlineClient{t: t, deadline: deadline}
	p := &PrometheusClient{client: client, v1api: v1.NewAPI(client)}
	profile, err := p.QueryProfileDataContext(ctx, "workspace", []string{"pod"}, time.Unix(1, 0), time.Unix(30, 0))
	if profile != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("profile query did not stop at parent deadline: %+v, %v", profile, err)
	}
}
