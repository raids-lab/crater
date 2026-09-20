package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestKthenaProxyForwardsBeforeUpstreamCompletes(t *testing.T) {
	reader, writer := io.Pipe()
	release := make(chan struct{})
	defer close(release)
	defer reader.Close()
	go func() {
		defer writer.Close()
		_, _ = io.WriteString(writer, "data: first\n\n")
		<-release
		_, _ = io.WriteString(writer, "data: second\n\n")
	}()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		forwardKthenaResponse(w, &http.Response{StatusCode: http.StatusOK,
			Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader})
	}))
	defer server.Close()
	client := &http.Client{Timeout: time.Second}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if got := response.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %s", got)
	}
	first := make([]byte, len("data: first\n\n"))
	if _, err := io.ReadFull(response.Body, first); err != nil {
		t.Fatal(err)
	}
	if string(first) != "data: first\n\n" {
		t.Fatalf("first event = %q", first)
	}
	// The second event is still blocked, so this proves incremental delivery.
	_ = reader.Close()
}

func TestKthenaRouterPreservesStatusAndDoesNotForwardCallerCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Authorization"), "caller-secret") || r.Header.Get("Cookie") != "" {
			t.Error("caller credentials leaked to the Kubernetes proxy")
		}
		if !strings.Contains(r.URL.Path, "/services/"+kthenaRouterService+":http/proxy/v1/chat/completions") {
			t.Errorf("unexpected proxy path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":"busy"}`)
	}))
	defer server.Close()
	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	manager := &KthenaMgr{kubeClient: client}
	response, err := manager.openKthenaRouterResponse(t.Context(), http.MethodPost, "v1/chat/completions", []byte(`{}`),
		http.Header{"Authorization": []string{"Bearer caller-secret"}, "Cookie": []string{"session=private"}})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	recorder := httptest.NewRecorder()
	forwardKthenaResponse(recorder, response)
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Content-Type") != "application/problem+json" ||
		recorder.Body.String() != `{"error":"busy"}` {
		t.Fatalf("proxy response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestKthenaResourceAssociationRequiresExactOwnership(t *testing.T) {
	booster := &unstructured.Unstructured{}
	booster.SetName("alice-qwen")
	booster.SetNamespace("jobs")
	booster.SetUID(types.UID("alice-uid"))
	object := &unstructured.Unstructured{}
	object.SetNamespace("jobs")
	for _, name := range []string{"other-qwen-route", "alice-qwen-other", "qwen"} {
		object.SetName(name)
		if isRelatedKthenaObject(object, booster, "qwen") {
			t.Fatalf("unowned resource %q matched", name)
		}
	}
	object.SetOwnerReferences([]metav1.OwnerReference{{Kind: kthenaKindModelBooster, Name: booster.GetName(), UID: booster.GetUID()}})
	if !isRelatedKthenaObject(object, booster, "qwen") {
		t.Fatal("exact owner did not match")
	}
	object.SetNamespace("other")
	if isRelatedKthenaObject(object, booster, "qwen") {
		t.Fatal("cross-namespace resource matched")
	}
}

func TestKthenaProxyModelIsBoundToAuthorizedDeployment(t *testing.T) {
	for _, body := range []string{`{"messages":[]}`, `{"model":"alice-route"}`} {
		raw, err := withDefaultModel([]byte(body), "alice-route")
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["model"] != "alice-route" {
			t.Fatalf("model = %v", payload["model"])
		}
	}
	for _, body := range []string{`null`, `{"model":"bob-private-route"}`} {
		if _, err := withDefaultModel([]byte(body), "alice-route"); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestHealthyKthenaContainerDoesNotFetchLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("healthy container triggered a Kubernetes API request")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	manager := &KthenaMgr{kubeClient: client}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "healthy", Namespace: "jobs"}, Status: corev1.PodStatus{
		Phase:             corev1.PodRunning,
		ContainerStatuses: []corev1.ContainerStatus{{Name: "vllm", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}},
	}}
	if got := manager.diagnosticsFromPod(t.Context(), pod); len(got) != 0 {
		t.Fatalf("healthy diagnostics = %v", got)
	}
}
