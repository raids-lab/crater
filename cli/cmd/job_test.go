package cmd

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/spf13/cobra"
)

func TestValidateDistributedRequestRejectsBackfill(t *testing.T) {
	backfill := scheduleBackfill
	req := api.CreateDistributedJobRequest{
		JobCommonRequest: api.JobCommonRequest{
			Name:         "demo",
			ScheduleType: &backfill,
		},
		Tasks: []api.TaskRequest{{
			Name:     "worker",
			Replicas: 1,
			Resource: api.ResourceList{"cpu": "1", "memory": "1Gi"},
			Image:    api.ImageBaseInfo{ImageLink: "example/image:latest"},
		}},
	}
	err := validateDistributedRequest(req)
	if err == nil || !strings.Contains(err.Error(), "backfill") {
		t.Fatalf("error = %v, want backfill validation error", err)
	}
}

func TestValidateInteractiveRequestAcceptsBackendMountDTO(t *testing.T) {
	req := api.CreateInteractiveJobRequest{
		JobCommonRequest: api.JobCommonRequest{
			Name: "demo",
			VolumeMounts: []api.VolumeMount{
				{Type: volumeTypeFile, SubPath: "workspace", MountPath: "/workspace"},
				{Type: volumeTypeDataset, DatasetID: 7, MountPath: "/data"},
			},
			Forwards: []api.Forward{{Name: "metrics", Port: 8080}},
		},
		Resource: api.ResourceList{"cpu": "1", "memory": "1Gi"},
		Image:    api.ImageBaseInfo{ImageLink: "example/image:latest"},
	}
	if err := validateInteractiveRequest(req); err != nil {
		t.Fatalf("validateInteractiveRequest: %v", err)
	}
}

func TestParseForwardFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("forward", nil, "")
	if err := cmd.Flags().Set("forward", "metrics:8080"); err != nil {
		t.Fatal(err)
	}
	forwards, err := parseForwardFlags(cmd)
	if err != nil {
		t.Fatalf("parseForwardFlags: %v", err)
	}
	if len(forwards) != 1 || forwards[0].Name != "metrics" || forwards[0].Port != 8080 {
		t.Fatalf("forwards = %#v", forwards)
	}
}

func TestReadJSONFileRejectsUnknownFields(t *testing.T) {
	path := t.TempDir() + "/request.json"
	if err := os.WriteFile(path, []byte(`{"name":"demo","unknown":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var req api.CreateInteractiveJobRequest
	err := readJSONFile(path, &req)
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("error = %v, want unknown-field error", err)
	}
}

func TestValidMountPath(t *testing.T) {
	for _, path := range []string{"/workspace", "/data/input"} {
		if !validMountPath(path) {
			t.Errorf("validMountPath(%q) = false", path)
		}
	}
	for _, path := range []string{"relative", "/", "/data/../secret", "/data//input"} {
		if validMountPath(path) {
			t.Errorf("validMountPath(%q) = true", path)
		}
	}
}

func TestJobListFilterIssuesAggregate(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("days", 0, "")
	cmd.Flags().StringSlice("status", nil, "")
	cmd.Flags().StringSlice("type", nil, "")
	cmd.Flags().StringSlice("schedule", nil, "")
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().Bool("batch", false, "")
	cmd.Flags().String("from", "", "")
	cmd.Flags().String("to", "", "")
	for name, value := range map[string]string{
		"days":        "-2",
		"status":      "invalid,also-invalid",
		"type":        "invalid",
		"schedule":    "invalid",
		"interactive": "true",
		"batch":       "true",
		"from":        "2026-07-12",
		"to":          "2026-07-11",
	} {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	issues := jobListFilterIssues(cmd)
	if len(issues) != 7 {
		t.Fatalf("issues = %#v, want 7 aggregated issues", issues)
	}
}

func TestNormalizeListFlagValuesTrimsAndDeduplicates(t *testing.T) {
	got := normalizeListFlagValues([]string{" Running ", "Pending", "Running", ""})
	want := []string{"Running", "Pending"}
	if !slices.Equal(got, want) {
		t.Fatalf("normalizeListFlagValues() = %#v, want %#v", got, want)
	}
}

func TestReadListFlagValuesSupportsSingleRepeatedAndCommaSeparatedValues(t *testing.T) {
	for name, test := range map[string]struct {
		args []string
		want []string
	}{
		"single": {
			args: []string{"--status", "Running"},
			want: []string{"Running"},
		},
		"repeated and comma-separated": {
			args: []string{"--status", "Running", "--status", "Pending,Failed"},
			want: []string{"Running", "Pending", "Failed"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().StringSlice("status", nil, "")
			if err := cmd.Flags().Parse(test.args); err != nil {
				t.Fatal(err)
			}
			got := normalizeListFlagValues(readListFlagValues(cmd, "status"))
			if !slices.Equal(got, test.want) {
				t.Fatalf("status values = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestJobListFilterIssuesEnforcesBackendLimits(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("days", 0, "")
	cmd.Flags().String("search", "", "")
	cmd.Flags().StringSlice("status", nil, "")
	cmd.Flags().StringSlice("type", nil, "")
	cmd.Flags().StringSlice("schedule", nil, "")
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().Bool("batch", false, "")
	cmd.Flags().String("from", "", "")
	cmd.Flags().String("to", "", "")

	tooMany := make([]string, 21)
	for index := range tooMany {
		tooMany[index] = jobStatuses[index%len(jobStatuses)]
	}
	if err := cmd.Flags().Set("status", strings.Join(tooMany, ",")); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("search", strings.Repeat("界", 129)); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("schedule", "0"); err != nil {
		t.Fatal(err)
	}

	issues := jobListFilterIssues(cmd)
	if len(issues) != 3 {
		t.Fatalf("issues = %#v, want search, status count, and schedule errors", issues)
	}
}

func TestParseJobListScheduleTypeUsesNamesOnly(t *testing.T) {
	for input, want := range map[string]int{"normal": scheduleNormal, "Backfill": scheduleBackfill} {
		got, ok := parseJobListScheduleType(input)
		if !ok || got != want {
			t.Fatalf("parseJobListScheduleType(%q) = (%d, %t), want (%d, true)", input, got, ok, want)
		}
	}
	for _, input := range []string{"", "0", "1", "invalid"} {
		if _, ok := parseJobListScheduleType(input); ok {
			t.Fatalf("parseJobListScheduleType(%q) unexpectedly succeeded", input)
		}
	}
}
