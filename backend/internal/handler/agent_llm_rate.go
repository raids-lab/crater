// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package handler

import "time"

const (
	maxLLMRequestsPerMinute  = 60
	maxConcurrentLLMRequests = 2
)

type agentLLMRateState struct {
	windowStarted time.Time
	requests      int
	active        int
}

// acquire is called while AgentMgr.llmRateMu is held.
func (state *agentLLMRateState) acquire(now time.Time) bool {
	if now.Sub(state.windowStarted) >= time.Minute {
		state.windowStarted = now
		state.requests = 0
	}
	if state.requests >= maxLLMRequestsPerMinute || state.active >= maxConcurrentLLMRequests {
		return false
	}
	state.requests++
	state.active++
	return true
}
