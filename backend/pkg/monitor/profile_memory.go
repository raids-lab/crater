package monitor

import "context"

const (
	bytesPerMiB     = 1024 * 1024
	cpuMemoryMetric = "container_memory_usage_bytes"
	gpuFBFree       = "DCGM_FI_DEV_FB_FREE"
	gpuFBUsed       = "DCGM_FI_DEV_FB_USED"
)

func (p *PrometheusClient) queryMemorySummary(
	ctx context.Context, window profileWindow, gpu bool,
) (used, capacity *metricSummary, err error) {
	metric := cpuMemoryMetric
	if gpu {
		metric = gpuFBUsed + "|" + gpuFBFree
	}
	timeline := memoryTimeline{devices: make(map[string]*memoryDevice), gpu: gpu}
	if err := p.readProfileSamples(ctx, window, metric, !gpu, timeline.consume); err != nil {
		return nil, nil, err
	}
	used, capacity = timeline.summary()
	return used, capacity, nil
}
