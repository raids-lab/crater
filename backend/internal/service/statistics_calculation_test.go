package service

import (
	"fmt"
	"math"
	"testing"
	"time"

	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/monitor"
)

func number(v float32) *float32 { return &v }
func testJob(name string, start, end time.Time) model.Job {
	p := datatypes.NewJSONType(&monitor.ProfileData{
		CPUUsageAvg: number(1), CPUMemAvg: number(2048), GPUUtilAvg: number(.25)})
	return model.Job{Name: name, JobName: name, UserID: 1, AccountID: 1, JobType: model.JobTypeCustom,
		Status: batch.Completed, CreationTimestamp: start, RunningTimestamp: start, CompletedTimestamp: end,
		Resources: datatypes.NewJSONType(corev1.ResourceList{"cpu": resource.MustParse("2"),
			"memory": resource.MustParse("4Gi"), "vendor.example/accelerator": resource.MustParse("4"),
			"looks-like-gpu": resource.MustParse("8")}), ProfileData: &p}
}
func testResources() map[string]*model.Resource {
	kind := model.ResourceTypeGPU
	return map[string]*model.Resource{"vendor.example/accelerator": {ResourceName: "vendor.example/accelerator", Type: &kind}}
}
func closeTo(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("got %v, want %v", got, want)
	}
}
func TestDailyCalculation(t *testing.T) {
	day := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	job := testJob("cross-day", day.Add(-time.Hour), day.Add(time.Hour))
	a := dayAccumulator{day: day, cutoff: day.Add(24 * time.Hour), resources: testResources(), rows: make(map[usageKey]*model.ResourceUsageDaily)}
	a.add(&job)
	for name, want := range map[string][3]float64{"cpu": {7200, 3600, 7200}, "memory": {14400, 7200, 14400},
		"vendor.example/accelerator": {14400, 3600, 14400}, "looks-like-gpu": {28800, 0, 0}} {
		row := a.rows[usageKey{1, 1, name}]
		closeTo(t, row.UsageSeconds, want[0])
		closeTo(t, row.EffectiveUsageSeconds, want[1])
		closeTo(t, row.ProfiledUsageSeconds, want[2])
	}
	for _, kind := range []string{"running", "tensorboard", "never-started"} {
		t.Run(kind, func(t *testing.T) {
			j := job
			switch kind {
			case "running":
				j.CompletedTimestamp = time.Time{}
			case "tensorboard":
				j.JobType = model.JobTypeTensorboard
			case "never-started":
				j.RunningTimestamp = time.Time{}
			}
			a := dayAccumulator{day: day, cutoff: day.Add(time.Hour), resources: testResources(), rows: make(map[usageKey]*model.ResourceUsageDaily)}
			a.add(&j)
			if kind == "running" {
				closeTo(t, a.rows[usageKey{1, 1, "cpu"}].UsageSeconds, 7200)
				closeTo(t, a.rows[usageKey{1, 1, "cpu"}].ProfiledUsageSeconds, 0)
			} else if len(a.rows) != 0 {
				t.Fatal("auxiliary or unstarted job counted")
			}
		})
	}
}
func TestStoredProfileValues(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		value *float32
	}{{"missing", nil}, {"zero", number(0)}, {"negative", number(-1)}, {"above_allocation", number(10)}} {
		t.Run(tc.name, func(t *testing.T) {
			job := testJob("stored", start, start.Add(time.Hour))
			job.ProfileData.Data().CPUUsageAvg = tc.value
			got, ok := evaluatedAmount(&job, "cpu", ResourceAmounts(job.Resources.Data()), testResources())
			if ok != (tc.value != nil) {
				t.Fatalf("available=%v, value=%v", ok, tc.value)
			}
			if ok {
				closeTo(t, got, float64(*tc.value))
			}
		})
	}
	for _, profile := range []*datatypes.JSONType[*monitor.ProfileData]{nil, new(datatypes.JSONType[*monitor.ProfileData])} {
		job := testJob("missing-profile", start, start.Add(time.Hour))
		job.ProfileData = profile
		if _, ok := evaluatedAmount(&job, "cpu", ResourceAmounts(job.Resources.Data()), testResources()); ok {
			t.Fatal("missing profile counted as a measured zero")
		}
	}
}

func TestStatisticsIncludesAllProfileVersions(t *testing.T) {
	for _, version := range []int{0, monitor.ProfileDataVersion} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			day := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
			job := testJob("profile", day, day.Add(time.Hour))
			job.ProfileData.Data().Version = version
			a := dayAccumulator{day: day, cutoff: day.AddDate(0, 0, 1), resources: testResources(), rows: make(map[usageKey]*model.ResourceUsageDaily)}
			a.add(&job)
			for name, want := range map[string][3]float64{"cpu": {7200, 3600, .5}, "memory": {14400, 7200, .5}, "vendor.example/accelerator": {14400, 3600, .25}} {
				row := a.rows[usageKey{1, 1, name}]
				closeTo(t, row.UsageSeconds, want[0])
				closeTo(t, row.ProfiledUsageSeconds, want[0])
				closeTo(t, row.EffectiveUsageSeconds, want[1])
				detail := resourceDetail(row, name, a.resources[name], false)
				if detail.Utilization == nil {
					t.Fatalf("%s has no utilization", name)
				}
				closeTo(t, *detail.Utilization, want[2])
			}
		})
	}
}

func TestStoredMeansIgnoreTemplateAndDuration(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, template := range []*batch.Job{nil, {Spec: batch.JobSpec{Tasks: []batch.TaskSpec{{
		Replicas: 3, Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main"}, {Name: "sidecar"}},
		}},
	}}}}} {
		job := testJob("historical", start, start.Add(55*24*time.Hour))
		job.Attributes = datatypes.NewJSONType(template)
		must(t, job.ProfileData.UnmarshalJSON([]byte(`{"cpu_usage_avg":1,"cpu_mem_avg":2048,"gpu_util_avg":0.25}`)))
		amounts := ResourceAmounts(job.Resources.Data())
		for name, want := range map[string]float64{"cpu": 1, "memory": 2, "vendor.example/accelerator": 1} {
			got, ok := evaluatedAmount(&job, name, amounts, testResources())
			if !ok {
				t.Fatalf("stored %s mean was rejected", name)
			}
			closeTo(t, got, want)
		}
	}
}

func TestGPUStoredMeanUsesResourceDictionary(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	job := testJob("multiple-gpu-types", start, start.Add(time.Hour))
	job.ProfileData.Data().GPUUtilAvg = number(1.5)
	amounts, metadata := ResourceAmounts(job.Resources.Data()), testResources()
	if _, ok := evaluatedAmount(&job, "looks-like-gpu", amounts, metadata); ok {
		t.Fatal("resource name was used to guess GPU type")
	}
	kind := model.ResourceTypeGPU
	metadata["looks-like-gpu"] = &model.Resource{Type: &kind}
	for name, want := range map[string]float64{"vendor.example/accelerator": 6, "looks-like-gpu": 12} {
		got, ok := evaluatedAmount(&job, name, amounts, metadata)
		if !ok {
			t.Fatalf("stored GPU mean rejected for %s", name)
		}
		closeTo(t, got, want)
	}
}
