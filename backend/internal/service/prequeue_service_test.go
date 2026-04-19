package service

import (
	"testing"

	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
)

func TestResolveQueueQuotaName(t *testing.T) {
	t.Parallel()

	if got := resolveQueueQuotaName("", model.DefaultAccountID, 7); got != "default" {
		t.Fatalf("expected default queue, got %q", got)
	}

	if got := resolveQueueQuotaName("", 12, 34); got != "q-a12-u34" {
		t.Fatalf("expected user queue, got %q", got)
	}

	if got := resolveQueueQuotaName("custom-queue", 12, 34); got != "custom-queue" {
		t.Fatalf("expected explicit queue name, got %q", got)
	}
}

func TestBuildUserResourceUsageSummary_WithQuota(t *testing.T) {
	t.Parallel()

	summary := buildUserResourceUsageSummary(
		&ResolvedQueueQuota{
			Name:    "q-a1-u2",
			Enabled: true,
			Quota: map[string]string{
				"cpu":                  "20",
				"memory":               "40Gi",
				"nvidia.com/a100":      "50",
				"nvidia.com/v100":      "60",
				"huawei.com/ascend910": "16",
			},
		},
		[]*model.Job{
			newJob("q-a1-u2", batch.Running, map[v1.ResourceName]string{
				v1.ResourceCPU:                     "4",
				v1.ResourceMemory:                  "8Gi",
				v1.ResourceName("nvidia.com/a100"): "6",
				v1.ResourceName("nvidia.com/p100"): "9",
			}),
			newJob("q-a1-u2", batch.Pending, map[v1.ResourceName]string{
				v1.ResourceCPU:                          "6",
				v1.ResourceMemory:                       "12Gi",
				v1.ResourceName("nvidia.com/a100"):      "4",
				v1.ResourceName("huawei.com/ascend910"): "8",
			}),
			newJob("q-a1-u2", model.Prequeue, map[v1.ResourceName]string{
				v1.ResourceCPU:                     "99",
				v1.ResourceName("nvidia.com/a100"): "99",
			}),
			newJob("other-queue", batch.Running, map[v1.ResourceName]string{
				v1.ResourceCPU:                     "2",
				v1.ResourceName("nvidia.com/a100"): "2",
			}),
			newJob("q-a1-u2", batch.Restarting, map[v1.ResourceName]string{
				v1.ResourceMemory:                  "4Gi",
				v1.ResourceName("nvidia.com/v100"): "9",
			}),
		},
	)

	if !summary.QuotaEnabled {
		t.Fatalf("expected quota to be enabled")
	}
	if summary.QueueName != "q-a1-u2" {
		t.Fatalf("expected resolved queue name, got %q", summary.QueueName)
	}
	if summary.OccupiedJobs != 2 {
		t.Fatalf("expected occupied running or pending jobs 2, got %d", summary.OccupiedJobs)
	}

	assertUsageItem(t, summary.Resources["cpu"], "cpu", "10", "20", true)
	assertUsageItem(t, summary.Resources["memory"], "memory", "24Gi", "40Gi", true)
	assertUsageItem(t, summary.Resources["nvidia.com/a100"], "nvidia.com/a100", "10", "50", true)
	assertUsageItem(t, summary.Resources["nvidia.com/v100"], "nvidia.com/v100", "9", "60", true)
	assertUsageItem(t, summary.Resources["nvidia.com/p100"], "nvidia.com/p100", "9", "", false)
	assertUsageItem(
		t,
		summary.Resources["huawei.com/ascend910"],
		"huawei.com/ascend910",
		"8",
		"16",
		true,
	)
}

func TestBuildUserResourceUsageSummary_WithoutQuota(t *testing.T) {
	t.Parallel()

	summary := buildUserResourceUsageSummary(
		&ResolvedQueueQuota{
			Name:    "q-a3-u5",
			Enabled: false,
			Quota: map[string]string{
				"cpu": "20",
			},
		},
		[]*model.Job{
			newJob("q-a3-u5", batch.Running, map[v1.ResourceName]string{
				v1.ResourceCPU:                     "2",
				v1.ResourceMemory:                  "6Gi",
				v1.ResourceName("nvidia.com/a100"): "3",
			}),
		},
	)

	if summary.QuotaEnabled {
		t.Fatalf("expected quota to be disabled")
	}
	if summary.OccupiedJobs != 1 {
		t.Fatalf("expected occupied running jobs 1, got %d", summary.OccupiedJobs)
	}

	assertUsageItem(t, summary.Resources["cpu"], "cpu", "2", "", false)
	assertUsageItem(t, summary.Resources["memory"], "memory", "6Gi", "", false)
	assertUsageItem(t, summary.Resources["nvidia.com/a100"], "nvidia.com/a100", "3", "", false)
}

func TestBuildQueueResourceUsageSummary_WithCapability(t *testing.T) {
	t.Parallel()

	summary := BuildQueueResourceUsageSummary(
		"q-a31",
		v1.ResourceList{
			v1.ResourceCPU:                     apiresource.MustParse("4"),
			v1.ResourceMemory:                  apiresource.MustParse("8Gi"),
			v1.ResourceName("nvidia.com/a100"): apiresource.MustParse("2"),
			v1.ResourcePods:                    apiresource.MustParse("3"),
			v1.ResourceName("attachable-volumes-csi.example.com"): apiresource.MustParse("1"),
		},
		v1.ResourceList{
			v1.ResourceCPU:                     apiresource.MustParse("10"),
			v1.ResourceMemory:                  apiresource.MustParse("16Gi"),
			v1.ResourceName("nvidia.com/a100"): apiresource.MustParse("4"),
			v1.ResourcePods:                    apiresource.MustParse("10"),
		},
		2,
	)

	if !summary.QuotaEnabled {
		t.Fatalf("expected quota to be enabled")
	}
	if summary.QueueName != "q-a31" {
		t.Fatalf("expected queue name q-a31, got %q", summary.QueueName)
	}
	if summary.OccupiedJobs != 2 {
		t.Fatalf("expected occupied running jobs 2, got %d", summary.OccupiedJobs)
	}

	assertUsageItem(t, summary.Resources["cpu"], "cpu", "4", "10", true)
	assertUsageItem(t, summary.Resources["memory"], "memory", "8Gi", "16Gi", true)
	assertUsageItem(t, summary.Resources["nvidia.com/a100"], "nvidia.com/a100", "2", "4", true)

	if _, exists := summary.Resources["pods"]; exists {
		t.Fatalf("expected pods resource to be filtered out")
	}
	if _, exists := summary.Resources["attachable-volumes-csi.example.com"]; exists {
		t.Fatalf("expected attachable volumes resource to be filtered out")
	}
}

func TestBuildQueueResourceUsageSummary_WithoutCapability(t *testing.T) {
	t.Parallel()

	summary := BuildQueueResourceUsageSummary(
		"q-a32",
		v1.ResourceList{
			v1.ResourceCPU:                     apiresource.MustParse("4"),
			v1.ResourceMemory:                  apiresource.MustParse("4Gi"),
			v1.ResourceName("nvidia.com/p100"): apiresource.MustParse("1"),
			v1.ResourcePods:                    apiresource.MustParse("1"),
		},
		nil,
		1,
	)

	if summary.QuotaEnabled {
		t.Fatalf("expected quota to be disabled")
	}

	assertUsageItem(t, summary.Resources["cpu"], "cpu", "4", "", false)
	assertUsageItem(t, summary.Resources["memory"], "memory", "4Gi", "", false)
	assertUsageItem(t, summary.Resources["nvidia.com/p100"], "nvidia.com/p100", "1", "", false)

	if _, exists := summary.Resources["pods"]; exists {
		t.Fatalf("expected pods resource to be filtered out")
	}
}

func newJob(
	queue string,
	status batch.JobPhase,
	resources map[v1.ResourceName]string,
) *model.Job {
	resourceList := make(v1.ResourceList, len(resources))
	for name, quantity := range resources {
		resourceList[name] = apiresource.MustParse(quantity)
	}

	return &model.Job{
		Queue:     queue,
		Status:    status,
		Resources: datatypes.NewJSONType(resourceList),
	}
}

func assertUsageItem(
	t *testing.T,
	item UserResourceUsageSummaryItem,
	resource string,
	used string,
	limit string,
	hasLimit bool,
) {
	t.Helper()

	if item.Resource != resource {
		t.Fatalf("expected resource %q, got %q", resource, item.Resource)
	}
	if item.Used != used {
		t.Fatalf("expected used %q for %s, got %q", used, resource, item.Used)
	}
	if item.Limit != limit {
		t.Fatalf("expected limit %q for %s, got %q", limit, resource, item.Limit)
	}
	if item.HasLimit != hasLimit {
		t.Fatalf("expected hasLimit=%t for %s, got %t", hasLimit, resource, item.HasLimit)
	}
}
