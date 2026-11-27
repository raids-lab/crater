package monitor

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

type cpuRateChange struct {
	cores  float64
	active int
}

type cpuCounters struct {
	last    map[string]prompb.Sample
	changes map[int64]cpuRateChange
	used    float64
}

func (c *cpuCounters) consume(series []*prompb.TimeSeries, _ int64) error {
	for _, stream := range series {
		key := profileSeriesKey(stream, false)
		previous, valid := c.last[key]
		for _, sample := range stream.Samples {
			if !isFinite(sample.Value) || sample.Value < 0 {
				valid = false
				continue
			}
			if valid {
				delta := sample.Value - previous.Value
				if delta < 0 {
					delta = sample.Value // counter reset within the observed interval
				}
				seconds := float64(sample.Timestamp-previous.Timestamp) / float64(time.Second/time.Millisecond)
				rate := delta / seconds
				c.addRateChange(previous.Timestamp, rate, 1)
				c.addRateChange(sample.Timestamp, -rate, -1)
				c.used += delta
			}
			previous, valid = sample, true
		}
		if valid {
			c.last[key] = previous
		} else {
			delete(c.last, key)
		}
	}
	return nil
}

func (c *cpuCounters) addRateChange(at int64, rate float64, active int) {
	change := c.changes[at]
	change.cores += rate
	change.active += active
	c.changes[at] = change
}

func (p *PrometheusClient) queryCPUUsage(ctx context.Context, window profileWindow) (*metricSummary, error) {
	cpu := cpuCounters{last: make(map[string]prompb.Sample), changes: make(map[int64]cpuRateChange)}
	err := p.readProfileSamples(ctx, window, "container_cpu_usage_seconds_total", true, cpu.consume)
	if err != nil {
		return nil, err
	}

	// A rate applies only to the interval that the two counter samples bracket.
	// Sweep their actual boundaries so overlapping containers add together even
	// when scraped at different times. No rate is held after the last sample.
	var stats metricMoments
	var cores float64
	var active int
	var previous int64
	for _, at := range slices.Sorted(maps.Keys(cpu.changes)) {
		if active > 0 {
			stats.add(max(0, cores), float64(at-previous))
		}
		change := cpu.changes[at]
		active += change.active
		cores += change.cores
		if active == 0 {
			cores = 0
		}
		previous = at
	}
	summary := stats.summary()
	if summary != nil {
		summary.avg = cpu.used / window.seconds
	}
	return summary, nil
}
