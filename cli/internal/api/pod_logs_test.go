package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"testing"
)

func TestGetPodContainers(t *testing.T) {
	client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/namespaces/crater-workspace/pods/job-pod/containers" {
			t.Errorf("path = %s", r.URL.Path)
		}
		writeJobTestResponse(t, w, map[string]interface{}{
			"containers": []map[string]interface{}{
				{"name": "trainer", "image": "example/train:latest", "isInitContainer": false},
			},
		})
	})

	containers, err := client.GetPodContainers("crater-workspace", "job-pod")
	if err != nil {
		t.Fatalf("GetPodContainers: %v", err)
	}
	if len(containers) != 1 || containers[0].Name != "trainer" || containers[0].IsInitContainer {
		t.Fatalf("containers = %#v", containers)
	}
}

func TestGetPodLogsDecodesBackendBase64(t *testing.T) {
	want := []byte("第一行\nsecond line\n")
	client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/namespaces/crater-workspace/pods/job-pod/containers/trainer/log" {
			t.Errorf("path = %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("tailLines") != "20" ||
			query.Get("timestamps") != "true" ||
			query.Get("previous") != "true" {
			t.Errorf("query = %v", query)
		}
		writeJobTestResponse(t, w, base64.StdEncoding.EncodeToString(want))
	})

	got, err := client.GetPodLogs(
		"crater-workspace",
		"job-pod",
		"trainer",
		PodLogOptions{TailLines: 20, Timestamps: true, Previous: true},
	)
	if err != nil {
		t.Fatalf("GetPodLogs: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("logs = %q, want %q", got, want)
	}
}

func TestGetPodLogsRejectsInvalidBase64(t *testing.T) {
	client := jobTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJobTestResponse(t, w, "not base64!")
	})

	_, err := client.GetPodLogs("ns", "pod", "container", PodLogOptions{})
	var protocolErr *PodLogProtocolError
	if !errors.As(err, &protocolErr) {
		t.Fatalf("error = %v, want PodLogProtocolError", err)
	}
}

func TestStreamPodLogsDecodesLines(t *testing.T) {
	first := base64.StdEncoding.EncodeToString([]byte("one\n"))
	second := base64.StdEncoding.EncodeToString([]byte("第二行\n"))
	client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/namespaces/ns/pods/pod/containers/trainer/log/stream" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("tailLines") != "5" || r.URL.Query().Get("timestamps") != "true" {
			t.Errorf("query = %v", r.URL.Query())
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte(first + "\n" + second + "\n"))
	})

	var got bytes.Buffer
	err := client.StreamPodLogs(
		context.Background(),
		&got,
		"ns",
		"pod",
		"trainer",
		PodLogOptions{TailLines: 5, Timestamps: true},
	)
	if err != nil {
		t.Fatalf("StreamPodLogs: %v", err)
	}
	if got.String() != "one\n第二行\n" {
		t.Fatalf("logs = %q", got.String())
	}
}
