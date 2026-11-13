// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package handler

import (
	"testing"
	"time"
)

func TestAgentLLMRateWindowPreservesActiveRequests(t *testing.T) {
	t.Parallel()
	start := time.Now()
	state := &agentLLMRateState{windowStarted: start}
	for range maxConcurrentLLMRequests {
		if !state.acquire(start) {
			t.Fatal("initial requests should be accepted")
		}
	}
	if state.acquire(start.Add(time.Minute)) || state.active != maxConcurrentLLMRequests || state.requests != 0 {
		t.Fatalf("window rollover lost active requests: %+v", state)
	}
	// One old request finishes; only one slot may be reused in the new window.
	state.active--
	if !state.acquire(start.Add(time.Minute)) || state.acquire(start.Add(time.Minute)) {
		t.Fatalf("concurrent request limit was not preserved: %+v", state)
	}
}

func TestAgentLLMRateWindowResetsOnlyRequestCount(t *testing.T) {
	t.Parallel()
	start := time.Now()
	state := &agentLLMRateState{windowStarted: start, requests: maxLLMRequestsPerMinute, active: 1}
	if state.acquire(start.Add(time.Minute - time.Nanosecond)) {
		t.Fatal("minute request limit should apply until the window expires")
	}
	if !state.acquire(start.Add(time.Minute)) || state.requests != 1 || state.active != maxConcurrentLLMRequests {
		t.Fatalf("unexpected state after window rollover: %+v", state)
	}
}
