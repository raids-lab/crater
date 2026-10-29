// Copyright 2026 The Crater Project Team, RAIDS-Lab

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestGetNodePodsReadsCompleteArray(t *testing.T) {
	for _, count := range []int{0, 201} {
		t.Run(fmt.Sprintf("%d pods", count), func(t *testing.T) {
			requests := 0
			client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != http.MethodGet || r.URL.Path != NodeListPath+"/node-a/pods" || r.URL.RawQuery != "" {
					t.Errorf("node Pods must use the complete-list request: %s %s", r.Method, r.URL)
				}
				items := make([]PodInfo, count)
				for i := range items {
					items[i] = PodInfo{Name: fmt.Sprintf("pod-%03d", i), Namespace: "test"}
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(Response[[]PodInfo]{Data: items}); err != nil {
					t.Errorf("encode Pods: %v", err)
				}
			})
			items, err := client.GetNodePods("node-a")
			if err != nil || items == nil || len(items) != count || requests != 1 {
				t.Fatalf("incomplete node Pods: count=%d requests=%d error=%v", len(items), requests, err)
			}
			if count > 0 && items[count-1].Name != fmt.Sprintf("pod-%03d", count-1) {
				t.Fatal("last Pod missing")
			}
		})
	}
}

func TestGetNodePodsPreservesReadFailure(t *testing.T) {
	client := jobTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(Response[any]{Code: 50003, Message: "list node pods failed"})
	})
	items, err := client.GetNodePods("node-a")
	var requestError *RequestError
	if items != nil || !errors.As(err, &requestError) || requestError.HTTPStatus != 500 || requestError.CraterCode != 50003 || requestError.Msg != "list node pods failed" {
		t.Fatalf("lost node Pod failure: items=%v error=%v", items, err)
	}
}
