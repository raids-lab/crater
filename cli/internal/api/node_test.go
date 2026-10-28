// Copyright 2026 The Crater Project Team, RAIDS-Lab

package api

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func nodePodPage(next string, names ...string) NodePodsSnapshotPage {
	page := NodePodsSnapshotPage{SnapshotID: "node-a-rv100", ResourceVersion: "100", Items: []PodInfo{}}
	if next != "" {
		page.NextCursor, page.HasMore = &next, true
	}
	for _, name := range names {
		page.Items = append(page.Items, PodInfo{Name: name, Namespace: "test"})
	}
	return page
}

func TestGetNodePodsContinuesWithinSnapshot(t *testing.T) {
	// Continue tokens are opaque; empty intermediate pages are valid Kubernetes responses.
	pages := []NodePodsSnapshotPage{
		nodePodPage("opaque+/=?&", "pod-a"),
		nodePodPage("next-empty-page"),
		nodePodPage("", "pod-b"),
	}
	requests := 0
	client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if requests >= len(pages) {
			t.Fatal("requested a page after completion")
		}
		wantCursor := ""
		if requests > 0 {
			wantCursor = *pages[requests-1].NextCursor
		}
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/nodes/node-a/pods" ||
			r.URL.Query().Get("limit") != "200" || r.URL.Query().Get("cursor") != wantCursor {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		// The continue token carries the snapshot; do not add a new resourceVersion constraint.
		if r.URL.Query().Has("resource_version") {
			t.Fatal("unexpected resource_version on continuation request")
		}
		writeJobTestResponse(t, w, pages[requests])
		requests++
	})
	items, err := client.GetNodePods("node-a")
	want := []PodInfo{{Name: "pod-a", Namespace: "test"}, {Name: "pod-b", Namespace: "test"}}
	if err != nil || !reflect.DeepEqual(items, want) || requests != len(pages) {
		t.Fatalf("items=%v err=%v requests=%d", items, err, requests)
	}
}

func TestGetNodePodsRejectsInconsistentPages(t *testing.T) {
	changed := nodePodPage("", "pod-b")
	changed.SnapshotID = "node-a-rv101"
	missing := nodePodPage("", "pod-b")
	missing.SnapshotID = ""
	missingCursor := nodePodPage("", "pod-b")
	missingCursor.HasMore = true
	unexpectedCursor := nodePodPage("next", "pod-b")
	unexpectedCursor.HasMore = false
	for _, tc := range []struct {
		name  string
		pages []NodePodsSnapshotPage
		want  string
	}{
		{"repeated cursor", []NodePodsSnapshotPage{nodePodPage("a", "pod-a"), nodePodPage("a", "pod-b")}, "repeated pagination cursor"},
		{"cursor cycle", []NodePodsSnapshotPage{nodePodPage("a"), nodePodPage("b"), nodePodPage("a")}, "repeated pagination cursor"},
		{"changed snapshot", []NodePodsSnapshotPage{nodePodPage("a", "pod-a"), changed}, "snapshot ID changed"},
		{"missing snapshot", []NodePodsSnapshotPage{missing}, "missing snapshot ID"},
		{"missing later snapshot", []NodePodsSnapshotPage{nodePodPage("a", "pod-a"), missing}, "missing snapshot ID"},
		{"missing cursor", []NodePodsSnapshotPage{nodePodPage("a", "pod-a"), missingCursor}, "has_more does not match"},
		{"unexpected cursor", []NodePodsSnapshotPage{unexpectedCursor}, "has_more does not match"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			client := jobTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				if requests >= len(tc.pages) {
					t.Fatal("requested a page after invalid response")
				}
				writeJobTestResponse(t, w, tc.pages[requests])
				requests++
			})
			items, err := client.GetNodePods("node-a")
			if err == nil || !strings.Contains(err.Error(), tc.want) || items != nil || requests != len(tc.pages) {
				t.Fatalf("items=%v err=%v requests=%d; want nil, %q, %d", items, err, requests, tc.want, len(tc.pages))
			}
		})
	}
}

func TestGetNodePodsBoundsRequests(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(fmt.Sprintf("completes=%t", complete), func(t *testing.T) {
			requests := 0
			client := jobTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				requests++
				if requests > MaxPageRequests {
					t.Fatal("exceeded maximum page requests")
				}
				next := fmt.Sprintf("cursor-%d", requests)
				if complete && requests == MaxPageRequests {
					next = ""
				}
				writeJobTestResponse(t, w, nodePodPage(next, "pod"))
			})
			items, err := client.GetNodePods("node-a")
			if requests != MaxPageRequests {
				t.Fatalf("requests=%d, want %d", requests, MaxPageRequests)
			}
			if complete {
				if err != nil || len(items) != MaxPageRequests {
					t.Fatalf("items=%d err=%v", len(items), err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "exceeded maximum") || items != nil {
				t.Fatalf("items=%d err=%v; want nil and maximum request error", len(items), err)
			}
		})
	}
}

func TestGetNodePodsRejectsExpiredSnapshotWithoutPartialResults(t *testing.T) {
	requests := 0
	client := jobTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests == 1 {
			writeJobTestResponse(t, w, nodePodPage("expired", "pod-a"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":40906,"msg":"snapshot cursor expired; refresh and retry"}`))
	})
	items, err := client.GetNodePods("node-a")
	if err == nil || !strings.Contains(err.Error(), "snapshot cursor expired") || items != nil || requests != 2 {
		t.Fatalf("items=%v err=%v requests=%d", items, err, requests)
	}
	var requestError *RequestError
	if !errors.As(err, &requestError) || requestError.HTTPStatus != http.StatusConflict || requestError.CraterCode != 40906 {
		t.Fatalf("expired snapshot error lost HTTP status or business code: %v", err)
	}
}
