package handler

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestKthenaCompletionStreamRequiresCompletion(t *testing.T) {
	chunk := "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"
	var output string
	raw,
		err := readKthenaCompletionStream(strings.NewReader(chunk+"data: [DONE]\n\n"),
		func(text string) error { output += text; return nil })
	if err != nil || output != "hello" || !strings.Contains(string(raw), `"content":"hello"`) {
		t.Fatalf("completion %s, %s, %v", output, raw, err)
	}
	if _,
		err = readKthenaCompletionStream(strings.NewReader(chunk),
		func(string) error { return nil }); !errors.Is(err,
		io.ErrUnexpectedEOF) {
		t.Fatalf("truncated stream: %v", err)
	}
	if _,
		err = readKthenaCompletionStream(strings.NewReader(chunk),
		func(string) error { return context.Canceled }); !errors.Is(err,
		context.Canceled) {
		t.Fatalf("canceled stream: %v", err)
	}
}

func TestKthenaRepairResumesCreationAndRejectsForeignOwnership(t *testing.T) {
	req := nativeTestRequest(t)
	serving := nativeTestServing(t, req)
	hash, err := kthenaRequestHash(req)
	if err != nil {
		t.Fatal(err)
	}
	annotations := serving.GetAnnotations()
	annotations[kthenaRequestHashAnnotation] = hash
	serving.SetAnnotations(annotations)
	mgr := &KthenaMgr{client: nativeTestClient(serving), namespace: serving.GetNamespace()}
	if err = mgr.reconcileKthenaNetworking(t.Context(), serving); err != nil {
		t.Fatal(err)
	}
	if err = mgr.reconcileKthenaNetworking(t.Context(), serving); err != nil {
		t.Fatal(err)
	}
	resources, err := mgr.rawKthenaResources(t.Context(), serving)
	if err != nil || len(resources["items"].([]any)) != 3 {
		t.Fatalf("resources=%v, err=%v", resources, err)
	}
	foreign := serving.DeepCopy()
	foreign.SetUID("other-uid")
	if err = mgr.reconcileKthenaNetworking(t.Context(), foreign); err == nil {
		t.Fatal("adopted resources from another serving UID")
	}
}
