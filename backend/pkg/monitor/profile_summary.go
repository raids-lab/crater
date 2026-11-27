package monitor

import "math"

type metricSummary struct {
	avg, max, std float64
}

type metricMoments struct {
	weight, mean, m2, peak float64
}

func (m *metricMoments) add(sample, weight float64) {
	if m.weight == 0 {
		m.peak = sample
	}
	m.weight += weight
	delta := sample - m.mean
	m.mean += delta * (weight / m.weight)
	m.m2 += weight * delta * (sample - m.mean)
	m.peak = math.Max(m.peak, sample)
}

func (m *metricMoments) summary() *metricSummary {
	if m.weight == 0 {
		return nil
	}
	return &metricSummary{avg: m.mean, max: m.peak, std: math.Sqrt(max(0, m.m2/m.weight))}
}

type profileFields struct {
	avg, max, std **float32
}

func (f profileFields) store(summary *metricSummary, scale float64) {
	if summary == nil {
		return
	}
	*f.avg = finiteFloat32(summary.avg * scale)
	*f.max = finiteFloat32(summary.max * scale)
	if f.std != nil {
		*f.std = finiteFloat32(summary.std * scale)
	}
}
