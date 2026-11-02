package reconciler

import (
	"encoding/json"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResolveJobRunningTimestamp(t *testing.T) {
	start := time.Date(2026, 10, 7, 10, 14, 27, 0, time.UTC)
	later := start.Add(time.Second)
	states := []v1.ContainerStateTerminated{
		{StartedAt: metav1.NewTime(later)},
		{},
		{StartedAt: metav1.NewTime(start)},
	}
	tests := []struct {
		name               string
		observed, recorded time.Time
		terminated         []v1.ContainerStateTerminated
		want               time.Time
	}{
		{name: "missing Running transition uses earliest container", terminated: states, want: start},
		{name: "Running transition retains precedence", observed: later, recorded: start, terminated: states, want: later},
		{name: "recorded start survives missing Running transition", recorded: later, terminated: states, want: later},
		{name: "recorded start survives pod cleanup", recorded: start, want: start},
		{name: "never started has no fabricated timestamp"},
		{name: "missing container start stays unknown", terminated: []v1.ContainerStateTerminated{{FinishedAt: metav1.NewTime(later)}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveJobRunningTimestamp(tt.observed, tt.recorded, tt.terminated)
			if !got.Equal(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecoverShortJobStartFromPersistedContainerStates(t *testing.T) {
	// These Completed jobs had no Running condition. Persisted container states
	// remain usable after Volcano deletes the pods, including same-second exits.
	fixtures := []struct{ name, started, finished string }{
		{"8556c", "2026-10-07T10:14:27Z", "2026-10-07T10:14:28Z"},
		{"c9f72", "2026-10-07T10:14:31Z", "2026-10-07T10:14:31Z"},
		{"bd25f", "2026-10-07T10:14:35Z", "2026-10-07T10:14:36Z"},
		{"cd102", "2026-10-07T10:14:40Z", "2026-10-07T10:14:40Z"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			stored := `[{"reason":"Completed","exitCode":0,"startedAt":"` + fixture.started + `","finishedAt":"` + fixture.finished + `"}]`
			var states []v1.ContainerStateTerminated
			if err := json.Unmarshal([]byte(stored), &states); err != nil {
				t.Fatal(err)
			}
			got := resolveJobRunningTimestamp(time.Time{}, time.Time{}, states)
			if got.UTC().Format(time.RFC3339) != fixture.started {
				t.Fatalf("got %v, want %s", got, fixture.started)
			}
		})
	}
}
