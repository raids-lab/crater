package monitor

import (
	"context"
	"fmt"
	"strings"

	"github.com/prometheus/prometheus/prompb"
)

const gpuUtilMetric = "DCGM_FI_DEV_GPU_UTIL"

type gpuProfileMetric struct {
	metric string
	fields profileFields
	scale  float64
}

func gpuProfileMetrics(p *ProfileData) []gpuProfileMetric {
	return []gpuProfileMetric{
		{gpuUtilMetric, profileFields{&p.GPUUtilAvg, &p.GPUUtilMax, &p.GPUUtilStd}, 0.01},
		{"DCGM_FI_DEV_MEM_COPY_UTIL", profileFields{&p.MemCopyUtilAvg, &p.MemCopyUtilMax, &p.MemCopyUtilStd}, 0.01},
		{"DCGM_FI_PROF_PCIE_TX_BYTES", profileFields{&p.PCIETxAvg, &p.PCIETxMax, nil}, 1.0 / bytesPerMiB},
		{"DCGM_FI_PROF_PCIE_RX_BYTES", profileFields{&p.PCIERxAvg, &p.PCIERxMax, nil}, 1.0 / bytesPerMiB},
		{"DCGM_FI_PROF_SM_ACTIVE", profileFields{&p.SMActiveAvg, &p.SMActiveMax, &p.SMActiveStd}, 1},
		{"DCGM_FI_PROF_SM_OCCUPANCY", profileFields{&p.SMOccupancyAvg, &p.SMOccupancyMax, &p.SMOccupancyStd}, 1},
		{"DCGM_FI_PROF_DRAM_ACTIVE", profileFields{&p.DramUtilAvg, &p.DramUtilMax, &p.DramUtilStd}, 1},
		{"DCGM_FI_PROF_DRAM_ACTIVE", profileFields{&p.DramActiveAvg, &p.DramActiveMax, &p.DramActiveStd}, 1},
		{"DCGM_FI_PROF_PIPE_TENSOR_ACTIVE", profileFields{&p.TensorActiveAvg, &p.TensorActiveMax, &p.TensorActiveStd}, 1},
		{"DCGM_FI_PROF_PIPE_FP64_ACTIVE", profileFields{&p.Fp64ActiveAvg, &p.Fp64ActiveMax, &p.Fp64ActiveStd}, 1},
		{"DCGM_FI_PROF_PIPE_FP32_ACTIVE", profileFields{&p.Fp32ActiveAvg, &p.Fp32ActiveMax, &p.Fp32ActiveStd}, 1},
		{"DCGM_FI_PROF_PIPE_FP16_ACTIVE", profileFields{&p.Fp16ActiveAvg, &p.Fp16ActiveMax, &p.Fp16ActiveStd}, 1},
	}
}

type gpuUtilSamples struct {
	last     map[string]prompb.Sample
	observed metricMoments
	timed    metricMoments
}

func (g *gpuUtilSamples) consume(stream *prompb.TimeSeries) {
	key := profileSeriesKey(stream, false)
	previous, valid := g.last[key]
	for _, sample := range stream.Samples {
		finite := isFinite(sample.Value)
		if finite {
			g.observed.add(sample.Value, 1)
		}
		if valid {
			span := sample.Timestamp - previous.Timestamp
			if span <= memoryLookback.Milliseconds() && finite {
				// Give each endpoint half of the observed interval. Gaps longer
				// than lookback do not establish a duration for either sample.
				g.timed.add(previous.Value, float64(span)/2)
				g.timed.add(sample.Value, float64(span)/2)
			} else if span <= memoryLookback.Milliseconds() {
				// The terminal observation ends the previous sample's lifetime.
				g.timed.add(previous.Value, float64(span))
			}
		}
		previous, valid = sample, finite
	}
	if valid {
		g.last[key] = previous
	} else {
		delete(g.last, key)
	}
}

func (g *gpuUtilSamples) summary() *metricSummary {
	// A single observation has no duration, but still establishes a sample
	// and can record the peak even when other GPUs have measured intervals.
	if summary := g.timed.summary(); summary != nil {
		summary.max = g.observed.peak
		return summary
	}
	return g.observed.summary()
}

func (p *PrometheusClient) queryGPUProfile(ctx context.Context, window profileWindow, profile *ProfileData) error {
	fields := gpuProfileMetrics(profile)
	regular := make(map[string]*metricMoments, len(fields))
	names := []string{gpuUtilMetric}
	for _, field := range fields {
		if field.metric != gpuUtilMetric && regular[field.metric] == nil {
			names = append(names, field.metric)
			regular[field.metric] = &metricMoments{}
		}
	}

	util := gpuUtilSamples{last: make(map[string]prompb.Sample)}
	err := p.readProfileSamples(ctx, window, strings.Join(names, "|"), false, func(series []*prompb.TimeSeries, _ int64) error {
		for _, stream := range series {
			metric := profileMetricName(stream)
			if metric == gpuUtilMetric {
				util.consume(stream)
				continue
			}
			stats := regular[metric]
			if stats == nil {
				return fmt.Errorf("unexpected GPU profile metric %q", metric)
			}
			for _, sample := range stream.Samples {
				if isFinite(sample.Value) {
					stats.add(sample.Value, 1)
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	usage := util.summary()
	for _, field := range fields {
		summary := usage
		if field.metric != gpuUtilMetric {
			summary = regular[field.metric].summary()
		}
		field.fields.store(summary, field.scale)
	}
	return nil
}
