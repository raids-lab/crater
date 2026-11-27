package monitor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/prompb"
	"golang.org/x/sync/semaphore"
)

const (
	profileReadWindow      = 6 * time.Hour
	profileReadPath        = "/api/v1/read"
	profileReadConcurrency = 4
)

var (
	errProfileSampleLimit = errors.New("remote read sample limit")
	profileReadSlots      = semaphore.NewWeighted(profileReadConcurrency)
)

type profileWindow struct {
	namespace, pods string
	start, end      int64
	seconds         float64
}

func newProfileWindow(namespace, pods string, from, to time.Time) profileWindow {
	// Prometheus timestamps are integer milliseconds. Round inward at both ends.
	start := from.UnixMilli()
	if time.UnixMilli(start).Before(from) {
		start++
	}
	return profileWindow{namespace, pods, start, to.UnixMilli(), to.Sub(from).Seconds()}
}

func (w profileWindow) matchers(metric string, containers bool) []*prompb.LabelMatcher {
	matchers := []*prompb.LabelMatcher{
		{Type: prompb.LabelMatcher_RE, Name: "__name__", Value: metric},
		{Type: prompb.LabelMatcher_EQ, Name: "namespace", Value: w.namespace},
		{Type: prompb.LabelMatcher_RE, Name: "pod", Value: w.pods},
	}
	if containers {
		matchers = append(matchers,
			&prompb.LabelMatcher{Type: prompb.LabelMatcher_NEQ, Name: "container", Value: "POD"},
			&prompb.LabelMatcher{Type: prompb.LabelMatcher_NEQ, Name: "container", Value: ""})
	}
	return matchers
}

// Remote Read preserves original timestamps and stale markers. Consume each
// inclusive slice before fetching the next so response size is bounded by time.
func (p *PrometheusClient) readProfileSamples(
	ctx context.Context, window profileWindow, metric string, containers bool,
	consume func([]*prompb.TimeSeries, int64) error,
) error {
	matchers := window.matchers(metric, containers)
	width := profileReadWindow.Milliseconds()
	for start, end := window.start, window.end; start <= end; {
		stop := min(start+width-1, end)
		series, err := p.readProfileChunk(ctx, &prompb.Query{
			StartTimestampMs: start, EndTimestampMs: stop, Matchers: matchers,
		})
		if errors.Is(err, errProfileSampleLimit) && stop > start {
			width = (stop - start + 1) / 2
			continue
		}
		if err == nil {
			err = consume(series, stop)
		}
		if err != nil {
			return fmt.Errorf("%s samples [%d, %d]: %w", metric, start, stop, err)
		}
		start = stop + 1
	}
	return nil
}

func (p *PrometheusClient) readProfileChunk(ctx context.Context, query *prompb.Query) ([]*prompb.TimeSeries, error) {
	request := prompb.ReadRequest{
		Queries:               []*prompb.Query{query},
		AcceptedResponseTypes: []prompb.ReadRequest_ResponseType{prompb.ReadRequest_SAMPLES},
	}
	body, err := request.Marshal()
	if err != nil {
		return nil, err
	}
	if err := profileReadSlots.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer profileReadSlots.Release(1)
	// Waiting for a process-wide slot uses only the overall profile budget.
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.client.URL(profileReadPath, nil).String(),
		bytes.NewReader(snappy.Encode(nil, body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Read-Version", "0.1.0")
	response, body, err := p.client.Do(ctx, req) //nolint:bodyclose // api.Client.Do reads and closes the response body.
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusBadRequest && bytes.HasPrefix(body, []byte("exceeded sample limit (")) {
		return nil, fmt.Errorf("%w: %s", errProfileSampleLimit, bytes.TrimSpace(body))
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote read: HTTP %d: %s", response.StatusCode, body)
	}
	return decodeProfileRead(body, query)
}

func decodeProfileRead(body []byte, query *prompb.Query) ([]*prompb.TimeSeries, error) {
	body, err := snappy.Decode(nil, body)
	if err != nil {
		return nil, fmt.Errorf("decode remote read: %w", err)
	}
	var result prompb.ReadResponse
	if err := result.Unmarshal(body); err != nil {
		return nil, fmt.Errorf("parse remote read: %w", err)
	}
	if len(result.Results) != 1 {
		return nil, fmt.Errorf("remote read: expected one query result, got %d", len(result.Results))
	}

	series := result.Results[0].Timeseries
	for _, stream := range series {
		var previous int64
		for i, sample := range stream.Samples {
			if sample.Timestamp < query.StartTimestampMs || sample.Timestamp > query.EndTimestampMs ||
				(i > 0 && sample.Timestamp <= previous) {
				return nil, fmt.Errorf("unexpected remote-read sample timestamp %d", sample.Timestamp)
			}
			previous = sample.Timestamp
		}
	}
	return series, nil
}

func profileSeriesKey(series *prompb.TimeSeries, omitMetric bool) string {
	var builder labels.ScratchBuilder
	metric := series.ToLabels(&builder, nil)
	if omitMetric {
		metric = metric.DropMetricName()
	}
	return metric.String()
}

func profileMetricName(series *prompb.TimeSeries) string {
	for _, label := range series.Labels {
		if label.Name == "__name__" {
			return label.Value
		}
	}
	return ""
}
