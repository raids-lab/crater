// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"fmt"

	"github.com/raids-lab/crater/pkg/constants"
)

const (
	llmConfigScopePersonal = "personal"
	llmConfigScopePlatform = "platform"
)

type llmConfigAuditEvent struct {
	Target  string
	Status  string
	Message string
	Details map[string]any
}

// Accept only operation metadata, never configurations or provider errors that may echo credentials.
func newLLMConfigAuditEvent(opType, scope string, userID uint, keySubmitted, success bool) llmConfigAuditEvent {
	event := llmConfigAuditEvent{
		Target: "platform-llm-config",
		Status: constants.OpStatusSuccess,
		Details: map[string]any{
			"scope": scope,
		},
	}
	if scope == llmConfigScopePersonal {
		event.Target = fmt.Sprintf("personal-llm-config:user:%d", userID)
		event.Details["userId"] = userID
	}
	if opType == constants.OpTypeUpdateLLMConfig {
		event.Details["keySubmitted"] = keySubmitted
	}
	if !success {
		event.Status = constants.OpStatusFailed
		event.Message = "LLM configuration change failed; no credentials are recorded"
	}
	return event
}

func hasSubmittedLLMKey(apiKey, maskedPlaceholder string) bool {
	return apiKey != "" && apiKey != maskedPlaceholder
}
