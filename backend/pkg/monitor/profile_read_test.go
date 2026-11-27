package monitor

import (
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/prometheus/prometheus/prompb"
)

func newProfileReadServer(t *testing.T, series []*prompb.TimeSeries) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, _, err := readProfileTestRequest(t, r)
		if err != nil {
			t.Error(err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		var result []*prompb.TimeSeries
		for _, stream := range series {
			if !matchesProfileRequest(stream, query.Matchers) {
				continue
			}
			selection := &prompb.TimeSeries{Labels: stream.Labels}
			for _, sample := range stream.Samples {
				if sample.Timestamp >= query.StartTimestampMs && sample.Timestamp <= query.EndTimestampMs {
					selection.Samples = append(selection.Samples, sample)
				}
			}
			if len(selection.Samples) != 0 {
				result = append(result, selection)
			}
		}
		writeProfileReadResponse(t, w, result)
	}))
	t.Cleanup(server.Close)
	return server
}

func matchesProfileRequest(stream *prompb.TimeSeries, matchers []*prompb.LabelMatcher) bool {
	labels := make(map[string]string, len(stream.Labels))
	for _, label := range stream.Labels {
		labels[label.Name] = label.Value
	}
	for _, matcher := range matchers {
		value := labels[matcher.Name]
		switch matcher.Type {
		case prompb.LabelMatcher_EQ:
			if value != matcher.Value {
				return false
			}
		case prompb.LabelMatcher_NEQ:
			if value == matcher.Value {
				return false
			}
		case prompb.LabelMatcher_RE:
			if !regexp.MustCompile("^(?:" + matcher.Value + ")$").MatchString(value) {
				return false
			}
		default:
			panic(matcher.Type)
		}
	}
	return true
}

func profileSeries(metric, pod, label, labelValue string, points ...float64) *prompb.TimeSeries {
	stream := memorySeries(points...)
	stream.Labels = []prompb.Label{
		{Name: "__name__", Value: metric}, {Name: "namespace", Value: "workspace"}, {Name: "pod", Value: pod},
		{Name: label, Value: labelValue},
	}
	return stream
}

func assertSummary(t *testing.T, got, want *metricSummary) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
		return
	}
	if math.Abs(got.avg-want.avg) > 1e-9 || math.Abs(got.max-want.max) > 1e-9 || math.Abs(got.std-want.std) > 1e-9 {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
