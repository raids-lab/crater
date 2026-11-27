package monitor

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	profileQueryConcurrency = 4
	profileQueryTimeout     = 3 * time.Minute
)

func (p *PrometheusClient) QueryProfileDataContext(
	ctx context.Context, namespace string, podNames []string, from, to time.Time,
) (*ProfileData, error) {
	ctx, cancel := context.WithTimeout(ctx, profileQueryTimeout)
	defer cancel()

	if len(podNames) == 0 || from.IsZero() || to.IsZero() || to.Sub(from) < time.Millisecond {
		return nil, nil
	}

	window := newProfileWindow(namespace, profilePodPattern(podNames), from, to)
	promRange := fmt.Sprintf("%dms", max(1, window.end-window.start))
	profile := &ProfileData{}
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(profileQueryConcurrency)
	for _, resource := range []struct {
		metric, name string
		scale        float64
		field        **float32
	}{
		{"kube_pod_container_resource_requests", "cpu", 1, &profile.CPURequest},
		{"kube_pod_container_resource_limits", "cpu", 1, &profile.CPULimit},
		{"kube_pod_container_resource_requests", "memory", 1.0 / bytesPerMiB, &profile.MemRequest},
		{"kube_pod_container_resource_limits", "memory", 1.0 / bytesPerMiB, &profile.MemLimit},
	} {
		group.Go(func() error {
			expression := profileResourceQuery(resource.metric, resource.name, namespace, window.pods, promRange)
			vector, err := p.queryVectorAt(ctx, expression, to)
			if err != nil {
				return fmt.Errorf("profile %s %s: %w", resource.metric, resource.name, err)
			}
			switch len(vector) {
			case 0:
			case 1:
				*resource.field = finiteFloat32(float64(vector[0].Value) * resource.scale)
			default:
				return fmt.Errorf("profile %s %s: expected a single aggregate, got %d samples", resource.metric, resource.name, len(vector))
			}
			return nil
		})
	}
	group.Go(func() error {
		cpu, err := p.queryCPUUsage(ctx, window)
		profileFields{&profile.CPUUsageAvg, &profile.CPUUsageMax, &profile.CPUUsageStd}.store(cpu, 1)
		return err
	})
	group.Go(func() error {
		memory, _, err := p.queryMemorySummary(ctx, window, false)
		profileFields{&profile.CPUMemAvg, &profile.CPUMemMax, &profile.CPUMemStd}.store(memory, 1.0/bytesPerMiB)
		return err
	})
	group.Go(func() error {
		used, capacity, err := p.queryMemorySummary(ctx, window, true)
		profileFields{&profile.GPUMemAvg, &profile.GPUMemMax, &profile.GPUMemStd}.store(used, 1)
		if capacity != nil {
			profile.GPUMemTotal = finiteFloat32(capacity.max)
		}
		return err
	})
	group.Go(func() error {
		return p.queryGPUProfile(ctx, window, profile)
	})
	if err := group.Wait(); err != nil {
		return nil, err
	}
	if *profile == (ProfileData{}) {
		return nil, nil
	}
	profile.Version = ProfileDataVersion
	return profile, nil
}
